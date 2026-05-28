# FusionMail 开发环境脚本

## 📋 脚本列表

### 1. dev-start.sh - 启动开发环境基础设施

启动本地 Docker 容器（PostgreSQL + Redis）。

**功能**：
- 检查 Docker 是否运行
- 启动 PostgreSQL 和 Redis 容器
- 等待服务就绪
- 显示连接信息

**使用方法**：

```bash
# 赋予执行权限（首次使用）
chmod +x scripts/dev-start.sh

# 运行脚本
./scripts/dev-start.sh
```

### 2. dev-stop.sh - 停止开发环境基础设施

停止本地 Docker 容器。

**使用方法**：

```bash
./scripts/dev-stop.sh
```

### 3. start.sh - 一键启动完整项目

启动所有服务（Docker 基础设施 + 后端 + 前端）。

**使用方法**：

```bash
# 完整启动
./start.sh

# 开发模式（监听文件变化）
./start.sh -w

# 仅启动后端
./start.sh -b

# 仅启动前端
./start.sh -f

# 清理数据后启动
./start.sh -c

# 查看帮助
./start.sh -h
```

### 4. stop.sh - 停止项目

停止所有服务。

**使用方法**：

```bash
# 停止前后端（保留 Docker）
./stop.sh

# 停止所有服务（包括 Docker）
./stop.sh -a

# 停止并清理数据
./stop.sh -c
```

### 5. check-quality.sh - 运行质量检查

执行后端和前端的本地质量门禁。

**默认检查**：
- 后端：`go test ./...`
- 前端：`npm run lint`
- 前端：`npm test`
- 前端：`npm run build`

**使用方法**：

```bash
# 运行全部检查
./scripts/check-quality.sh

# 仅检查后端
./scripts/check-quality.sh --backend-only

# 仅检查前端
./scripts/check-quality.sh --frontend-only
```

## 🚀 快速开始

### 完整设置流程

```bash
# 1. 一键启动项目（自动启动 Docker + 后端 + 前端）
./start.sh

# 2. 访问应用
# 前端: http://localhost:4444
# API: http://localhost:3333
```

### 仅启动基础设施

如果你只需要启动数据库和 Redis：

```bash
./scripts/dev-start.sh
```

### 清理数据重新开始

```bash
# 停止并清理所有数据
./stop.sh -c

# 重新启动
./start.sh
```

## 🔧 配置信息

### 数据库配置（本地 Docker）

- **主机**: localhost
- **端口**: 5432
- **用户**: fusionmail
- **密码**: fusionmail_dev_password
- **数据库**: fusionmail
- **连接字符串**: `postgresql://fusionmail:fusionmail_dev_password@localhost:5432/fusionmail`

### Redis 配置（本地 Docker）

- **主机**: localhost
- **端口**: 6379
- **密码**: fusionmail_redis_password
- **数据库**: 0
- **连接字符串**: `redis://:fusionmail_redis_password@localhost:6379/0`

## 📝 注意事项

1. **Docker 必须运行**: 确保 Docker Desktop 或 Docker 服务已启动
2. **端口占用**: 确保端口 5432、6379、3333、4444 未被占用
3. **数据持久化**: 数据存储在 Docker 卷中，停止容器不会丢失数据
4. **清理数据**: 使用 `./stop.sh -c` 会删除所有数据

## 🐛 故障排查

### 问题：Docker 未运行

```bash
# macOS
open -a Docker

# Linux
sudo systemctl start docker
```

### 问题：端口被占用

```bash
# 查看端口占用
lsof -i :5432
lsof -i :6379
lsof -i :3333
lsof -i :4444

# 终止占用进程
kill -9 <PID>
```

### 问题：容器启动失败

```bash
# 查看容器日志
docker-compose -f docker-compose.dev.yml logs

# 重新创建容器
docker-compose -f docker-compose.dev.yml down
docker-compose -f docker-compose.dev.yml up -d
```

### 问题：数据库连接失败

```bash
# 检查容器状态
docker ps | grep fusionmail

# 进入 PostgreSQL 容器
docker exec -it fusionmail-postgres psql -U fusionmail -d fusionmail

# 进入 Redis 容器
docker exec -it fusionmail-redis redis-cli -a fusionmail_redis_password
```

## 📚 相关文档

- [快速开始指南](../docs/quick-start.md)
- [Docker Compose 配置](../docker-compose.dev.yml)
- [后端环境变量](../backend/.env.example)

---

**更新时间**: 2024-12-10  
**维护者**: FusionMail Team
