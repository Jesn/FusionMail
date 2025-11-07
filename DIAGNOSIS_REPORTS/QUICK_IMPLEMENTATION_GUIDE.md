# FusionMail 快速实现指南

**目的**: 快速查看每个改进的具体实现步骤  
**使用**: 开发人员在实现时参考

---

## 🚀 快速导航

### P0 任务 (立即开始)

| 任务 | 文件 | 工作量 | 难度 |
|------|------|--------|------|
| OAuth2 Token 刷新 | `oauth2_service.go` | 2-3 天 | 中 |
| 错误处理统一 | `dto/error.go` | 1-2 天 | 低 |
| 前端功能完成 | `pages/` | 3-5 天 | 中 |
| 账户失败计数 | `sync_service.go` | 2-3 天 | 中 |

### P1 任务 (本周处理)

| 任务 | 文件 | 工作量 | 难度 |
|------|------|--------|------|
| 缓存策略 | `pkg/cache/` | 3-5 天 | 中 |
| 数据库优化 | `migrations/` | 2-3 天 | 低 |
| 性能监控 | `middleware/metrics.go` | 2-3 天 | 中 |
| 测试覆盖 | `*_test.go` | 5-7 天 | 高 |

---

## 📝 P0 任务详细步骤

### 任务 1: OAuth2 Token 自动刷新

**步骤 1**: 修改 `oauth2_service.go`

```go
// 添加新方法
func (s *OAuth2Service) EnsureValidToken(ctx context.Context, account *model.Account) error {
    // 检查 token 是否即将过期 (5分钟内)
    if time.Now().Add(5 * time.Minute).Before(account.TokenExpiresAt) {
        return nil
    }
    
    // 执行刷新...
}
```

**步骤 2**: 在 `sync_service.go` 中调用

```go
// 在 SyncEmails 方法开始处添加
if err := s.oauth2Service.EnsureValidToken(ctx, account); err != nil {
    return err
}
```

**步骤 3**: 编写测试

```bash
cd backend
go test ./internal/service -run TestEnsureValidToken -v
```

**验证**: 
- [ ] Token 在即将过期时自动刷新
- [ ] 刷新失败时标记账户错误
- [ ] 测试覆盖率 > 80%

---

### 任务 2: 统一错误处理

**步骤 1**: 创建 `backend/internal/dto/error.go`

```go
package dto

type ErrorCode int

const (
    ErrInvalidCredentials ErrorCode = 1001
    ErrAccountNotFound    ErrorCode = 1002
    ErrTokenExpired       ErrorCode = 1003
    ErrSyncFailed         ErrorCode = 2001
)

type APIError struct {
    Code    ErrorCode              `json:"code"`
    Message string                 `json:"message"`
    Details map[string]interface{} `json:"details,omitempty"`
}
```

**步骤 2**: 创建错误中间件 `backend/internal/middleware/error_handler.go`

```go
func ErrorHandlerMiddleware() gin.HandlerFunc {
    return func(c *gin.Context) {
        c.Next()
        if len(c.Errors) > 0 {
            err := c.Errors.Last()
            if apiErr, ok := err.Err.(*dto.APIError); ok {
                c.JSON(http.StatusBadRequest, apiErr)
            }
        }
    }
}
```

**步骤 3**: 在 `router.go` 中注册中间件

```go
router.Use(middleware.ErrorHandlerMiddleware())
```

**验证**:
- [ ] 所有错误返回统一格式
- [ ] 错误码正确
- [ ] 错误消息清晰

---

### 任务 3: 前端功能完成

**步骤 1**: 创建 `frontend/src/pages/EmailDetailPage.tsx`

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

**步骤 2**: 在 `frontend/src/App.tsx` 中添加路由

```typescript
<Route path="/emails/:id" element={<EmailDetailPage />} />
```

**验证**:
- [ ] 页面加载正常
- [ ] 显示邮件详情
- [ ] 错误处理正确

---

## 📊 P1 任务详细步骤

### 任务 4: 缓存策略

**步骤 1**: 创建 `backend/pkg/cache/cache.go`

```go
type CacheService struct {
    redis *redis.Client
}

func (c *CacheService) GetAccountList(ctx context.Context, userID string) ([]*model.Account, error) {
    key := fmt.Sprintf("accounts:%s", userID)
    
    // 尝试从缓存获取
    val, err := c.redis.Get(ctx, key).Result()
    if err == nil {
        var accounts []*model.Account
        json.Unmarshal([]byte(val), &accounts)
        return accounts, nil
    }
    
    // 从数据库获取
    accounts, err := c.accountRepo.GetByUserID(ctx, userID)
    if err != nil {
        return nil, err
    }
    
    // 存入缓存 (TTL: 5分钟)
    data, _ := json.Marshal(accounts)
    c.redis.Set(ctx, key, data, 5*time.Minute)
    
    return accounts, nil
}
```

**步骤 2**: 在 `account_service.go` 中使用

```go
// 替换原来的查询
accounts, err := s.cacheService.GetAccountList(ctx, userID)
```

**验证**:
- [ ] 缓存命中率 > 80%
- [ ] 响应时间降低 10 倍
- [ ] 缓存失效正确

---

### 任务 5: 数据库优化

**步骤 1**: 创建迁移文件 `backend/migrations/006_add_indexes.sql`

```sql
CREATE INDEX idx_emails_account_id ON emails(account_id);
CREATE INDEX idx_emails_created_at ON emails(created_at DESC);
CREATE INDEX idx_rules_user_id ON rules(user_id);
CREATE INDEX idx_attachments_email_id ON attachments(email_id);
CREATE INDEX idx_emails_account_created ON emails(account_id, created_at DESC);
```

**步骤 2**: 运行迁移

```bash
cd backend
make migrate
```

**步骤 3**: 优化查询 `repository/email_repository.go`

```go
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

**验证**:
- [ ] 查询时间降低 50%
- [ ] 没有 N+1 查询
- [ ] 索引被正确使用

---

## 🧪 测试命令

```bash
# 运行所有测试
cd backend
make test

# 运行特定测试
go test ./internal/service -run TestOAuth2 -v

# 生成覆盖率报告
go test ./... -cover

# 性能测试
go test -bench=. -benchmem ./...
```

---

## ✅ 完成检查清单

### P0 完成标准
- [ ] OAuth2 自动刷新工作正常
- [ ] 错误处理统一
- [ ] 前端所有页面完成
- [ ] 账户失败计数和禁用工作
- [ ] 所有 P0 测试通过
- [ ] 代码审查通过

### P1 完成标准
- [ ] 缓存命中率 > 80%
- [ ] 数据库查询时间 < 100ms
- [ ] 性能监控正常工作
- [ ] 测试覆盖率 > 80%
- [ ] 所有 P1 测试通过

---

**建议**: 按照上述步骤逐个实现，每个任务完成后进行测试和性能验证


