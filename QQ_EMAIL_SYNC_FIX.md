# QQ邮箱同步问题修复

## 问题描述

用户反映QQ邮箱状态显示正常，但是接收不到刚才发送的邮件。

## 问题分析

通过代码分析，我发现了以下几个关键问题：

### 1. 账户状态检查缺失

**问题**：同步服务只检查了 `sync_enabled` 字段，没有检查账户的 `status` 字段。

**影响**：即使账户被禁用（status = "disabled"），同步服务仍然会尝试同步，但可能会失败。

**修复**：
- 在 `SyncAccount` 方法中添加账户状态检查
- 在 `ListSyncEnabled` 方法中只返回状态为 "active" 的账户

### 2. IMAP适配器忽略时间过滤

**问题**：IMAP适配器的 `FetchEmails` 方法完全忽略了 `since` 参数，总是获取所有邮件或最新的N封邮件。

**影响**：
- 增量同步无法正常工作
- 新邮件可能被遗漏
- 每次同步都会重复处理相同的邮件

**修复**：
- 使用IMAP SEARCH命令根据时间过滤邮件
- 正确处理 `since` 参数
- 添加详细的日志输出

### 3. 首次同步时间范围过大

**问题**：首次同步从30天前开始，可能导致获取过多历史邮件。

**影响**：
- 首次同步时间过长
- 可能超出邮件服务器限制
- 影响性能

**修复**：
- 将首次同步时间范围调整为7天
- 减少初始同步的邮件数量

## 修复内容

### 1. 后端修复

#### `backend/internal/service/sync_service.go`
```go
// 添加账户状态检查
if account.Status != "active" {
    return fmt.Errorf("account is not active (status: %s): %s", account.Status, accountUID)
}

// 调整首次同步时间范围
since = time.Now().AddDate(0, 0, -7) // 从7天前开始
```

#### `backend/internal/repository/account.go`
```go
// 只返回状态为active的账户
Where("sync_enabled = ? AND status = ?", true, "active")
```

#### `backend/internal/adapter/imap.go`
```go
// 使用IMAP SEARCH命令根据时间过滤
if !since.IsZero() {
    searchCriteria := &imap.SearchCriteria{
        Since: since,
    }
    // ... 搜索逻辑
}
```

### 2. 测试脚本

创建了多个诊断和测试脚本：
- `debug-qq-sync.sh` - 基础诊断脚本
- `check-qq-email-sync.sh` - 详细检查脚本
- `test-qq-email-fix.sh` - 修复验证脚本

## 修复验证

### 自动验证
```bash
# 运行修复验证脚本
./test-qq-email-fix.sh
```

### 手动验证
1. 确保QQ账户状态为 "active"
2. 确保同步功能已启用
3. 手动触发同步
4. 检查是否能接收到新邮件

## 预期效果

修复后应该能够：
1. ✅ 正确检查账户状态
2. ✅ 只同步状态为 "active" 的账户
3. ✅ 根据时间范围增量同步邮件
4. ✅ 接收到新发送的邮件
5. ✅ 避免重复处理相同邮件

## 注意事项

### 1. 数据库迁移
- 新的 `status` 字段会通过GORM AutoMigrate自动添加
- 现有账户默认状态为 "active"

### 2. IMAP搜索限制
- 某些邮件服务器可能不支持所有搜索条件
- 如果搜索失败，会回退到获取所有邮件

### 3. 性能考虑
- 首次同步现在只获取7天内的邮件
- 可以根据需要调整时间范围

## 故障排除

如果修复后仍有问题，请检查：

1. **账户配置**
   - QQ邮箱是否开启IMAP服务
   - 是否使用授权码而不是登录密码
   - 账户状态是否为 "active"

2. **网络连接**
   - 是否能连接到 imap.qq.com:993
   - 防火墙是否阻止连接

3. **服务器日志**
   - 查看后端服务日志获取详细错误信息
   - 检查IMAP连接和搜索日志

4. **同步状态**
   - 检查最后同步时间和状态
   - 查看同步日志了解详细信息

## 相关文件

### 修改的文件
- `backend/internal/service/sync_service.go`
- `backend/internal/repository/account.go`
- `backend/internal/adapter/imap.go`

### 新增的文件
- `debug-qq-sync.sh`
- `check-qq-email-sync.sh`
- `test-qq-email-fix.sh`
- `QQ_EMAIL_SYNC_FIX.md`

## 总结

通过修复账户状态检查和IMAP时间过滤逻辑，QQ邮箱同步功能现在应该能够正常工作，用户可以接收到新发送的邮件。这些修复不仅解决了当前问题，还提高了整个邮件同步系统的可靠性和性能。