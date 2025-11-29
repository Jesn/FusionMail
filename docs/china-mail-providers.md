# 中国邮箱服务商配置指南

## 📋 支持的邮箱服务商

FusionMail 已针对中国主流邮箱服务商进行了优化配置。

### 1. 139 邮箱（中国移动）

**服务器配置**：

| 协议 | 服务器地址 | 端口（非SSL） | 端口（SSL） |
|------|-----------|--------------|------------|
| IMAP | imap.139.com | 143 | 993 |
| POP3 | pop.139.com | 110 | 995 |
| SMTP | smtp.139.com | 25 | 465 |

**特殊配置**：
- 使用宽松的 TLS 配置（支持 TLS 1.0-1.2）
- 跳过证书验证（InsecureSkipVerify）
- 需要使用授权码而非登录密码

**获取授权码**：
1. 登录 139 邮箱网页版
2. 进入"设置" → "账户安全"
3. 开启 IMAP/SMTP 服务
4. 生成授权码

### 2. QQ 邮箱（腾讯）

**服务器配置**：

| 协议 | 服务器地址 | 端口（非SSL） | 端口（SSL） |
|------|-----------|--------------|------------|
| IMAP | imap.qq.com | 143 | 993 |
| POP3 | pop.qq.com | 110 | 995 |
| SMTP | smtp.qq.com | 25/587 | 465 |

**特殊配置**：
- 支持 TLS 1.0+
- 需要使用授权码

**获取授权码**：
1. 登录 QQ 邮箱网页版
2. 进入"设置" → "账户"
3. 开启 IMAP/SMTP 服务
4. 生成授权码

### 3. 163 邮箱（网易）

**服务器配置**：

| 协议 | 服务器地址 | 端口（非SSL） | 端口（SSL） |
|------|-----------|--------------|------------|
| IMAP | imap.163.com | 143 | 993 |
| POP3 | pop.163.com | 110 | 995 |
| SMTP | smtp.163.com | 25 | 465/994 |

**特殊配置**：
- 支持 TLS 1.0+
- 需要使用授权码
- 需要发送 IMAP ID 信息

**获取授权码**：
1. 登录 163 邮箱网页版
2. 进入"设置" → "POP3/SMTP/IMAP"
3. 开启 IMAP/SMTP 服务
4. 设置客户端授权密码

### 4. 126 邮箱（网易）

**服务器配置**：

| 协议 | 服务器地址 | 端口（非SSL） | 端口（SSL） |
|------|-----------|--------------|------------|
| IMAP | imap.126.com | 143 | 993 |
| POP3 | pop.126.com | 110 | 995 |
| SMTP | smtp.126.com | 25 | 465/994 |

**特殊配置**：
- 与 163 邮箱相同
- 需要使用授权码

### 5. 189 邮箱（中国电信）

**服务器配置**：

| 协议 | 服务器地址 | 端口（非SSL） | 端口（SSL） |
|------|-----------|--------------|------------|
| IMAP | imap.189.cn | 143 | 993 |
| POP3 | pop.189.cn | 110 | 995 |
| SMTP | smtp.189.cn | 25 | 465 |

**特殊配置**：
- 支持 TLS 1.0+
- 需要使用授权码

## 🔧 TLS 配置说明

### 为什么需要特殊配置？

中国的一些邮箱服务商使用了较旧的 TLS 版本或特殊的证书配置，标准的 TLS 配置可能无法连接。FusionMail 针对这些服务商进行了以下优化：

1. **支持较旧的 TLS 版本**：允许 TLS 1.0 和 1.1
2. **宽松的证书验证**：对于某些服务商跳过证书验证
3. **限制最高 TLS 版本**：某些服务商不支持 TLS 1.3

### 安全性说明

⚠️ **注意**：跳过证书验证会降低安全性，但这是连接某些中国邮箱服务商的必要措施。

**建议**：
- 仅在受信任的网络环境中使用
- 使用授权码而非登录密码
- 定期更换授权码

## 📝 配置示例

### 添加 139 邮箱账户

```json
{
  "email": "your_phone@139.com",
  "provider": "139",
  "protocol": "imap",
  "credentials": {
    "host": "imap.139.com",
    "port": 993,
    "email": "your_phone@139.com",
    "password": "your_authorization_code",
    "tls": true
  }
}
```

### 添加 QQ 邮箱账户

```json
{
  "email": "your_qq@qq.com",
  "provider": "qq",
  "protocol": "imap",
  "credentials": {
    "host": "imap.qq.com",
    "port": 993,
    "email": "your_qq@qq.com",
    "password": "your_authorization_code",
    "tls": true
  }
}
```

### 添加 163 邮箱账户

```json
{
  "email": "your_name@163.com",
  "provider": "163",
  "protocol": "imap",
  "credentials": {
    "host": "imap.163.com",
    "port": 993,
    "email": "your_name@163.com",
    "password": "your_authorization_code",
    "tls": true
  }
}
```

## 🐛 常见问题

### 问题 1: TLS 握手失败

**错误信息**：
```
failed to connect: failed to connect to IMAP server: remote error: tls: handshake failure
```

**解决方案**：
1. 确认使用的是 SSL 端口（IMAP: 993, POP3: 995）
2. 确认服务器地址正确
3. 检查是否使用了授权码而非登录密码
4. 确保 FusionMail 已更新到最新版本（包含 TLS 优化）

### 问题 2: 认证失败

**错误信息**：
```
failed to login: authentication failed
```

**解决方案**：
1. 确认已开启 IMAP/POP3 服务
2. 使用授权码而非登录密码
3. 检查邮箱地址是否正确
4. 确认授权码未过期

### 问题 3: 连接超时

**错误信息**：
```
failed to connect: dial tcp: i/o timeout
```

**解决方案**：
1. 检查网络连接
2. 确认防火墙未阻止相关端口
3. 尝试使用代理
4. 检查邮箱服务商是否有地区限制

## 🔍 调试技巧

### 查看详细日志

启动后端时查看日志：

```bash
tail -f logs/backend.log | grep -E "IMAP|POP3|TLS"
```

### 测试连接

使用 openssl 测试 TLS 连接：

```bash
# 测试 139 邮箱 IMAP
openssl s_client -connect imap.139.com:993 -showcerts

# 测试 QQ 邮箱 IMAP
openssl s_client -connect imap.qq.com:993 -showcerts

# 测试 163 邮箱 IMAP
openssl s_client -connect imap.163.com:993 -showcerts
```

### 检查 TLS 版本支持

```bash
# 测试 TLS 1.0
openssl s_client -connect imap.139.com:993 -tls1

# 测试 TLS 1.1
openssl s_client -connect imap.139.com:993 -tls1_1

# 测试 TLS 1.2
openssl s_client -connect imap.139.com:993 -tls1_2
```

## 📚 参考资料

### 官方文档

- [139 邮箱帮助中心](https://mail.10086.cn/help/)
- [QQ 邮箱帮助中心](https://service.mail.qq.com/)
- [163 邮箱帮助中心](https://help.mail.163.com/)
- [126 邮箱帮助中心](https://help.mail.126.com/)
- [189 邮箱帮助中心](https://mail.189.cn/help/)

### 授权码获取指南

各邮箱服务商的授权码获取方式略有不同，请参考各自的帮助文档。

## 🔄 更新日志

### 2024-11-29
- ✅ 添加 139 邮箱（中国移动）支持
- ✅ 优化 TLS 配置，支持较旧版本
- ✅ 添加证书验证跳过选项
- ✅ 改进错误提示信息

---

**维护者**: FusionMail Team  
**更新时间**: 2024-11-29
