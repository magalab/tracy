# Tracy 开发规则

- 默认保持单 Go binary + 两个 SQLite 文件的部署模型。
- Metadata 与 Trace 存储必须保持独立；不要把 SQL wrapper 暴露到业务层。
- HTTP DTO 不得直接作为 domain 或 storage entity。
- 新 endpoint 必须有黑盒 contract test；新 storage backend 必须复用同一 contract 和 benchmark。
- ingest 的成功语义是进入有界队列，不是已经落盘；队列满必须返回可识别的错误。
- API Key 只存 hash，明文只在创建时交付一次。
- 暂不引入 Redis、MQ、ClickHouse、MinIO 或分布式锁。
- 运行测试时如果默认 Go cache 不可写，使用 `GOCACHE=/private/tmp/tracy-gocache`。
