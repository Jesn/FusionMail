# Token 刷新机制实现文档

## 概述

本文档描述了 FusionMail 短效邮箱适配器中实现的 token 刷新机制。该机制确保在访问 Microsoft Graph API 时始终使用有效的访问令牌。

## 核心特性

### 1. 线程安全的 Token 管理

使用 `sync.RWMutex` 确保多协程环境下的 token 安全访问：

```go
type GraphQuickAdapter struct {
    // ... 其他字段
    accessToken string
    tokenExpiry time.Time
    tokenMutex  sync.RWMutex
}
```

- **读锁**：用于检查 token 状态和获取 token 值
- **写锁**：用于刷新 token 时的独占访问

### 2. 自动 Token 刷新

#### 触发条件

Token 刷新在以下情况下自动触发：

1. **首次连接**：`Connect()` 方法调用时
2. **Token 不存在**：`accessToken` 为空
3. **Token 即将过期**：距离过期时间少于 5 分钟

#### 刷新流程

```go
func (a *GraphQuickAdapter) refreshAccessToken(ctx context.Context) error {
    // 1. 获取写锁
    a.tokenMutex.Lock()
    defer a.tokenMutex.Unlock()
    
    // 2. 双重检查（避免重复刷新）
    if a.accessToken != "" && time.Now().Add(5*time.Minute).Before(a.tokenExpiry) {
        return nil
    }
    
    // 3. 执行刷新逻辑（包含重试机制）
    // 4. 更新 token 和过期时间
}
```

### 3. 重试机制

#### 重试策略

- **最大重试次数**：3 次
- **退避策略**：指数退避（1s, 2s, 3s）
- **不可重试错误**：配置错误（如 `invalid_client`、`invalid_grant`）

#### 错误分类

```go
// 不可重试的错误码
nonRetryableCodes := []string{
    "invalid_client",      // 客户端 ID 无效
    "invalid_grant",       // refresh_token 无效
    "unsupported_grant_type", // 不支持的授权类型
    "invalid_scope",       // 无效的权限范围
}
```

### 4. 智能请求处理

#### 自动认证请求

`makeAuthenticatedRequest` 方法提供了自动处理 token 刷新的 HTTP 请求：

```go
func (a *GraphQuickAdapter) makeAuthenticatedRequest(ctx context.Context, method, url string, body io.Reader) (*http.Response, error) {
    maxRetries := 2
    
    for attempt := 1; attempt <= maxRetries; attempt++ {
        // 1. 确保有有效的 token
        if err := a.ensureValidToken(ctx); err != nil {
            return nil, err
        }
        
        // 2. 发送请求
        resp, err := a.httpClient.Do(req)
        
        // 3. 处理 401 错误（自动重试）
        if resp.StatusCode == http.StatusUnauthorized && attempt < maxRetries {
            // 强制刷新 token 并重试
            continue
        }
        
        return resp, nil
    }
}
```

## API 接口

### 核心方法

#### 1. Token 状态检查

```go
// IsTokenValid 检查当前 token 是否有效
func (a *GraphQuickAdapter) IsTokenValid() bool

// GetTokenExpiry 获取 token 过期时间
func (a *GraphQuickAdapter) GetTokenExpiry() time.Time

// GetTokenInfo 获取 token 详细信息（用于监控）
func (a *GraphQuickAdapter) GetTokenInfo() map[string]interface{}
```

#### 2. Token 管理

```go
// RefreshTokenIfNeeded 如果需要则刷新 token
func (a *GraphQuickAdapter) RefreshTokenIfNeeded(ctx context.Context) error

// ForceRefreshToken 强制刷新 token
func (a *GraphQuickAdapter) ForceRefreshToken(ctx context.Context) error

// ClearToken 清除当前 token
func (a *GraphQuickAdapter) ClearToken()
```

### 错误处理

#### TokenError 类型

```go
type TokenError struct {
    Code        string // 错误码
    Description string // 错误描述
    StatusCode  int    // HTTP 状态码
}
```

#### TokenRefreshedError 类型

用于指示 token 已刷新，调用方可以重试请求：

```go
type TokenRefreshedError struct {
    OriginalError error
}
```

## 使用示例

### 基本使用

```go
// 创建适配器
config := &adapter.Config{
    Email:    "user@outlook.com",
    Provider: "outlook",
    AuthType: "quick",
    Credentials: &adapter.Credentials{
        ClientID:     "your_client_id",
        RefreshToken: "your_refresh_token",
    },
    Timeout: 30 * time.Second,
}

quickAdapter, err := adapter.NewGraphQuickAdapter(config)
if err != nil {
    log.Fatal(err)
}

// 连接（自动获取 token）
ctx := context.Background()
if err := quickAdapter.Connect(ctx); err != nil {
    log.Fatal(err)
}

// 获取邮件（自动处理 token 刷新）
emails, err := quickAdapter.FetchEmails(ctx, time.Now().Add(-24*time.Hour), 10)
if err != nil {
    log.Fatal(err)
}
```

### 监控 Token 状态

```go
// 检查 token 状态
if !quickAdapter.IsTokenValid() {
    log.Println("Token 无效，需要刷新")
}

// 获取详细信息
tokenInfo := quickAdapter.GetTokenInfo()
fmt.Printf("Token 信息: %+v\n", tokenInfo)

// 输出示例：
// {
//   "has_token": true,
//   "expires_at": "2024-01-15T10:30:00Z",
//   "is_valid": true,
//   "expires_in": 3456,
//   "token_preview": "eyJ0eXAiOi..."
// }
```

### 手动刷新

```go
// 主动刷新 token
if err := quickAdapter.RefreshTokenIfNeeded(ctx); err != nil {
    log.Printf("Token 刷新失败: %v", err)
}

// 强制刷新
if err := quickAdapter.ForceRefreshToken(ctx); err != nil {
    log.Printf("强制刷新失败: %v", err)
}
```

## 性能考虑

### 1. 锁的使用

- **读锁优先**：token 检查使用读锁，允许并发访问
- **写锁最小化**：只在实际刷新时使用写锁
- **双重检查**：避免不必要的刷新操作

### 2. 缓存策略

- **提前刷新**：在 token 过期前 5 分钟自动刷新
- **避免频繁刷新**：通过双重检查机制避免重复刷新

### 3. 网络优化

- **超时控制**：所有 HTTP 请求都有超时限制
- **重试限制**：最多重试 3 次，避免无限重试
- **错误分类**：区分可重试和不可重试错误

## 安全考虑

### 1. 敏感信息保护

- **Token 预览**：日志中只显示 token 的前 10 个字符
- **内存清理**：`ClearToken()` 方法清除内存中的敏感信息

### 2. 错误信息

- **详细错误**：提供足够的错误信息用于调试
- **敏感信息过滤**：避免在错误信息中暴露完整的 token

## 测试

### 单元测试

```bash
# 运行 token 刷新相关测试
go test ./internal/adapter -run TestGraphQuickAdapter_TokenRefresh

# 运行并发安全测试
go test ./internal/adapter -run TestGraphQuickAdapter_ConcurrentTokenRefresh

# 性能测试
go test ./internal/adapter -bench BenchmarkGraphQuickAdapter_TokenValidation
```

### 集成测试

```bash
# 运行完整的 token 刷新测试
go run backend/test_token_refresh.go
```

## 故障排除

### 常见问题

1. **Token 刷新失败**
   - 检查 `refresh_token` 是否有效
   - 验证 `client_id` 是否正确
   - 确认网络连接正常

2. **并发问题**
   - 检查是否正确使用了锁机制
   - 验证没有死锁情况

3. **性能问题**
   - 监控 token 刷新频率
   - 检查是否有不必要的重复刷新

### 调试技巧

```go
// 启用详细日志
tokenInfo := adapter.GetTokenInfo()
log.Printf("Token 状态: %+v", tokenInfo)

// 检查刷新历史
// （可以通过添加刷新计数器来实现）
```

## 未来改进

1. **异步刷新**：在后台异步刷新 token
2. **刷新预测**：基于使用模式预测刷新时机
3. **多实例协调**：在多实例环境下协调 token 刷新
4. **监控指标**：添加 token 刷新的监控指标

---

**注意**：本文档描述的是 token 刷新机制的核心实现。在生产环境中使用时，请确保正确配置认证参数并处理所有可能的错误情况。