# CozeLoop 兼容层

Tracy 的兼容边界位于 `compat/cozeloop`，不让 CozeLoop DTO 进入 domain 或 storage。

当前依据官方 `coze-dev/cozeloop-go` 的公开 exporter 实现固化了以下协议：

```text
POST /v1/loop/traces/ingest
Authorization: Bearer <api-token>
Content-Type: application/json
body: { "spans": [...] }
```

官方 `UploadSpan` 字段映射到 Tracy `trace.Span`：

```text
started_at_micros → start_time
duration_micros   → duration
span_name         → name
span_type         → kind
parent_id         → parent_span_id
tags_*            → attributes
workspace_id      → compatibility metadata
```

上传成功响应使用兼容格式：

```json
{"code":0,"msg":""}
```

协议来源：

- https://github.com/coze-dev/cozeloop-go/blob/main/internal/trace/exporter.go
- https://github.com/coze-dev/cozeloop-go/blob/main/entity/export.go

兼容测试分为 fixture contract test 和官方 SDK e2e。fixture 测试默认运行；官方 SDK e2e 需要显式启动服务并设置 `COZELOOP_API_BASE_URL`、`COZELOOP_WORKSPACE_ID` 和 `COZELOOP_API_TOKEN`。

官方 SDK e2e 测试位于 `tests/cozeloop-e2e`，它是独立 Go module，不会把 SDK 作为 Tracy 服务的运行时依赖：

```bash
cd tests/cozeloop-e2e
go test ./...
```
