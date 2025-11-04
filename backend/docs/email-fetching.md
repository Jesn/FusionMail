# 邮件获取功能文档

## 概述

GraphQuickAdapter 提供了完整的邮件获取功能，支持多种获取方式和过滤条件。该功能基于 Microsoft Graph API，提供高性能的邮件数据访问能力。

## 核心功能

### 1. 基本邮件获取

#### FetchEmails - 获取邮件列表

```go
func (a *GraphQuickAdapter) FetchEmails(ctx context.Context, since time.Time, limit int) ([]*Email, error)
```

**参数：**
- `ctx`：上下文，用于超时控制
- `since`：起始时间，获取此时间之后的邮件
- `limit`：限制数量，最大 100

**特性：**
- 按接收时间倒序排列
- 自动字段选择优化性能
- 详细的日志记录
- 错误处理和重试机制

**使用示例：**
```go
// 获取最近 24 小时的邮件，最多 10 封
since := time.Now().Add(-24 * time.Hour)
emails, err := adapter.FetchEmails(ctx, since, 10)
if err != nil {
    log.Printf("获取邮件失败: %v", err)
    return
}

for _, email := range emails {
    fmt.Printf("主题: %s, 发件人: %s\n", email.Subject, email.FromAddress)
}
```

#### FetchEmailDetail - 获取邮件详情

```go
func (a *GraphQuickAdapter) FetchEmailDetail(ctx context.Context, providerID string) (*Email, error)
```

**参数：**
- `ctx`：上下文
- `providerID`：邮件的提供商 ID

**特性：**
- 获取完整的邮件内容
- 自动获取附件信息
- 支持 HTML 和纯文本正文
- 完整的收件人信息

**使用示例：**
```go
email, err := adapter.FetchEmailDetail(ctx, "message_id_123")
if err != nil {
    log.Printf("获取邮件详情失败: %v", err)
    return
}

fmt.Printf("主题: %s\n", email.Subject)
fmt.Printf("发件人: %s (%s)\n", email.FromName, email.FromAddress)
fmt.Printf("HTML正文: %s\n", email.HTMLBody)

if email.HasAttachments {
    fmt.Printf("附件数量: %d\n", len(email.Attachments))
    for _, att := range email.Attachments {
        fmt.Printf("  - %s (%d bytes)\n", att.Filename, att.SizeBytes)
    }
}
```

### 2. 高级邮件获取

#### FetchEmailsWithFilter - 过滤邮件获取

```go
func (a *GraphQuickAdapter) FetchEmailsWithFilter(ctx context.Context, filter *EmailFilter, limit int) ([]*Email, error)
```

**EmailFilter 结构：**
```go
type EmailFilter struct {
    Since      time.Time // 开始时间
    Until      time.Time // 结束时间
    IsRead     *bool     // 已读状态
    HasAttach  *bool     // 是否有附件
    FromEmail  string    // 发件人邮箱
    Subject    string    // 主题关键词
    Folder     string    // 文件夹名称
    Categories []string  // 分类标签
}
```

**使用示例：**
```go
// 获取未读邮件
isRead := false
filter := &EmailFilter{
    Since:  time.Now().Add(-7 * 24 * time.Hour), // 最近7天
    IsRead: &isRead,
}

emails, err := adapter.FetchEmailsWithFilter(ctx, filter, 20)
if err != nil {
    log.Printf("过滤邮件获取失败: %v", err)
    return
}

// 获取有附件的邮件
hasAttach := true
filter = &EmailFilter{
    HasAttach: &hasAttach,
    Since:     time.Now().Add(-30 * 24 * time.Hour),
}

emails, err = adapter.FetchEmailsWithFilter(ctx, filter, 10)

// 按发件人过滤
filter = &EmailFilter{
    FromEmail: "important@company.com",
    Since:     time.Now().Add(-90 * 24 * time.Hour),
}

emails, err = adapter.FetchEmailsWithFilter(ctx, filter, 50)

// 按主题关键词过滤
filter = &EmailFilter{
    Subject: "报告",
    Since:   time.Now().Add(-7 * 24 * time.Hour),
}

emails, err = adapter.FetchEmailsWithFilter(ctx, filter, 15)
```

#### FetchEmailsWithPagination - 分页邮件获取

```go
func (a *GraphQuickAdapter) FetchEmailsWithPagination(ctx context.Context, pageSize int, nextPageToken string) (*EmailPage, error)
```

**EmailPage 结构：**
```go
type EmailPage struct {
    Emails        []*Email `json:"emails"`
    NextPageToken string   `json:"next_page_token"`
    HasNextPage   bool     `json:"has_next_page"`
    PageSize      int      `json:"page_size"`
}
```

**使用示例：**
```go
// 获取第一页
page, err := adapter.FetchEmailsWithPagination(ctx, 20, "")
if err != nil {
    log.Printf("分页获取失败: %v", err)
    return
}

fmt.Printf("当前页邮件数: %d\n", page.PageSize)
fmt.Printf("是否有下一页: %t\n", page.HasNextPage)

// 获取下一页
if page.HasNextPage {
    nextPage, err := adapter.FetchEmailsWithPagination(ctx, 20, page.NextPageToken)
    if err != nil {
        log.Printf("获取下一页失败: %v", err)
        return
    }
    
    fmt.Printf("下一页邮件数: %d\n", nextPage.PageSize)
}

// 遍历所有页面
var allEmails []*Email
nextToken := ""

for {
    page, err := adapter.FetchEmailsWithPagination(ctx, 50, nextToken)
    if err != nil {
        break
    }
    
    allEmails = append(allEmails, page.Emails...)
    
    if !page.HasNextPage {
        break
    }
    
    nextToken = page.NextPageToken
}

fmt.Printf("总共获取邮件: %d\n", len(allEmails))
```

#### FetchEmailsByFolder - 按文件夹获取邮件

```go
func (a *GraphQuickAdapter) FetchEmailsByFolder(ctx context.Context, folderName string, limit int) ([]*Email, error)
```

**支持的文件夹：**
- `inbox` - 收件箱
- `sent` - 已发送
- `drafts` - 草稿箱
- `deleted` - 已删除
- `junk` - 垃圾邮件
- `outbox` - 发件箱
- `archive` - 存档
- 自定义文件夹名称

**使用示例：**
```go
// 获取收件箱邮件
inboxEmails, err := adapter.FetchEmailsByFolder(ctx, "inbox", 20)
if err != nil {
    log.Printf("获取收件箱邮件失败: %v", err)
    return
}

// 获取已发送邮件
sentEmails, err := adapter.FetchEmailsByFolder(ctx, "sent", 10)

// 获取草稿邮件
draftEmails, err := adapter.FetchEmailsByFolder(ctx, "drafts", 5)

// 获取自定义文件夹邮件
customEmails, err := adapter.FetchEmailsByFolder(ctx, "重要邮件", 15)
```

### 3. 辅助功能

#### GetEmailCount - 获取邮件总数

```go
func (a *GraphQuickAdapter) GetEmailCount(ctx context.Context) (int, error)
```

**使用示例：**
```go
count, err := adapter.GetEmailCount(ctx)
if err != nil {
    log.Printf("获取邮件总数失败: %v", err)
    return
}

fmt.Printf("邮箱中共有 %d 封邮件\n", count)
```

## 邮件数据结构

### Email 结构体

```go
type Email struct {
    // 基本信息
    ProviderID   string   // 邮箱服务商原生 ID
    MessageID    string   // 邮件 Message-ID
    Subject      string   // 主题
    FromAddress  string   // 发件人地址
    FromName     string   // 发件人名称
    ToAddresses  []string // 收件人地址列表
    CcAddresses  []string // 抄送地址列表
    BccAddresses []string // 密送地址列表
    ReplyTo      string   // 回复地址

    // 邮件内容
    TextBody string // 纯文本正文
    HTMLBody string // HTML 正文
    Snippet  string // 摘要

    // 源邮箱状态
    SourceIsRead *bool    // 源邮箱已读状态
    SourceLabels []string // 源邮箱标签
    SourceFolder string   // 源邮箱文件夹

    // 附件信息
    HasAttachments   bool         // 是否有附件
    AttachmentsCount int          // 附件数量
    Attachments      []Attachment // 附件列表

    // 时间信息
    SentAt     time.Time // 发送时间
    ReceivedAt time.Time // 接收时间

    // 元数据
    SizeBytes  int64  // 邮件大小（字节）
    ThreadID   string // 会话 ID
    InReplyTo  string // 回复的邮件 ID
    References string // 引用的邮件 ID 列表
}
```

### Attachment 结构体

```go
type Attachment struct {
    Filename    string // 文件名
    ContentType string // 内容类型
    SizeBytes   int64  // 大小（字节）
    Content     []byte // 内容（可选，用于下载）
    IsInline    bool   // 是否内联
    ContentID   string // 内容 ID
}
```

## 性能优化

### 1. 字段选择

所有邮件获取方法都使用 `$select` 参数来只获取需要的字段，减少网络传输和解析时间：

```go
params.Set("$select", "id,subject,bodyPreview,from,toRecipients,ccRecipients,bccRecipients,replyTo,sentDateTime,receivedDateTime,hasAttachments,internetMessageId,conversationId,isRead,categories,inferenceClassification")
```

### 2. 分页处理

- 默认每次最多获取 100 封邮件
- 支持自定义页面大小
- 使用 Graph API 的 `@odata.nextLink` 进行分页

### 3. 缓存策略

- Token 自动刷新和缓存
- 连接复用
- 请求去重

### 4. 并发控制

- 所有方法都是并发安全的
- 支持多协程同时调用
- 自动处理速率限制

## 错误处理

### 常见错误类型

1. **认证错误**
   - Token 过期或无效
   - 权限不足

2. **网络错误**
   - 连接超时
   - 网络不可达

3. **API 错误**
   - 邮件不存在 (404)
   - 请求过于频繁 (429)
   - 服务不可用 (503)

4. **数据错误**
   - 响应格式错误
   - 字段缺失

### 错误处理示例

```go
emails, err := adapter.FetchEmails(ctx, since, limit)
if err != nil {
    // 检查具体错误类型
    if strings.Contains(err.Error(), "401") {
        log.Println("认证失败，需要重新获取 token")
    } else if strings.Contains(err.Error(), "429") {
        log.Println("请求过于频繁，需要等待")
        time.Sleep(time.Minute)
        // 重试
    } else if strings.Contains(err.Error(), "404") {
        log.Println("邮件不存在")
    } else {
        log.Printf("其他错误: %v", err)
    }
    return
}
```

## 使用最佳实践

### 1. 合理设置获取数量

```go
// ✅ 好的做法
emails, err := adapter.FetchEmails(ctx, since, 50) // 适中的数量

// ❌ 避免的做法
emails, err := adapter.FetchEmails(ctx, since, 1000) // 数量过大
```

### 2. 使用过滤条件

```go
// ✅ 好的做法 - 使用过滤条件减少数据传输
filter := &EmailFilter{
    Since:  time.Now().Add(-7 * 24 * time.Hour),
    IsRead: &isRead,
}
emails, err := adapter.FetchEmailsWithFilter(ctx, filter, 20)

// ❌ 避免的做法 - 获取所有邮件后再过滤
allEmails, err := adapter.FetchEmails(ctx, time.Time{}, 1000)
// 然后在客户端过滤...
```

### 3. 分页处理大量数据

```go
// ✅ 好的做法 - 使用分页
var allEmails []*Email
nextToken := ""

for {
    page, err := adapter.FetchEmailsWithPagination(ctx, 50, nextToken)
    if err != nil {
        break
    }
    
    // 处理当前页的邮件
    processEmails(page.Emails)
    
    if !page.HasNextPage {
        break
    }
    
    nextToken = page.NextPageToken
}

// ❌ 避免的做法 - 一次获取所有邮件
allEmails, err := adapter.FetchEmails(ctx, time.Time{}, 10000)
```

### 4. 错误重试

```go
// ✅ 好的做法 - 实现重试机制
func fetchEmailsWithRetry(adapter *GraphQuickAdapter, ctx context.Context, since time.Time, limit int) ([]*Email, error) {
    maxRetries := 3
    for i := 0; i < maxRetries; i++ {
        emails, err := adapter.FetchEmails(ctx, since, limit)
        if err == nil {
            return emails, nil
        }
        
        if strings.Contains(err.Error(), "429") {
            // 速率限制，等待后重试
            time.Sleep(time.Duration(i+1) * time.Second)
            continue
        }
        
        // 其他错误直接返回
        return nil, err
    }
    
    return nil, fmt.Errorf("max retries exceeded")
}
```

### 5. 上下文超时

```go
// ✅ 好的做法 - 设置合理的超时
ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
defer cancel()

emails, err := adapter.FetchEmails(ctx, since, limit)

// ❌ 避免的做法 - 不设置超时
emails, err := adapter.FetchEmails(context.Background(), since, limit)
```

## 监控和调试

### 日志记录

所有邮件获取方法都包含详细的日志记录：

```go
a.logger.Info("开始获取邮件列表", 
    "email", a.config.Email, 
    "since", since.Format(time.RFC3339), 
    "limit", limit)

a.logger.Info("成功获取邮件列表", 
    "email", a.config.Email, 
    "count", len(emails),
    "has_next", messageList.NextLink != "")
```

### 性能监控

```go
// 监控响应时间
start := time.Now()
emails, err := adapter.FetchEmails(ctx, since, limit)
duration := time.Since(start)

log.Printf("邮件获取耗时: %v", duration)
```

### 调试技巧

1. **启用详细日志**
2. **检查 API 响应**
3. **监控网络请求**
4. **验证过滤条件**

## 测试

### 单元测试

```bash
# 运行邮件获取相关测试
go test ./internal/adapter -run TestGraphQuickAdapter_FetchEmails

# 运行所有邮件相关测试
go test ./internal/adapter -run Email

# 运行性能基准测试
go test ./internal/adapter -bench BenchmarkGraphQuickAdapter_FetchEmails
```

### 集成测试

```bash
# 设置测试环境变量
export TEST_EMAIL=your@outlook.com
export TEST_CLIENT_ID=your_client_id
export TEST_REFRESH_TOKEN=your_refresh_token

# 运行集成测试
go run backend/scripts/test_email_fetch.go
```

---

**注意**：邮件获取功能是短效邮箱适配器的核心功能，确保在生产环境中正确处理各种边界情况和错误场景。