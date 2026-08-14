# Design: Public Email APIs

## Architecture

所有新接口添加到现有的 `PublicHandler` 中，路由注册在 `router.go` 的 `public` 路由组下（`apiKeyMiddleware.RequireAPIKeyOnly()` 保护）。

## Data Flow

### 发送邮件 (`POST /api/v1/public/mail/send`)

```
Client (API Key)
  → APIKeyMiddleware.RequireAPIKeyOnly()
  → PublicHandler.SendMail()
    → AccountService.GetByEmail(from)  // 查找发件账户
    → SendService.SendEmail(req)       // 复用现有发送逻辑
    → 返回 { success, data: { message_id, sent_email_id, sender_type } }
```

### 获取邮件详情 (`GET /api/v1/public/mail/detail`)

```
Client (API Key)
  → APIKeyMiddleware.RequireAPIKeyOnly()
  → PublicHandler.GetMailDetail()
    → AccountService.GetByEmail(email)     // 查找账户
    → EmailService.GetEmailByID(id)        // 获取邮件
    → 校验 email.AccountUID == account.UID // 越权检查
    → 返回 { success, data: EmailDetailResponse }
```

### 删除邮件 (`DELETE /api/v1/public/mail/delete`)

```
Client (API Key)
  → APIKeyMiddleware.RequireAPIKeyOnly()
  → PublicHandler.DeleteMail()
    → AccountService.GetByEmail(email)
    → EmailRepo.FindByID(id)               // 获取邮件验证归属
    → EmailService.DeleteEmail(id)         // 软删除
    → 返回 { success, data: { message } }
```

### 清空收件箱 (`DELETE /api/v1/public/mail/clear`)

```
Client (API Key)
  → APIKeyMiddleware.RequireAPIKeyOnly()
  → PublicHandler.ClearMailbox()
    → AccountService.GetByEmail(email)
    → EmailRepo.BatchSoftDeleteByAccountUID(account.UID)  // 需新增 repo 方法
    → 返回 { success, data: { count } }
```

### 已发送邮件列表 (`GET /api/v1/public/mail/sent`)

```
Client (API Key)
  → APIKeyMiddleware.RequireAPIKeyOnly()
  → PublicHandler.ListSentEmails()
    → AccountService.GetByEmail(email)
    → SentEmailService.ListSentEmails(req with account_uid)
    → 返回 { success, data: { emails, total, ... } }
```

## Component Changes

### 1. `PublicHandler` (handler/public_handler.go)

新增依赖：`sendService`、`sentEmailService`、`emailRepo`

构造函数 `NewPublicHandler` 增加参数：
- `sendService *service.SendService`
- `sentEmailService *service.SentEmailService`
- `emailRepo repository.EmailRepository`

新增方法：
- `SendMail(c *gin.Context)` — 发送邮件
- `GetMailDetail(c *gin.Context)` — 获取邮件详情
- `DeleteMail(c *gin.Context)` — 删除邮件
- `ClearMailbox(c *gin.Context)` — 清空收件箱
- `ListSentEmails(c *gin.Context)` — 已发送列表

### 2. `EmailRepository` (repository/email.go)

新增方法：
- `BatchSoftDeleteByAccountUID(ctx, accountUID string) (int64, error)` — 批量软删除指定账户下未删除的邮件

### 3. Router (router/router.go)

在 `public/mail` 路由组下新增：
```go
mail.POST("/send", publicHandler.SendMail)
mail.GET("/detail", publicHandler.GetMailDetail)
mail.DELETE("/delete", publicHandler.DeleteMail)
mail.DELETE("/clear", publicHandler.ClearMailbox)
mail.GET("/sent", publicHandler.ListSentEmails)
```

### 4. main.go

`NewPublicHandler` 调用处增加 `sendService`、`sentEmailService`、`emailRepo` 参数。

## Request/Response Contracts

### SendMail Request

```json
{
  "from": "sender@example.com",
  "to": ["recipient@example.com"],
  "cc": [],
  "bcc": [],
  "subject": "Subject",
  "text_body": "Plain text",
  "html_body": "<p>HTML</p>",
  "reply_to": ""
}
```

### SendMail Response

```json
{
  "success": true,
  "data": {
    "message_id": "<xxx@mail.example.com>",
    "sent_email_id": 123,
    "sender_type": "smtp"
  }
}
```

### GetMailDetail Response

```json
{
  "success": true,
  "data": {
    "id": 123,
    "subject": "...",
    "from_address": "...",
    "text_body": "...",
    "html_body": "...",
    ...
  }
}
```

### ClearMailbox Response

```json
{
  "success": true,
  "data": {
    "count": 42
  }
}
```

## Security Considerations

- 越权防护：所有邮件 ID 操作必须验证 `email.AccountUID == account.UID`
- 账户状态检查：`account.Status` 必须为 `active`
- 发送邮件需要账户有有效的 SMTP/OAuth2 配置（由 `SendService` 内部处理）
- 限速由路由组级别的 `publicRatePerMin` 控制

## Compatibility

- 不修改现有接口签名
- `NewPublicHandler` 构造函数签名变更，需同步更新 `main.go` 中的调用
- 新增 repo 方法不影响现有方法