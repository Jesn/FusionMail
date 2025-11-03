# Microsoft Graph OAuth2 Azure 配置指南

## 概述

本文档详细说明如何在Azure Portal中配置Microsoft Graph OAuth2应用，以便FusionMail能够安全地访问用户的Outlook/Hotmail邮箱。

## 前置条件

- Microsoft账号（个人账号或工作账号）
- Azure Portal访问权限
- FusionMail项目已部署并可访问

## 配置步骤

### 1. 创建Azure AD应用注册

#### 1.1 访问Azure Portal
1. 打开 [Azure Portal](https://portal.azure.com/)
2. 使用Microsoft账号登录

#### 1.2 导航到应用注册
1. 在顶部搜索栏输入"应用注册"
2. 点击"Azure Active Directory"
3. 在左侧菜单中选择"应用注册"

#### 1.3 创建新应用
1. 点击"+ 新注册"
2. 填写应用信息：
   ```
   名称: FusionMail
   支持的账户类型: 任何组织目录(任何 Azure AD 目录 - 多租户)中的账户和个人 Microsoft 账户
   重定向 URI (可选): 
     - 平台: Web
     - URI: http://localhost:3333/api/v1/auth/microsoft/callback
   ```
3. 点击"注册"

### 2. 配置API权限

#### 2.1 添加Microsoft Graph权限
1. 在应用页面，点击左侧"API 权限"
2. 点击"+ 添加权限"
3. 选择"Microsoft Graph"
4. 选择"委托的权限"
5. 搜索并添加以下权限：

**必需权限：**
- `Mail.ReadWrite` - 读取和写入用户邮件
- `User.Read` - 登录并读取用户配置文件  
- `offline_access` - 维持对数据的访问权限

#### 2.2 授予管理员同意（推荐）
1. 点击"为 [组织名称] 授予管理员同意"
2. 在弹出对话框中点击"是"
3. 确认所有权限状态显示为"已授予"

### 3. 创建客户端密钥

#### 3.1 生成密钥
1. 点击左侧"证书和密钥"
2. 在"客户端密钥"部分，点击"+ 新客户端密钥"
3. 填写信息：
   ```
   说明: FusionMail Client Secret
   过期时间: 24个月 (推荐)
   ```
4. 点击"添加"

#### 3.2 保存重要信息
**⚠️ 重要：立即复制并保存以下信息，密钥值只显示一次！**

```
应用程序(客户端) ID: xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx
客户端密钥值: xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx
目录(租户) ID: xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx
```

### 4. 配置身份验证

#### 4.1 设置重定向URI
1. 点击左侧"身份验证"
2. 在"平台配置"部分，确认Web平台已添加
3. 添加重定向URI：
   ```
   开发环境: http://localhost:3333/api/v1/auth/microsoft/callback
   生产环境: https://your-domain.com/api/v1/auth/microsoft/callback
   ```

#### 4.2 高级设置
在"隐式授权和混合流"部分：
- ✅ 启用"访问令牌"
- ✅ 启用"ID令牌"

### 5. 更新FusionMail配置

#### 5.1 环境变量配置
将获取的信息更新到 `backend/.env` 文件：

```bash
# Microsoft Graph API 配置
MICROSOFT_CLIENT_ID=xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx
MICROSOFT_CLIENT_SECRET=xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx
MICROSOFT_REDIRECT_URI=http://localhost:3333/api/v1/auth/microsoft/callback
```

#### 5.2 生产环境配置
对于生产环境，确保：
1. 使用HTTPS重定向URI
2. 客户端密钥安全存储
3. 定期轮换密钥

## 测试配置

### 1. 启动FusionMail
```bash
cd backend
go run ./cmd/server
```

### 2. 测试OAuth2流程
1. 访问 FusionMail 前端
2. 点击"添加账户"
3. 选择"Microsoft/Outlook"
4. 点击"使用 Outlook 账号登录"
5. 完成Microsoft授权流程

### 3. 验证权限
确认应用能够：
- 获取用户基本信息
- 读取邮件列表
- 访问邮件详情

## 常见问题

### Q1: 授权时提示"AADSTS50011"错误
**原因：** 重定向URI不匹配
**解决：** 确保Azure应用注册中的重定向URI与代码中配置的完全一致

### Q2: 无法获取邮件权限
**原因：** API权限配置不正确
**解决：** 检查是否添加了 `Mail.ReadWrite` 权限并授予了管理员同意

### Q3: Token刷新失败
**原因：** 缺少 `offline_access` 权限
**解决：** 确保添加了 `offline_access` 权限

### Q4: 个人账户无法登录
**原因：** 账户类型配置错误
**解决：** 确保选择了"任何组织目录中的账户和个人 Microsoft 账户"

## 安全最佳实践

### 1. 密钥管理
- 定期轮换客户端密钥（建议每12个月）
- 使用Azure Key Vault存储生产环境密钥
- 不要在代码中硬编码密钥

### 2. 权限最小化
- 只请求应用实际需要的权限
- 定期审查权限使用情况
- 移除不再需要的权限

### 3. 监控和审计
- 启用Azure AD登录日志
- 监控异常的API调用
- 设置权限变更告警

## 支持的账户类型

### 个人Microsoft账户
- @outlook.com
- @hotmail.com  
- @live.com
- @msn.com

### 工作或学校账户
- 企业Azure AD账户
- 教育机构账户
- 政府机构账户

## 相关链接

- [Microsoft Graph API 文档](https://docs.microsoft.com/en-us/graph/)
- [Azure AD 应用注册文档](https://docs.microsoft.com/en-us/azure/active-directory/develop/quickstart-register-app)
- [Microsoft Graph 权限参考](https://docs.microsoft.com/en-us/graph/permissions-reference)
- [OAuth 2.0 授权代码流](https://docs.microsoft.com/en-us/azure/active-directory/develop/v2-oauth2-auth-code-flow)

---

**文档版本：** v1.0  
**创建日期：** 2025-01-31  
**适用版本：** FusionMail v0.1.0+