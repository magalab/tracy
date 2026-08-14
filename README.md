# Tracy

Tracy 是一个单进程优先、自托管的 LLM/Agent Trace Observability 服务。当前实现处于 MVP-0：Go HTTP 服务、SQLite metadata/trace 存储、Project/API Key 隔离、批量 ingest 和 GetTrace。

## 快速开始

需要 Go 1.26 或更高版本。

```bash
go run ./cmd/server
```

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

查询 Trace 列表（支持 `status`、`kind`、`name`、`trace_id`、`limit` 和不透明 `cursor`）：

```bash
curl 'http://localhost:8080/api/v1/traces?limit=20&status=ok' \
  -H 'Authorization: Bearer tr_dev_key'
```

## HTTP contract

- `GET /healthz` 和 `GET /readyz` 返回 200。
- `POST /api/v1/ingest` 接受 `{ "spans": [...] }`，成功返回 `202`，表示数据已进入内存队列。
- `GET /api/v1/traces` 返回当前 Project 的 Trace 列表；`GET /api/v1/traces/{trace_id}` 返回单个 Trace。
- 错误格式固定为 `{ "error": { "code": "...", "message": "..." } }`。
- API 使用 `Authorization: Bearer <token>`；Key 只绑定一个 Project。

## 配置

完整示例见 [`config.example.yaml`](config.example.yaml)。当前配置通过环境变量读取：`TRACY_ADDR`、`TRACY_META_DB`、`TRACY_TRACE_DB`、`TRACY_BATCH_SIZE`、`TRACY_FLUSH_INTERVAL`、`TRACY_QUEUE_SIZE`。

## 开发

```bash
go test ./...
go vet ./...
```

架构和后续阶段见 [`tmp/plan.md`](tmp/plan.md)，协议边界和本地开发约定见 [`docs/development.md`](docs/development.md)。
