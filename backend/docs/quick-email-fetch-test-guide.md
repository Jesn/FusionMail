# 短效邮箱邮件获取测试指南

## 概述

本指南将帮助您测试短效邮箱适配器的邮件获取功能，验证是否能够成功从 Microsoft Outlook/Hotmail 账户中获取邮件。

## 测试前准备

### 1. 获取必要的凭据

要测试短效邮箱功能，您需要以下信息：

- **邮箱地址**: Outlook/Hotmail 邮箱 (如: user@outlook.com)
- **Client ID**: Azure 应用的客户端 ID
- **Refresh Token**: OAuth2 刷新令牌

### 2. 设置 Azure 应用权限

确保您的 Azure 应用具有以下权限：
- `Mail.Read` - 读取邮件 (必需)
- `Mail.ReadWrite` - 读写邮件 (可选)
- `User.Read` - 读取用户信息 (推荐)

## 测试方法

### 方法 1: 快速测试 (推荐)

使用简化的测试脚本进行快速验证：

```bash
cd backend

# 方法 1a: 使用命令行参数
go run scripts/quick_email_test.go "your@outlook.com----password----refresh_token----client_id"

# 方法 1b: 使用环境变量
export QUICK_ACCOUNT_STRING="your@outlook.com----password----refresh_token----client_id"
go run scripts/quick_email_test.go
```

**预期输出**:
```
⚡ 短效邮箱快速测试
==============================
📧 邮箱: your@outlook.com
🔍 测试连接... ✅ 成功
📬 获取邮件... ✅ 获取到 5 封邮件

📧 邮件列表:
  1. Important Meeting Tomorrow
     发件人: boss@company.com | 时间: 11-03 14:30
  2. Weekly Newsletter
     发件人: newsletter@service.com | 时间: 11-02 09:15

🔍 获取邮件详情... ✅ 成功 (正文长度: 1234 字符)

🎉 测试完成! 短效邮箱功能正常
```

### 方法 2: 详细测试

使用完整的测试脚本进行全面验证：

```bash
cd backend

# 方法 2a: 交互式输入
go run scripts/test_quick_email_fetch.go

# 方法 2b: 环境变量
export QUICK_EMAIL="your@outlook.com"
export QUICK_CLIENT_ID="your_client_id"
export QUICK_REFRESH_TOKEN="your_refresh_token"
go run scripts/test_quick_email_fetch.go

# 方法 2c: 账户字符串
go run scripts/test_quick_email_fetch.go "your@outlook.com----password----refresh_token----client_id"
```

### 方法 3: 演示模式

查看短效邮箱功能演示（不需要真实凭据）：

```bash
cd backend
go run scripts/demo_quick_email.go
```

## 账户字符串格式

短效邮箱使用特定的字符串格式：

```
email----password----refresh_token----client_id
```

**字段说明**:
- `email`: Outlook/Hotmail 邮箱地址
- `password`: 可以为空，用 `----` 占位
- `refresh_token`: OAuth2 刷新令牌
- `client_id`: Azure 应用的客户端 ID

**示例**:
```
user@outlook.com--------eyJ0eXAiOiJKV1QiLCJhbGc...----12345678-1234-1234-1234-123456789abc
```

## 测试场景

### 场景 1: 基础连接测试

验证短效适配器能否成功连接到 Microsoft Graph API：

```bash
# 测试连接
go run scripts/quick_email_test.go "your_account_string"
```

**验证点**:
- ✅ 连接测试成功
- ✅ Token 刷新正常
- ✅ 用户信息获取成功

### 场景 2: 邮件列表获取

验证能否获取不同时间范围的邮件：

```bash
# 详细邮件测试
go run scripts/test_quick_email_fetch.go "your_account_string"
```

**验证点**:
- ✅ 获取最近24小时邮件
- ✅ 获取最近7天邮件
- ✅ 获取最近30天邮件
- ✅ 邮件数据格式正确

### 场景 3: 邮件详情获取

验证能否获取单封邮件的完整信息：

**验证点**:
- ✅ 邮件基本信息 (主题、发件人、时间)
- ✅ 邮件正文内容
- ✅ 附件信息 (如果有)
- ✅ 邮件状态 (已读/未读)

### 场景 4: 错误处理测试

测试各种错误情况的处理：

```bash
# 使用无效的凭据测试
go run scripts/quick_email_test.go "invalid----account----string----format"
```

**验证点**:
- ✅ 无效凭据错误处理
- ✅ 网络错误重试机制
- ✅ 权限错误提示
- ✅ 友好的错误消息

## 测试结果分析

### 成功指标

1. **连接成功**: 能够成功连接到 Microsoft Graph API
2. **Token 有效**: Refresh Token 能够成功获取 Access Token
3. **邮件获取**: 能够获取到邮件列表和详情
4. **数据完整**: 邮件数据包含所有必要字段
5. **性能良好**: 响应时间在可接受范围内 (< 5秒)

### 常见问题排查

#### 1. 连接失败

**错误**: `invalid_grant: The provided authorization grant is invalid`

**原因**: 
- Refresh Token 已过期
- Client ID 不正确
- 应用权限配置问题

**解决方案**:
```bash
# 检查 Token 状态
curl -X POST https://login.microsoftonline.com/common/oauth2/v2.0/token \
  -H "Content-Type: application/x-www-form-urlencoded" \
  -d "grant_type=refresh_token&refresh_token=YOUR_REFRESH_TOKEN&client_id=YOUR_CLIENT_ID"
```

#### 2. 权限不足

**错误**: `Insufficient privileges to complete the operation`

**解决方案**:
1. 在 Azure 门户中添加 `Mail.Read` 权限
2. 确保管理员已同意权限
3. 重新生成 Refresh Token

#### 3. 网络问题

**错误**: `context deadline exceeded` 或连接超时

**解决方案**:
1. 检查网络连接
2. 配置代理设置 (如果需要)
3. 增加超时时间

#### 4. 邮件获取为空

**现象**: 连接成功但获取不到邮件

**排查步骤**:
1. 检查邮箱是否有邮件
2. 调整时间范围 (扩大到30天或更长)
3. 检查邮件权限范围
4. 验证邮箱地址是否正确

## 性能基准

### 预期性能指标

- **连接时间**: < 2 秒
- **Token 刷新**: < 1 秒  
- **邮件列表获取**: < 3 秒 (10封邮件)
- **邮件详情获取**: < 2 秒 (单封邮件)
- **总测试时间**: < 10 秒

### 性能优化建议

1. **Token 缓存**: 避免频繁刷新 Token
2. **并发控制**: 限制同时连接数
3. **分页获取**: 大量邮件分批获取
4. **超时设置**: 合理设置超时时间

## 自动化测试

### 持续集成测试

创建自动化测试脚本：

```bash
#!/bin/bash
# test-quick-email-ci.sh

set -e

echo "🚀 开始短效邮箱自动化测试"

# 检查环境变量
if [ -z "$QUICK_ACCOUNT_STRING" ]; then
    echo "❌ 缺少 QUICK_ACCOUNT_STRING 环境变量"
    exit 1
fi

# 运行测试
cd backend
go run scripts/quick_email_test.go "$QUICK_ACCOUNT_STRING"

echo "✅ 短效邮箱测试通过"
```

### 定期验证

设置定期测试任务验证短效邮箱功能：

```bash
# 每日验证脚本
0 9 * * * /path/to/test-quick-email-ci.sh
```

## 安全注意事项

1. **凭据保护**: 不要在日志中记录敏感信息
2. **环境隔离**: 测试和生产环境分离
3. **权限最小化**: 只授予必要的权限
4. **定期轮换**: 定期更新 Refresh Token
5. **监控告警**: 监控异常访问和失败率

## 故障排除工具

### 调试模式

启用详细日志进行调试：

```bash
export LOG_LEVEL=debug
go run scripts/test_quick_email_fetch.go "your_account_string"
```

### 网络诊断

测试网络连接：

```bash
# 测试 Microsoft Graph API 连通性
curl -v https://graph.microsoft.com/v1.0/me

# 测试 OAuth2 端点
curl -v https://login.microsoftonline.com/common/oauth2/v2.0/token
```

### Token 验证

验证 Access Token 有效性：

```bash
# 使用 Access Token 测试 API 调用
curl -H "Authorization: Bearer YOUR_ACCESS_TOKEN" \
     https://graph.microsoft.com/v1.0/me
```

## 总结

通过以上测试方法，您可以全面验证短效邮箱适配器的邮件获取功能。建议按照以下顺序进行测试：

1. **快速测试** - 验证基本功能
2. **详细测试** - 验证完整功能
3. **错误测试** - 验证异常处理
4. **性能测试** - 验证性能指标
5. **自动化测试** - 集成到 CI/CD

如果所有测试都通过，说明短效邮箱适配器可以正常获取邮件，可以继续进行后续的集成和部署工作。

---

*更新时间: 2025-11-03*
*版本: 1.0*