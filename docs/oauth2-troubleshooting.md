# Microsoft OAuth2 授权失败排查指南

## 🚨 当前发现的问题

根据日志分析，发现Microsoft OAuth2授权失败，错误类型为 `server_error`。

### 日志信息分析

```
[INFO] Starting OAuth2 auth URL generation (provider=microsoft, email=jesn2013@hotmail.com)
[DEBUG] Generated OAuth2 state (state=2vVKXscdEqOC4EdIvHBdCMBA1YwOSMz2DfkDNyqKo2s=)
[DEBUG] OAuth2 config retrieved (client_id=0ec56a84-6012-4ac5-81a5-e61f6a1f4438)
[INFO] OAuth2 auth URL generated successfully
[ERROR] Microsoft OAuth2 authorization error (error=server_error)
```

## 🔍 问题分析

### 1. `server_error` 错误类型

`server_error` 通常表示以下问题之一：

1. **Azure应用配置问题**
   - 客户端ID或密钥不正确
   - 重定向URI配置不匹配
   - API权限配置不正确

2. **Microsoft服务问题**
   - Microsoft服务临时不可用
   - 账户类型不支持
   - 租户配置问题

3. **网络连接问题**
   - 网络代理配置
   - DNS解析问题
   - 防火墙阻止

## 🛠️ 解决方案

### 步骤1：验证Azure应用配置

1. **检查客户端ID和密钥**
   ```bash
   # 检查当前配置
   grep MICROSOFT backend/.env
   ```

2. **验证重定向URI**
   - Azure Portal中配置：`http://localhost:3333/api/v1/auth/microsoft/callback`
   - 环境变量中配置：`http://localhost:3333/api/v1/auth/microsoft/callback`
   - 确保完全一致（包括协议、域名、端口、路径）

3. **检查API权限**
   - Mail.ReadWrite ✅
   - User.Read ✅
   - offline_access ✅
   - 确保已授予管理员同意

### 步骤2：检查账户类型支持

当前配置支持的账户类型：
```
任何组织目录(任何 Azure AD 目录 - 多租户)中的账户和个人 Microsoft 账户
```

测试邮箱 `jesn2013@hotmail.com` 是个人Microsoft账户，应该被支持。

### 步骤3：网络连接测试

```bash
# 测试Microsoft登录端点连接
curl -I "https://login.microsoftonline.com/common/oauth2/v2.0/authorize"

# 测试Graph API端点连接
curl -I "https://graph.microsoft.com/v1.0/"
```

### 步骤4：详细错误信息获取

修改日志级别获取更详细的错误信息：

1. **临时启用详细日志**
   ```bash
   # 在 backend/.env 中设置
   LOG_LEVEL=debug
   ```

2. **重启服务并重试授权**

3. **查看详细错误信息**
   ```bash
   tail -f logs/backend.log | grep -i "oauth\|microsoft\|error"
   ```

## 🔧 快速修复尝试

### 方法1：重新创建Azure应用

如果配置看起来正确但仍然失败，尝试重新创建Azure应用：

1. 在Azure Portal中删除当前应用
2. 重新创建应用注册
3. 重新配置权限和重定向URI
4. 更新环境变量

### 方法2：使用不同的重定向URI

尝试使用不同的重定向URI格式：

```bash
# 当前配置
MICROSOFT_REDIRECT_URI=http://localhost:3333/api/v1/auth/microsoft/callback

# 尝试使用127.0.0.1
MICROSOFT_REDIRECT_URI=http://127.0.0.1:3333/api/v1/auth/microsoft/callback
```

### 方法3：检查客户端密钥

客户端密钥可能已过期或不正确：

1. 在Azure Portal中生成新的客户端密钥
2. 更新环境变量
3. 重启服务

## 📋 完整排查清单

### ✅ Azure Portal 检查
- [ ] 应用注册存在且状态正常
- [ ] 客户端ID正确 (0ec56a84-6012-4ac5-81a5-e61f6a1f4438)
- [ ] 客户端密钥有效且未过期
- [ ] 重定向URI完全匹配
- [ ] API权限正确配置
- [ ] 管理员同意已授予

### ✅ 环境配置检查
- [ ] MICROSOFT_CLIENT_ID 设置正确
- [ ] MICROSOFT_CLIENT_SECRET 设置正确
- [ ] MICROSOFT_REDIRECT_URI 设置正确
- [ ] 服务器端口配置正确 (3333)

### ✅ 网络连接检查
- [ ] 可以访问 login.microsoftonline.com
- [ ] 可以访问 graph.microsoft.com
- [ ] 防火墙允许HTTPS连接
- [ ] 代理配置正确（如果使用）

### ✅ 服务状态检查
- [ ] 后端服务正常运行
- [ ] 数据库连接正常
- [ ] Redis连接正常
- [ ] 日志级别设置为debug

## 🧪 测试步骤

### 1. 基础连接测试
```bash
# 测试授权URL生成
curl "http://localhost:3333/api/v1/auth/microsoft/authorize?email=test@hotmail.com"
```

### 2. 手动授权测试
1. 复制生成的授权URL
2. 在浏览器中访问
3. 观察Microsoft登录页面是否正常显示
4. 尝试登录并观察错误信息

### 3. 回调测试
```bash
# 模拟回调请求（使用实际的code和state）
curl "http://localhost:3333/api/v1/auth/microsoft/callback?code=TEST_CODE&state=TEST_STATE"
```

## 📞 获取更多帮助

如果问题仍然存在：

1. **查看Microsoft文档**
   - [OAuth 2.0 错误代码](https://docs.microsoft.com/en-us/azure/active-directory/develop/reference-aadsts-error-codes)
   - [Microsoft Graph 错误处理](https://docs.microsoft.com/en-us/graph/errors)

2. **检查Microsoft服务状态**
   - [Azure状态页面](https://status.azure.com/)
   - [Microsoft 365状态页面](https://portal.office.com/servicestatus)

3. **联系支持**
   - 提供完整的错误日志
   - 包含Azure应用配置截图
   - 说明重现步骤

---

**文档版本：** v1.0  
**创建日期：** 2025-01-31  
**最后更新：** 2025-01-31