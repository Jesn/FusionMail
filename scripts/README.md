# FusionMail 开发环境脚本

## 📋 脚本列表

### 1. setup-dev-db.sh - 开发数据库快速设置

自动创建和配置开发环境数据库。

**功能**：
- 检查网络连接和端口可用性
- 创建 PostgreSQL 数据库
- 安装必要的数据库扩展
- 验证数据库连接

**使用方法**：

```bash
# 赋予执行权限（首次使用）
chmod +x scripts/setup-dev-db.sh

# 运行脚本
./scripts/setup-dev-db.sh
```

**前置要求**：
- 已安装 PostgreSQL 客户端 (psql)
- 能够访问远程数据库服务器 (192.168.2.200)

**安装 PostgreSQL 客户端**：

```bash
# macOS
brew install postgresql

# Ubuntu/Debian
sudo apt-get install postgresql-client

# CentOS/RHEL
sudo yum install postgresql
```

### 2. create-dev-database.sql - SQL 创建脚本

手动执行的 SQL 脚本，用于创建数据库。

**使用方法**：

```bash
# 使用 psql 执行
PGPASSWORD=8QMZn3yfrbkVG7 psql -h 192.168.2.200 -p 5432 -U postgres -f scripts/create-dev-database.sql
```

**或者使用数据库管理工具**：
- 连接到 PostgreSQL 服务器
- 打开 `scripts/create-dev-database.sql`
- 执行 SQL 脚本

## 🚀 快速开始

### 完整设置流程

```bash
# 1. 创建开发数据库
./scripts/setup-dev-db.sh

# 2. 启动项目
./start.sh

# 3. 访问应用
# 前端: http://localhost:4444
# API: http://localhost:3333
```

### 仅创建数据库

如果你只需要创建数据库（不启动项目）：

```bash
./scripts/setup-dev-db.sh
```

### 重新创建数据库

如果需要清空数据并重新开始：

```bash
# 脚本会提示是否删除现有数据库
./scripts/setup-dev-db.sh

# 或者手动删除
PGPASSWORD=8QMZn3yfrbkVG7 psql -h 192.168.2.200 -p 5432 -U postgres -c "DROP DATABASE IF EXISTS \"fusionmail-dev\";"

# 然后重新创建
./scripts/setup-dev-db.sh
```

## 🔧 配置信息

### 数据库配置

- **主机**: 192.168.2.200
- **端口**: 5432
- **用户**: postgres
- **密码**: 8QMZn3yfrbkVG7
- **数据库**: fusionmail-dev

### Redis 配置

- **主机**: 192.168.2.200
- **端口**: 6379
- **数据库**: 6
- **密码**: (无)

## 📝 注意事项

1. **网络访问**: 确保你的开发机器能够访问 192.168.2.200
2. **防火墙**: 确保端口 5432 (PostgreSQL) 和 6379 (Redis) 已开放
3. **数据隔离**: 使用独立的数据库名称避免与其他环境冲突
4. **数据备份**: 开发环境数据可能会被清理，不要存储重要数据

## 🐛 故障排查

### 问题：无法连接到数据库服务器

```bash
# 检查网络连接
ping 192.168.2.200

# 检查端口是否开放
nc -z 192.168.2.200 5432
telnet 192.168.2.200 5432
```

### 问题：psql 命令未找到

安装 PostgreSQL 客户端：

```bash
# macOS
brew install postgresql

# Ubuntu/Debian
sudo apt-get install postgresql-client

# CentOS/RHEL
sudo yum install postgresql
```

### 问题：权限不足

确保使用的数据库用户有创建数据库的权限：

```bash
# 使用 postgres 超级用户
PGPASSWORD=8QMZn3yfrbkVG7 psql -h 192.168.2.200 -p 5432 -U postgres
```

## 📚 相关文档

- [开发环境数据库迁移说明](../docs/dev-database-migration.md)
- [项目启动脚本说明](../start.sh)
- [Docker Compose 配置](../docker-compose.yml)

---

**更新时间**: 2024-11-29  
**维护者**: FusionMail Team
