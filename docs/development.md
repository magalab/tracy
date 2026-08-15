# 开发说明

## 边界

`internal/storage/meta` 负责 Workspace（当前数据库仍沿用 `projects` 表名）、User、Workspace Member、Session 和 API Key；`internal/storage/trace` 只负责 Span。HTTP DTO 通过 `internal/trace` domain model 进入存储，CozeLoop 兼容代码应放在 `compat/cozeloop`。

服务使用两个独立 SQLite 文件：`meta.db` 和 `traces.db`。两者不做跨库事务，Project 权限由 API 层和 Service 层保证。Trace writer 同时受队列条数和 `TRACY_QUEUE_BYTES` 在途字节预算限制，默认预算为 512 MiB。反向代理部署时，仅将受信任代理的 IP/CIDR 配置到 `TRACY_TRUSTED_PROXIES`，登录限流才会读取 `X-Forwarded-For`。

## Ingest 语义

ingest 返回 `202` 仅表示 span 进入有界内存队列。队列满返回 `429`。服务优雅退出时会等待 writer flush；重复的 `project_id + trace_id + span_id` 使用 upsert 幂等写入。

Writer 会对失败批次进行有限重试，并通过 `GET /api/v1/ingest/stats` 暴露写入错误和丢弃数量。enqueue 会在入队前检查整批容量，避免一批请求部分成功后客户端重试造成意外重复。

Span 校验限制单个 input/output 为 1 MiB、attributes 为 256 KiB / 128 项，并要求有效的 `start_time` 和非负 duration。超限请求返回 413。

当前查询 API 包括 `GET /api/v1/traces` 和 `GET /api/v1/traces/{trace_id}`。列表支持 status、kind、name、时间范围、duration、token 数量过滤和不透明 cursor 分页。Trace List 使用 `trace_summaries` 物化摘要，摘要在 span upsert 的同一事务中更新。

Dashboard 使用 `GET /api/v1/dashboard`，默认聚合最近 24 小时，也支持 RFC3339 的 `start_time` / `end_time`。请求量、错误率、Token 和延迟分位数从 `trace_summaries` 聚合，span kind 用量从 spans 聚合；查询始终绑定认证 Key 的 ProjectID。

Default API Key 是兼容用的 admin key，可以创建 Workspace 和 workspace-scoped API Key。新 Key 的明文 token 只在创建响应中返回一次；数据库只存 token hash。撤销通过 `POST /api/v1/keys/{keyID}/revoke` 完成。

Web 用户通过 `POST /api/v1/auth/login` 登录，服务创建 24 小时的有状态 User Session；当前 Session 绑定一个 Workspace。`POST /api/v1/auth/logout` 会在服务端撤销当前 Session，`GET /api/v1/auth/me` 返回当前用户和 Workspace。SDK、CozeLoop ingest 和其他机器调用继续使用 Workspace API Key / PAT；两者不能混淆。首次启动会创建 `TRACY_ADMIN_EMAIL` / `TRACY_ADMIN_PASSWORD` 对应的 owner 用户；未配置密码时生成随机密码并写入启动日志。

Web 登录后可以通过 `GET /api/v1/workspaces` 查看成员可访问的 Workspace，使用 `POST /api/v1/workspaces` 创建 Workspace，使用 `POST /api/v1/workspaces/{workspaceID}/switch` 切换当前 Session。未登录的 Web 页面不会读取旧 API Key，也不会请求 Trace 数据。

JWT OAuth Compatibility 使用 metadata DB 的 `oauth_apps` 和 `oauth_access_tokens` 表。Admin 通过 `/api/v1/oauth/apps` 注册 `client_id`、Workspace、`public_key_id` 和 RSA PEM 公钥；`/api/permission/oauth2/token` 使用 `golang-jwt/jwt` 校验 Bearer JWT 的 RS256 签名及 `iss`、`aud`、`kid`、`iat`、`exp`、`jti` 后签发短期 workspace-scoped access token。JWT audience 必须等于客户端请求的 API host，access token 继续复用现有 `Authorization: Bearer` 认证链路。

## Web 开发

HTTP contract 的机器可读描述位于 [`docs/openapi.yaml`](openapi.yaml)。新增 endpoint 时同步更新该文件，并保持错误响应和 Project 隔离语义与 contract tests 一致。

前端源码位于 `web/`，构建产物输出到被 Git 忽略的 `internal/web/dist/`，由 Go `embed.FS` 编入 binary。构建 Go binary 前必须先运行 `make build-web`；`make build` 已包含该步骤。不要为该目录添加 `.gitkeep`，因为它不能替代真实的前端入口文件。开发前端时可运行：

```bash
cd web
npm install --include=dev
npm run dev
```

Vite 会把 `/api` 请求代理到本地 Go 服务。发布前运行 `make build`。

## Docker

Docker 使用多阶段构建：Node.js 阶段构建 `web/`，Go 阶段编译单 binary，最终镜像只包含 binary、CA 证书和 `/data` 数据目录。前端构建产物不提交到 Git，由 Docker build 在镜像构建过程中生成。

本地构建和运行：

```bash
make docker-build
docker run --rm -p 8080:8080 \
  -v "$(pwd)/data:/data" \
  tracy:local
```

GitHub Actions 会在 push 和 pull request 中运行 Go 测试、静态检查、前端检查、Go binary 构建和 Docker 镜像构建。

推送形如 `v1.2.3` 的 Git tag 后，Release workflow 会生成 `linux/amd64`、`linux/arm64` 和 `darwin/arm64` 二进制压缩包，并将 `linux/amd64` 与 `linux/arm64` 多架构镜像推送到 `ghcr.io/<owner>/<repository>`。仓库需要允许 GitHub Actions 使用 `packages: write` 权限。

## OpenAPI

`docs/openapi.yaml` 是 HTTP contract 的显式源文件，不从 handler 注解自动生成。当前服务使用标准库 `net/http` 的路由和独立 HTTP DTO；强行引入注解生成器会增加路由、模型和响应定义的第二套来源，且无法替代现有黑盒 contract tests。修改 endpoint 时应同步更新 OpenAPI 文件和 contract tests。

前端工程化检查使用 Oxlint 和 Oxfmt：

```bash
cd web
npm run lint          # 静态检查
npm run format        # 格式化源码
npm run format:check  # 只检查格式，不修改文件
npm run check         # 格式检查 + lint + TypeScript/Vite 构建
```

仓库根目录也提供 `make web-check`。依赖版本固定在 `web/package-lock.json`，避免不同开发机使用不同工具版本。

## 测试约定

契约测试只依赖 `BASE_URL`，不导入服务内部包。每新增 endpoint，至少增加请求、认证失败、Project 隔离和错误响应测试。后续 Go/Rust 或不同 TraceStore 应复用同一套黑盒测试。
