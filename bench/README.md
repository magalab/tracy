# SQLite benchmark

基准测试只测当前 TraceStore capability，不把 SQL wrapper 暴露给上层。默认 `go test ./...` 会跳过长时间 workload；显式设置环境变量才运行规模测试：

```bash
GOCACHE=/private/tmp/tracy-gocache go test ./bench -bench=. -benchtime=3s
TRACY_BENCH_SPANS=1000000 GOCACHE=/private/tmp/tracy-gocache go test ./bench -run TestWorkloadSmoke -v
TRACY_BENCH_SPANS=10000000 GOCACHE=/private/tmp/tracy-gocache go test ./bench -run TestWorkloadSmoke -v
TRACY_BENCH_CONCURRENT=10 GOCACHE=/private/tmp/tracy-gocache go test ./bench -run TestConcurrentReadWrite -v
```

`TestWorkloadSmoke` 会输出 ingest 吞吐、Trace List p50/p95/p99、SQLite/WAL 文件大小和 Go 内存统计。结果应保存到发布记录中；正式比较 DuckDB 前，先用相同 workload 建立 SQLite 基线。

## 当前基线（2026-08-14，Apple M1 arm64）

workload 使用 batch size 256、单 writer、每个 trace 10 个 spans，payload 为小型文本和一个 attribute：

| spans | ingest | ingest rate | List p50 | List p95 | List p99 | DB | WAL |
|---:|---:|---:|---:|---:|---:|---:|---:|
| 1M | 32.15s | 31.1k/s | 75.5µs | 120.5µs | 163.9µs | 253.79MiB | 4.15MiB |
| 10M | 5m40.07s | 29.4k/s | 77.4µs | 136.2µs | 164.6µs | 2.63GiB | 4.16MiB |

结论：在这个 workload 下，10M spans 仍保持接近线性 ingest 吞吐，摘要列表查询维持亚毫秒级；目前没有足够证据为了 MVP 引入 DuckDB。50M 只保留命令，不在开发机上直接运行，以避免占用约十几 GiB 临时磁盘。
