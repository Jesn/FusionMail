# FusionMail 启动脚本使用指南

## 📋 脚本概述

本项目提供了三个便捷的启动脚本，用于管理 FusionMail 项目的完整生命周期：

- `start.sh` - 完整启动脚本
- `stop.sh` - 优雅停止脚本  
- `restart.sh` - 重启脚本

## 🚀 快速开始

### 启动项目

```bash
./start.sh
```

### 停止项目

```bash
./stop.sh
```

### 重启项目

```bash
./restart.sh
```

## 📖 详细说明

### start.sh - 启动脚本

**功能特性：**
- ✅ 自动检查系统依赖 (Docker, Node.js, Go, lsof)
- ✅ 智能端口冲突检测和处理
- ✅ 自动终止占用端口的进程
- ✅ 按顺序启动所有服务
- ✅ 健康检查确保服务正常运行
- ✅ 详细的启动日志和状态显示

**启动顺序：**
1. 检查系统依赖
2. 检查并清理端口占用
3. 启动基础设施 (PostgreSQL + Redis)
4. 启动后端服务 (Go API)
5. 启动前端服务 (React + Vite)

**涉及端口：**
- `3000` - 前端开发服务器
- `8080` - 后端 API 服务
- `5432` - PostgreSQL 数据库
- `6379` - Redis 缓存

### stop.sh - 停止脚本

**功能特性：**
- ✅ 优雅停止所有服务
- ✅ 自动清理进程 PID 文件
- ✅ 强制终止无响应进程
- ✅ 停止 Docker 容器
- ✅ 清理端口占用

**停止顺序：**
1. 停止前端服务
2. 停止后端服务
3. 停止基础设施服务
4. 清理端口占用

### restart.sh - 重启脚本

**功能特性：**
- ✅ 调用 stop.sh 停止服务
- ✅ 等待 3 秒确保完全停止
- ✅ 调用 start.sh 重新启动
- ✅ 完整的重启流程

## 🔧 系统要求

### 必需依赖

- **Docker** - 用于运行 PostgreSQL 和 Redis
- **Docker Compose** - 用于管理容器编排
- **Node.js** (≥20.19.0) - 用于运行前端服务
- **Go** (≥1.21) - 用于构建和运行后端服务
- **lsof** - 用于端口检查 (通常系统自带)

### 安装依赖

**macOS:**
```bash
# 安装 Docker Desktop
brew install --cask docker

# 安装 Node.js
brew install node

# 安装 Go
brew install go
```

**Ubuntu/Debian:**
```bash
# 安装 Docker
curl -fsSL https://get.docker.com -o get-docker.sh
sudo sh get-docker.sh

# 安装 Docker Compose
sudo apt-get install docker-compose

# 安装 Node.js
curl -fsSL https://deb.nodesource.com/setup_20.x | sudo -E bash -
sudo apt-get install -y nodejs

# 安装 Go
sudo apt-get install golang-go
```

## 📊 服务信息

### 启动后的访问地址

| 服务 | 地址 | 说明 |
|------|------|------|
| 前端界面 | http://localhost:3000 | React 应用主界面 |
| 后端 API | http://localhost:8080 | Go API 服务 |
| 健康检查 | http://localhost:8080/api/v1/health | API 健康状态 |

### 默认账号信息

| 项目 | 账号/用户名 | 密码 |
|------|-------------|------|
| 管理员账号 | admin@fusionmail.local | FusionMail2024! |
| PostgreSQL | fusionmail | fusionmail_dev_password |
| Redis | - | fusionmail_redis_password |

### 数据库连接信息

**PostgreSQL:**
```
Host: localhost
Port: 5432
Database: fusionmail
Username: fusionmail
Password: fusionmail_dev_password
URL: postgresql://fusionmail:fusionmail_dev_password@localhost:5432/fusionmail
```

**Redis:**
```
Host: localhost
Port: 6379
Password: fusionmail_redis_password
URL: redis://:fusionmail_redis_password@localhost:6379/0
```

## 📝 日志管理

### 日志文件位置

- `logs/frontend.log` - 前端服务日志
- `logs/backend.log` - 后端服务日志
- Docker 日志通过 `docker-compose logs` 查看

### 查看日志命令

```bash
# 查看前端日志
tail -f logs/frontend.log

# 查看后端日志
tail -f logs/backend.log

# 查看所有应用日志
tail -f logs/*.log

# 查看 Docker 服务日志
docker-compose -f docker-compose.dev.yml logs -f

# 查看特定服务日志
docker-compose -f docker-compose.dev.yml logs -f postgres
docker-compose -f docker-compose.dev.yml logs -f redis
```

## 🛠️ 故障排除

### 常见问题

**1. 端口被占用**
```bash
# 脚本会自动检测并询问是否终止占用进程
# 也可以手动检查端口占用
lsof -i :3000
lsof -i :8080
lsof -i :5432
lsof -i :6379
```

**2. Docker 服务未启动**
```bash
# 启动 Docker 服务
sudo systemctl start docker  # Linux
# 或启动 Docker Desktop (macOS/Windows)
```

**3. 依赖缺失**
```bash
# 脚本会自动检查并提示缺失的依赖
# 按照提示安装相应依赖即可
```

**4. 服务启动超时**
```bash
# 检查日志文件
tail -f logs/backend.log
tail -f logs/frontend.log

# 检查 Docker 服务状态
docker-compose -f docker-compose.dev.yml ps
```

### 手动操作

**手动停止服务:**
```bash
# 停止应用服务
pkill -f "fusionmail"
pkill -f "vite"

# 停止 Docker 服务
docker-compose -f docker-compose.dev.yml down
```

**手动清理:**
```bash
# 清理日志文件
rm -f logs/*.log
rm -f logs/*.pid

# 清理 Docker 数据 (谨慎操作)
docker-compose -f docker-compose.dev.yml down -v
```

## 🔄 开发工作流

### 日常开发

```bash
# 启动开发环境
./start.sh

# 开发过程中...
# 前端代码会自动热重载
# 后端代码修改后需要重启

# 重启后端服务
./restart.sh

# 完成开发后停止
./stop.sh
```

### 生产部署

```bash
# 生产环境建议使用 Docker Compose
docker-compose up -d

# 或使用专门的生产启动脚本 (需要创建)
./start-prod.sh
```

## 📞 技术支持

如果遇到问题：

1. 查看日志文件确定错误原因
2. 检查系统依赖是否完整安装
3. 确认端口没有被其他程序占用
4. 重启 Docker 服务
5. 联系技术支持团队

---

**享受使用 FusionMail！** 🎉