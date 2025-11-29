# FusionMail 开发环境配置检查清单

## 📋 配置完成状态

### ✅ 已完成的配置

- [x] 更新 `backend/.env` 数据库配置
- [x] 更新 `docker-compose.yml` 配置
- [x] 更新 `start.sh` 启动脚本
- [x] 更新 `backend/config/config.go` 支持 Redis DB 配置
- [x] 创建数据库创建脚本
- [x] 创建配置文档

### 📝 待完成的步骤

- [ ] 创建远程数据库 `fusionmail-dev`
- [ ] 启动项目并验证连接
- [ ] 测试基本功能

## 🚀 快速开始指南

### 步骤 1: 测试连接

```bash
./scripts/test-connection.sh
```

**预期结果**: 所有连接测试通过

### 步骤 2: 创建数据库

选择以下任一方法：

**方法 A: 使用数据库管理工具（推荐）**
- 参考: [手动创建数据库指南](docs/manual-database-creation.md)
- 工具: pgAdmin、DBeaver、TablePlus 等

**方法 B: 使用 Docker（如果 Docker 可用）**
```bash
./scripts/create-db-docker.sh
```

**方法 C: 使用 psql（如果已安装）**
```bash
./scripts/setup-dev-db.sh
```

### 步骤 3: 启动项目

```bash
# 完整启动（前端 + 后端）
./start.sh

# 或仅启动后端测试
./start.sh -b
```

### 步骤 4: 验证

#### 4.1 检查后端日志
```bash
tail -f logs/backend.log
```

**预期输出**:
```
[INFO] 数据库连接成功
[INFO] 数据库迁移完成
[INFO] Redis 连接成功
[INFO] 服务器启动在 :3333
```

#### 4.2 测试健康检查
```bash
curl http://localhost:3333/api/v1/health
```

**预期输出**:
```json
{
  "status": "ok",
  "database": "connected",
  "redis": "connected"
}
```

#### 4.3 访问前端
打开浏览器访问: http://localhost:4444

**预期结果**: 看到登录页面

## 📊 配置信息汇总

### 数据库连接信息

| 服务 | 地址 | 端口 | 用户 | 密码 | 数据库/DB |
|------|------|------|------|------|-----------|
| PostgreSQL | 192.168.2.200 | 5432 | postgres | 8QMZn3yfrbkVG7 | fusionmail-dev |
| Redis | 192.168.2.200 | 6379 | - | (无) | 6 |

### 应用访问地址

| 服务 | 地址 | 说明 |
|------|------|------|
| 前端 | http://localhost:4444 | Web 界面 |
| 后端 API | http://localhost:3333 | RESTful API |
| API 文档 | http://localhost:3333/docs | Swagger 文档（如果启用） |
| 健康检查 | http://localhost:3333/api/v1/health | 服务状态 |

### 默认管理员账号

- **用户名**: admin
- **密码**: 存储在 `backend/passwd` 文件中（首次启动自动生成）

## 🔧 配置文件位置

| 文件 | 说明 |
|------|------|
| `backend/.env` | 后端环境变量配置 |
| `docker-compose.yml` | Docker Compose 配置 |
| `start.sh` | 项目启动脚本 |
| `backend/config/config.go` | 后端配置代码 |

## 📚 相关文档

| 文档 | 说明 |
|------|------|
| [开发环境配置总结](docs/dev-environment-setup-summary.md) | 完整的配置变更说明 |
| [数据库迁移文档](docs/dev-database-migration.md) | 详细的迁移指南 |
| [手动创建数据库](docs/manual-database-creation.md) | 手动创建数据库的方法 |
| [脚本使用说明](scripts/README.md) | 脚本工具使用指南 |

## 🐛 故障排查

### 问题 1: 连接测试失败

```bash
# 检查网络
ping 192.168.2.200

# 检查端口
nc -z 192.168.2.200 5432
nc -z 192.168.2.200 6379
```

### 问题 2: 数据库创建失败

- 检查是否有权限创建数据库
- 检查数据库是否已存在
- 参考 [手动创建数据库指南](docs/manual-database-creation.md)

### 问题 3: 后端启动失败

```bash
# 查看详细日志
tail -f logs/backend.log

# 检查配置文件
cat backend/.env | grep DB_
cat backend/.env | grep REDIS_
```

### 问题 4: 前端无法访问

```bash
# 检查前端日志
tail -f logs/frontend.log

# 检查端口占用
lsof -i :4444
```

## ✅ 验证清单

完成以下所有项目表示配置成功：

- [ ] 连接测试通过 (`./scripts/test-connection.sh`)
- [ ] 数据库创建成功
- [ ] 后端启动成功
- [ ] 前端启动成功
- [ ] 健康检查返回正常
- [ ] 可以访问登录页面
- [ ] 可以成功登录

## 🎯 下一步

配置完成后，你可以：

1. ✅ 添加邮箱账户
2. ✅ 测试邮件同步
3. ✅ 开发新功能
4. ✅ 运行测试

## 📞 获取帮助

如果遇到问题：

1. 查看相关文档（见上方"相关文档"部分）
2. 检查日志文件 (`logs/backend.log`, `logs/frontend.log`)
3. 运行连接测试 (`./scripts/test-connection.sh`)
4. 查看故障排查部分

---

**配置日期**: 2024-11-29  
**维护者**: FusionMail Team  
**版本**: 1.0.0
