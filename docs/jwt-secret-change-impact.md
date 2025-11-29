# JWT_SECRET 变更影响分析

## 📋 概述

`JWT_SECRET` 是用于签名和验证 JWT（JSON Web Token）的密钥。变更此密钥会导致所有现有的 JWT token 失效。

---

## 🔍 JWT_SECRET 的作用

### 1. Token 签名（生成 Token）

**位置**：`backend/internal/handler/auth.go`

```go
// 登录时生成 token
token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
    "sub":      strconv.FormatInt(user.ID, 10),
    "exp":      expiresAt.Unix(),
    "iat":      time.Now().Unix(),
    "username": user.Username,
    "role":     user.Role,
})

tokenString, err := token.SignedString([]byte(h.jwtSecret))
```

**说明**：使用 `JWT_SECRET` 对 token 进行 HMAC-SHA256 签名。

### 2. Token 验证（验证 Token）

**位置**：
- `backend/internal/middleware/auth.go` - 认证中间件
- `backend/internal/handler/auth.go` - 认证处理器
- `backend/internal/handler/sse_handler.go` - SSE 处理器

```go
// 验证 token
token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
    return []byte(h.jwtSecret), nil
})

if err != nil || !token.Valid {
    // Token 无效
}
```

**说明**：使用相同的 `JWT_SECRET` 验证 token 签名是否有效。

---

## ⚠️ 变更 JWT_SECRET 的影响

### 1. 所有现有 Token 立即失效

**影响范围**：
- ✅ 所有已登录用户的 Token
- ✅ 所有 API 请求的 Authorization Token
- ✅ 所有 Cookie 中的 Session Token
- ✅ 所有 SSE 连接的认证 Token

**结果**：
- ❌ 所有用户被强制登出
- ❌ 所有 API 请求返回 401 Unauthorized
- ❌ 所有 SSE 连接断开

### 2. 用户需要重新登录

**影响**：
- 用户访问任何需要认证的页面时，会被重定向到登录页
- 用户需要重新输入用户名和密码
- 登录后会生成新的 Token（使用新的 JWT_SECRET 签名）

### 3. 不影响数据库数据

**不受影响的内容**：
- ✅ 用户账号数据
- ✅ 邮件数据
- ✅ 邮箱账户配置
- ✅ 规则和 Webhook 配置
- ✅ 所有业务数据

**说明**：JWT_SECRET 只用于 Token 签名，不涉及数据加密。

---

## 🔄 变更场景分析

### 场景 1：首次部署（设置 JWT_SECRET）

**影响**：✅ 无影响

**说明**：首次部署时设置 JWT_SECRET，不存在旧 Token。

---

### 场景 2：运行中变更 JWT_SECRET

**影响**：⚠️ 所有用户被强制登出

**步骤**：
1. 修改 `.env.prod` 中的 `JWT_SECRET`
2. 重启服务：`docker-compose restart fusionmail`
3. 所有现有 Token 失效
4. 用户需要重新登录

**建议**：
- 在维护窗口期间进行
- 提前通知用户
- 准备好管理员密码（用户可能忘记密码）

---

### 场景 3：定期轮换 JWT_SECRET（安全最佳实践）

**影响**：⚠️ 所有用户被强制登出

**建议频率**：
- 一般情况：6-12 个月
- 安全事件后：立即更换
- 密钥泄露：立即更换

**操作流程**：
1. 生成新密钥：`openssl rand -base64 32`
2. 更新配置文件
3. 在维护窗口重启服务
4. 通知用户重新登录

---

### 场景 4：密钥泄露（紧急情况）

**影响**：⚠️ 所有用户被强制登出

**紧急处理**：
1. **立即**生成新密钥
2. **立即**更新配置并重启服务
3. 检查是否有异常登录
4. 通知所有用户修改密码
5. 审查安全日志

---

## 🆚 JWT_SECRET vs ENCRYPTION_KEY

### JWT_SECRET
- **用途**：JWT Token 签名和验证
- **影响**：Token 失效，用户需要重新登录
- **数据影响**：无，不涉及数据加密
- **可以变更**：✅ 是（会强制用户登出）

### ENCRYPTION_KEY
- **用途**：敏感数据加密（邮箱密码、OAuth Token 等）
- **影响**：已加密数据无法解密
- **数据影响**：⚠️ 严重，所有加密数据丢失
- **可以变更**：❌ 否（设置后不可更改）

---

## 📝 变更操作指南

### 步骤 1：生成新密钥

```bash
# 生成新的 JWT_SECRET
NEW_JWT_SECRET=$(openssl rand -base64 32)
echo "新的 JWT_SECRET: $NEW_JWT_SECRET"
```

### 步骤 2：备份当前配置

```bash
# 备份 .env.prod
cp .env.prod .env.prod.backup.$(date +%Y%m%d)
```

### 步骤 3：更新配置

```bash
# 编辑 .env.prod
vim .env.prod

# 修改 JWT_SECRET
JWT_SECRET=your-new-jwt-secret-here
```

### 步骤 4：重启服务

```bash
# 重启 FusionMail 服务
docker-compose -f docker-compose.prod.yml --env-file .env.prod restart fusionmail

# 或者完全重启
docker-compose -f docker-compose.prod.yml --env-file .env.prod down
docker-compose -f docker-compose.prod.yml --env-file .env.prod up -d
```

### 步骤 5：验证

```bash
# 检查服务状态
docker-compose -f docker-compose.prod.yml ps

# 查看日志
docker-compose -f docker-compose.prod.yml logs -f fusionmail

# 测试登录
curl -X POST http://192.168.2.200:3333/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{"username":"admin","password":"your-password"}'
```

---

## 🔒 安全建议

### 1. 定期轮换

**建议**：每 6-12 个月轮换一次 JWT_SECRET

**原因**：
- 降低密钥泄露风险
- 符合安全最佳实践
- 强制清理长期有效的 Token

### 2. 密钥强度

**要求**：
- 至少 32 字符
- 使用随机生成
- 不要使用可预测的字符串

**生成命令**：
```bash
openssl rand -base64 32
```

### 3. 密钥存储

**建议**：
- 使用环境变量，不要硬编码
- 限制 `.env.prod` 文件权限：`chmod 600 .env.prod`
- 不要提交到 Git 仓库
- 使用密钥管理服务（如 AWS Secrets Manager）

### 4. 监控异常

**监控指标**：
- 异常登录尝试
- Token 验证失败率
- 来自异常 IP 的请求

---

## 🚨 紧急情况处理

### 情况 1：JWT_SECRET 泄露

**立即行动**：
1. 生成新的 JWT_SECRET
2. 更新配置并重启服务
3. 检查访问日志，查找异常活动
4. 通知所有用户修改密码
5. 审查安全策略

### 情况 2：大量 Token 验证失败

**可能原因**：
- JWT_SECRET 配置错误
- 服务重启时 JWT_SECRET 变更
- 时钟不同步（Token 过期时间）

**排查步骤**：
1. 检查 JWT_SECRET 配置是否正确
2. 检查服务器时间是否同步
3. 查看错误日志
4. 测试新登录是否正常

---

## 📊 影响对比表

| 操作 | JWT_SECRET 变更 | ENCRYPTION_KEY 变更 |
|------|----------------|---------------------|
| **用户登录状态** | ❌ 全部失效 | ✅ 不影响 |
| **需要重新登录** | ✅ 是 | ❌ 否 |
| **邮箱账户配置** | ✅ 不影响 | ❌ 全部丢失 |
| **邮件数据** | ✅ 不影响 | ✅ 不影响 |
| **OAuth Token** | ✅ 不影响 | ❌ 全部丢失 |
| **可以回滚** | ✅ 是 | ⚠️ 困难 |
| **数据恢复** | ✅ 无需恢复 | ❌ 无法恢复 |

---

## ✅ 最佳实践总结

1. **首次部署**：使用强随机密钥
2. **定期轮换**：每 6-12 个月轮换一次
3. **安全存储**：使用环境变量，限制文件权限
4. **变更通知**：提前通知用户，选择维护窗口
5. **备份配置**：变更前备份配置文件
6. **监控日志**：变更后监控异常活动
7. **应急预案**：准备密钥泄露的应急流程

---

## 🔗 相关文档

- [环境变量配置说明](./environment-variables.md)
- [生产环境部署检查清单](./production-deployment-checklist.md)
- [安全配置指南](./security-best-practices.md)（待创建）

---

## 💡 常见问题

### Q1: 变更 JWT_SECRET 后，用户数据会丢失吗？

**A**: 不会。JWT_SECRET 只用于 Token 签名，不涉及数据加密。所有用户数据、邮件数据都不受影响。

### Q2: 可以在不重启服务的情况下变更 JWT_SECRET 吗？

**A**: 不可以。JWT_SECRET 在服务启动时加载，必须重启服务才能生效。

### Q3: 变更 JWT_SECRET 后，可以回滚吗？

**A**: 可以。只需将 JWT_SECRET 改回旧值并重启服务即可。但这会使新生成的 Token 失效。

### Q4: 多久应该轮换一次 JWT_SECRET？

**A**: 建议每 6-12 个月轮换一次。如果发生安全事件或密钥泄露，应立即更换。

### Q5: JWT_SECRET 和 ENCRYPTION_KEY 可以使用相同的值吗？

**A**: 不建议。它们用途不同，应该使用不同的密钥。而且 ENCRYPTION_KEY 必须是 32 字节，JWT_SECRET 没有此限制。
