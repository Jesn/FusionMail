# OAuth2 Token 刷新 - 具体实现代码

**目的**: 提供可直接使用的实现代码  
**日期**: 2025-11-05

---

## 📝 第 1 步：扩展 MailProvider 接口

**文件**: `backend/internal/adapter/adapter.go`

```go
// 在 MailProvider 接口中添加
type MailProvider interface {
    // ... 现有方法 ...
    
    // RefreshTokenIfNeeded 如果需要则刷新 token
    // 返回 nil 表示 token 有效或刷新成功
    // 返回 error 表示刷新失败
    RefreshTokenIfNeeded(ctx context.Context) error
    
    // GetTokenExpiry 获取 token 过期时间（用于监控）
    GetTokenExpiry() time.Time
}
```

---

## 📝 第 2 步：实现 GmailAdapter Token 刷新

**文件**: `backend/internal/adapter/gmail.go`

```go
// 添加字段到 GmailAdapter 结构体
type GmailAdapter struct {
    config *Config
    service *gmail.Service
    
    // 添加以下字段
    oauth2Service OAuth2Service  // 用于刷新 token
    accountRepo   AccountRepository
    encryptor     Encryptor
}

// 修改 NewGmailAdapter 构造函数
func NewGmailAdapter(config *Config, oauth2Service OAuth2Service, 
    accountRepo AccountRepository, encryptor Encryptor) (*GmailAdapter, error) {
    if config == nil {
        return nil, fmt.Errorf("config is required")
    }
    
    return &GmailAdapter{
        config:        config,
        oauth2Service: oauth2Service,
        accountRepo:   accountRepo,
        encryptor:     encryptor,
    }, nil
}

// 实现 RefreshTokenIfNeeded 方法
func (a *GmailAdapter) RefreshTokenIfNeeded(ctx context.Context) error {
    // 检查 token 是否即将过期 (5分钟内)
    if time.Now().Add(5 * time.Minute).Before(a.config.Credentials.TokenExpiry) {
        return nil // token 仍然有效
    }
    
    // token 即将过期，执行刷新
    return a.refreshToken(ctx)
}

// 实现 GetTokenExpiry 方法
func (a *GmailAdapter) GetTokenExpiry() time.Time {
    return a.config.Credentials.TokenExpiry
}

// 私有方法：刷新 token
func (a *GmailAdapter) refreshToken(ctx context.Context) error {
    // 获取账户信息（需要从 config 中获取 accountUID）
    accountUID := a.config.Credentials.AccountUID
    
    // 调用 OAuth2Service 刷新 token
    req := &OAuth2TokenRefreshRequest{
        AccountUID: accountUID,
    }
    
    resp, err := a.oauth2Service.RefreshToken(ctx, req)
    if err != nil {
        return fmt.Errorf("failed to refresh token: %w", err)
    }
    
    // 更新配置中的 token
    a.config.Credentials.AccessToken = resp.AccessToken
    a.config.Credentials.TokenExpiry = resp.ExpiresAt
    if resp.RefreshToken != "" {
        a.config.Credentials.RefreshToken = resp.RefreshToken
    }
    
    // 重新连接以使用新 token
    if err := a.Connect(ctx); err != nil {
        return fmt.Errorf("failed to reconnect with new token: %w", err)
    }
    
    return nil
}

// 修改 FetchEmails 方法，在开始时刷新 token
func (a *GmailAdapter) FetchEmails(ctx context.Context, since time.Time, limit int) ([]*Email, error) {
    // 预检查：刷新 token（如果需要）
    if err := a.RefreshTokenIfNeeded(ctx); err != nil {
        return nil, fmt.Errorf("token refresh failed: %w", err)
    }
    
    if a.service == nil {
        return nil, fmt.Errorf("not connected to Gmail API")
    }
    
    // ... 原有逻辑 ...
}

// 修改 FetchEmailDetail 方法
func (a *GmailAdapter) FetchEmailDetail(ctx context.Context, providerID string) (*Email, error) {
    // 预检查：刷新 token（如果需要）
    if err := a.RefreshTokenIfNeeded(ctx); err != nil {
        return nil, fmt.Errorf("token refresh failed: %w", err)
    }
    
    if a.service == nil {
        return nil, fmt.Errorf("not connected to Gmail API")
    }
    
    // ... 原有逻辑 ...
}
```

---

## 📝 第 3 步：在 sync_service 中集成

**文件**: `backend/internal/service/sync_service.go`

```go
// 修改 doSync 方法
func (s *syncService) doSync(ctx context.Context, account *model.Account, syncLog *model.SyncLog) error {
    log.Printf("[DEBUG] Starting sync for account %s", account.UID)
    
    // ... 现有代码：解析凭证、代理等 ...
    
    // 创建适配器
    provider, err := s.adapterFactory.CreateAdapter(config)
    if err != nil {
        return fmt.Errorf("failed to create adapter: %w", err)
    }
    
    // 连接到邮箱服务
    if err := provider.Connect(ctx); err != nil {
        return s.handleSyncError(ctx, account, fmt.Errorf("failed to connect: %w", err))
    }
    defer provider.Disconnect()
    
    // ✨ 新增：预检查 - 刷新 token（如果需要）
    if err := s.ensureValidToken(ctx, provider); err != nil {
        log.Printf("[WARN] Token refresh failed: %v", err)
        // 记录警告但继续，因为适配器内部也会尝试刷新
    }
    
    // 拉取邮件列表
    emails, err := provider.FetchEmails(ctx, since, 1000)
    if err != nil {
        return s.handleSyncError(ctx, account, fmt.Errorf("failed to fetch emails: %w", err))
    }
    
    // ... 原有逻辑 ...
}

// 新增方法：确保 token 有效
func (s *syncService) ensureValidToken(ctx context.Context, provider adapter.MailProvider) error {
    // 检查适配器是否支持 token 刷新
    if tokenRefresher, ok := provider.(interface{ RefreshTokenIfNeeded(context.Context) error }); ok {
        return tokenRefresher.RefreshTokenIfNeeded(ctx)
    }
    
    // 如果不支持，返回 nil（不是错误）
    return nil
}
```

---

## 📝 第 4 步：单元测试

**文件**: `backend/internal/adapter/gmail_test.go`

```go
func TestGmailAdapter_RefreshTokenIfNeeded(t *testing.T) {
    // 创建 mock OAuth2Service
    mockOAuth2Service := &MockOAuth2Service{
        refreshTokenFunc: func(ctx context.Context, req *OAuth2TokenRefreshRequest) (*OAuth2TokenRefreshResponse, error) {
            return &OAuth2TokenRefreshResponse{
                AccessToken: "new-token",
                ExpiresAt:   time.Now().Add(1 * time.Hour),
            }, nil
        },
    }
    
    // 创建一个即将过期的 token
    config := &Config{
        Credentials: &Credentials{
            AccessToken:  "old-token",
            TokenExpiry:  time.Now().Add(2 * time.Minute), // 2分钟后过期
            AccountUID:   "test-account",
        },
    }
    
    adapter := &GmailAdapter{
        config:        config,
        oauth2Service: mockOAuth2Service,
    }
    
    // 调用 RefreshTokenIfNeeded
    err := adapter.RefreshTokenIfNeeded(context.Background())
    
    // 验证
    assert.NoError(t, err)
    assert.Equal(t, "new-token", adapter.config.Credentials.AccessToken)
}

func TestGmailAdapter_TokenStillValid(t *testing.T) {
    // 创建一个有效的 token（1小时后过期）
    config := &Config{
        Credentials: &Credentials{
            AccessToken:  "valid-token",
            TokenExpiry:  time.Now().Add(1 * time.Hour),
        },
    }
    
    adapter := &GmailAdapter{
        config: config,
    }
    
    // 调用 RefreshTokenIfNeeded
    err := adapter.RefreshTokenIfNeeded(context.Background())
    
    // 验证：不应该刷新
    assert.NoError(t, err)
    assert.Equal(t, "valid-token", adapter.config.Credentials.AccessToken)
}
```

---

## 📝 第 5 步：集成测试

**文件**: `backend/internal/service/sync_service_test.go`

```go
func TestSyncService_TokenRefreshBeforeSync(t *testing.T) {
    // 创建 mock 适配器
    mockAdapter := &MockMailProvider{
        refreshTokenIfNeededFunc: func(ctx context.Context) error {
            // 模拟 token 刷新
            return nil
        },
        fetchEmailsFunc: func(ctx context.Context, since time.Time, limit int) ([]*adapter.Email, error) {
            return []*adapter.Email{}, nil
        },
    }
    
    // 创建 sync service
    syncService := &syncService{
        // ... 初始化 ...
    }
    
    // 执行同步
    err := syncService.doSync(context.Background(), &model.Account{
        UID: "test-account",
    }, &model.SyncLog{})
    
    // 验证：RefreshTokenIfNeeded 被调用
    assert.NoError(t, err)
    assert.True(t, mockAdapter.refreshTokenIfNeededCalled)
}
```

---

## 🔍 验证步骤

```bash
# 1. 运行单元测试
cd backend
go test ./internal/adapter -run TestGmailAdapter_RefreshTokenIfNeeded -v

# 2. 运行集成测试
go test ./internal/service -run TestSyncService_TokenRefreshBeforeSync -v

# 3. 运行所有测试
make test

# 4. 检查代码覆盖率
go test ./... -cover
```

---

**下一步**: 按照上述步骤逐个实现，每个步骤完成后运行测试验证


