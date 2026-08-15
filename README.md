# Tracy

Tracy 是一个单进程优先、自托管的 LLM/Agent Trace Observability 服务。当前实现已覆盖 MVP-1：Go HTTP 服务、SQLite metadata/trace 存储、Project/API Key 隔离、批量 ingest、Trace Explorer 和嵌入式 Web UI。

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
  -v "$(pwd)/data:/data" \
  tracy:local
```

Release builds are produced from version tags and include `linux/amd64`, `linux/arm64`, and `darwin/arm64` archives. The same release workflow publishes a `linux/amd64` and `linux/arm64` image to `ghcr.io/<owner>/<repository>`.

首次启动会创建 `data/meta.db` 和 `data/traces.db`，并将初始 API Key 只输出到启动日志。也可以预先设置 `TRACY_API_KEY`：

```bash
TRACY_API_KEY=tr_dev_key go run ./cmd/server
```

发送 Trace：

```bash
curl -i http://localhost:8080/api/v1/ingest \
  -H 'Authorization: Bearer tr_dev_key' \
  -H 'Content-Type: application/json' \
  -d '{"spans":[{"trace_id":"trace-1","span_id":"span-1","name":"chat","kind":"llm","start_time":"2026-01-01T00:00:00Z","duration":1200000,"status":"ok","input":"hello","output":"world"}]}'
```

查询 Trace：

```bash
curl http://localhost:8080/api/v1/traces/trace-1 \
  -H 'Authorization: Bearer tr_dev_key'
```

查询 Trace 列表（支持 `status`、`kind`、`name`、`trace_id`、`start_time`、`end_time`、`min_duration_ms`、`max_duration_ms`、`min_tokens`、`max_tokens`、`limit` 和不透明 `cursor`）：

```bash
curl 'http://localhost:8080/api/v1/traces?limit=20&status=ok' \
  -H 'Authorization: Bearer tr_dev_key'
```

查询 Dashboard 指标（默认最近 24 小时，也可传 `start_time` / `end_time` RFC3339）：

```bash
curl http://localhost:8080/api/v1/dashboard \
  -H 'Authorization: Bearer tr_dev_key'
```

CozeLoop JWT OAuth 需要先由 Admin API Key 注册 OAuth App（`public_key` 使用 RSA PEM 公钥）：

```bash
curl -X POST http://localhost:8080/api/v1/oauth/apps \
  -H 'Authorization: Bearer tr_dev_key' \
  -H 'Content-Type: application/json' \
  -d '{"client_id":"coze-client","project_id":"default","public_key_id":"key-1","public_key":"-----BEGIN PUBLIC KEY-----\\n...\\n-----END PUBLIC KEY-----"}'
```

注册后，官方 SDK 可使用 `COZELOOP_JWT_OAUTH_CLIENT_ID`、`COZELOOP_JWT_OAUTH_PRIVATE_KEY` 和 `COZELOOP_JWT_OAUTH_PUBLIC_KEY_ID`，Tracy 会兼容 `/api/permission/oauth2/token` 的 JWT bearer 换 token 流程。

## HTTP contract

- `GET /healthz` 和 `GET /readyz` 返回 200。
- `POST /api/v1/ingest` 接受 `{ "spans": [...] }`，成功返回 `202`，表示数据已进入内存队列。
- `POST /v1/loop/traces/ingest` 接受官方 CozeLoop Go SDK 的 `{ "spans": [...] }` payload，成功返回 `{"code":0,"msg":""}`。
- `POST /api/v1/auth/logout` 撤销当前 Web Session；API 未匹配路径返回 JSON 404/405，不会回退到 SPA。
- `GET /api/v1/traces` 返回当前 Project 的 Trace 列表；`GET /api/v1/traces/{trace_id}` 返回单个 Trace。
- `GET /api/v1/dashboard` 返回当前 Project 的请求量、错误率、Token 汇总、延迟分位数和按 span kind 的用量分布。
- `POST /api/permission/oauth2/token` 支持 CozeLoop JWT bearer grant；Admin API Key 可通过 `POST/GET /api/v1/oauth/apps` 管理 OAuth App。
- `/` 和其它前端路径返回嵌入式 React Trace Explorer。
- Admin API Key 可调用 `GET /api/v1/projects`、`POST /api/v1/projects`、项目 Key 列表/创建和 `POST /api/v1/keys/{id}/revoke`。
- Admin API Key 调用 `GET /api/v1/ingest/stats` 可查看 accepted、written、dropped、queue depth 和 write errors。
- 单个 input/output 最大 1 MiB，attributes 最大 256 KiB / 128 项；超限返回 `413 payload_too_large`。
- 错误格式固定为 `{ "error": { "code": "...", "message": "..." } }`。
- API 使用 `Authorization: Bearer <token>`；Key 只绑定一个 Project。

## 配置

完整示例见 [`config.example.yaml`](config.example.yaml)。当前配置通过环境变量读取：`TRACY_ADDR`、`TRACY_META_DB`、`TRACY_TRACE_DB`、`TRACY_BATCH_SIZE`、`TRACY_FLUSH_INTERVAL`、`TRACY_QUEUE_SIZE`、`TRACY_QUEUE_BYTES`。如果服务位于反向代理后，可通过 `TRACY_TRUSTED_PROXIES` 配置可信代理 IP/CIDR，登录限流才会使用代理转发的客户端 IP；不要把该配置指向不受信任的网络。

## 开发

```bash
go test ./...
go vet ./...
```

协议边界和本地开发约定见 [`docs/development.md`](docs/development.md)，项目开发规则见 [`AGENTS.md`](AGENTS.md)。CozeLoop 兼容协议说明见 [`docs/cozeloop-compat.md`](docs/cozeloop-compat.md)。
SQLite benchmark 说明见 [`bench/README.md`](bench/README.md)。
HTTP API 的 OpenAPI 3.0 描述见 [`docs/openapi.yaml`](docs/openapi.yaml)。
