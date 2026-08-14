# SQLite benchmark

基准测试只测当前 TraceStore capability，不把 SQL wrapper 暴露给上层。默认 `go test ./...` 会跳过长时间 workload；显式设置环境变量才运行规模测试：

```bash
GOCACHE=/private/tmp/tracy-gocache go test ./bench -bench=. -benchtime=3s
TRACY_BENCH_SPANS=1000000 GOCACHE=/private/tmp/tracy-gocache go test ./bench -run TestWorkloadSmoke -v
```

建议在记录结果时同时保存：机器信息、Go/SQLite 版本、batch size、WAL 配置、p50/p95/p99 查询延迟、RSS、数据库大小和 WAL 大小。正式比较 DuckDB 前，先用相同 workload 建立 SQLite 基线。
