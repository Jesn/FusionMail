# Add public email send/receive APIs with API Key auth

## Goal

为 FusionMail 增加对外 API，允许第三方通过 API Key 发送邮件、获取邮件详情、删除邮件、清空收件箱、查看已发送邮件。API 格式参考 GPTMail（`https://mail.chatgpt.org.uk/zh/api/`），适配 FusionMail 多账户邮件管理架构。

## Background

- API Keys 功能已完整实现（CRUD + 中间件认证），使用 `Authorization: Bearer <API_KEY>` 格式
- 已有 3 个公共邮件接口：`GET /api/v1/public/mail/receive`、`GET /api/v1/public/mail/search`、`POST /api/v1/public/mail/mark-read`
- 邮件发送功能已实现（`SendService.SendEmail`），但仅 JWT 保护
- 需要复用现有 service 层，不新建数据模型

## Requirements

### 功能需求

1. **发送邮件** — `POST /api/v1/public/mail/send`
   - 请求体包含 `from`（发件邮箱地址）、`to`、`cc`、`bcc`、`subject`、`text_body`、`html_body`、`reply_to`
   - 系统通过 `from` 邮箱地址查找对应账户，调用 `SendService.SendEmail`
   - 返回 `message_id`、`sent_email_id`、`sender_type`

2. **获取单封邮件详情** — `GET /api/v1/public/mail/detail`
   - 参数：`email`（邮箱地址）、`id`（邮件 ID）
   - 必须校验邮件属于该邮箱对应的账户
   - 返回 `EmailDetailResponse`（已有 DTO）

3. **删除单封邮件** — `DELETE /api/v1/public/mail/delete`
   - 参数：`email`（邮箱地址）、`id`（邮件 ID）
   - 必须校验邮件属于该邮箱对应的账户
   - 软删除（复用 `EmailService.DeleteEmail`）

4. **清空收件箱** — `DELETE /api/v1/public/mail/clear`
   - 参数：`email`（邮箱地址）
   - 删除该邮箱账户下的所有未删除邮件（软删除）
   - 返回删除数量

5. **已发送邮件列表** — `GET /api/v1/public/mail/sent`
   - 参数：`email`（邮箱地址）、`page`、`page_size`、可选 `status`、可选 `search`
   - 复用 `SentEmailService.ListSentEmails`

### 安全需求

- 所有新接口使用 `apiKeyMiddleware.RequireAPIKeyOnly()` 认证
- 所有涉及邮件操作的新接口必须校验 `email` 参数对应的账户存在且 `status = active`
- 邮件 ID 操作必须验证邮件 `account_uid` 与请求的 `email` 对应账户匹配
- 公共接口限速由路由层 `publicRatePerMin` 控制（已有）

### 非功能需求

- 响应格式与现有公共接口一致：`{ success, data }`
- 不引入新的数据库表或迁移
- 不修改现有 JWT 保护的接口

## Acceptance Criteria

- [ ] `POST /api/v1/public/mail/send` — 使用 API Key 成功发送邮件，返回 `message_id`
- [ ] `GET /api/v1/public/mail/detail` — 使用 API Key 获取指定邮件详情，跨账户访问返回 404
- [ ] `DELETE /api/v1/public/mail/delete` — 使用 API Key 删除指定邮件，跨账户删除返回 404
- [ ] `DELETE /api/v1/public/mail/clear` — 使用 API Key 清空收件箱，返回删除数量
- [ ] `GET /api/v1/public/mail/sent` — 使用 API Key 获取已发送邮件列表
- [ ] 无 API Key 或无效 API Key 访问所有新接口返回 401
- [ ] `go build ./...` 通过
- [ ] `go test ./...` 通过
- [ ] 现有公共接口不受影响

## Out of Scope

- 生成临时邮箱地址（GPTMail 特有概念，不适用于 FusionMail）
- 公共统计接口
- 前端 API 文档页面（可后续迭代）
- API Key 权限粒度控制（`permissions` 字段后续启用）