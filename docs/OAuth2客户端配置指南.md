# OAuth2 客户端配置指南

## 📋 概述

OAuth2 客户端配置用于 Gmail 和 Outlook 账户的 OAuth2 认证。系统已经移除了占位符数据，现在需要您配置真实的客户端凭据。

## 🗑️ 清理现有占位符数据

如果数据库中存在占位符数据（`your-gmail-client-id`、`your-outlook-client-id`），请通过以下方式清理：

### 方法一：前端界面删除
1. 访问 http://localhost:4444/oauth2-clients
2. 点击占位符配置项的"删除"按钮
3. 确认删除

### 方法二：数据库直接清理
```sql
DELETE FROM oauth2_clients
WHERE client_id LIKE 'your-%-client-id';
```

## 🔑 获取 OAuth2 客户端凭据

### Gmail (Google OAuth2)

1. **访问 Google Cloud Console**
   - 打开 https://console.cloud.google.com/
   - 登录您的 Google 账户

2. **创建或选择项目**
   - 创建新项目或选择现有项目

3. **启用 Gmail API**
   - 转到"API 和服务" > "库"
   - 搜索"Gmail API"
   - 点击并启用

4. **创建 OAuth2 凭据**
   - 转到"API 和服务" > "凭据"
   - 点击"创建凭据" > "OAuth 客户端 ID"
   - 选择"Web 应用程序"
   - 名称：输入任意名称（如：FusionMail Gmail Client）
   - **授权的重定向 URI**：
     ```
     http://localhost:3333/api/v1/auth/google/callback
     ```

5. **获取凭据**
   - 创建后，复制**客户端 ID**和**客户端密钥**

### Outlook (Microsoft OAuth2)

1. **访问 Microsoft Azure Portal**
   - 打开 https://portal.azure.com/
   - 登录您的 Microsoft 账户

2. **注册应用程序**
   - 转到"Azure Active Directory" > "应用注册"
   - 点击"新注册"

3. **配置应用**
   - 名称：输入任意名称（如：FusionMail Outlook Client）
   - **重定向 URI**：
     ```
     http://localhost:3333/api/v1/auth/microsoft/callback
     ```
   - 点击"注册"

4. **配置 API 权限**
   - 转到"API 权限"
   - 点击"添加权限" > "Microsoft Graph"
   - 选择以下权限：
     - `offline_access`
     - `Mail.Read`
     - `Mail.ReadWrite`
     - `Mail.Send`
   - 点击"授予管理员同意"

5. **创建客户端密钥**
   - 转到"证书和密码"
   - 点击"新客户端密码"
   - 描述：输入任意描述
   - 到期：选择时长
   - 点击"添加"
   - **立即复制客户端密码**（只显示一次）

6. **获取凭据**
   - 在"概述"页面，复制**应用程序(客户端) ID**
   - 客户端密码已在步骤5中获取

## ⚙️ 配置 OAuth2 客户端

### 通过前端界面配置

1. **访问 OAuth2 客户端管理页面**
   - http://localhost:4444/oauth2-clients

2. **创建新配置**
   - 点击"新增客户端"按钮

3. **填写配置信息**
   ```
   邮箱提供商：gmail 或 outlook
   配置名称：输入描述性名称（如：我的 Gmail 配置）
   客户端ID：粘贴从步骤获取的 Client ID
   客户端密钥：粘贴从步骤获取的 Client Secret
   重定向URI：系统已自动填充，无需修改
   配额设置：
     - 日配额：100（可选）
     - 月配额：2000（可选）
   ```

4. **保存配置**
   - 点击"创建"按钮

5. **设置为默认（可选）**
   - 在配置列表中，点击"设为默认"按钮
   - 每个提供商只能有一个默认配置

## ✅ 验证配置

配置完成后，您可以：

1. **测试 OAuth2 流程**
   - 访问 http://localhost:4444/accounts
   - 点击"添加账户"
   - 选择 Gmail 或 Outlook
   - 选择 OAuth2 协议
   - 点击"通过 OAuth2 登录"

2. **检查配置使用情况**
   - 在 OAuth2 客户端列表中查看"使用次数"统计
   - 查看"最后使用时间"了解最近使用情况

## 🔧 常见问题

### Q: 提示"redirect_uri_mismatch"错误？
A: 检查重定向 URI 是否正确设置为：
   - Gmail: `http://localhost:3333/api/v1/auth/google/callback`
   - Outlook: `http://localhost:3333/api/v1/auth/microsoft/callback`

### Q: 提示"invalid_client"错误？
A: 检查客户端 ID 和客户端密钥是否正确粘贴，没有多余的空格或换行。

### Q: 可以使用多个配置吗？
A: 可以！系统支持为每个提供商创建多个 OAuth2 客户端配置，并支持智能切换和配额管理。

### Q: 如何备份配置？
A: OAuth2 客户端配置存储在数据库的 `oauth2_clients` 表中，建议定期备份数据库。

## 📚 参考资料

- [Google OAuth2 文档](https://developers.google.com/identity/protocols/oauth2)
- [Microsoft OAuth2 文档](https://docs.microsoft.com/en-us/azure/active-directory/develop/)
- [OAuth2 标准规范](https://oauth.net/2/)

## 🆘 需要帮助？

如果遇到问题，请：
1. 检查浏览器控制台的网络请求错误
2. 查看后端日志文件
3. 确认 OAuth2 提供商（Google/Microsoft）的配置正确
