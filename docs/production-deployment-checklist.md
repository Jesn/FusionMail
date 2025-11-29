# 生产环境部署检查清单

## 📋 部署前检查

### ✅ 必须完成的配置

#### 1. 环境变量配置

- [ ] **复制环境变量模板**
  ```bash
  cp .env.prod.example .env.prod
  ```

- [ ] **生成安全密钥**
  ```bash
  # JWT 密钥（至少 32 字符）
  echo "JWT_SECRET=$(openssl rand -base64 32)"
  
  # 加密密钥（必须 32 字节，设置后不可更改！）
  echo "ENCRYPTION_KEY=$(openssl rand -base64 32 | cut -c1-32)"
  
  # 数据库密码
  echo "DB_PASSWORD=$(openssl rand -base64 24)"
  ```

- [ ] **修改 `.env.prod` 中的所有 REQUIRED 项**
  - `DB_PASSWORD` - 数据库密码
  - `JWT_SECRET` - JWT 密钥
  - `ENCRYPTION_KEY` - 加密密钥（⚠️ 设置后不可更改）

#### 2. 数据库准备

- [ ] **创建数据库**
  ```sql
  CREATE DATABASE fusionmail;
  CREATE USER fusionmail WITH PASSWORD 'your-password';
  GRANT ALL PRIVILEGES ON DATABASE fusionmail TO fusionmail;
  ```

- [ ] **测试数据库连接**
  ```bash
  psql -h 192.168.2.200 -U postgres -d fusionmail -c "SELECT version();"
  ```

#### 3. Redis 准备

- [ ] **测试 Redis 连接**
  ```bash
  redis-cli -h 192.168.2.200 ping
  # 如果有密码：redis-cli -h 192.168.2.200 -a your-password ping
  ```

#### 4. 存储准备

- [ ] **创建数据目录**
  ```bash
  mkdir -p /data/docekr/fusionmail
  chmod 755 /data/docekr/fusionmail
  ```

---

## 🚀 部署步骤

### 1. 拉取最新代码

```bash
cd /path/to/FusionMail
git pull
```

### 2. 构建镜像

```bash
# 如果使用本地构建
docker-compose -f docker-compose.prod.yml build --no-cache

# 如果使用远程镜像
docker-compose -f docker-compose.prod.yml pull
```

### 3. 启动服务

```bash
docker-compose -f docker-compose.prod.yml --env-file .env.prod up -d
```

### 4. 查看日志

```bash
docker-compose -f docker-compose.prod.yml logs -f fusionmail
```

### 5. 获取管理员密码

```bash
# 如果没有设置 ADMIN_PASSWORD，系统会自动生成
docker-compose -f docker-compose.prod.yml logs fusionmail | grep "Initial password"

# 或者查看密码文件（如果 SAVE_PASSWORD_FILE=true）
docker exec fusionmail cat /app/passwd
```

---

## 🔍 部署后验证

### 1. 健康检查

```bash
curl http://192.168.2.200:3333/api/v1/health
# 预期输出：{"service":"fusionmail","status":"ok","version":"0.1.0"}
```

### 2. 访问前端

浏览器访问：`http://192.168.2.200:3333`

### 3. 测试登录

- 用户名：`admin`
- 密码：从日志中获取或使用设置的 `ADMIN_PASSWORD`

### 4. 检查容器状态

```bash
docker-compose -f docker-compose.prod.yml ps
# 所有服务应该是 Up 状态
```

### 5. 检查数据库连接

```bash
docker-compose -f docker-compose.prod.yml logs fusionmail | grep "Database initialization completed"
```

### 6. 检查 Redis 连接

```bash
docker-compose -f docker-compose.prod.yml logs fusionmail | grep "Redis connection established"
```

---

## 🔒 安全加固

### 1. 修改默认密码

- [ ] 首次登录后立即修改管理员密码
- [ ] 使用强密码（至少 12 位，包含大小写字母、数字、特殊字符）

### 2. 防火墙配置

- [ ] 仅开放必要端口（3333）
- [ ] 限制数据库和 Redis 的访问来源

### 3. HTTPS 配置（推荐）

- [ ] 配置 Nginx 反向代理
- [ ] 申请 SSL 证书（Let's Encrypt 或购买）
- [ ] 设置 `COOKIE_SECURE=true`

### 4. 备份策略

- [ ] 配置数据库定期备份
- [ ] 备份 `/data/docekr/fusionmail` 目录
- [ ] 备份 `.env.prod` 文件（安全存储）

---

## 📊 监控配置

### 1. 日志监控

```bash
# 实时查看日志
docker-compose -f docker-compose.prod.yml logs -f fusionmail

# 查看错误日志
docker-compose -f docker-compose.prod.yml logs fusionmail | grep -i error
```

### 2. 资源监控

```bash
# 查看容器资源使用
docker stats fusionmail
```

### 3. 健康检查

```bash
# 定期检查服务健康状态
watch -n 10 'curl -s http://192.168.2.200:3333/api/v1/health'
```

---

## 🔧 常见问题

### Q1: 容器启动失败

**检查步骤**：
1. 查看日志：`docker-compose -f docker-compose.prod.yml logs fusionmail`
2. 检查环境变量是否正确配置
3. 确认数据库和 Redis 可以连接

### Q2: 无法连接数据库

**解决方案**：
1. 检查数据库是否运行：`psql -h 192.168.2.200 -U postgres -l`
2. 检查防火墙规则
3. 确认数据库密码正确

### Q3: 登录失败

**解决方案**：
1. 检查管理员密码是否正确
2. 查看日志中的密码：`docker-compose -f docker-compose.prod.yml logs fusionmail | grep password`
3. 尝试重置密码（删除容器重新创建）

### Q4: CORS 错误

**解决方案**：
1. 检查 `CORS_ALLOWED_ORIGINS` 配置
2. 确保包含实际访问的域名/IP
3. 重启容器使配置生效

---

## 🔄 更新部署

### 1. 拉取最新代码

```bash
cd /path/to/FusionMail
git pull
```

### 2. 备份数据

```bash
# 备份数据库
pg_dump -h 192.168.2.200 -U postgres fusionmail > fusionmail_backup_$(date +%Y%m%d).sql

# 备份附件
tar -czf fusionmail_data_$(date +%Y%m%d).tar.gz /data/docekr/fusionmail
```

### 3. 重新构建并启动

```bash
docker-compose -f docker-compose.prod.yml --env-file .env.prod down
docker-compose -f docker-compose.prod.yml build --no-cache
docker-compose -f docker-compose.prod.yml --env-file .env.prod up -d
```

### 4. 验证更新

```bash
# 检查版本
curl http://192.168.2.200:3333/api/v1/health

# 检查日志
docker-compose -f docker-compose.prod.yml logs -f fusionmail
```

---

## 📝 维护命令

### 启动服务
```bash
docker-compose -f docker-compose.prod.yml --env-file .env.prod up -d
```

### 停止服务
```bash
docker-compose -f docker-compose.prod.yml down
```

### 重启服务
```bash
docker-compose -f docker-compose.prod.yml restart fusionmail
```

### 查看日志
```bash
docker-compose -f docker-compose.prod.yml logs -f fusionmail
```

### 进入容器
```bash
docker exec -it fusionmail sh
```

### 清理旧镜像
```bash
docker image prune -a
```

---

## 🆘 紧急恢复

### 数据库恢复

```bash
psql -h 192.168.2.200 -U postgres fusionmail < fusionmail_backup_20250129.sql
```

### 附件恢复

```bash
tar -xzf fusionmail_data_20250129.tar.gz -C /
```

### 重置管理员密码

```bash
# 停止容器
docker-compose -f docker-compose.prod.yml down

# 删除数据库中的用户表（谨慎操作！）
psql -h 192.168.2.200 -U postgres fusionmail -c "DELETE FROM users WHERE role='admin';"

# 重新启动，系统会创建新的管理员
docker-compose -f docker-compose.prod.yml --env-file .env.prod up -d
```

---

## 📞 技术支持

如遇问题，请提供以下信息：

1. 错误日志：`docker-compose -f docker-compose.prod.yml logs fusionmail`
2. 容器状态：`docker-compose -f docker-compose.prod.yml ps`
3. 环境信息：操作系统、Docker 版本
4. 配置信息：`.env.prod`（隐藏敏感信息）

---

## ✅ 部署完成检查清单

- [ ] 所有环境变量已正确配置
- [ ] 数据库连接正常
- [ ] Redis 连接正常
- [ ] 服务健康检查通过
- [ ] 前端页面可以访问
- [ ] 登录功能正常
- [ ] 管理员密码已修改
- [ ] 备份策略已配置
- [ ] 监控已设置
- [ ] 文档已归档

**恭喜！FusionMail 已成功部署到生产环境！** 🎉
