# 开发环境数据库迁移说明

## 📋 变更概述

将 FusionMail 开发环境从本地 Docker 数据库迁移到远程数据库服务器。

### 变更内容

- **PostgreSQL**: 从本地 Docker 容器迁移到 `192.168.2.200:5432`
- **Redis**: 从本地 Docker 容器迁移到 `192.168.2.200:6379` (DB 6)
- **数据库名称**: `fusionmail-dev`

## 🔧 配置变更

### 1. 后端环境变量 (`backend/.env`)

```bash
# 数据库配置
DB_HOST=192.168.2.200
DB_PORT=5432
DB_USER=postgres
DB_PASSWORD=8QMZn3yfrbkVG7
DB_NAME=fusionmail-dev
DB_SSLMODE=disable

# Redis 配置
REDIS_HOST=192.168.2.200
REDIS_PORT=6379
REDIS_PASSWORD=
REDIS_DB=6
```

### 2. Docker Compose 配置 (`docker-compose.yml`)

- 已注释掉本地 PostgreSQL 和 Redis 容器
- 应用容器环境变量已更新为远程数据库配置

### 3. 启动脚本 (`start.sh`)

- 移除了本地 Docker 容器的启动和检查逻辑
- 移除了 Docker 和 Docker Compose 依赖检查
- 更新了数据库连接信息显示

## 🚀 使用步骤

### 步骤 1: 创建远程数据库

使用以下任一方法创建数据库：

**方法 1: 使用 psql 命令行**

```bash
# 如果已安装 psql
PGPASSWORD=8QMZn3yfrbkVG7 psql -h 192.168.2.200 -p 5432 -U postgres -c "CREATE DATABASE \"fusionmail-dev\" OWNER postgres;"
```

**方法 2: 使用 SQL 脚本**

```bash
# 执行创建脚本
PGPASSWORD=8QMZn3yfrbkVG7 psql -h 192.168.2.200 -p 5432 -U postgres -f scripts/create-dev-database.sql
```

**方法 3: 使用数据库管理工具**

使用 pgAdmin、DBeaver 或其他数据库管理工具连接到服务器并创建数据库：

- 主机: `192.168.2.200`
- 端口: `5432`
- 用户: `postgres`
- 密码: `8QMZn3yfrbkVG7`
- 数据库名: `fusionmail-dev`

### 步骤 2: 验证 Redis 连接

确保 Redis 服务器可访问：

```bash
# 使用 redis-cli 测试（如果已安装）
redis-cli -h 192.168.2.200 -p 6379 -n 6 ping

# 应该返回: PONG
```

### 步骤 3: 启动项目

```bash
# 完整启动
./start.sh

# 或仅启动后端
./start.sh -b

# 或仅启动前端
./start.sh -f
```

## 📊 数据库连接信息

### PostgreSQL

- **主机**: 192.168.2.200
- **端口**: 5432
- **用户**: postgres
- **密码**: 8QMZn3yfrbkVG7
- **数据库**: fusionmail-dev
- **连接字符串**: `postgresql://postgres:8QMZn3yfrbkVG7@192.168.2.200:5432/fusionmail-dev`

### Redis

- **主机**: 192.168.2.200
- **端口**: 6379
- **密码**: (无)
- **数据库**: 6
- **连接字符串**: `redis://192.168.2.200:6379/6`

## 🔍 验证步骤

### 1. 检查数据库是否创建成功

```bash
PGPASSWORD=8QMZn3yfrbkVG7 psql -h 192.168.2.200 -p 5432 -U postgres -l | grep fusionmail-dev
```

### 2. 检查后端是否能连接数据库

启动后端后，查看日志：

```bash
tail -f logs/backend.log
```

应该看到类似以下的成功日志：

```
[INFO] 数据库连接成功
[INFO] 数据库迁移完成
[INFO] 服务器启动在 :3333
```

### 3. 检查健康检查接口

```bash
curl http://localhost:3333/api/v1/health
```

应该返回：

```json
{
  "status": "ok",
  "database": "connected",
  "redis": "connected"
}
```

## ⚠️ 注意事项

### 数据隔离

- 使用独立的数据库名称 `fusionmail-dev` 避免与其他环境冲突
- Redis 使用 DB 6，避免与其他应用的数据混淆

### 网络访问

- 确保开发机器能够访问 `192.168.2.200`
- 检查防火墙规则是否允许 5432 和 6379 端口

### 数据备份

- 远程数据库应该有定期备份策略
- 开发环境数据可能会被清理，不要存储重要数据

### 清理数据

如需清理开发数据库：

```bash
# 删除并重新创建数据库
PGPASSWORD=8QMZn3yfrbkVG7 psql -h 192.168.2.200 -p 5432 -U postgres -c "DROP DATABASE IF EXISTS \"fusionmail-dev\";"
PGPASSWORD=8QMZn3yfrbkVG7 psql -h 192.168.2.200 -p 5432 -U postgres -c "CREATE DATABASE \"fusionmail-dev\" OWNER postgres;"
```

清理 Redis 数据：

```bash
# 清理 DB 6 的所有数据
redis-cli -h 192.168.2.200 -p 6379 -n 6 FLUSHDB
```

## 🐛 故障排查

### 问题 1: 无法连接到数据库

**症状**: 后端启动失败，日志显示数据库连接错误

**解决方案**:

1. 检查网络连接：`ping 192.168.2.200`
2. 检查端口是否开放：`telnet 192.168.2.200 5432`
3. 验证数据库凭证是否正确
4. 检查 PostgreSQL 服务器的 `pg_hba.conf` 是否允许远程连接

### 问题 2: Redis 连接失败

**症状**: 后端启动成功但 Redis 功能异常

**解决方案**:

1. 检查 Redis 端口：`telnet 192.168.2.200 6379`
2. 验证 Redis 是否需要密码
3. 确认使用的是 DB 6

### 问题 3: 数据库已存在

**症状**: 创建数据库时提示已存在

**解决方案**:

```bash
# 如果需要重新创建，先删除
PGPASSWORD=8QMZn3yfrbkVG7 psql -h 192.168.2.200 -p 5432 -U postgres -c "DROP DATABASE IF EXISTS \"fusionmail-dev\";"

# 然后重新创建
PGPASSWORD=8QMZn3yfrbkVG7 psql -h 192.168.2.200 -p 5432 -U postgres -f scripts/create-dev-database.sql
```

## 📝 回滚方案

如需回滚到本地 Docker 数据库：

1. 恢复 `backend/.env` 中的数据库配置为 `localhost`
2. 取消注释 `docker-compose.yml` 中的 PostgreSQL 和 Redis 服务
3. 恢复 `start.sh` 中的 Docker 容器启动逻辑
4. 运行 `docker-compose -f docker-compose.dev.yml up -d`

## 🎯 下一步

配置完成后，你可以：

1. 启动项目：`./start.sh`
2. 访问前端：http://localhost:4444
3. 访问 API：http://localhost:3333
4. 查看 API 文档：http://localhost:3333/docs (如果启用了 Swagger)

---

**更新时间**: 2024-11-29  
**维护者**: FusionMail Team
