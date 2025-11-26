# OAuth2 客户端使用次数统计修复

## 问题描述

用户反馈：
- 配置了 2 个 Gmail 账号，但 Gmail OAuth2 客户端显示使用次数为 1
- 配置了 3 个 Outlook 账号，但 Outlook OAuth2 客户端显示使用次数为 0

## 根本原因

在 OAuth2 认证流程中，`oauth2config.Provider` 的 `GetOAuth2Config` 和 `GetOAuth2ConfigForClient` 方法只是选择了客户端配置，但**没有调用 `IncrementUsage` 来增加使用计数**。

### 代码流程分析

```
用户添加账号
  ↓
OAuth2Handler.GoogleCallback / MicrosoftCallback
  ↓
OAuth2Service.HandleCallback
  ↓
OAuth2Service.getOAuth2Config
  ↓
oauth2config.Provider.GetOAuth2Config
  ↓
选择客户端配置 ← 这里缺少 IncrementUsage 调用
  ↓
返回 OAuth2 配置
```

## 修复方案

在 `backend/pkg/oauth2config/provider.go` 中的两个方法中添加使用计数逻辑：

### 1. GetOAuth2Config 方法

```go
// 选择默认客户端或第一个可用客户端
var selectedClient *model.OAuth2Client
// ... 选择逻辑 ...

// ✅ 新增：增加使用计数
if err := p.clientRepo.IncrementUsage(ctx, selectedClient.ID); err != nil {
    p.logger.Error("Failed to increment OAuth2 client usage",
        "client_id", selectedClient.ID,
        "error", err)
    // 不返回错误，继续执行
} else {
    p.logger.Info("OAuth2 client usage incremented",
        "client_id", selectedClient.ID,
        "client_name", selectedClient.Name)
}
```

### 2. GetOAuth2ConfigForClient 方法

```go
// 获取客户端配置
client, err := p.clientRepo.FindByID(ctx, clientID)
// ... 错误处理 ...

// ✅ 新增：增加使用计数
if err := p.clientRepo.IncrementUsage(ctx, client.ID); err != nil {
    p.logger.Error("Failed to increment OAuth2 client usage",
        "client_id", client.ID,
        "error", err)
    // 不返回错误，继续执行
} else {
    p.logger.Info("OAuth2 client usage incremented",
        "client_id", client.ID,
        "client_name", client.Name)
}
```

## 修复效果

修复后，每次使用 OAuth2 客户端配置时都会正确增加使用计数：

### 使用场景

1. **添加新账号**：OAuth2 认证时 +1
2. **Token 刷新**：刷新访问令牌时 +1
3. **邮件同步**：每次同步可能刷新 token 时 +1

### 预期结果

```
初始状态：
- Gmail OAuth2 客户端: usage_count = 0
- Outlook OAuth2 客户端: usage_count = 0

添加第 1 个 Gmail 账号：
- Gmail OAuth2 客户端: usage_count = 1 ✅

添加第 2 个 Gmail 账号：
- Gmail OAuth2 客户端: usage_count = 2 ✅

添加第 1 个 Outlook 账号：
- Outlook OAuth2 客户端: usage_count = 1 ✅

添加第 2 个 Outlook 账号：
- Outlook OAuth2 客户端: usage_count = 2 ✅

添加第 3 个 Outlook 账号：
- Outlook OAuth2 客户端: usage_count = 3 ✅
```

## 验证方法

### 1. 重启后端服务

```bash
# 停止现有服务
pkill -f "fusionmail"

# 重新编译
cd backend
go build -o ../bin/fusionmail ./cmd/server

# 启动服务
../bin/fusionmail
```

### 2. 测试新账号添加

1. 访问 `/oauth2-clients` 页面，记录当前使用次数
2. 添加一个新的 Gmail 或 Outlook 账号
3. 刷新 `/oauth2-clients` 页面，验证使用次数是否 +1

### 3. 测试 Token 刷新

```bash
# 手动触发 token 刷新
curl -X POST http://localhost:3333/api/v1/auth/google/refresh/{account_uid}
```

### 4. 查看日志

日志中应该能看到：

```
[INFO] Selected OAuth2 client for usage client_id=1 client_name="Gmail 生产环境" is_default=true provider_type=1
[INFO] OAuth2 client usage incremented client_id=1 client_name="Gmail 生产环境"
```

## 注意事项

1. **不影响现有功能**：即使 `IncrementUsage` 失败，也不会影响 OAuth2 认证流程
2. **日志记录完善**：添加了详细的日志，方便排查问题
3. **向后兼容**：对现有账号没有影响，只影响新的使用计数

## 相关文件

- `backend/pkg/oauth2config/provider.go` - 主要修复文件
- `backend/internal/repository/oauth2_client.go` - IncrementUsage 实现
- `backend/internal/model/oauth2_client.go` - OAuth2Client 模型

## 测试建议

建议在以下场景测试：

1. ✅ 添加新的 Gmail 账号
2. ✅ 添加新的 Outlook 账号
3. ✅ 刷新现有账号的 token
4. ✅ 邮件同步过程中的 token 刷新
5. ✅ 多个账号共享同一个 OAuth2 客户端配置

## 后续优化建议

1. **配额管理**：当使用次数达到配额限制时，自动切换到其他可用客户端
2. **使用统计**：在管理页面显示每个账号使用的客户端配置
3. **重置计数**：提供按天/月重置使用计数的功能
4. **告警机制**：当使用次数接近配额时发送告警

---

**修复时间**：2025-11-26  
**修复人员**：Kiro AI Assistant  
**影响范围**：OAuth2 认证流程  
**风险等级**：低（仅增加计数逻辑，不影响核心功能）
