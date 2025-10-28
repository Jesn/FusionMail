# FusionMail API 测试示例

## 前置条件

1. 启动服务器：
```bash
cd backend
./bin/server
```

2. 服务器默认运行在 `http://localhost:8080`

## 1. 健康检查

```bash
curl http://localhost:8080/api/v1/health
```

## 2. 添加邮箱账户

### QQ 邮箱示例

```bash
curl -X POST http://localhost:8080/api/v1/accounts \
  -H "Content-Type: application/json" \
  -d '{
    "email": "your@qq.com",
    "provider": "qq",
    "protocol": "imap",
    "auth_type": "password",
    "password": "your_authorization_code",
    "sync_enabled": true,
    "sync_interval": 5
  }'
```

**注意**：QQ 邮箱需要使用授权码，不是登录密码！

获取 QQ 邮箱授权码：
1. 登录 QQ 邮箱网页版
2. 设置 → 账户 → POP3/IMAP/SMTP/Exchange/CardDAV/CalDAV服务
3. 开启 IMAP/SMTP 服务
4. 生成授权码

### 163 邮箱示例

```bash
curl -X POST http://localhost:8080/api/v1/accounts \
  -H "Content-Type: application/json" \
  -d '{
    "email": "your@163.com",
    "provider": "163",
    "protocol": "imap",
    "auth_type": "password",
    "password": "your_authorization_code",
    "sync_enabled": true,
    "sync_interval": 5
  }'
```

**注意**：163 邮箱也需要使用授权码！

获取 163 邮箱授权码：
1. 登录 163 邮箱网页版
2. 设置 → POP3/SMTP/IMAP
3. 开启 IMAP/SMTP 服务
4. 设置客户端授权密码

### Gmail 示例

```bash
curl -X POST http://localhost:8080/api/v1/accounts \
  -H "Content-Type: application/json" \
  -d '{
    "email": "your@gmail.com",
    "provider": "gmail",
    "protocol": "imap",
    "auth_type": "password",
    "password": "your_app_password",
    "sync_enabled": true,
    "sync_interval": 5
  }'
```

**注意**：Gmail 需要使用应用专用密码！

获取 Gmail 应用专用密码：
1. 开启两步验证
2. Google 账户 → 安全性 → 应用专用密码
3. 生成新的应用专用密码

### iCloud 邮箱示例

```bash
curl -X POST http://localhost:8080/api/v1/accounts \
  -H "Content-Type: application/json" \
  -d '{
    "email": "your@icloud.com",
    "provider": "icloud",
    "protocol": "imap",
    "auth_type": "password",
    "password": "your_app_password",
    "sync_enabled": true,
    "sync_interval": 5
  }'
```

**注意**：iCloud 需要使用应用专用密码！

获取 iCloud 应用专用密码：
1. 登录 appleid.apple.com
2. 安全 → 应用专用密码
3. 生成新密码

## 3. 查看账户列表

```bash
curl http://localhost:8080/api/v1/accounts
```

## 4. 查看账户详情

```bash
# 替换 {account_uid} 为实际的账户 UID
curl http://localhost:8080/api/v1/accounts/{account_uid}
```

## 5. 测试账户连接

```bash
# 替换 {account_uid} 为实际的账户 UID
curl -X POST http://localhost:8080/api/v1/accounts/{account_uid}/test
```

## 6. 手动触发同步

### 同步指定账户

```bash
# 替换 {account_uid} 为实际的账户 UID
curl -X POST http://localhost:8080/api/v1/sync/accounts/{account_uid}
```

### 同步所有账户

```bash
curl -X POST http://localhost:8080/api/v1/sync/all
```

## 7. 查看同步状态

```bash
curl http://localhost:8080/api/v1/sync/status
```

## 8. 更新账户

```bash
# 替换 {account_uid} 为实际的账户 UID
curl -X PUT http://localhost:8080/api/v1/accounts/{account_uid} \
  -H "Content-Type: application/json" \
  -d '{
    "sync_enabled": false
  }'
```

## 9. 删除账户

```bash
# 替换 {account_uid} 为实际的账户 UID
curl -X DELETE http://localhost:8080/api/v1/accounts/{account_uid}
```

## 完整测试流程

```bash
# 1. 检查服务器
curl http://localhost:8080/api/v1/health

# 2. 添加账户（以 QQ 邮箱为例）
RESPONSE=$(curl -s -X POST http://localhost:8080/api/v1/accounts \
  -H "Content-Type: application/json" \
  -d '{
    "email": "your@qq.com",
    "provider": "qq",
    "protocol": "imap",
    "auth_type": "password",
    "password": "your_authorization_code",
    "sync_enabled": true,
    "sync_interval": 5
  }')

echo $RESPONSE

# 3. 提取账户 UID（需要 jq 工具）
ACCOUNT_UID=$(echo $RESPONSE | jq -r '.data.uid')
echo "账户 UID: $ACCOUNT_UID"

# 4. 测试连接
curl -X POST http://localhost:8080/api/v1/accounts/$ACCOUNT_UID/test

# 5. 手动触发同步
curl -X POST http://localhost:8080/api/v1/sync/accounts/$ACCOUNT_UID

# 6. 等待几秒后查看同步状态
sleep 5
curl http://localhost:8080/api/v1/sync/status

# 7. 查看账户详情（包含同步状态）
curl http://localhost:8080/api/v1/accounts/$ACCOUNT_UID
```

## 常见问题

### Q: 连接测试失败？

**A:** 检查以下几点：
1. 邮箱地址是否正确
2. 是否使用了授权码/应用专用密码（而非登录密码）
3. 是否开启了 IMAP 服务
4. 网络连接是否正常

### Q: 同步没有拉取到邮件？

**A:** 可能的原因：
1. 首次同步只拉取最近 30 天的邮件
2. 邮箱中可能没有符合条件的邮件
3. 查看服务器日志确认是否有错误

### Q: 如何查看同步日志？

**A:** 查看服务器控制台输出，或者后续实现日志查询 API

## 使用测试脚本

我们提供了一个交互式测试脚本：

```bash
./test-account.sh
```

脚本会引导你：
1. 输入邮箱信息
2. 自动添加账户
3. 测试连接
4. 触发同步
5. 查看结果

## 支持的邮箱提供商

| 提供商 | Provider | IMAP 服务器 | 端口 | 需要授权码 |
|--------|----------|------------|------|-----------|
| QQ 邮箱 | qq | imap.qq.com | 993 | ✅ |
| 163 邮箱 | 163 | imap.163.com | 993 | ✅ |
| Gmail | gmail | imap.gmail.com | 993 | ✅ |
| iCloud | icloud | imap.mail.me.com | 993 | ✅ |
| Outlook | outlook | outlook.office365.com | 993 | ❌ |

## 下一步

- 实现邮件查询 API
- 实现邮件搜索功能
- 添加前端界面
