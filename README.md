# Tracy

Tracy 是一个单进程优先、自托管的 LLM/Agent Trace Observability 服务。当前实现已覆盖 MVP-1：Go HTTP 服务、SQLite metadata/trace 存储、Workspace/API Key 隔离、批量 ingest、Trace Explorer 和嵌入式 Web UI。

## 快速开始

需要 Go 1.26 或更高版本。

```bash
go run ./cmd/server
```

如果修改了前端源码，重新生成嵌入资源：

```bash
make build
```

### Docker

Build the image:

```bash
make docker-build
```

Run the container with SQLite files persisted under `./data`:

```bash
docker run --rm -p 8080:8080 \
  --user "$(id -u):$(id -g)" \
  -v "$(pwd)/data:/data" \
  tracy:local
```

镜像默认以 root 运行。绑定挂载宿主机目录时，用 `--user "$(id -u):$(id -g)"` 覆盖为宿主机用户，`/data` 下的 SQLite 文件就会以宿主机用户属主落盘；不传时容器以 root 运行，能写入任意目录，但数据文件属主为 root。

Release builds are produced from version tags and include `linux/amd64`, `linux/arm64`, and `darwin/arm64` archives. The same release workflow publishes only the semver tag (for example, `1.2.3`) and `latest` tags for the `linux/amd64` and `linux/arm64` image at `ghcr.io/<owner>/<repository>`.

首次启动会创建 `data/meta.db` 和 `data/traces.db`，并创建 `TRACY_ADMIN_EMAIL` / `TRACY_ADMIN_PASSWORD` 对应的管理员用户。服务不会自动创建 API Key；登录 Web 后可在当前 Workspace 的 API Keys 页面手动创建。

发送 Trace：

```bash
curl -i http://localhost:8080/api/v1/ingest \
  -H 'Authorization: Bearer <WORKSPACE_API_KEY>' \
  -H 'Content-Type: application/json' \
  -d '{"spans":[{"trace_id":"trace-1","span_id":"span-1","name":"chat","kind":"llm","start_time":"2026-01-01T00:00:00Z","duration":1200000,"status":"ok","input":"hello","output":"world"}]}'
```

查询 Trace：

```bash
curl http://localhost:8080/api/v1/traces/trace-1 \
  -H 'Authorization: Bearer <WORKSPACE_API_KEY>'
```

查询 Trace 列表（支持 `status`、`kind`、`name`、`trace_id`、`start_time`、`end_time`、`min_duration_ms`、`max_duration_ms`、`min_tokens`、`max_tokens`、`limit` 和不透明 `cursor`）：

```bash
curl 'http://localhost:8080/api/v1/traces?limit=20&status=ok' \
  -H 'Authorization: Bearer <WORKSPACE_API_KEY>'
```

查询 Dashboard 指标（默认最近 24 小时，也可传 `start_time` / `end_time` RFC3339）：

```bash
curl http://localhost:8080/api/v1/dashboard \
  -H 'Authorization: Bearer <WORKSPACE_API_KEY>'
```

CozeLoop JWT OAuth 需要先由 Workspace Owner 注册 OAuth App（`public_key` 使用 RSA PEM 公钥）：

```bash
curl -X POST http://localhost:8080/api/v1/oauth/apps \
  -H 'Authorization: Bearer <USER_SESSION>' \
  -H 'X-Tracy-Workspace-ID: default' \
  -H 'Content-Type: application/json' \
  -d '{"client_id":"coze-client","workspace_id":"default","public_key_id":"key-1","public_key":"-----BEGIN PUBLIC KEY-----\\n...\\n-----END PUBLIC KEY-----"}'
```

注册后，官方 SDK 可使用 `COZELOOP_JWT_OAUTH_CLIENT_ID`、`COZELOOP_JWT_OAUTH_PRIVATE_KEY` 和 `COZELOOP_JWT_OAUTH_PUBLIC_KEY_ID`，Tracy 会兼容 `/api/permission/oauth2/token` 的 JWT bearer 换 token 流程。

## HTTP contract

- `GET /healthz` 和 `GET /readyz` 返回 200。
- `POST /api/v1/ingest` 接受 `{ "spans": [...] }`，成功返回 `202`，表示数据已进入内存队列。
- `POST /v1/loop/traces/ingest` 接受官方 CozeLoop Go SDK 的 `{ "spans": [...] }` payload，成功返回 `{"code":0,"msg":""}`。
- `POST /api/v1/auth/logout` 撤销当前 Web Session；API 未匹配路径返回 JSON 404/405，不会回退到 SPA。
- `GET /api/v1/traces` 返回当前 Workspace 的 Trace 列表；`GET /api/v1/traces/{trace_id}` 返回单个 Trace。
- Trace 详情响应包含整条 Trace 的 `start_time`、`end_time` 和 `span_count`，同时通过 `spans` 返回当前分页；Web Trace Explorer 支持树视图和时间线视图。
- `GET /api/v1/dashboard` 返回当前 Workspace 的请求量、错误率、Token 汇总、延迟分位数和按 span kind 的用量分布。
- `POST /api/permission/oauth2/token` 支持 CozeLoop JWT bearer grant；Workspace Owner 可通过 `POST/GET /api/v1/oauth/apps` 管理当前 Workspace 的 OAuth App。
- `/` 和其它前端路径返回嵌入式 React Trace Explorer。
- Workspace Owner 可调用 `GET /api/v1/workspaces`、`POST /api/v1/workspaces` 以及当前 Workspace 的 API Key 列表、创建和撤销接口。
- `GET /api/v1/ingest/stats` 在 Workspace Owner 通过 `X-Tracy-Workspace-ID` 完成权限校验后，返回进程级全局 writer 指标，不是 Workspace 级统计。
- 单个 input/output 最大 1 MiB，attributes 最大 256 KiB / 128 项；超限返回 `413 payload_too_large`。
- 错误格式固定为 `{ "error": { "code": "...", "message": "..." } }`。
- API 使用 `Authorization: Bearer <token>`；用户 Session 访问 Workspace-scoped 接口时还必须发送 `X-Tracy-Workspace-ID`；API Key 只绑定一个 Workspace。
- `GET /api/v1/auth/me` 只返回当前用户信息，不保存或返回当前 Workspace；Workspace 选择由客户端维护，并通过 `X-Tracy-Workspace-ID` 发送。

## 配置

完整示例见 [`config.example.yaml`](config.example.yaml)。

当前配置通过环境变量读取：`TRACY_ADDR`、`TRACY_META_DB`、`TRACY_TRACE_DB`、`TRACY_BATCH_SIZE`、`TRACY_FLUSH_INTERVAL`、`TRACY_QUEUE_SIZE`、`TRACY_QUEUE_BYTES`。

如果服务位于反向代理后，可通过 `TRACY_TRUSTED_PROXIES` 配置可信代理 IP/CIDR，登录限流才会使用代理转发的客户端 IP；不要把该配置指向不受信任的网络。

## 开发

```bash
go test ./...
go vet ./...
```

协议边界和本地开发约定见 [`docs/development.md`](docs/development.md)，项目开发规则见 [`AGENTS.md`](AGENTS.md)。

CozeLoop 兼容协议说明见 [`docs/cozeloop-compat.md`](docs/cozeloop-compat.md)。
SQLite benchmark 说明见 [`bench/README.md`](bench/README.md)。
HTTP API 的 OpenAPI 3.0 描述见 [`docs/openapi.yaml`](docs/openapi.yaml)。
