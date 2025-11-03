# Microsoft Graph OAuth2 配置检查清单

## 📋 Azure Portal 配置检查清单

### ✅ 应用注册基本信息
- [ ] 应用名称：FusionMail
- [ ] 支持的账户类型：任何组织目录中的账户和个人 Microsoft 账户
- [ ] 重定向URI已正确配置

### ✅ API 权限配置
- [ ] Microsoft Graph - Mail.ReadWrite (委托)
- [ ] Microsoft Graph - User.Read (委托)  
- [ ] Microsoft Graph - offline_access (委托)
- [ ] 已授予管理员同意（状态显示为"已授予"）

### ✅ 身份验证配置
- [ ] Web 平台已配置
- [ ] 重定向URI格式正确：`http://localhost:3333/api/v1/auth/microsoft/callback`
- [ ] 生产环境使用HTTPS重定向URI
- [ ] 启用了访问令牌和ID令牌

### ✅ 证书和密钥
- [ ] 已创建客户端密钥
- [ ] 密钥说明：FusionMail Client Secret
- [ ] 密钥过期时间：24个月
- [ ] 已安全保存客户端密钥值

### ✅ 重要信息记录
```
应用程序(客户端) ID: ________________
客户端密钥值: ________________
目录(租户) ID: ________________
重定向URI: ________________
```

## 🔧 FusionMail 配置检查清单

### ✅ 环境变量配置
- [ ] `MICROSOFT_CLIENT_ID` 已设置
- [ ] `MICROSOFT_CLIENT_SECRET` 已设置  
- [ ] `MICROSOFT_REDIRECT_URI` 已设置
- [ ] 重定向URI与Azure配置完全一致

### ✅ 代码配置验证
- [ ] 后端编译成功
- [ ] OAuth2服务正常启动
- [ ] API端点可访问

### ✅ 网络配置
- [ ] 防火墙允许相应端口
- [ ] 代理配置正确（如果使用）
- [ ] DNS解析正常

## 🧪 功能测试检查清单

### ✅ 基本功能测试
- [ ] 授权URL生成成功
- [ ] 授权页面可正常访问
- [ ] 用户可完成授权流程
- [ ] 回调处理成功
- [ ] Token获取成功

### ✅ 邮件功能测试
- [ ] 可获取用户基本信息
- [ ] 可读取邮件列表
- [ ] 可获取邮件详情
- [ ] 可处理附件

### ✅ Token管理测试
- [ ] Token自动刷新
- [ ] 刷新失败处理
- [ ] Token撤销功能

## 🚨 常见问题排查

### 问题1：AADSTS50011 - 重定向URI不匹配
**检查项：**
- [ ] Azure应用注册中的重定向URI
- [ ] 环境变量中的重定向URI
- [ ] URI格式是否完全一致（包括协议、域名、端口、路径）

### 问题2：AADSTS65001 - 用户未同意
**检查项：**
- [ ] API权限是否正确配置
- [ ] 是否授予了管理员同意
- [ ] 权限范围是否正确

### 问题3：invalid_client - 客户端认证失败
**检查项：**
- [ ] 客户端ID是否正确
- [ ] 客户端密钥是否正确
- [ ] 密钥是否已过期

### 问题4：insufficient_claims - 权限不足
**检查项：**
- [ ] Mail.ReadWrite权限是否已添加
- [ ] offline_access权限是否已添加
- [ ] 权限是否已授予管理员同意

## 🔍 验证命令

### 快速配置验证
```bash
# 运行配置验证脚本
./scripts/setup-microsoft-oauth2.sh
```

### 手动验证步骤
```bash
# 1. 检查环境变量
grep MICROSOFT backend/.env

# 2. 编译后端
cd backend && go build -o fusionmail ./cmd/server

# 3. 启动服务
./fusionmail

# 4. 测试健康检查
curl http://localhost:3333/api/v1/health

# 5. 测试授权URL生成
curl http://localhost:3333/api/v1/auth/microsoft/authorize
```

## 📞 获取帮助

如果遇到问题，请检查：

1. **Azure Portal 日志**
   - Azure AD > 登录日志
   - 应用注册 > 身份验证日志

2. **FusionMail 日志**
   - 后端服务日志
   - 浏览器开发者工具

3. **文档资源**
   - [Microsoft Graph 文档](https://docs.microsoft.com/en-us/graph/)
   - [Azure AD 应用注册](https://docs.microsoft.com/en-us/azure/active-directory/develop/)

---

**检查清单版本：** v1.0  
**创建日期：** 2025-01-31  
**适用版本：** FusionMail v0.1.0+