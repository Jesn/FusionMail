# 139 邮箱故障排查指南

## 🐛 问题：TLS 握手失败

### 症状

```json
{
  "error": "failed to connect: failed to connect to IMAP server: remote error: tls: handshake failure",
  "success": false
}
```

### 根本原因

139 邮箱账户的 `encryption` 字段设置错误。

**错误配置**：
- `encryption: "starttls"` - 用于 143 端口（先明文连接再升级）
- `imap_port: 993` - SSL/TLS 直连端口

**正确配置**：
- `encryption: "ssl"` 或 `"tls"`
- `imap_port: 993`

### 解决方案

#### 方案 1：重新添加账户（推荐）

1. **删除旧账户**
   - 在前端界面删除 139 邮箱账户
   - 或使用 API：
     ```bash
     curl -X DELETE http://localhost:3333/api/v1/accounts/{account_uid} \
       -H "Authorization: Bearer {token}"
     ```

2. **重新添加账户**
   - 在前端"添加账户"页面
   - 选择"139 邮箱"
   - 输入邮箱地址和授权码
   - **重要**：确保加密方式选择 "SSL/TLS"
   - 端口：993

3. **API 添加示例**
   ```bash
   curl -X POST http://localhost:3333/api/v1/accounts \
     -H "Content-Type: application/json" \
     -H "Authorization: Bearer {token}" \
     -d '{
       "email": "your_phone@139.com",
       "provider": "139",
       "protocol": "imap",
       "credentials": {
         "host": "imap.139.com",
         "port": 993,
         "email": "your_phone@139.com",
         "password": "your_authorization_code",
         "tls": true
       },
       "sync_enabled": true,
       "sync_interval": 5
     }'
   ```

#### 方案 2：直接修改数据库

如果已安装 PostgreSQL 客户端：

```bash
# 连接数据库
PGPASSWORD=8QMZn3yfrbkVG7 psql -h 192.168.2.200 -p 5432 -U postgres -d fusionmail-dev

# 更新加密方式
UPDATE email_accounts 
SET encryption = 'ssl' 
WHERE email = '15026732619@139.com';

# 退出
\q
```

然后重启后端服务：
```bash
./start.sh -b
```

## 📋 正确的 139 邮箱配置

### IMAP 配置

| 项目 | 值 |
|------|---|
| 服务器 | imap.139.com |
| 端口 | 993 |
| 加密方式 | SSL/TLS（不是 STARTTLS） |
| 认证方式 | 密码（使用授权码） |

### POP3 配置

| 项目 | 值 |
|------|---|
| 服务器 | pop.139.com |
| 端口 | 995 |
| 加密方式 | SSL/TLS |
| 认证方式 | 密码（使用授权码） |

## 🔍 验证配置

### 1. 检查账户配置

```bash
curl http://localhost:3333/api/v1/accounts/{account_uid} \
  -H "Authorization: Bearer {token}" | jq '{
    email,
    imap_host,
    imap_port,
    encryption,
    protocol
  }'
```

**预期输出**：
```json
{
  "email": "your_phone@139.com",
  "imap_host": "imap.139.com",
  "imap_port": 993,
  "encryption": "ssl",
  "protocol": "imap"
}
```

### 2. 测试连接

```bash
curl -X POST http://localhost:3333/api/v1/accounts/{account_uid}/test \
  -H "Authorization: Bearer {token}"
```

**成功响应**：
```json
{
  "success": true,
  "message": "连接测试成功"
}
```

### 3. 触发同步

```bash
curl -X POST http://localhost:3333/api/v1/sync/accounts/{account_uid} \
  -H "Authorization: Bearer {token}"
```

**成功响应**：
```json
{
  "success": true,
  "message": "同步成功",
  "synced_count": 10
}
```

### 4. 查看日志

```bash
tail -f logs/backend.log | grep -E "IMAP|139|TLS"
```

**成功日志**：
```
[IMAP] Detected 139 Mail (China Mobile), using relaxed TLS config
[IMAP] Login successful
[IMAP] Successfully fetched X emails
```

## 🔧 加密方式说明

### SSL/TLS（推荐用于 993/995 端口）

- **端口**：IMAP 993, POP3 995
- **连接方式**：直接建立 TLS 加密连接
- **安全性**：高
- **兼容性**：所有现代邮件服务器

### STARTTLS（用于 143/110 端口）

- **端口**：IMAP 143, POP3 110
- **连接方式**：先明文连接，再升级到 TLS
- **安全性**：中（可能被降级攻击）
- **兼容性**：较好

### 无加密（不推荐）

- **端口**：IMAP 143, POP3 110
- **连接方式**：明文传输
- **安全性**：低（密码和邮件内容可被窃听）
- **兼容性**：最好

## ⚠️ 常见错误

### 错误 1：端口和加密方式不匹配

```
❌ 端口 993 + STARTTLS
✅ 端口 993 + SSL/TLS
✅ 端口 143 + STARTTLS
```

### 错误 2：使用登录密码而非授权码

```
❌ 密码：登录密码
✅ 密码：授权码（在邮箱设置中生成）
```

### 错误 3：未开启 IMAP 服务

```
❌ IMAP 服务：关闭
✅ IMAP 服务：开启
```

## 📚 相关文档

- [中国邮箱服务商配置指南](./china-mail-providers.md)
- [139 邮箱修复总结](./139-mail-fix-summary.md)
- [快速配置参考](../CHINA_MAIL_QUICK_REFERENCE.md)

---

**更新时间**: 2024-11-29  
**维护者**: FusionMail Team
