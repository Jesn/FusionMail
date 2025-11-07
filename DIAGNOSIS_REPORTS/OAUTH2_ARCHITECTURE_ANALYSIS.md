# OAuth2 Token 刷新机制 - 架构分析

**目的**: 分析 OAuth2 Token 刷新应该在哪一层实现  
**日期**: 2025-11-05

---

## 📊 当前架构现状

### 1. 现有实现

#### GraphQuickAdapter (已实现 ✅)
```
✓ RefreshTokenIfNeeded()      - 主动刷新
✓ ForceRefreshToken()         - 强制刷新
✓ ClearToken()                - 清除 token
✓ handleTokenError()          - 错误处理
✓ ensureValidToken()          - 内部自动刷新
✓ makeAuthenticatedRequest()  - 请求时自动刷新
```

#### GmailAdapter (未实现 ❌)
```
✗ 没有 token 刷新机制
✗ 在 Connect 时创建 token，之后不管理
✗ 如果 token 过期，直接失败
```

#### sync_service (未实现 ❌)
```
✗ 没有 token 刷新逻辑
✗ 直接调用 provider.FetchEmails()
✗ 如果 token 过期，同步失败
```

---

## 🏗️ 两种实现方案对比

### 方案 A：在适配器层实现 (推荐 ⭐)

**实现位置**: `GmailAdapter`, `IMAPAdapter`, `POP3Adapter` 等

**优点**:
- ✅ 符合适配器模式的职责分离
- ✅ 每个适配器可以有自己的刷新策略
- ✅ 已经在 GraphQuickAdapter 中有成熟实现
- ✅ 适配器内部可以自动处理 token 过期
- ✅ 不同提供商的 token 管理方式不同，适配器层最合适
- ✅ 代码复用性好（通过继承或组合）

**缺点**:
- ❌ 需要在每个适配器中实现
- ❌ 代码可能有重复

**代码示例**:
```go
// GmailAdapter 中添加
func (a *GmailAdapter) RefreshTokenIfNeeded(ctx context.Context) error {
    if time.Now().Add(5 * time.Minute).Before(a.config.Credentials.TokenExpiry) {
        return nil // token 仍然有效
    }
    
    // 调用 OAuth2Service 刷新 token
    return a.refreshToken(ctx)
}

// 在 FetchEmails 前调用
func (a *GmailAdapter) FetchEmails(ctx context.Context, since time.Time, limit int) ([]*Email, error) {
    if err := a.RefreshTokenIfNeeded(ctx); err != nil {
        return nil, fmt.Errorf("token refresh failed: %w", err)
    }
    
    // 继续原有逻辑...
}
```

---

### 方案 B：在同步服务层实现

**实现位置**: `sync_service.go` 中的 `doSync()` 方法

**优点**:
- ✅ 集中管理，代码不重复
- ✅ 可以统一处理所有提供商
- ✅ 更容易添加日志和监控

**缺点**:
- ❌ 违反了适配器模式的职责分离
- ❌ 同步服务需要了解 token 细节
- ❌ 不同提供商的刷新策略难以定制
- ❌ 只在同步时刷新，其他操作（如测试连接）无法刷新
- ❌ 如果适配器内部需要刷新，无法处理

**代码示例**:
```go
// sync_service.go 中
func (s *syncService) doSync(ctx context.Context, account *model.Account, syncLog *model.SyncLog) error {
    // ... 创建适配器 ...
    
    // 在同步前刷新 token
    if err := s.ensureValidToken(ctx, account); err != nil {
        return fmt.Errorf("token refresh failed: %w", err)
    }
    
    // 继续原有逻辑...
}
```

---

## 🎯 推荐方案：混合实现

**最佳实践** = 适配器层 + 同步服务层 + 接口扩展

### 1️⃣ 扩展 MailProvider 接口

**文件**: `backend/internal/adapter/adapter.go`

```go
// 添加到 MailProvider 接口
type MailProvider interface {
    // ... 现有方法 ...
    
    // RefreshTokenIfNeeded 如果需要则刷新 token（可选）
    RefreshTokenIfNeeded(ctx context.Context) error
}
```

### 2️⃣ 在适配器层实现

**GmailAdapter**:
```go
func (a *GmailAdapter) RefreshTokenIfNeeded(ctx context.Context) error {
    // 检查 token 是否即将过期
    if time.Now().Add(5 * time.Minute).Before(a.config.Credentials.TokenExpiry) {
        return nil
    }
    
    // 刷新 token（需要调用 OAuth2Service）
    return a.refreshToken(ctx)
}
```

**IMAPAdapter**:
```go
func (a *IMAPAdapter) RefreshTokenIfNeeded(ctx context.Context) error {
    // IMAP 可能不需要刷新（取决于认证方式）
    return nil
}
```

### 3️⃣ 在同步服务层添加预检查

**文件**: `backend/internal/service/sync_service.go`

```go
func (s *syncService) doSync(ctx context.Context, account *model.Account, syncLog *model.SyncLog) error {
    // ... 创建适配器 ...
    
    // 预检查：刷新 token（如果需要）
    if tokenRefresher, ok := provider.(interface{ RefreshTokenIfNeeded(context.Context) error }); ok {
        if err := tokenRefresher.RefreshTokenIfNeeded(ctx); err != nil {
            return fmt.Errorf("token refresh failed: %w", err)
        }
    }
    
    // 继续原有逻辑...
    emails, err := provider.FetchEmails(ctx, since, 1000)
    // ...
}
```

---

## 📋 实现步骤

### 第 1 步：扩展接口 (1 小时)
- 在 `MailProvider` 接口中添加 `RefreshTokenIfNeeded()` 方法

### 第 2 步：实现 GmailAdapter (2-3 小时)
- 添加 token 刷新逻辑
- 添加错误处理
- 编写单元测试

### 第 3 步：实现其他适配器 (2-3 小时)
- IMAPAdapter
- POP3Adapter
- GraphAdapter

### 第 4 步：在同步服务中集成 (1-2 小时)
- 在 `doSync()` 中添加预检查
- 添加日志记录
- 编写集成测试

### 第 5 步：测试和验证 (2-3 小时)
- 单元测试
- 集成测试
- 性能测试

---

## 🔄 执行流程图

```
同步流程:
┌─────────────────────────────────────────────────────────┐
│ 1. sync_service.SyncAccount()                           │
└────────────────────┬────────────────────────────────────┘
                     │
                     ▼
┌─────────────────────────────────────────────────────────┐
│ 2. 创建适配器 (GmailAdapter/GraphQuickAdapter/etc)      │
└────────────────────┬────────────────────────────────────┘
                     │
                     ▼
┌─────────────────────────────────────────────────────────┐
│ 3. 预检查：RefreshTokenIfNeeded()                       │
│    (同步服务层)                                         │
└────────────────────┬────────────────────────────────────┘
                     │
                     ▼
┌─────────────────────────────────────────────────────────┐
│ 4. 适配器内部自动刷新：ensureValidToken()              │
│    (在 FetchEmails/FetchEmailDetail 时)                │
│    (适配器层)                                           │
└────────────────────┬────────────────────────────────────┘
                     │
                     ▼
┌─────────────────────────────────────────────────────────┐
│ 5. 发送 API 请求                                        │
└────────────────────┬────────────────────────────────────┘
                     │
                     ▼
┌─────────────────────────────────────────────────────────┐
│ 6. 如果收到 401 错误，自动刷新并重试                   │
│    (适配器层错误处理)                                   │
└─────────────────────────────────────────────────────────┘
```

---

## ✅ 完成标准

- [ ] MailProvider 接口扩展完成
- [ ] GmailAdapter 实现 token 刷新
- [ ] 其他适配器实现 token 刷新
- [ ] sync_service 集成预检查
- [ ] 单元测试覆盖率 > 80%
- [ ] 集成测试通过
- [ ] 性能测试通过

---

**建议**: 采用混合方案，既在适配器层实现自动刷新，也在同步服务层添加预检查，形成双重保障。


