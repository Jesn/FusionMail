---
inclusion: conditional
fileMatchPattern: "**/{docker,script,setup,config}/**/*"
---

# FusionMail 开发环境配置

## 快速启动

### 一键启动基础设施

项目根目录提供了快速启动脚本，可以一键启动所有基础设施服务（PostgreSQL、Redis）。

**启动命令**：
```bash
# 在项目根目录执行
./scripts/dev-start.sh

# 或者使用 npm/yarn（如果配置了）
npm run dev:infra
```

**停止命令**：
```bash
./scripts/dev-stop.sh

# 或者
npm run dev:infra:stop
```

### Docker Compose 配置

**配置文件位置**：`docker-compose.dev.yml`

**手动启动**：
```bash
# 启动基础设施（PostgreSQL + Redis）
docker-compose -f docker-compose.dev.yml up -d

# 查看日志
docker-compose -f docker-compose.dev.yml logs -f

# 停止服务
docker-compose -f docker-compose.dev.yml down

# 停止并删除数据卷（慎用）
docker-compose -f docker-compose.dev.yml down -v
```

## 基础设施服务

### PostgreSQL 数据库

**版本**：15-alpine

**连接信息**：
```bash
Host: localhost
Port: 5432
Database: fusionmail
Username: fusionmail
Password: fusionmail_dev_password
```

**连接字符串**：
```
postgresql://fusionmail:fusionmail_dev_password@localhost:5432/fusionmail?sslmode=disable
```

**管理工具**：
- **命令行**：`docker exec -it fusionmail-postgres psql -U fusionmail -d fusionmail`
- **GUI 工具**：DBeaver、pgAdmin、TablePlus 等

**数据持久化**：
- 数据卷：`fusionmail_postgres_data`
- 挂载路径：`./data/postgres`（可选）

### Redis 缓存

**版本**：7-alpine

**连接信息**：
```bash
Host: localhost
Port: 6379
Password: fusionmail_redis_password
Database: 0
```

**连接字符串**：
```
redis://:fusionmail_redis_password@localhost:6379/0
```

**管理工具**：
- **命令行**：`docker exec -it fusionmail-redis redis-cli -a fusionmail_redis_password`
- **GUI 工具**：RedisInsight、Another Redis Desktop Manager

**数据持久化**：
- 数据卷：`fusionmail_redis_data`
- 持久化策略：RDB + AOF

## 环境变量配置

### 后端环境变量

**文件位置**：`backend/.env`

**示例配置**：
```bash
# 数据库配置
DATABASE_URL=postgresql://fusionmail:fusionmail_dev_password@localhost:5432/fusionmail?sslmode=disable
DATABASE_MAX_OPEN_CONNS=25
DATABASE_MAX_IDLE_CONNS=5

# Redis 配置
REDIS_URL=redis://:fusionmail_redis_password@localhost:6379/0
REDIS_MAX_RETRIES=3

# 服务器配置
SERVER_PORT=8080
SERVER_HOST=0.0.0.0
SERVER_ENV=development

# JWT 配置
JWT_SECRET=your-development-jwt-secret-key-change-in-production
JWT_EXPIRY=24h

# 加密配置
ENCRYPTION_KEY=your-32-byte-encryption-key-dev

# 日志配置
LOG_LEVEL=debug
LOG_FORMAT=json

# 存储配置
STORAGE_TYPE=local
STORAGE_PATH=./data/attachments

# 同步配置
SYNC_WORKER_COUNT=5
SYNC_DEFAULT_INTERVAL=5
```

### 前端环境变量

**文件位置**：`frontend/.env`

**示例配置**：
```bash
VITE_API_BASE_URL=http://localhost:8080/api/v1
VITE_WS_URL=ws://localhost:8080/ws
VITE_APP_NAME=FusionMail
VITE_APP_VERSION=1.0.0
```

## 开发工作流

### 1. 初始化项目

```bash
# 克隆项目
git clone <repository-url>
cd fusionmail

# 启动基础设施
./scripts/dev-start.sh

# 等待服务启动（约 10-15 秒）
docker-compose -f docker-compose.dev.yml ps

# 初始化数据库
cd backend
go run cmd/server/main.go migrate

# 或者使用迁移脚本
./scripts/migrate.sh
```

### 2. 启动后端开发服务器

```bash
cd backend

# 安装依赖
go mod download

# 运行开发服务器
go run cmd/server/main.go

# 或者使用 air 热重载（推荐）
air
```

### 3. 启动前端开发服务器

```bash
cd frontend

# 安装依赖
npm install
# 或
yarn install

# 运行开发服务器
npm run dev
# 或
yarn dev
```

### 4. 访问应用

- **前端**：http://localhost:3000
- **后端 API**：http://localhost:8080/api/v1
- **API 文档**：http://localhost:8080/swagger/index.html（如果配置了 Swagger）

## 数据库管理

### 数据库迁移

**迁移文件位置**：`backend/migrations/`

**执行迁移**：
```bash
# 使用 Go 代码执行迁移
cd backend
go run cmd/server/main.go migrate

# 或者使用迁移脚本
./scripts/migrate.sh up

# 回滚迁移
./scripts/migrate.sh down
```

**创建新迁移**：
```bash
./scripts/migrate.sh create <migration_name>
```

### 数据库备份与恢复

**备份**：
```bash
# 备份数据库
docker exec fusionmail-postgres pg_dump -U fusionmail fusionmail > backup.sql

# 或使用脚本
./scripts/db-backup.sh
```

**恢复**：
```bash
# 恢复数据库
docker exec -i fusionmail-postgres psql -U fusionmail fusionmail < backup.sql

# 或使用脚本
./scripts/db-restore.sh backup.sql
```

### 重置数据库

```bash
# 停止服务
docker-compose -f docker-compose.dev.yml down

# 删除数据卷
docker volume rm fusionmail_postgres_data

# 重新启动
docker-compose -f docker-compose.dev.yml up -d

# 等待启动后执行迁移
./scripts/migrate.sh up
```

## Redis 管理

### 清空缓存

```bash
# 清空所有缓存
docker exec fusionmail-redis redis-cli -a fusionmail_redis_password FLUSHALL

# 清空当前数据库
docker exec fusionmail-redis redis-cli -a fusionmail_redis_password FLUSHDB

# 查看所有键
docker exec fusionmail-redis redis-cli -a fusionmail_redis_password KEYS '*'
```

### 监控 Redis

```bash
# 实时监控命令
docker exec fusionmail-redis redis-cli -a fusionmail_redis_password MONITOR

# 查看信息
docker exec fusionmail-redis redis-cli -a fusionmail_redis_password INFO
```

## 故障排查

### 服务无法启动

**检查端口占用**：
```bash
# 检查 PostgreSQL 端口
lsof -i :5432

# 检查 Redis 端口
lsof -i :6379

# 检查后端端口
lsof -i :8080

# 检查前端端口
lsof -i :3000
```

**查看服务日志**：
```bash
# 查看所有服务日志
docker-compose -f docker-compose.dev.yml logs

# 查看特定服务日志
docker-compose -f docker-compose.dev.yml logs postgres
docker-compose -f docker-compose.dev.yml logs redis
```

### 数据库连接失败

**检查数据库状态**：
```bash
# 检查容器状态
docker ps | grep fusionmail-postgres

# 测试连接
docker exec fusionmail-postgres pg_isready -U fusionmail

# 查看数据库日志
docker logs fusionmail-postgres
```

### Redis 连接失败

**检查 Redis 状态**：
```bash
# 检查容器状态
docker ps | grep fusionmail-redis

# 测试连接
docker exec fusionmail-redis redis-cli -a fusionmail_redis_password PING

# 查看 Redis 日志
docker logs fusionmail-redis
```

## 性能优化建议

### PostgreSQL 优化

**配置调整**（在 `docker-compose.dev.yml` 中）：
```yaml
environment:
  - POSTGRES_SHARED_BUFFERS=256MB
  - POSTGRES_EFFECTIVE_CACHE_SIZE=1GB
  - POSTGRES_WORK_MEM=16MB
  - POSTGRES_MAINTENANCE_WORK_MEM=128MB
```

### Redis 优化

**配置调整**（在 `docker-compose.dev.yml` 中）：
```yaml
command: >
  redis-server
  --maxmemory 512mb
  --maxmemory-policy allkeys-lru
  --save 60 1000
  --appendonly yes
```

## 开发工具推荐

### 数据库工具
- **DBeaver**：免费、跨平台、功能强大
- **TablePlus**：界面美观、性能好（macOS/Windows）
- **pgAdmin**：PostgreSQL 官方工具

### Redis 工具
- **RedisInsight**：Redis 官方工具，功能全面
- **Another Redis Desktop Manager**：开源、跨平台
- **Medis**：macOS 专用，界面简洁

### API 测试工具
- **Postman**：功能强大、团队协作
- **Insomnia**：界面简洁、开源
- **HTTPie**：命令行工具

### 日志查看工具
- **Lnav**：命令行日志查看器
- **Papertrail**：云端日志管理
- **Docker Desktop**：内置日志查看

## 常用命令速查

```bash
# 启动所有服务
./scripts/dev-start.sh

# 停止所有服务
./scripts/dev-stop.sh

# 查看服务状态
docker-compose -f docker-compose.dev.yml ps

# 查看日志
docker-compose -f docker-compose.dev.yml logs -f

# 重启服务
docker-compose -f docker-compose.dev.yml restart

# 进入 PostgreSQL
docker exec -it fusionmail-postgres psql -U fusionmail -d fusionmail

# 进入 Redis
docker exec -it fusionmail-redis redis-cli -a fusionmail_redis_password

# 数据库迁移
./scripts/migrate.sh up

# 数据库备份
./scripts/db-backup.sh

# 清空 Redis 缓存
docker exec fusionmail-redis redis-cli -a fusionmail_redis_password FLUSHALL
```

---

**注意**：
1. 开发环境的密码仅用于本地开发，生产环境必须使用强密码
2. 定期备份开发数据库，避免数据丢失
3. 使用 `.env` 文件管理环境变量，不要提交到 Git
4. 遇到问题先查看日志，大部分问题都能从日志中找到原因
