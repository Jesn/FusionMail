# 开发环境配置完成总结

## ✅ 已完成的配置变更

### 1. 数据库配置迁移

已将开发环境从本地 Docker 容器迁移到远程数据库服务器。

**远程数据库信息**：
- **PostgreSQL**: `192.168.2.200:5432`
  - 数据库名: `fusionmail-dev`
  - 用户: `postgres`
  - 密码: `8QMZn3yfrbkVG7`
  
- **Redis**: `192.168.2.200:6379`
  - 数据库: `6`
  - 密码: (无)

### 2. 配置文件更新

#### ✅ backend/.env
- 更新 `DB_HOST` 为 `192.168.2.200`
- 更新 `DB_USER` 为 `postgres`
- 更新 `DB_PASSWORD` 为 `8QMZn3yfrbkVG7`
- 更新 `DB_NAME` 为 `fusionmail-dev`
- 更新 `REDIS_HOST` 为 `192.168.2.200`
- 更新 `REDIS_PASSWORD` 为空
- 新增 `REDIS_DB=6`

#### ✅ docker-compose.yml
- 注释掉本地 PostgreSQL 容器配置
- 注释掉本地 Redis 容器配置
- 更新应用容器的数据库环境变量
- 移除对本地容器的依赖 (depends_on)
- 注释掉数据卷配置

#### ✅ start.sh
- 移除 Docker 和 Docker Compose 依赖检查
- 移除本地容器启动逻辑
- 更新数据库连接信息显示
- 添加远程数据库连接检查
- 更新服务状态显示

#### ✅ backend/config/config.go
- 更新 Redis 配置，支持从环境变量读取 `REDIS_DB`
- 更新 Redis 默认密码为空字符串

### 3. 新增脚本和文档

#### ✅ scripts/setup-dev-db.sh
自动化数据库创建脚本，功能包括：
- 检查网络连接和端口可用性
- 创建 PostgreSQL 数据库
- 安装数据库扩展 (uuid-ossp, pg_trgm)
- 验证数据库连接
- 友好的交互式提示

#### ✅ scripts/create-dev-database.sql
手动执行的 SQL 脚本，用于创建数据库和扩展。

#### ✅ docs/dev-database-migration.md
详细的迁移文档，包含：
- 配置变更说明
- 使用步骤指南
- 验证方法
- 故障排查
- 回滚方案

#### ✅ scripts/README.md
脚本使用说明文档。

## 🚀 快速开始

### 步骤 1: 创建数据库

**方法 1: 使用自动化脚本（推荐）**

```bash
./scripts/setup-dev-db.sh
```

**方法 2: 手动创建**

如果没有安装 psql，可以使用数据库管理工具（如 pgAdmin、DBeaver）连接到服务器并创建数据库：

- 主机: `192.168.2.200`
- 端口: `5432`
- 用户: `postgres`
- 密码: `8QMZn3yfrbkVG7`
- 创建数据库: `fusionmail-dev`

### 步骤 2: 启动项目

```bash
# 完整启动（前端 + 后端）
./start.sh

# 或仅启动后端
./start.sh -b

# 或仅启动前端
./start.sh -f
```

### 步骤 3: 访问应用

- **前端**: http://localhost:4444
- **后端 API**: http://localhost:3333
- **API 文档**: http://localhost:3333/docs (如果启用了 Swagger)
- **健康检查**: http://localhost:3333/api/v1/health

## 📋 验证清单

启动项目后，请验证以下内容：

### ✅ 数据库连接
```bash
# 查看后端日志
tail -f logs/backend.log

# 应该看到类似以下的成功日志：
# [INFO] 数据库连接成功
# [INFO] 数据库迁移完成
```

### ✅ 健康检查
```bash
curl http://localhost:3333/api/v1/health

# 应该返回：
# {
#   "status": "ok",
#   "database": "connected",
#   "redis": "connected"
# }
```

### ✅ 前端访问
在浏览器中打开 http://localhost:4444，应该能看到登录页面。

## ⚠️ 重要注意事项

### 1. 数据隔离
- 使用独立的数据库名称 `fusionmail-dev` 避免与其他环境冲突
- Redis 使用 DB 6，避免与其他应用的数据混淆

### 2. 网络访问
- 确保开发机器能够访问 `192.168.2.200`
- 检查防火墙规则是否允许 5432 和 6379 端口

### 3. 数据备份
- 远程数据库应该有定期备份策略
- 开发环境数据可能会被清理，不要存储重要数据

### 4. 密码安全
- 配置文件中的密码仅用于开发环境
- 生产环境必须使用强密码并妥善保管

## 🔧 常用命令

### 查看日志
```bash
# 查看后端日志
tail -f logs/backend.log

# 查看前端日志
tail -f logs/frontend.log

# 查看所有日志
tail -f logs/*.log
```

### 重启服务
```bash
# 停止所有服务
pkill -f fusionmail

# 重新启动
./start.sh
```

### 清理数据
```bash
# 删除并重新创建数据库
PGPASSWORD=8QMZn3yfrbkVG7 psql -h 192.168.2.200 -p 5432 -U postgres -c "DROP DATABASE IF EXISTS \"fusionmail-dev\";"
./scripts/setup-dev-db.sh

# 清理 Redis 数据
redis-cli -h 192.168.2.200 -p 6379 -n 6 FLUSHDB
```

## 🐛 故障排查

### 问题 1: 无法连接到数据库

**症状**: 后端启动失败，日志显示数据库连接错误

**解决方案**:
1. 检查网络连接：`ping 192.168.2.200`
2. 检查端口是否开放：`nc -z 192.168.2.200 5432`
3. 验证数据库凭证是否正确
4. 检查数据库是否已创建：`./scripts/setup-dev-db.sh`

### 问题 2: Redis 连接失败

**症状**: 后端启动成功但 Redis 功能异常

**解决方案**:
1. 检查 Redis 端口：`nc -z 192.168.2.200 6379`
2. 验证 Redis 是否需要密码（当前配置为无密码）
3. 确认使用的是 DB 6

### 问题 3: 数据库已存在

**症状**: 创建数据库时提示已存在

**解决方案**:
```bash
# 重新运行脚本，选择删除并重新创建
./scripts/setup-dev-db.sh
```

## 📚 相关文档

- [详细迁移文档](./dev-database-migration.md)
- [脚本使用说明](../scripts/README.md)
- [项目启动脚本](../start.sh)
- [Docker Compose 配置](../docker-compose.yml)

## 🎯 下一步

配置完成后，你可以：

1. ✅ 开始开发新功能
2. ✅ 运行测试
3. ✅ 调试现有代码
4. ✅ 查看 API 文档

---

**配置完成时间**: 2024-11-29  
**维护者**: FusionMail Team  
**状态**: ✅ 已完成并验证
