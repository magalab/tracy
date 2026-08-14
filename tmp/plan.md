# Trace Server 开发计划

> 执行原则：先验证最小可用链路和 CozeLoop 协议，再扩展 UI、性能和多后端。未来扩展不能阻塞 MVP。

## 当前实现状态（2026-08-14）

已完成：

```text
✓ Git 仓库和 Go 单 binary 骨架
✓ 两个 SQLite 数据库、迁移和 WAL 配置
✓ Default Project、API Key hash、Project 隔离
✓ 有界队列、批量写入、幂等 ingest、GetTrace
✓ Trace List、基础过滤和 cursor 分页
✓ CozeLoop Go SDK trace ingest 兼容层
✓ React Trace Explorer 和 Go embed.FS
✓ SQLite benchmark harness 和基础 workload
✓ payload 大小校验、writer 重试和 ingest stats
✓ admin Project/API Key 管理与 revoke
✓ trace_summaries 物化摘要和时间/duration/token 查询过滤
✓ Feedback/Annotation API、Project 隔离和 Trace Explorer 展示
✓ Dashboard API、24 小时默认时间范围、请求/错误/Token/延迟/用量指标和前端概览卡片
✓ README、AGENTS.md、开发说明和 contract test 基础
```

已通过验证：

```text
go test ./...
go vet ./...
go build ./cmd/server
npm run build
官方 CozeLoop Go SDK → 本地 Tracy → SQLite → Trace List
```

下一阶段：

```text
✓ SQLite 1M/10M 多规模性能基线（当前 workload 下暂不需要 DuckDB）
✓ Dashboard Phase 13（当前用 span kind 作为用量分组维度）
✓ JWT OAuth Compatibility：OAuth App metadata、RS256 JWT bearer exchange、project-scoped access token
□ DuckDB/VictoriaTraces 仅在出现明确容量或查询瓶颈后评估
```

## 1. 项目目标

开发一个轻量、自托管、单进程优先的 LLM/Agent Trace Observability 产品。

产品定位类似一个轻量版 LangSmith，第一阶段重点解决：

```text
SDK / Application
        │
        ▼
    Trace Ingest
        │
        ▼
   Trace Storage
        │
        ▼
Trace List / Search / Detail
        │
        ▼
      Web UI
```

设计优先级：

```text
部署简单
    >
代码结构清晰
    >
可扩展
    >
分布式能力
```

默认部署目标：

```text
一个 Go binary
+
meta.db
+
traces.db
```

不默认依赖：

```text
Redis
MQ
ClickHouse
MySQL
PostgreSQL
MinIO
Nginx
```

---

## 2. 核心工程原则

### 2.1 不以 CozeLoop 为代码基础

CozeLoop 仅作为：

```text
API 行为参考
官方 SDK 协议参考
Trace UI / UX 参考
Trace 数据模型参考
```

尽量不要直接复制 CozeLoop 后端代码。

优先根据需求重新实现。

允许参考：

```text
API path
request / response schema
filter semantics
SDK authentication behavior
Trace UI interaction
```

不要继承：

```text
原有 DDD 层级
Wire / RPC infrastructure
Thrift codegen
Redis abstractions
RocketMQ
ClickHouse DAO
Evaluation / Dataset / Prompt / LLM
```

---

## 3. 技术栈

第一主线：

```text
Backend
Go

Frontend
React + TypeScript

Metadata DB
SQLite

Trace DB
SQLite

API
HTTP + JSON

API Docs
OpenAPI

Frontend API
手写 TypeScript client
不做 TS client codegen
```

后备技术路线：

```text
Backend
Rust + Salvo

Metadata
SQLite

Trace
SQLite / DuckDB / VictoriaTraces
```

Rust 第一阶段不开发完整实现。

先让 Go 版把以下内容稳定下来：

```text
API contract
domain model
storage contract
benchmark workload
```

再根据这些契约实现 Rust 版。

---

## 4. 推荐仓库结构

第一版建议：

```text
trace-server/
├── cmd/
│   └── server/
│
├── internal/
│   ├── api/
│   │   ├── auth/
│   │   ├── project/
│   │   └── trace/
│   │
│   ├── auth/
│   ├── project/
│   ├── trace/
│   │   ├── model.go
│   │   ├── service.go
│   │   ├── query.go
│   │   └── filter.go
│   │
│   └── storage/
│       ├── meta/
│       │   ├── store.go
│       │   └── sqlite/
│       │
│       └── trace/
│           ├── store.go
│           └── sqlite/
│
├── compat/
│   └── cozeloop/
│       ├── auth/
│       └── trace/
│
├── migrations/
│   ├── meta/
│   └── trace/
│
├── web/
│
├── bench/
│
├── tests/
│   ├── integration/
│   └── contract/
│
├── docs/
│
├── go.mod
└── Makefile
```

后续扩展：

```text
internal/storage/meta/
├── sqlite/
├── postgres/
└── mysql/

internal/storage/trace/
├── sqlite/
├── duckdb/
└── victoria/
```

---

## 5. 数据库边界

这是整个项目必须严格保持的边界。

### 5.1 Metadata Store

存储：

```text
User
Project
API Key
OAuth App
Session
Settings
Saved View
```

定义领域级接口，不暴露 SQL：

```go
type MetaStore interface {
    Users() UserRepository
    Projects() ProjectRepository
    APIKeys() APIKeyRepository
    OAuthApps() OAuthAppRepository
}
```

默认：

```text
SQLiteMetaStore
```

文件：

```text
data/meta.db
```

未来允许：

```text
PostgreSQL
MySQL
```

但当前阶段不实现。

### 5.2 Trace Store

Trace 数据库单独抽象。

建议核心接口：

```go
type TraceStore interface {
    AppendSpans(
        ctx context.Context,
        spans []Span,
    ) error

    GetTrace(
        ctx context.Context,
        projectID ProjectID,
        traceID TraceID,
    ) ([]Span, error)

    ListTraces(
        ctx context.Context,
        query TraceQuery,
    ) (TracePage, error)

    QuerySpans(
        ctx context.Context,
        query SpanQuery,
    ) (SpanPage, error)

    Metrics(
        ctx context.Context,
        query MetricsQuery,
    ) (MetricsResult, error)
}
```

默认：

```text
SQLiteTraceStore
```

文件：

```text
data/traces.db
```

以后实现：

```text
DuckDBTraceStore
VictoriaTraceStore
```

不要设计 SQL-level 通用接口。

抽象的是：

```text
Trace capability
```

不是：

```text
database query language
```

### 5.3 两个 SQLite 数据库的边界

`meta.db` 与 `traces.db` 不做跨数据库事务，也不建立跨库 foreign key。

约定：

```text
MetaStore 负责用户、项目、Key 和管理数据
TraceStore 负责 span、trace summary 和 trace 查询
Service 层负责 Project 存在性、权限和删除编排
```

删除 Project 时由 Service 执行显式清理流程；备份必须同时备份两个数据库。TraceStore 不直接查询 MetaStore。

---

## 6. Phase 0：项目骨架

目标只建立工程结构，不实现完整业务。

完成：

```text
Go module
配置加载
logger
HTTP server
graceful shutdown
/healthz
/readyz
meta SQLite 初始化
trace SQLite 初始化
migration runner
基本测试框架
Makefile
```

配置例如：

```yaml
server:
  addr: ":8080"

metadata:
  driver: sqlite
  sqlite:
    path: ./data/meta.db

trace:
  driver: sqlite
  sqlite:
    path: ./data/traces.db
```

要求：

```text
./trace-server
```

即可启动。

首次运行自动创建：

```text
data/meta.db
data/traces.db
```

验收：

```text
GET /healthz
→ 200

GET /readyz
→ 200
```

### Phase 0.1：HTTP Contract 和兼容性探测

在业务实现前先固定 `/api/v1` 的最小 HTTP 行为：

```text
错误响应格式
时间格式和 ID 格式
API Key 认证结果
ingest 的 accepted 语义
cursor 的不透明语义
```

同时使用官方 CozeLoop Go SDK 对本地请求做探测，记录实际使用的 endpoint、headers、workspace 参数、request/response JSON、错误码和重试行为。探测结果固化为 fixture，避免凭猜测实现兼容层。

---

## 7. Phase 1：Project + API Key

先不要做复杂账号系统。

第一阶段 Metadata Model：

```text
Project
├ id
├ name
├ created_at
└ updated_at

APIKey
├ id
├ project_id
├ name
├ token_hash
├ created_at
├ expires_at
└ last_used_at
```

Project 是产品内部概念。

CozeLoop SDK 的：

```text
workspace_id
```

进入兼容层以后映射：

```text
workspace_id
    ↓
project_id
```

API Key：

```http
Authorization: Bearer <token>
```

数据库只保存：

```text
token hash
```

绝不保存明文 API Key。

第一阶段可以首次启动自动创建 `Default Project`，并提供 CLI 或 API 创建 API Key。

验收条件：

```text
合法 API key
→ Principal{ProjectID}

非法 API key
→ 401

Project A API key
→ 无法读取 Project B 数据
```

安全约定：

```text
token 只在创建时完整返回一次
数据库只保存 token hash
支持 revoke 和 expires_at
last_used_at 异步更新
```

MVP 可以只有一种 API Key，但权限模型预留 ingest、read、admin 三类能力。首次启动自动创建 Default Project 时，初始 Key 必须通过明确的 CLI 输出或受控文件交付，不能写入普通日志。

---

## 8. Phase 2：Trace Domain Model

先定义干净的内部模型。

不要直接使用 CozeLoop DTO。

建议：

```go
type Span struct {
    ProjectID ProjectID

    TraceID      string
    SpanID       string
    ParentSpanID string

    Name string
    Kind SpanKind

    StartTime time.Time
    Duration  time.Duration

    Status SpanStatus

    Input  string
    Output string

    InputTokens  int64
    OutputTokens int64

    Attributes map[string]Value
}
```

模型约定：

```text
所有时间使用 UTC
数据库时间使用 integer timestamp，统一微秒或纳秒
Duration 使用整数，不使用浮点数
Span 的业务唯一键为 project_id + trace_id + span_id
重复 ingest 必须幂等
允许 span 乱序到达
```

建议补充 `ReceivedAt`、`ErrorType`、`ErrorMessage`、`StatusMessage`、`ModelName`、`Provider`、`Events`、`Links` 和 `Resource/ServiceName`。输入输出和 attributes 必须定义大小限制及超限行为；MVP 默认截断并记录指标，不允许单个 payload 无限增长。

Trace List 不应长期依赖扫描所有 span 聚合。预留 `trace_summaries`，包含 project_id、trace_id、start_time、end_time、root_span_id、span_count、status、input_tokens 和 output_tokens。第一版可以延后完整优化，但必须明确其生成和幂等更新规则。

动态 attributes 类型至少支持：

```text
string
integer
float
boolean
```

内部模型不应该出现：

```text
Enterprise
Space
Evaluation
Prompt
Dataset
Coze platform-specific enums
```

---

## 9. Phase 3：SQLite Trace Store

这是第一个真正重要的实现。

初始 schema 大致：

```text
spans
├ project_id
├ trace_id
├ span_id
├ parent_span_id
├ name
├ span_kind
├ start_time
├ duration
├ status
├ input
├ output
├ input_tokens
├ output_tokens
└ attributes_json
```

核心索引：

```text
(project_id, start_time)
(project_id, trace_id)
(project_id, trace_id, span_id)
(project_id, status, start_time)
(project_id, span_kind, start_time)
```

SQLite 启用：

```text
WAL
foreign_keys
busy_timeout
合理的 synchronous
```

Trace ingest 不允许每个 span 单独开启事务。

应该实现：

```text
HTTP ingest
    ↓
bounded queue
    ↓
single TraceWriter
    ↓
batch
    ↓
transaction
    ↓
SQLite
```

例如：

```text
batch size = 128 / 256
flush interval = 20~100ms
```

这些参数全部可配置。

### 9.1 Ingest 可靠性约定

```text
队列满：返回 429，并记录 dropped / rejected 指标
成功响应：表示已进入内存队列，不承诺已经落盘
优雅退出：停止接收后等待队列 flush，超时则报告未落盘数量
SQLite 写失败：有限次数重试，之后记录错误并触发告警
批内非法 span：隔离非法 span，不让单条数据拖垮整批
重复 span：按 project_id + trace_id + span_id 幂等处理
```

至少增加 `accepted`、`rejected`、`dropped`、`queue depth`、`batch size`、`flush latency` 和 `write error` 指标。

---

## 10. Phase 4：CozeLoop SDK Compatibility

这是 MVP 的核心能力。

目标：

> 官方 CozeLoop SDK 只修改 Base URL，即可把 Trace 发到本项目。

第一阶段优先支持：

```text
COZELOOP_API_BASE_URL
COZELOOP_WORKSPACE_ID
COZELOOP_API_TOKEN
```

兼容：

```text
Authorization: Bearer <api-token>
workspace_id
Trace ingest HTTP path
Trace ingest request JSON
Trace ingest response JSON
```

设计：

```text
CozeLoop DTO
     ↓
compat/cozeloop
     ↓
mapper
     ↓
Internal Span
     ↓
TraceService
```

禁止：

```text
CozeLoop DTO
     ↓
直接进入 storage
```

验收必须使用官方 SDK。

至少 CI 测：

```text
official Go SDK
→ localhost server
→ ingest span
→ SQLite
→ GET trace
→ 数据正确
```

兼容层分两步：

```text
fixture contract test：默认 CI 运行，快速稳定
official SDK e2e：验证真实 SDK，允许单独运行
```

只有在协议探测完成后才实现 mapper；兼容 DTO 不得进入 domain 或 storage。

之后增加：

```text
Python SDK
JavaScript SDK
```

---

## 11. Phase 5：Trace Query API

实现最小产品 API。

第一批：

```text
List Traces
Get Trace
List/Search Spans
Trace Field Metadata
```

支持：

```text
Project
时间范围
status
span kind
name
duration
token count
trace_id
span_id
```

分页使用 cursor，而不是 offset。

例如 cursor 包含：

```text
start_time
trace_id/span_id
```

动态 attributes 查询第一阶段可以有限支持。

不要一开始实现完整 CozeLoop Filter DSL。

API 统一使用 `/api/v1`。Cursor 对客户端不透明，不允许客户端依赖内部字段。错误响应统一为：

```json
{
  "error": {
    "code": "invalid_api_key",
    "message": "..."
  }
}
```

---

## 12. Phase 6：前端

前端：

```text
React
TypeScript
```

不生成 API client。

结构：

```text
web/src/
├── api/
│   ├── client.ts
│   ├── auth.ts
│   ├── projects.ts
│   └── traces.ts
│
├── models/
│
├── features/
│   └── traces/
│       ├── trace-list/
│       ├── trace-detail/
│       ├── span-tree/
│       └── filters/
│
├── components/
│   └── ui/
│
└── app/
```

API client 手写：

```ts
traceApi.list()
traceApi.get()
traceApi.search()
```

Trace UI 可以参考/移植 CozeLoop 的交互设计，但应逐步解除：

```text
@cozeloop/api-schema
@coze-arch/coze-design
Prompt
Evaluation
Tag
Enterprise
Space
```

的依赖。

核心值得复用的是：

```text
Trace list
Trace detail
Span tree
Resizable panels
JSON viewer
Virtualized list
Filter UX
```

而不是 CozeLoop 整个 Console。

---

## 13. Phase 7：Web UI 内嵌

生产构建：

```text
React
 ↓
dist/
 ↓
Go embed.FS
 ↓
single binary
```

路由：

```text
/api/*
→ Go API

/*
→ React SPA
```

SPA fallback：

```text
未知静态路径
→ index.html
```

最终部署：

```bash
./trace-server
```

然后访问：

```text
http://localhost:8080
```

不需要 Nginx。

---

## 14. Phase 8：Trace Benchmark Harness

这一阶段不要等 SQLite 出问题才做。

尽早建立标准 workload。

数据规模：

```text
1M spans
10M spans
50M spans
```

主要 workload：

```text
Ingest
recent traces
get trace
status filter
span kind filter
dynamic attribute filter
latency filter
token filter
aggregation
concurrent read + write
```

测试必须包括：

```text
持续 ingest
+
同时 UI 查询
```

而不是静态数据库 benchmark。

记录：

```text
ingest throughput
p50/p95/p99
query latency
CPU
RSS
disk size
WAL size
checkpoint behavior
startup time
```

Benchmark 必须定义通过标准，而不只是收集数字。至少记录：

```text
目标吞吐
ingest/query p50/p95/p99 上限
最大 RSS
最大可接受队列积压
允许的数据丢失行为
读写并发时的查询上限
```

每组测试都必须包含持续写入和同时查询，数据集、随机种子、机器信息和配置要可复现。

---

## 15. Phase 9：SQLite Trace 优化实验

按实验结果逐步添加，不提前复杂化。

实验：

```text
普通 JSON attributes
json_extract
expression index
generated columns
FTS5
partial indexes
不同 batch size
不同 transaction size
不同 WAL checkpoint 参数
```

优先验证：

> SQLite 实际能支撑多大的真实 Trace workload。

不要先假设 SQLite 不够用。

---

## 16. Phase 10：DuckDB TraceStore

只有 SQLite benchmark 基本稳定后再开发。

要求实现同一 `TraceStore` interface。

不要修改：

```text
TraceService
API
Frontend
Auth
```

运行完全相同的 benchmark。

比较：

```text
SQLiteTraceStore
vs
DuckDBTraceStore
```

重点：

```text
大范围 aggregation
大量 attributes filter
10M+
50M+ spans
读写并发
```

---

## 17. Phase 11：VictoriaTraces

再实现：

```text
VictoriaTraceStore
```

这一层与 DuckDB 不同，它是外部 Trace backend。

Adapter 负责：

```text
ingest mapping
query mapping
trace retrieval
filter translation
```

同样要求：

```text
上层 API 不感知 backend
```

配置：

```yaml
trace:
  driver: victoria
  victoria:
    endpoint: http://...
```

---

## 18. Phase 12：Feedback / Annotation

等 Trace Explorer 稳定后增加。

简单数据模型即可：

```text
Annotation
├ id
├ project_id
├ trace_id
├ span_id
├ key
├ score
├ label
├ comment
├ created_by
├ created_at
└ updated_at
```

先不要引入：

```text
Evaluator
Dataset
Evaluation Pipeline
```

Feedback 本身独立存在。

---

## 19. Phase 13：Dashboard

基于：

```text
TraceStore.Metrics()
```

实现：

```text
request count
error rate
token usage
latency
model usage
```

SQLite 和 DuckDB/Victoria 各自实现最适合自己的查询方式。

API 不暴露数据库细节。

当前实现补充约束：

```text
默认时间范围：最近 24 小时
显式时间范围：start_time/end_time，RFC3339
聚合粒度：trace_summaries；用量分组暂按 span kind
权限边界：始终按当前 API Key 的 project_id 过滤
```

Phase 14 已完成：OAuth App 通过 Admin API 注册，JWT bearer exchange 复用现有 project-scoped authentication；下一阶段继续保持 SQLite 单进程边界，等待 benchmark 暴露真实瓶颈后再评估后备 Trace backend。

---

## 20. Phase 14：JWT OAuth Compatibility

API Token 稳定后再支持 CozeLoop JWT OAuth。

实现兼容：

```text
/api/permission/oauth2/token
```

支持：

```text
urn:ietf:params:oauth:grant-type:jwt-bearer
```

Metadata 增加：

```text
OAuthApp
├ client_id
├ project_id
├ public_key_id
├ public_key
└ enabled
```

然后官方 SDK 可以使用：

```text
COZELOOP_JWT_OAUTH_CLIENT_ID
COZELOOP_JWT_OAUTH_PRIVATE_KEY
COZELOOP_JWT_OAUTH_PUBLIC_KEY_ID
```

---

## 21. Rust 后备实现

Rust 不和 Go 同步开发。

启动条件建议是：

```text
Go API 基本稳定
TraceStore interface 稳定
SQLite schema 稳定
Contract tests 稳定
Benchmark workload 稳定
```

Rust 技术栈：

```text
Rust
Salvo
Tokio
Serde
SQLite
```

Rust 实现不共享 Go 代码。

共享的是：

```text
HTTP behavior
database semantics
contract tests
benchmark workload
```

目标：

```text
Go Server
    ↓
contract suite
    ✓

Rust Server
    ↓
same contract suite
    ✓
```

Rust 版首先实现：

```text
health
auth
project
SQLite MetaStore
SQLite TraceStore
ingest
trace list/detail
```

然后再评估是否替换 Go 主线。

---

## 22. Contract Tests 是最高优先级基础设施

独立黑盒测试，不 import server 内部代码。

Contract test 必须从第一个 endpoint 开始，而不是等所有 API 完成后再补。最小顺序是：

```text
health → auth → ingest → GetTrace → ListTraces → filters
```

只接受：

```text
BASE_URL
```

测试：

```text
Health
API Key auth
Project isolation
CozeLoop ingest
duplicate ingest
Trace list
Trace detail
pagination
filters
invalid requests
authorization
```

以后：

```text
Go
Rust
SQLite
DuckDB
Victoria
```

各种组合都跑同一套测试。

---

## 23. Codex 开发规则

建议同步复制到仓库根目录 `AGENTS.md`。

1. 不直接复制 CozeLoop 后端实现。
2. 可以参考 CozeLoop API contract、SDK behavior 和 UI behavior。
3. 不引入 Redis、MQ、ClickHouse、MinIO 作为默认依赖。
4. Metadata Store 与 Trace Store 必须保持独立。
5. 默认数据库是 SQLite。
6. 不为未来 PostgreSQL/MySQL 人为限制 SQLite 实现。
7. Storage abstraction 必须是 domain capability，而非通用 SQL wrapper。
8. HTTP DTO 不得直接作为 domain entity 或 storage entity。
9. CozeLoop compatibility 必须放在 `compat/cozeloop` 边界。
10. Frontend API client 手写，不生成 TypeScript API client。
11. 新增 endpoint 必须有 integration/contract test。
12. 新增 storage backend 必须跑同一 benchmark 和 contract suite。
13. 优先简单实现，不提前建设分布式能力。
14. 不因为未来可能需要集群而引入 Redis/distributed lock。
15. 尽量保持 server 可作为单 binary 运行。
16. 新增失败场景必须定义 HTTP 状态、错误 code 和重试语义。
17. ingest 成功只代表进入队列，除非 API 明确声明已落盘。
18. 所有新增数据字段必须定义大小限制、时间单位和幂等语义。

---

## 24. MVP 分层和明确截止线

为了尽早得到可运行结果，MVP 分三层。每一层都应可独立验收。

### MVP-0：最小垂直链路

```text
✓ Go server
✓ SQLite meta.db
✓ SQLite traces.db
✓ Default Project
✓ API Key
✓ Trace Detail API
✓ contract tests
```

验收链路：

```text
API Key → ingest span → SQLite → GetTrace
```

### MVP-1：Trace Explorer

```text
✓ buffered/batched Trace writer
✓ Trace List API
✓ basic filters
✓ cursor pagination
✓ React Trace List
✓ Span Tree
✓ Trace Detail
```

### MVP-2：可交付单 binary

```text
✓ 官方 CozeLoop Go SDK ingest
✓ fixture contract test
✓ official SDK e2e test
✓ embedded frontend
✓ single binary
✓ SQLite benchmark 和基础优化
```

暂时不做：

```text
✗ Rust 完整版
✗ MySQL
✗ PostgreSQL
✗ DuckDB 正式实现
✗ VictoriaTraces 正式实现
✗ JWT OAuth
✗ Evaluation
✗ Dataset
✗ Prompt Management
✗ LLM Proxy
✗ distributed deployment
✗ Redis
✗ MQ
```

这个边界非常重要，否则 Codex 很容易顺手把未来扩展一起实现。

---

## 25. 给 Codex 的实际执行顺序

严格按这个顺序推进：

```text
01 repository skeleton
02 config + logging + server lifecycle
03 SQLite migration infrastructure
04 HTTP v1 error/auth/ingest contract
05 health contract test
06 MetaStore interface + SQLiteMetaStore
07 Project model + API Key authentication
08 auth and project-isolation contract tests
09 Trace domain model and size/idempotency rules
10 TraceStore interface + SQLite schema
11 minimal internal ingest API
12 GetTrace
13 ingest/GetTrace contract tests
14 single-writer + batch ingestion
15 queue-full/shutdown/write-failure tests
16 ListTraces
17 filters + cursor pagination
18 ListTraces/filter contract tests
19 CozeLoop SDK HTTP request probe
20 compat mapper + fixture contract test
21 official SDK e2e integration test
22 React shell + handwritten frontend API
23 trace list
24 trace detail + span tree
25 embed frontend into Go binary
26 reproducible SQLite benchmark harness
27 benchmark + optimize SQLite
28 decide whether DuckDB implementation is necessary
✓ 29 Dashboard metrics API and Trace Explorer overview
✓ 30 Dashboard time-range and Project-isolation contract tests
✓ 31 stabilize API Token behavior
✓ 32 CozeLoop JWT OAuth compatibility
```

第 28 步之前不要实现 DuckDB / VictoriaTraces。

真正未知的问题是：

> SQLite 到底什么时候不够用。

应该让 benchmark 数据回答，而不是靠架构直觉回答。

---

## 26. 架构边界总结

必须长期保持三个边界：

```text
CozeLoop
    ↓
只保留 protocol compatibility

Metadata
    ↓
MetaStore

Trace
    ↓
TraceStore
```

只要这三个边界不被写穿，后续：

```text
Go → Rust
SQLite Metadata → PostgreSQL / MySQL
SQLite Trace → DuckDB / VictoriaTraces
```

都能维持相对可控的迁移成本。
