# 批量启用/禁用账号功能实现文档

## 📋 功能概述

在账号管理页面（`/accounts`）添加了批量启用和批量禁用功能，允许用户一次性对多个账号进行状态切换操作。

## 🎯 实现内容

### 后端实现

#### 1. Service 层（`backend/internal/service/account_service.go`）

**新增接口方法：**
```go
// BatchEnableAccounts 批量启用账户
BatchEnableAccounts(ctx context.Context, uids []string) (*BatchOperationResult, error)

// BatchDisableAccounts 批量禁用账户
BatchDisableAccounts(ctx context.Context, uids []string) (*BatchOperationResult, error)
```

**新增数据结构：**
```go
// BatchOperationResult 批量操作结果
type BatchOperationResult struct {
    Success      int                       `json:"success"`       // 成功数量
    Failed       int                       `json:"failed"`        // 失败数量
    Total        int                       `json:"total"`         // 总数量
    FailedItems  []BatchOperationFailedItem `json:"failed_items"` // 失败项详情
}

// BatchOperationFailedItem 批量操作失败项
type BatchOperationFailedItem struct {
    UID   string `json:"uid"`   // 账户 UID
    Email string `json:"email"` // 邮箱地址
    Error string `json:"error"` // 错误信息
}
```

**实现逻辑：**
- 遍历所有账号 UID
- 对每个账号调用单个启用/禁用方法
- 收集成功和失败的结果
- 返回详细的操作报告

#### 2. Handler 层（`backend/internal/handler/account_handler.go`）

**新增 API 端点：**
```go
// POST /api/v1/accounts/batch/enable
func (h *AccountHandler) BatchEnableAccounts(c *gin.Context)

// POST /api/v1/accounts/batch/disable
func (h *AccountHandler) BatchDisableAccounts(c *gin.Context)
```

**请求格式：**
```json
{
  "uids": ["uid1", "uid2", "uid3"]
}
```

**响应格式：**
```json
{
  "success": true,
  "message": "批量启用完成: 成功 3 个，失败 0 个",
  "data": {
    "success": 3,
    "failed": 0,
    "total": 3,
    "failed_items": []
  }
}
```

#### 3. 路由注册（`backend/internal/router/router.go`）

```go
// 批量操作
accounts.POST("/batch/enable", accountHandler.BatchEnableAccounts)   // 批量启用账户
accounts.POST("/batch/disable", accountHandler.BatchDisableAccounts) // 批量禁用账户
```

### 前端实现

#### 1. Service 层（`frontend/src/services/accountService.ts`）

**新增方法：**
```typescript
// 批量启用账户
batchEnable: async (uids: string[]): Promise<BatchOperationResult>

// 批量禁用账户
batchDisable: async (uids: string[]): Promise<BatchOperationResult>
```

#### 2. 页面组件（`frontend/src/pages/AccountsPage.tsx`）

**新增 UI 元素：**
- 批量启用按钮：显示在表格头部，选中账号时可见
- 批量禁用按钮：显示在表格头部，选中账号时可见
- 按钮显示选中账号数量

**新增处理函数：**
```typescript
// 批量启用账户
const handleBatchEnable = async () => {
  // 调用 API
  // 显示结果提示
  // 刷新数据
}

// 批量禁用账户
const handleBatchDisable = async () => {
  // 调用 API
  // 显示结果提示
  // 刷新数据
}
```

**用户体验优化：**
- 操作过程中显示加载状态（按钮禁用）
- 操作完成后显示详细结果（成功/失败数量）
- 如有失败项，显示失败账号的邮箱地址
- 自动清空选择并刷新列表

## 🚀 使用方法

### 1. 选择账号
在账号管理页面的表格中，勾选需要操作的账号（支持多选）

### 2. 执行批量操作
点击表格头部的"批量启用"或"批量禁用"按钮

### 3. 查看结果
- 全部成功：显示绿色成功提示
- 部分失败：显示黄色警告提示，包含失败账号信息
- 全部失败：显示红色错误提示

## 📊 API 测试示例

### 批量启用账户

**请求：**
```bash
curl -X POST http://localhost:3333/api/v1/accounts/batch/enable \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer YOUR_TOKEN" \
  -d '{
    "uids": ["account-uid-1", "account-uid-2"]
  }'
```

**响应：**
```json
{
  "success": true,
  "message": "批量启用完成: 成功 2 个，失败 0 个",
  "data": {
    "success": 2,
    "failed": 0,
    "total": 2,
    "failed_items": []
  }
}
```

### 批量禁用账户

**请求：**
```bash
curl -X POST http://localhost:3333/api/v1/accounts/batch/disable \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer YOUR_TOKEN" \
  -d '{
    "uids": ["account-uid-1", "account-uid-2"]
  }'
```

## 🔍 技术细节

### 设计考虑

1. **事务处理**：当前实现为逐个处理，未使用数据库事务。这样设计的原因是：
   - 允许部分成功，不会因为一个账号失败导致全部回滚
   - 提供详细的失败信息，便于用户定位问题
   - 符合批量操作的常见场景

2. **错误处理**：
   - 每个账号的操作独立进行
   - 失败不影响其他账号的处理
   - 返回详细的失败信息（UID、邮箱、错误原因）

3. **性能优化**：
   - 后端循环处理，避免前端多次请求
   - 可以考虑后续优化为并发处理（使用 goroutine）

### 扩展建议

如果需要进一步优化性能，可以考虑：

1. **并发处理**：使用 goroutine 并发处理多个账号
2. **批量更新**：使用数据库的批量更新语句
3. **异步处理**：对于大量账号，可以改为异步任务

## ✅ 测试清单

- [x] 后端编译通过
- [x] 前端编译通过
- [ ] 批量启用功能测试
- [ ] 批量禁用功能测试
- [ ] 部分失败场景测试
- [ ] 权限验证测试
- [ ] 并发操作测试

## 📝 注意事项

1. **权限控制**：确保用户有权限操作这些账号
2. **并发安全**：多个用户同时操作同一账号时的处理
3. **日志记录**：批量操作会记录详细日志，便于审计
4. **状态刷新**：操作完成后自动刷新账号列表和分组统计

## 🎨 UI 展示

批量操作按钮位于表格头部，仅在选中账号时显示：

```
┌─────────────────────────────────────────────────────────────┐
│ 所有账号 [20 个账号]                                          │
│                                                               │
│ [加入当前分组] [移动到分组] [批量同步] [批量启用] [批量禁用]  │
└─────────────────────────────────────────────────────────────┘
```

## 🔗 相关文件

### 后端
- `backend/internal/service/account_service.go` - 服务层实现
- `backend/internal/handler/account_handler.go` - API 处理层
- `backend/internal/router/router.go` - 路由注册

### 前端
- `frontend/src/services/accountService.ts` - API 服务
- `frontend/src/pages/AccountsPage.tsx` - 页面组件

## 📅 更新日期

2026-01-08
