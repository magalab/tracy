# 开发说明

## 边界

`internal/storage/meta` 只负责 Project 和 API Key；`internal/storage/trace` 只负责 Span。HTTP DTO 通过 `internal/trace` domain model 进入存储，CozeLoop 兼容代码应放在 `compat/cozeloop`。

服务使用两个独立 SQLite 文件：`meta.db` 和 `traces.db`。两者不做跨库事务，Project 权限由 API 层和 Service 层保证。

## Ingest 语义

ingest 返回 `202` 仅表示 span 进入有界内存队列。队列满返回 `429`。服务优雅退出时会等待 writer flush；重复的 `project_id + trace_id + span_id` 使用 upsert 幂等写入。

当前查询 API 包括 `GET /api/v1/traces` 和 `GET /api/v1/traces/{trace_id}`。列表支持基础字段过滤和不透明 cursor 分页。

## Web 开发

前端源码位于 `web/`，构建产物输出到 `internal/web/dist/`，由 Go `embed.FS` 编入 binary。开发前端时可运行：

```bash
cd web
npm install --include=dev
npm run dev
```

Vite 会把 `/api` 请求代理到本地 Go 服务。发布前运行 `make build`。

## 测试约定

契约测试只依赖 `BASE_URL`，不导入服务内部包。每新增 endpoint，至少增加请求、认证失败、Project 隔离和错误响应测试。后续 Go/Rust 或不同 TraceStore 应复用同一套黑盒测试。
