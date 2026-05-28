# FusionMail 数据库迁移

## 目录约定

`backend/migrations` 根目录只存放版本化迁移文件，命名必须为 `NNN_*.sql`，编号不可重复。非版本化 SQL 必须放入子目录：

- `manual/`：一次性手工修复脚本，不会被常规迁移流程自动执行。
- `maintenance/`：运维维护脚本，例如索引整理、日志表优化，不表达业务 schema 版本。

## 执行方式

### 显式迁移命令

```bash
cd backend

# 创建或更新 GORM 模型覆盖的表结构，并初始化种子数据
go run cmd/migrate/main.go -action=up

# 检查数据库状态
go run cmd/migrate/main.go -action=status
```

### 启动时迁移策略

服务启动默认不在 release/test 模式执行 AutoMigrate，只检查数据库结构是否满足当前运行要求。开发模式默认允许 AutoMigrate，或显式设置：

```bash
cd backend
ENABLE_AUTO_MIGRATE=true go run cmd/server/main.go
```

### 手工 SQL

手工 SQL 只用于受控维护，不得替代版本化迁移。执行前必须备份数据库，并记录执行环境、时间和原因。

```bash
# 一次性修复脚本
psql -h localhost -U fusionmail -d fusionmail -f migrations/manual/add_account_status.sql

# 运维维护脚本
psql -h localhost -U fusionmail -d fusionmail -f migrations/maintenance/optimize_log_tables.sql
```

## 迁移内容

### 001_create_tables.sql

创建所有核心数据表：

- `users` - 用户表
- `accounts` - 邮箱账户表
- `emails` - 邮件主表
- `email_attachments` - 邮件附件表
- `email_labels` - 邮件标签表
- `email_label_relations` - 邮件-标签关联表
- `email_rules` - 邮件规则表
- `webhooks` - Webhook 配置表
- `webhook_logs` - Webhook 调用日志表
- `sync_logs` - 同步日志表
- `api_keys` - API 密钥表

## 数据库索引

### 主要索引

- **唯一索引**：
  - `users.username`
  - `users.email`
  - `accounts.uid`
  - `emails(provider_id, account_uid)` - 复合唯一索引
  - `email_labels.name`
  - `api_keys.key_hash`

- **查询索引**：
  - `emails.account_uid`
  - `emails.from_address`
  - `emails.sent_at` (降序)
  - `emails.is_read`
  - `emails.is_starred`
  - `emails.is_archived`
  - `emails.is_deleted`

- **全文搜索索引**：
  - `emails` - GIN 索引，用于全文搜索（subject + from_name + text_body）

## 环境变量

数据库连接配置通过环境变量设置：

```bash
DB_HOST=localhost
DB_PORT=5432
DB_USER=fusionmail
DB_PASSWORD=fusionmail_password
DB_NAME=fusionmail
DB_SSLMODE=disable
```

## 注意事项

1. **版本化优先**：业务 schema 变更应新增 `NNN_*.sql` 文件，禁止把新业务变更写进 `manual/` 或 `maintenance/`。
2. **AutoMigrate 边界**：`database.AutoMigrate()` 只作为开发和显式维护入口，不能替代生产迁移审计。
3. **数据安全**：生产迁移前必须备份数据库，并在测试环境验证。
4. **全文搜索**：PostgreSQL 全文搜索索引由迁移或显式维护脚本管理。
5. **回滚责任**：每个破坏性变更必须在变更说明中写明回滚方式。

## 回滚

当前迁移命令不提供自动回滚。需要回滚时优先恢复迁移前备份；如需手写回滚 SQL，应放入 `manual/` 并在执行记录中说明原因。

禁止在没有备份和影响评估的情况下直接删除生产表或列。

## 故障排查

### 连接失败

检查数据库是否运行：

```bash
docker-compose -f docker-compose.dev.yml ps
```

### 权限问题

确保数据库用户有足够的权限：

```sql
GRANT ALL PRIVILEGES ON DATABASE fusionmail TO fusionmail;
GRANT ALL PRIVILEGES ON ALL TABLES IN SCHEMA public TO fusionmail;
```

### 索引创建失败

如果全文搜索索引创建失败，可以手动创建：

```sql
CREATE INDEX idx_emails_fulltext_search ON emails 
USING gin(to_tsvector('english', 
    coalesce(subject, '') || ' ' || 
    coalesce(from_name, '') || ' ' || 
    coalesce(text_body, '')
));
```

## 参考文档

- [GORM 文档](https://gorm.io/docs/)
- [PostgreSQL 文档](https://www.postgresql.org/docs/)
- [FusionMail 设计文档](../../.kiro/specs/fusionmail/design.md)
