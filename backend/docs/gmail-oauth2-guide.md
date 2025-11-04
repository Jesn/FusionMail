# Gmail OAuth2 认证集成指南

## 概述

FusionMail 现已支持 Gmail OAuth2 认证，用户可以通过安全的 OAuth2 流程添加 Gmail 账户，无需提供密码或应用专用密码。

## 功能特性

- ✅ 安全的 OAuth2 认证流程
- ✅ 自动 Token 刷新机制
- ✅ PKCE 安全增强
- ✅ State 参数防 CSRF 攻击
- ✅ 支持多个 Gmail 账户
- ✅ Token 加密存储
- ✅ 授权撤销功能

## Google Cloud Console 配置

### 1. 创建项目和启用 API

1. 访问 [Google Cloud Console](https://console.cloud.google.com/)
2. 创建新项目或选择现有项目
3. 启用 Gmail API：
   - 导航到 "APIs & Services" > "Library"
   - 搜索 "Gmail API"
   - 点击启用

### 2. 创建 OAuth2 凭证

1. 导航到 "APIs & Services" > "Credentials"
2. 点击 "Create Credentials" > "OAuth 2.0 Client IDs"
3. 选择应用类型：Web application
4. 配置授权回调 URL：
   ```
   http://localhost:4444/auth/google/callback  # 开发环境
   https://your-domain.com/auth/google/callback  # 生产环境
   ```
5. 保存客户端 ID 和客户端密钥

### 3. 配置 OAuth 同意屏幕

1. 导航到 "APIs & Services" > "OAuth consent screen"
2. 选择用户类型（External 用于公开应用）
3. 填写应用信息：
   - 应用名称：FusionMail
   - 用户支持邮箱：your-email@example.com
   - 开发者联系信息：your-email@example.com
4. 添加作用域：
   - `https://www.googleapis.com/auth/gmail.readonly`
   - `https://www.googleapis.com/auth/gmail.modify`
   - `https://www.googleapis.com/auth/userinfo.email`
5. 添加测试用户（开发阶段）

## 后端配置

### 环境变量配置

在 `.env` 文件中添加以下配置：

```bash
# Google OAuth2 配置
GOOGLE_CLIENT_ID=your-google-client-id-here
GOOGLE_CLIENT_SECRET=your-google-client-secret-here
GOOGLE_REDIRECT_URL=http://localhost:4444/auth/google/callback
```

### 配置验证

启动服务器后，可以通过以下方式验证配置：

```bash
# 检查健康状态
curl http://localhost:3333/api/v1/health

# 测试 OAuth2 授权 URL 生成
curl "http://localhost:3333/api/v1/auth/google/authorize?email=test@gmail.com"
```

## API 端点

### 1. 生成授权 URL

```http
GET /api/v1/auth/google/authorize?email=user@gmail.com
```

**响应示例：**
```json
{
  "success": true,
  "data": {
    "auth_url": "https://accounts.google.com/o/oauth2/auth?client_id=...",
    "state": "random-state-string"
  }
}
```

### 2. 处理授权回调

```http
POST /api/v1/auth/google/callback?code=auth_code&state=state_string
```

**响应示例：**
```json
{
  "success": true,
  "data": {
    "account_uid": "account-uid-here",
    "email": "user@gmail.com",
    "access_token": "ya29.a0...",
    "refresh_token": "1//04...",
    "expires_at": "2024-01-01T12:00:00Z"
  }
}
```

### 3. 刷新访问令牌

```http
POST /api/v1/auth/google/refresh/{account_uid}
```

**响应示例：**
```json
{
  "success": true,
  "data": {
    "access_token": "ya29.a0...",
    "expires_at": "2024-01-01T13:00:00Z"
  }
}
```

### 4. 撤销授权

```http
POST /api/v1/auth/google/revoke/{account_uid}
```

**响应示例：**
```json
{
  "success": true,
  "data": "访问令牌已撤销"
}
```

## 前端集成流程

### 1. 发起授权

```javascript
// 获取授权 URL
const response = await fetch('/api/v1/auth/google/authorize?email=user@gmail.com');
const { data } = await response.json();

// 重定向到 Google 授权页面
window.location.href = data.auth_url;
```

### 2. 处理回调

```javascript
// 在回调页面处理授权码
const urlParams = new URLSearchParams(window.location.search);
const code = urlParams.get('code');
const state = urlParams.get('state');

if (code && state) {
  // 发送到后端处理
  const response = await fetch(`/api/v1/auth/google/callback?code=${code}&state=${state}`, {
    method: 'POST'
  });
  
  const result = await response.json();
  if (result.success) {
    // 账户添加成功
    console.log('Gmail 账户已添加:', result.data);
  }
}
```

## 安全特性

### 1. PKCE (Proof Key for Code Exchange)

- 使用 SHA256 哈希的代码验证器
- 防止授权码拦截攻击
- 增强移动和单页应用的安全性

### 2. State 参数

- 随机生成的 32 字节状态参数
- 存储在 Redis 中，5 分钟过期
- 防止 CSRF 攻击

### 3. Token 加密存储

- 使用 AES-256-GCM 加密存储 Token
- 密钥通过环境变量配置
- 支持 Token 自动刷新

### 4. 作用域限制

- `gmail.readonly`: 只读访问邮件
- `gmail.modify`: 修改邮件状态（标记已读等）
- `userinfo.email`: 获取用户邮箱地址

## 错误处理

### 常见错误及解决方案

1. **invalid_client**
   - 检查客户端 ID 和密钥是否正确
   - 确认回调 URL 配置正确

2. **access_denied**
   - 用户拒绝授权
   - 引导用户重新授权

3. **invalid_grant**
   - 授权码已过期或无效
   - 重新发起授权流程

4. **token_expired**
   - 访问令牌已过期
   - 自动使用刷新令牌更新

## 监控和日志

### 日志记录

系统会记录以下 OAuth2 相关事件：

- 授权 URL 生成
- 授权回调处理
- Token 刷新操作
- 授权撤销操作
- 错误和异常情况

### 监控指标

- OAuth2 授权成功率
- Token 刷新频率
- 授权错误统计
- 账户连接状态

## 开发和测试

### 本地开发

1. 配置 Google Cloud Console（使用 localhost 回调 URL）
2. 设置环境变量
3. 启动后端服务
4. 使用 Postman 或 curl 测试 API

### 测试用户

在开发阶段，需要在 Google Cloud Console 中添加测试用户：

1. 导航到 "OAuth consent screen"
2. 在 "Test users" 部分添加测试邮箱
3. 测试用户可以正常使用 OAuth2 流程

## 生产部署

### 环境配置

1. 使用生产域名配置回调 URL
2. 设置强密码的加密密钥
3. 配置 HTTPS
4. 启用日志记录

### 安全检查清单

- [ ] 客户端密钥安全存储
- [ ] 加密密钥定期轮换
- [ ] HTTPS 强制启用
- [ ] 回调 URL 白名单验证
- [ ] Token 过期时间合理设置
- [ ] 错误信息不泄露敏感数据

## 故障排除

### 调试步骤

1. 检查环境变量配置
2. 验证 Google Cloud Console 设置
3. 查看服务器日志
4. 测试网络连接
5. 验证 Redis 连接

### 常用调试命令

```bash
# 检查配置
curl http://localhost:3333/api/v1/health

# 测试授权 URL 生成
curl "http://localhost:3333/api/v1/auth/google/authorize"

# 查看日志
tail -f /var/log/fusionmail/app.log

# 检查 Redis 连接
redis-cli ping
```

## 更新日志

### v1.0.0 (2024-01-31)
- ✅ 初始 Gmail OAuth2 认证集成
- ✅ 支持授权 URL 生成和回调处理
- ✅ 实现 Token 自动刷新机制
- ✅ 添加授权撤销功能
- ✅ 完整的安全特性实现

---

**注意**：此功能需要 Google Cloud Console 配置和有效的 OAuth2 凭证。请确保在生产环境中使用安全的配置和 HTTPS 连接。