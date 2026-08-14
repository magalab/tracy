# 开发说明

## 边界

`internal/storage/meta` 只负责 Project 和 API Key；`internal/storage/trace` 只负责 Span。HTTP DTO 通过 `internal/trace` domain model 进入存储，CozeLoop 兼容代码应放在 `compat/cozeloop`。

服务使用两个独立 SQLite 文件：`meta.db` 和 `traces.db`。两者不做跨库事务，Project 权限由 API 层和 Service 层保证。

## Ingest 语义

ingest 返回 `202` 仅表示 span 进入有界内存队列。队列满返回 `429`。服务优雅退出时会等待 writer flush；重复的 `project_id + trace_id + span_id` 使用 upsert 幂等写入。

Writer 会对失败批次进行有限重试，并通过 `GET /api/v1/ingest/stats` 暴露写入错误和丢弃数量。enqueue 会在入队前检查整批容量，避免一批请求部分成功后客户端重试造成意外重复。

Span 校验限制单个 input/output 为 1 MiB、attributes 为 256 KiB / 128 项，并要求有效的 `start_time` 和非负 duration。超限请求返回 413。

当前查询 API 包括 `GET /api/v1/traces` 和 `GET /api/v1/traces/{trace_id}`。列表支持 status、kind、name、时间范围、duration、token 数量过滤和不透明 cursor 分页。Trace List 使用 `trace_summaries` 物化摘要，摘要在 span upsert 的同一事务中更新。

Dashboard 使用 `GET /api/v1/dashboard`，默认聚合最近 24 小时，也支持 RFC3339 的 `start_time` / `end_time`。请求量、错误率、Token 和延迟分位数从 `trace_summaries` 聚合，span kind 用量从 spans 聚合；查询始终绑定认证 Key 的 ProjectID。

Default API Key 是 admin key，可以创建 Project 和 project-scoped API Key。新 Key 的明文 token 只在创建响应中返回一次；数据库只存 token hash。撤销通过 `POST /api/v1/keys/{keyID}/revoke` 完成。

JWT OAuth Compatibility 使用 metadata DB 的 `oauth_apps` 和 `oauth_access_tokens` 表。Admin 通过 `/api/v1/oauth/apps` 注册 `client_id`、Project、`public_key_id` 和 RSA PEM 公钥；`/api/permission/oauth2/token` 校验 Bearer JWT 的 RS256 签名及 `iss`、`aud`、`kid`、`iat`、`exp`、`jti` 后签发短期 project-scoped access token。JWT audience 必须等于客户端请求的 API host，access token 继续复用现有 `Authorization: Bearer` 认证链路。

Annotation 存在 metadata DB 中，但所有查询都带当前 API Key 的 ProjectID。Annotation 的 `key` 必填，score 范围为 0 到 1；Trace Explorer 会在详情页加载、创建和删除 Annotation。

## Web 开发

HTTP contract 的机器可读描述位于 [`docs/openapi.yaml`](openapi.yaml)。新增 endpoint 时同步更新该文件，并保持错误响应和 Project 隔离语义与 contract tests 一致。

前端源码位于 `web/`，构建产物输出到 `internal/web/dist/`，由 Go `embed.FS` 编入 binary。开发前端时可运行：

```bash
cd web
npm install --include=dev
npm run dev
```

Vite 会把 `/api` 请求代理到本地 Go 服务。发布前运行 `make build`。

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
