# FusionMail 开发脚本

本目录包含 FusionMail 项目的开发和管理脚本。

## 快速开始

### 启动开发环境

```bash
# 在项目根目录执行
./scripts/dev-start.sh
```

这个脚本会：
- 检查 Docker 环境
- 检查端口占用情况
- 启动 PostgreSQL 和 Redis 服务
- 等待服务就绪
- 显示连接信息

### 停止开发环境

```bash
./scripts/dev-stop.sh
```

这个脚本会：
- 停止所有服务
- 询问是否删除数据卷

## 脚本列表

### dev-start.sh
启动开发环境基础设施（PostgreSQL + Redis）

**用法**：
```bash
./scripts/dev-start.sh
```

**功能**：
- 自动检查 Docker 环境
- 检查端口占用
- 启动服务并等待就绪
- 显示连接信息

### dev-stop.sh
停止开发环境基础设施

**用法**：
```bash
./scripts/dev-stop.sh
```

**功能**：
- 停止所有服务
- 可选删除数据卷

## 常见问题

### 端口被占用

如果端口被占用，脚本会提示你。你可以：

1. 停止占用端口的服务
2. 修改 `docker-compose.dev.yml` 中的端口映射

### Docker 未运行

确保 Docker Desktop 已启动：
- macOS: 打开 Docker Desktop 应用
- Linux: `sudo systemctl start docker`
- Windows: 打开 Docker Desktop 应用

### 服务启动失败

查看日志：
```bash
docker-compose -f docker-compose.dev.yml logs
```

## 手动操作

如果你想手动控制服务：

```bash
# 启动服务
docker-compose -f docker-compose.dev.yml up -d

# 查看日志
docker-compose -f docker-compose.dev.yml logs -f

# 停止服务
docker-compose -f docker-compose.dev.yml down

# 重启服务
docker-compose -f docker-compose.dev.yml restart

# 查看状态
docker-compose -f docker-compose.dev.yml ps
```

## 数据管理

### 备份数据

```bash
# PostgreSQL
docker exec fusionmail-postgres pg_dump -U fusionmail fusionmail > backup.sql

# Redis
docker exec fusionmail-redis redis-cli -a fusionmail_redis_password --rdb /data/dump.rdb
```

### 恢复数据

```bash
# PostgreSQL
docker exec -i fusionmail-postgres psql -U fusionmail fusionmail < backup.sql
```

### 清空数据

```bash
# 停止服务并删除数据卷
docker-compose -f docker-compose.dev.yml down -v

# 重新启动
./scripts/dev-start.sh
```

## 更多信息

查看完整的开发环境配置文档：`.kiro/steering/development-setup.md`
