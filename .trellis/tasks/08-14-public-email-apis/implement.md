# Implementation Plan: Public Email APIs

## Execution Checklist

### Step 1: Add `BatchSoftDeleteByAccountUID` to EmailRepository

- File: `backend/internal/repository/email.go`
- Add method `BatchSoftDeleteByAccountUID(ctx, accountUID string) (int64, error)`
- Implementation: `db.Model(&Email{}).Where("account_uid = ? AND is_deleted = false", accountUID).Update("is_deleted", true)`
- Return count of affected rows

### Step 2: Update `PublicHandler` struct and constructor

- File: `backend/internal/handler/public_handler.go`
- Add fields: `sendService *service.SendService`, `sentEmailService *service.SentEmailService`, `emailRepo repository.EmailRepository`
- Update `NewPublicHandler` signature to accept new dependencies

### Step 3: Implement `SendMail` handler

- File: `backend/internal/handler/public_handler.go`
- Bind JSON body (`from`, `to`, `cc`, `bcc`, `subject`, `text_body`, `html_body`, `reply_to`)
- `AccountService.GetByEmail(from)` → validate account exists and active
- Build `service.SendEmailRequest` with `AccountUID` from account
- Call `sendService.SendEmail()`
- Return `{ success, data: { message_id, sent_email_id, sender_type } }`

### Step 4: Implement `GetMailDetail` handler

- File: `backend/internal/handler/public_handler.go`
- Query params: `email`, `id`
- `AccountService.GetByEmail(email)` → validate
- `emailService.GetEmailByID(id)` → get email
- Verify `email.AccountUID == account.UID` (404 if mismatch)
- Return `{ success, data: email_detail }`

### Step 5: Implement `DeleteMail` handler

- File: `backend/internal/handler/public_handler.go`
- Query params: `email`, `id` (support both query and JSON body)
- `AccountService.GetByEmail(email)` → validate
- `emailRepo.FindByID(id)` → verify ownership
- `emailService.DeleteEmail(id)` → soft delete
- Return `{ success, data: { message: "Email deleted" } }`

### Step 6: Implement `ClearMailbox` handler

- File: `backend/internal/handler/public_handler.go`
- Query param: `email`
- `AccountService.GetByEmail(email)` → validate
- `emailRepo.BatchSoftDeleteByAccountUID(account.UID)` → batch soft delete
- Return `{ success, data: { count: N } }`

### Step 7: Implement `ListSentEmails` handler

- File: `backend/internal/handler/public_handler.go`
- Query params: `email`, `page`, `page_size`, `status`, `search`
- `AccountService.GetByEmail(email)` → validate
- Build `ListSentEmailsRequest` with `AccountUID`
- Call `sentEmailService.ListSentEmails()`
- Return `{ success, data: result }`

### Step 8: Register routes

- File: `backend/internal/router/router.go`
- In `public/mail` group, add:
  - `mail.POST("/send", publicHandler.SendMail)`
  - `mail.GET("/detail", publicHandler.GetMailDetail)`
  - `mail.DELETE("/delete", publicHandler.DeleteMail)`
  - `mail.DELETE("/clear", publicHandler.ClearMailbox)`
  - `mail.GET("/sent", publicHandler.ListSentEmails)`

### Step 9: Update `main.go` constructor call

- File: `backend/cmd/main.go` (or wherever `NewPublicHandler` is called)
- Pass `sendService`, `sentEmailService`, `emailRepo` to `NewPublicHandler`

### Step 10: Validate

- `go build ./...`
- `go test ./...`
- Manual smoke test with API Key

## Validation Commands

```bash
cd backend && go build ./...
cd backend && go test ./...
```

## Rollback Points

- After Step 1: repo method only, no callers — safe
- After Step 2: constructor change breaks compilation until Step 9 — must complete Steps 2→9 as a unit
- After Step 8: routes registered but handlers not implemented — must complete Steps 3→7 before building