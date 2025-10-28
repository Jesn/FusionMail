# FusionMail 数据库迁移

## 概述

FusionMail 使用 GORM 的 AutoMigrate 功能进行数据库表结构的自动迁移。此目录包含 SQL 迁移脚本作为文档和参考。

## 迁移方式

### 方式一：使用迁移工具（推荐）

```bash
cd backend

# 执行迁移（创建/更新表结构）
go run cmd/migrate/main.go -action=up

# 检查数据库状态
go run cmd/migrate/main.go -action=status
```

### 方式二：启动服务器时自动迁移

服务器启动时会自动执行数据库迁移：

```bash
cd backend
go run cmd/server/main.go
```

### 方式三：手动执行 SQL（不推荐）

如果需要手动执行 SQL 脚本：

```bash
# 连接到 PostgreSQL
docker exec -it fusionmail-postgres psql -U fusionmail -d fusionmail

# 执行 SQL 文件
\i /path/to/001_create_tables.sql
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

1. **自动迁移**：GORM AutoMigrate 会自动创建表和索引，但不会删除列或表
2. **数据安全**：迁移前建议备份数据库
3. **全文搜索**：PostgreSQL 全文搜索索引在 AutoMigrate 后单独创建
4. **软删除**：`accounts` 表使用软删除（`deleted_at` 字段）
5. **外键约束**：部分表设置了外键约束，删除时会级联删除相关数据

## 回滚

GORM AutoMigrate 不支持自动回滚。如需回滚，请：

1. 恢复数据库备份
2. 或手动删除表：

```sql
DROP TABLE IF EXISTS webhook_logs CASCADE;
DROP TABLE IF EXISTS webhooks CASCADE;
DROP TABLE IF EXISTS sync_logs CASCADE;
DROP TABLE IF EXISTS email_label_relations CASCADE;
DROP TABLE IF EXISTS email_labels CASCADE;
DROP TABLE IF EXISTS email_attachments CASCADE;
DROP TABLE IF EXISTS email_rules CASCADE;
DROP TABLE IF EXISTS emails CASCADE;
DROP TABLE IF EXISTS accounts CASCADE;
DROP TABLE IF EXISTS api_keys CASCADE;
DROP TABLE IF EXISTS users CASCADE;
```

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
