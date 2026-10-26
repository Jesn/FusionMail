# FusionMail 数据库迁移

## 目录约定

```
migrations/
├── 000001_baseline.up.sql      # schema 起点（唯一真相）
├── 000001_baseline.down.sql
├── <后续版本>.up.sql / .down.sql
├── legacy/                     # 历史 SQL，已失效，仅存档，不执行
├── manual/                     # 一次性手工修复脚本，不自动执行
└── maintenance/                # 运维脚本（索引整理、日志表优化等）
```

**根目录只放 golang-migrate 的版本化文件**，命名必须是
`{version}_{name}.up.sql` / `{version}_{name}.down.sql`。
迁移器会扫描整个根目录，任何非版本化文件都会让 `migrate -action=up` 直接失败。
`cmd/migrate/main_test.go` 会强制校验这条约定。

## 为什么换成 golang-migrate

2026-10 的架构体检发现，此前的迁移体系**没有任何单一真相**：

| 问题 | 证据 |
|------|------|
| 39 个 SQL 迁移**从未被任何代码执行** | 全仓 grep 无 Go 代码读取 `migrations/*.sql`；唯一入口 `migrate.sh` 只跑 001-003 且硬编码本地容器名 |
| 这套迁移链**无法从零建库** | 实测按序执行：24 成功 / 15 失败。`001_create_tables.sql:97` 用 PostgreSQL 保留字 `references` 作列名导致语法错误，后续全部级联失败；`033:127` 的 `DO $` 少了 `$$` |
| 线上 schema 实际由 **AutoMigrate 推导** | `migrations/039` 自己的注释承认「线上 migrate 工具走 GORM AutoMigrate，此前未注册该模型导致表缺失」 |
| 同表两种类型 | `030:43` = `account_uid UUID` vs `039:10` = `VARCHAR(64)`，而真实 UID 含 `webhook_*` 字符串 |
| 无版本表 / 无 checksum / 无 down | 无法审计、无法回滚 |
| **每次部署都在跑 AutoMigrate + seed** | `fly.toml` 的 `release_command` 指向 `migrate -action=up`，而该命令当时无条件调用 `AutoMigrate()` + `SeedInitialData()`，绕过 `ENABLE_AUTO_MIGRATE` |

## baseline 的来源

`000001_baseline.up.sql` 是对「GORM AutoMigrate 在干净 PostgreSQL 16 上产出的真实
schema」执行 `pg_dump --schema-only` 导出的结果，并做了语义级校验：

```
AutoMigrate 库 vs baseline 库
  列   350 / 350  ✅ 完全一致
  表    24 /  24  ✅
  索引  98 /  98  ✅
  约束  36 /  36  ✅
  序列  22 /  22  ✅
```

唯一差异是 5 个 CHECK 约束的文本表示（`ANY(ARRAY[...])` 的两种等价写法），语义相同。

## 执行方式

```bash
cd backend

# 应用全部待执行的版本化迁移（幂等）
go run cmd/migrate/main.go -action=up

# 查看当前 schema 版本与 dirty 标记
go run cmd/migrate/main.go -action=status

# 回滚（仅测试环境；生产请用备份恢复）
go run cmd/migrate/main.go -action=down -steps=1

# 仅初始化���子数据
go run cmd/migrate/main.go -action=seed

# 仅开发/测试：从 GORM 模型推导 schema（release 模式一律拒绝）
go run cmd/migrate/main.go -action=automigrate
```

环境变量 `MIGRATIONS_DIR` 可显式指定迁移目录；未设置时按
`migrations` → `backend/migrations` → `../migrations` 顺序探测。

## AutoMigrate 的边界

`AutoMigrate` 从 GORM 模型推导 schema，**绕过版本化迁移，使结构变更无法审计与回滚**。
因此它的适用边界是：

- ✅ 开发与测试环境（`GIN_MODE=debug`，且 `ENABLE_AUTO_MIGRATE` 未显式关闭）
- ❌ release 模式**一律拒绝**，即使显式设置 `ENABLE_AUTO_MIGRATE=true` 也不放行
- ❌ 生产 schema 变更只能通过新增版本化迁移文件

## 新增迁移的��程

1. 在 `migrations/` 下新建 `{下一个版本号}_{描述}.up.sql` 与 `.down.sql`
2. `up` 写正向变更，`down` 写可逆回滚（无法回滚的破坏性操作请在注释里说明）
3. 本地验证：
   ```bash
   go run cmd/migrate/main.go -action=up
   go run cmd/migrate/main.go -action=down -steps=1
   go run cmd/migrate/main.go -action=up
   ```
4. `go test ./cmd/migrate/` 会校验命名、版本唯一性与 up/down 成对

## 手工 SQL

手工 SQL 只用于受控维护，不得替代版本化迁移。执行前必须备份数据库，并记录执行环境、时间和原因。

```bash
psql -h localhost -U fusionmail -d fusionmail -f migrations/manual/add_account_status.sql
psql -h localhost -U fusionmail -d fusionmail -f migrations/maintenance/optimize_log_tables.sql
```

## 历史文件

`legacy/` 下的 39 个 `NNN_*.sql` 是 2025-10 ~ 2026-06 期间编写但从未执行的迁移。
保留它们是为了追溯设计意图，**不要执行**——它们无法从零建库，且部分脚本引用的表名
（`accounts`、`oauth2_clients`）早已被重命名为 `email_accounts`、`email_oauth2_tokens`。

`legacy/insert_oauth2_clients_from_env.sql` 内含真实 Google OAuth client_id，
属于历史遗留的凭据入库问题，已记录在 2.0 的 P0 修复清单中。

## 注意事项

1. **版本化优先**：业务 schema 变更必须新增 `{version}_{name}.up.sql` / `.down.sql`。
2. **AutoMigrate 边界**：只���为开发与测试入口，不能替代生产迁移审计。
3. **数据安全**：生产迁移前必须备份数据库，并在测试环境验证。
4. **回滚责任**：每个破坏性变更必须在 `.down.sql` 或注释中写明回滚方式。

## 参考文档

- [golang-migrate 文档](https://github.com/golang-migrate/migrate)
- [GORM 文档](https://gorm.io/docs/)
- [PostgreSQL 文档](https://www.postgresql.org/docs/)
