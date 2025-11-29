# FusionMail 文档目录

## 📚 文档索引

### 开发环境配置

| 文档 | 说明 | 适用场景 |
|------|------|----------|
| [开发环境配置总结](./dev-environment-setup-summary.md) | 完整的配置变更说明和快速开始指南 | 首次配置或了解配置变更 |
| [数据库迁移文档](./dev-database-migration.md) | 详细的数据库迁移步骤和故障排查 | 深入了解迁移过程 |
| [手动创建数据库](./manual-database-creation.md) | 使用各种工具手动创建数据库 | 自动化脚本无法使用时 |
| [配置检查清单](../SETUP_CHECKLIST.md) | 配置步骤检查清单 | 验证配置是否完成 |

### API 文档

| 文档 | 说明 |
|------|------|
| [Swagger 集成说明](./swagger-integration-summary.md) | Swagger API 文档配置 |
| [JWT 密钥变更影响](./jwt-secret-change-impact.md) | JWT 密钥变更说明 |
| [登录密码指南](./login-password-guide.md) | 管理员密码管理 |

## 🚀 快速开始

### 新开发者入门

1. **阅读配置总结**
   ```bash
   cat docs/dev-environment-setup-summary.md
   ```

2. **测试连接**
   ```bash
   ./scripts/test-connection.sh
   ```

3. **创建数据库**
   - 自动化: `./scripts/create-db-docker.sh`
   - 手动: 参考 [手动创建数据库](./manual-database-creation.md)

4. **启动项目**
   ```bash
   ./start.sh
   ```

5. **验证配置**
   - 查看 [配置检查清单](../SETUP_CHECKLIST.md)

### 常见任务

#### 重新创建数据库
```bash
# 使用 Docker
./scripts/create-db-docker.sh

# 或使用 psql
./scripts/setup-dev-db.sh
```

#### 查看日志
```bash
# 后端日志
tail -f logs/backend.log

# 前端日志
tail -f logs/frontend.log
```

#### 重启服务
```bash
# 停止
pkill -f fusionmail

# 启动
./start.sh
```

## 📋 配置信息

### 开发环境数据库

- **PostgreSQL**: `192.168.2.200:5432/fusionmail-dev`
- **Redis**: `192.168.2.200:6379/6`

### 应用访问

- **前端**: http://localhost:4444
- **后端**: http://localhost:3333
- **API 文档**: http://localhost:3333/docs

## 🔧 脚本工具

所有脚本位于 `scripts/` 目录：

| 脚本 | 说明 |
|------|------|
| `test-connection.sh` | 测试数据库连接 |
| `setup-dev-db.sh` | 自动创建数据库（需要 psql） |
| `create-db-docker.sh` | 使用 Docker 创建数据库 |
| `create-dev-database.sql` | SQL 创建脚本 |

详细说明参考: [scripts/README.md](../scripts/README.md)

## 🐛 故障排查

### 常见问题

1. **无法连接数据库**
   - 运行 `./scripts/test-connection.sh` 检查连接
   - 检查网络和防火墙设置

2. **数据库创建失败**
   - 参考 [手动创建数据库](./manual-database-creation.md)
   - 使用数据库管理工具手动创建

3. **后端启动失败**
   - 查看 `logs/backend.log`
   - 检查 `backend/.env` 配置

4. **前端无法访问**
   - 查看 `logs/frontend.log`
   - 检查端口 4444 是否被占用

## 📞 获取帮助

1. 查看相关文档
2. 检查日志文件
3. 运行连接测试
4. 查看配置检查清单

## 🔄 更新日志

### 2024-11-29
- ✅ 迁移到远程数据库
- ✅ 更新所有配置文件
- ✅ 创建自动化脚本
- ✅ 完善文档

---

**维护者**: FusionMail Team
