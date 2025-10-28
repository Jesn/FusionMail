# FusionMail 代码规范

## Go 后端代码规范

### 代码格式化

**使用 gofmt 和 goimports**：
```bash
# 格式化代码
gofmt -w .

# 自动整理导入
goimports -w .
```

**编辑器配置**：
- 使用 Tab 缩进（Go 标准）
- 每行最大长度：120 字符
- 文件末尾保留一个空行

### 命名规范

#### 包命名
```go
// ✅ 好的命名
package adapter
package service
package repository

// ❌ 避免的命名
package adapters      // 避免复数
package emailService  // 避免驼峰
package email_service // 避免下划线
```

#### 变量命名
```go
// ✅ 好的命名
var emailList []Email
var accountRepo AccountRepository
const MaxRetryCount = 3

// ❌ 避免的命名
var email_list []Email    // 避免下划线
var EmailList []Email     // 私有变量不应大写开头
var MAX_RETRY_COUNT = 3   // 常量使用驼峰而非全大写
```

#### 函数命名
```go
// ✅ 好的命名
func GetEmailList() []Email
func CreateAccount() error
func parseEmailBody() string  // 私有函数小写开头

// ❌ 避免的命名
func get_email_list() []Email  // 避免下划线
func getEmailList() []Email    // 公开函数应大写开头
```

#### 接口命名
```go
// ✅ 好的命名
type MailProvider interface {
    Connect() error
    Fetch() ([]Email, error)
}

type EmailRepository interface {
    Create(*Email) error
    FindByID(int64) (*Email, error)
}

// ❌ 避免的命名
type IMailProvider interface {}  // 避免 I 前缀
type MailProviderInterface interface {}  // 避免 Interface 后缀
```

### 注释规范

#### 包注释
```go
// Package adapter 提供邮箱协议适配器的统一接口和实现。
// 支持 Gmail API、Microsoft Graph、IMAP 和 POP3 协议。
package adapter
```

#### 函数注释
```go
// GetEmailList 获取邮件列表，支持分页和过滤。
// 参数：
//   - ctx: 请求上下文
//   - filter: 过滤条件
//   - page: 页码（从 1 开始）
//   - pageSize: 每页数量
// 返回：
//   - []Email: 邮件列表
//   - int: 总数
//   - error: 错误信息
func GetEmailList(ctx context.Context, filter *EmailFilter, page, pageSize int) ([]Email, int, error) {
    // 实现...
}
```

#### 结构体注释
```go
// Email 表示一封邮件的完整信息。
type Email struct {
    ID         int64     `json:"id" gorm:"primaryKey"`
    ProviderID string    `json:"provider_id" gorm:"uniqueIndex:idx_provider_account"`  // 邮箱服务商原生 ID
    AccountUID string    `json:"account_uid" gorm:"uniqueIndex:idx_provider_account"`  // 所属账户 UID
    Subject    string    `json:"subject"`                                               // 邮件主题
    CreatedAt  time.Time `json:"created_at"`                                            // 创建时间
}
```

### 错误处理

#### 错误定义
```go
// 使用 errors 包定义错误
var (
    ErrAccountNotFound     = errors.New("account not found")
    ErrInvalidCredentials  = errors.New("invalid credentials")
    ErrConnectionFailed    = errors.New("connection failed")
)

// 使用 fmt.Errorf 包装错误
func GetAccount(uid string) (*Account, error) {
    account, err := repo.FindByUID(uid)
    if err != nil {
        return nil, fmt.Errorf("failed to get account: %w", err)
    }
    return account, nil
}
```

#### 错误检查
```go
// ✅ 好的错误处理
result, err := doSomething()
if err != nil {
    return fmt.Errorf("failed to do something: %w", err)
}

// ✅ 使用 errors.Is 检查错误
if errors.Is(err, ErrAccountNotFound) {
    return nil, fmt.Errorf("account not found: %w", err)
}

// ❌ 避免忽略错误
result, _ := doSomething()  // 不要忽略错误
```

### Context 使用

```go
// ✅ 好的 Context 使用
func (s *EmailService) GetEmailList(ctx context.Context, filter *EmailFilter) ([]Email, error) {
    // 检查 Context 是否已取消
    select {
    case <-ctx.Done():
        return nil, ctx.Err()
    default:
    }
    
    // 传递 Context 到下层
    emails, err := s.repo.FindAll(ctx, filter)
    if err != nil {
        return nil, err
    }
    
    return emails, nil
}

// ❌ 避免创建新的 Context
func (s *EmailService) GetEmailList() ([]Email, error) {
    ctx := context.Background()  // 应该由调用者传入
    // ...
}
```

### 并发编程

#### Goroutine 使用
```go
// ✅ 好的 Goroutine 使用
func (e *SyncEngine) SyncAccounts(ctx context.Context, accounts []*Account) error {
    var wg sync.WaitGroup
    errChan := make(chan error, len(accounts))
    
    for _, account := range accounts {
        wg.Add(1)
        go func(acc *Account) {
            defer wg.Done()
            if err := e.syncAccount(ctx, acc); err != nil {
                errChan <- err
            }
        }(account)  // 传递参数避免闭包问题
    }
    
    wg.Wait()
    close(errChan)
    
    // 收集错误
    for err := range errChan {
        if err != nil {
            return err
        }
    }
    
    return nil
}
```

#### Channel 使用
```go
// ✅ 好的 Channel 使用
func worker(ctx context.Context, tasks <-chan Task, results chan<- Result) {
    for {
        select {
        case <-ctx.Done():
            return
        case task, ok := <-tasks:
            if !ok {
                return  // Channel 已关闭
            }
            result := processTask(task)
            results <- result
        }
    }
}
```

### 依赖注入

```go
// ✅ 好的依赖注入
type EmailService struct {
    repo      EmailRepository
    eventBus  *event.Bus
    logger    *logger.Logger
}

func NewEmailService(repo EmailRepository, eventBus *event.Bus, logger *logger.Logger) *EmailService {
    return &EmailService{
        repo:     repo,
        eventBus: eventBus,
        logger:   logger,
    }
}

// ❌ 避免全局变量
var globalRepo EmailRepository  // 避免全局变量

func GetEmails() []Email {
    return globalRepo.FindAll()  // 难以测试
}
```

### 测试规范

#### 单元测试
```go
// email_service_test.go
func TestEmailService_GetEmailList(t *testing.T) {
    // Arrange
    mockRepo := &MockEmailRepository{}
    service := NewEmailService(mockRepo, nil, nil)
    
    expectedEmails := []*Email{
        {ID: 1, Subject: "Test 1"},
        {ID: 2, Subject: "Test 2"},
    }
    mockRepo.On("FindAll").Return(expectedEmails, nil)
    
    // Act
    emails, err := service.GetEmailList(context.Background(), nil)
    
    // Assert
    assert.NoError(t, err)
    assert.Equal(t, 2, len(emails))
    assert.Equal(t, "Test 1", emails[0].Subject)
}
```

#### 表驱动测试
```go
func TestValidateEmail(t *testing.T) {
    tests := []struct {
        name    string
        email   string
        wantErr bool
    }{
        {"valid email", "user@example.com", false},
        {"invalid email", "invalid-email", true},
        {"empty email", "", true},
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            err := ValidateEmail(tt.email)
            if (err != nil) != tt.wantErr {
                t.Errorf("ValidateEmail() error = %v, wantErr %v", err, tt.wantErr)
            }
        })
    }
}
```

## TypeScript/React 前端代码规范

### 代码格式化

**使用 Prettier**：
```json
{
  "semi": true,
  "singleQuote": true,
  "tabWidth": 2,
  "trailingComma": "es5",
  "printWidth": 100
}
```

### 命名规范

#### 组件命名
```typescript
// ✅ 好的命名
export const EmailList: React.FC = () => { /* ... */ };
export const AccountCard: React.FC<AccountCardProps> = ({ account }) => { /* ... */ };

// ❌ 避免的命名
export const emailList = () => { /* ... */ };  // 组件应大写开头
export const Email_List = () => { /* ... */ };  // 避免下划线
```

#### 变量和函数命名
```typescript
// ✅ 好的命名
const emailList = [];
const isLoading = false;
const hasError = false;

function fetchEmails() { /* ... */ }
function handleClick() { /* ... */ }

// ❌ 避免的命名
const EmailList = [];  // 变量应小写开头
const loading = false;  // 布尔值应有 is/has 前缀
const email_list = [];  // 避免下划线
```

#### 类型命名
```typescript
// ✅ 好的命名
interface Email {
  id: number;
  subject: string;
}

type EmailFilter = {
  accountUid?: string;
  isRead?: boolean;
};

// ❌ 避免的命名
interface IEmail { /* ... */ }  // 避免 I 前缀
type TEmailFilter = { /* ... */ };  // 避免 T 前缀
```

### TypeScript 类型规范

#### 类型定义
```typescript
// ✅ 好的类型定义
interface Email {
  id: number;
  subject: string;
  from: EmailAddress;
  sentAt: string;
  isRead: boolean;
}

interface EmailListProps {
  emails: Email[];
  onEmailClick: (email: Email) => void;
  isLoading?: boolean;
}

// ❌ 避免使用 any
function processEmail(email: any) { /* ... */ }  // 应该定义具体类型
```

#### 类型导入
```typescript
// ✅ 好的类型导入
import type { Email, Account } from '@/types';

// 或者
import { type Email, type Account } from '@/types';
```

### React 组件规范

#### 函数组件
```typescript
// ✅ 好的组件定义
interface EmailListProps {
  emails: Email[];
  onEmailClick: (email: Email) => void;
}

export const EmailList: React.FC<EmailListProps> = ({ emails, onEmailClick }) => {
  const [selectedId, setSelectedId] = useState<number | null>(null);
  
  const handleClick = (email: Email) => {
    setSelectedId(email.id);
    onEmailClick(email);
  };
  
  return (
    <div className="email-list">
      {emails.map((email) => (
        <EmailItem key={email.id} email={email} onClick={handleClick} />
      ))}
    </div>
  );
};
```

#### Hooks 使用
```typescript
// ✅ 好的 Hooks 使用
export const useEmails = (accountUid?: string) => {
  const [emails, setEmails] = useState<Email[]>([]);
  const [isLoading, setIsLoading] = useState(false);
  const [error, setError] = useState<Error | null>(null);
  
  useEffect(() => {
    const fetchEmails = async () => {
      setIsLoading(true);
      try {
        const data = await emailService.getList({ accountUid });
        setEmails(data);
      } catch (err) {
        setError(err as Error);
      } finally {
        setIsLoading(false);
      }
    };
    
    fetchEmails();
  }, [accountUid]);
  
  return { emails, isLoading, error };
};
```

### 状态管理（Zustand）

```typescript
// ✅ 好的 Store 定义
interface EmailStore {
  emails: Email[];
  selectedEmail: Email | null;
  isLoading: boolean;
  
  // Actions
  setEmails: (emails: Email[]) => void;
  selectEmail: (email: Email) => void;
  fetchEmails: (filter?: EmailFilter) => Promise<void>;
}

export const useEmailStore = create<EmailStore>((set) => ({
  emails: [],
  selectedEmail: null,
  isLoading: false,
  
  setEmails: (emails) => set({ emails }),
  selectEmail: (email) => set({ selectedEmail: email }),
  
  fetchEmails: async (filter) => {
    set({ isLoading: true });
    try {
      const emails = await emailService.getList(filter);
      set({ emails, isLoading: false });
    } catch (error) {
      set({ isLoading: false });
      throw error;
    }
  },
}));
```

### 注释规范

```typescript
/**
 * 邮件列表组件
 * 
 * @param emails - 邮件列表数据
 * @param onEmailClick - 点击邮件时的回调函数
 * @param isLoading - 是否正在加载
 * 
 * @example
 * ```tsx
 * <EmailList
 *   emails={emails}
 *   onEmailClick={(email) => console.log(email)}
 *   isLoading={false}
 * />
 * ```
 */
export const EmailList: React.FC<EmailListProps> = ({ emails, onEmailClick, isLoading }) => {
  // 实现...
};
```

### 错误处理

```typescript
// ✅ 好的错误处理
try {
  const emails = await emailService.getList();
  setEmails(emails);
} catch (error) {
  if (error instanceof ApiError) {
    toast.error(error.message);
  } else {
    toast.error('获取邮件列表失败');
  }
  console.error('Failed to fetch emails:', error);
}

// ✅ 使用 ErrorBoundary
<ErrorBoundary fallback={<ErrorFallback />}>
  <EmailList emails={emails} />
</ErrorBoundary>
```

## 通用代码规范

### 代码复杂度
- 单个函数不超过 50 行
- 单个文件不超过 500 行
- 圈复杂度不超过 10

### 代码重复
- 遵循 DRY 原则（Don't Repeat Yourself）
- 重复代码超过 3 次应提取为函数
- 相似逻辑应抽象为通用函数

### 魔法数字
```go
// ❌ 避免魔法数字
if retryCount > 3 {
    return err
}

// ✅ 使用常量
const MaxRetryCount = 3

if retryCount > MaxRetryCount {
    return err
}
```

### 函数参数
- 函数参数不超过 5 个
- 超过 3 个参数考虑使用结构体/对象

```go
// ❌ 参数过多
func CreateAccount(email, provider, authType, accessToken, refreshToken string, syncEnabled bool, syncInterval int) error

// ✅ 使用结构体
type CreateAccountRequest struct {
    Email        string
    Provider     string
    AuthType     string
    AccessToken  string
    RefreshToken string
    SyncEnabled  bool
    SyncInterval int
}

func CreateAccount(req *CreateAccountRequest) error
```

## Git 提交规范

### 提交消息格式
```
<type>[scope]: <description>

[optional body]

[optional footer]
```

### Type 类型
- `feat`: 新功能
- `fix`: Bug 修复
- `refactor`: 重构
- `docs`: 文档更新
- `test`: 测试相关
- `chore`: 构建/工具相关
- `style`: 代码格式（不影响功能）
- `perf`: 性能优化

### 示例
```
feat(email): 添加邮件搜索功能

实现全文搜索和高级筛选功能，支持按发件人、日期范围等条件搜索。

Closes #123
```

---

**注意**：代码规范是团队协作的基础，请严格遵守。在代码审查时，规范性是重要的审查标准之一。
