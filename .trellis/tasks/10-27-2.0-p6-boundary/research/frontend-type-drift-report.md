# P6 遗留：前端类型代码生成漂移报告（2026-10-27）

> 执行：openapi-typescript 从 swagger.json 生成 `src/types/api.generated.ts`
> （7,850 行，115 paths / 72 schemas），替换 service 层手写 interface。
> 提交 `79a7370`。管线：`npm run generate:api-types`（swagger2openapi --patch
> → openapi-typescript，复跑 md5 一致）。

## A. 前端手写错 / 后端不收 → 运行时静默丢失（**3 个用户可见 bug**）

| # | 漂移 | 运行时影响 |
|---|------|-----------|
| 1 | `SendEmailRequest.attachments`：handler 层无此字段（send_handler.go:40 只绑定 reply_to 系） | **发邮件带附件被静默丢弃** |
| 2 | `SentEmail.attachments_count`：后端为 `attachment_count`（单数） | SentPage.tsx:502 恒 undefined，附件数永远显示回退 `\|\| 1` |
| 3 | `ProviderCreate/UpdateRequest.adapter_ids`：后端 ProviderRequest 无此字段 | ProvidersPage.tsx:497-505 适配器多选**提交即丢失**（功能不生效） |
| 4 | `SentEmailFilter.keyword`：后端 form 绑定 `search`（sent_email_service.go:31） | 已发送列表搜索**失效** |
| 5 | `SendEmailRequest.in_reply_to`/`references`：后端只绑定 `reply_to` | ComposeEmail.tsx:309 回复引用被丢弃 |
| 6 | `ReplyEmailRequest.attachments` / `ForwardEmailRequest.bcc`、`include_attachments` | 被丢弃 |
| 7 | `Provider.supported_adapters` 元素：前端 `Adapter[]`，后端 `ProviderAdapterResponse[]`（name 嵌套 `.adapter.name`） | ProvidersPage.tsx:350 徽章显示回退名 |
| 8 | `User.displayName`：后端 `display_name` | 永远收不到值 |
| 9 | `Account.total_count`：后端 `total_emails`；`consecutive_auth_failures`/`imap_host`/`starred_count` 等不在 swagger | 部分字段收不到 |
| 10 | `UpdateSMTPConfigRequest`：前端 `{smtp_enabled}` vs 后端 `{enabled}` | **无 bug** —— smtpService.ts:51 已在边界翻译 |

## B. swagger 与后端实现不一致 / 不完整（记录未改契约）

1. `OAuth2ClientListResponse`：swagger 注解 `data` 单对象，handler 实际返回
   分页列表（oauth2_client_handler.go:176-183）—— 前端手写类型与运行时一致，
   **swagger 注解错**
2. smart-select 路由：真实路由 `/smart-select/:provider_type`（router.go:322），
   swagger 丢路径参数写成 query
3. `SentEmailFilter.start_date/end_date`：后端存在但 swagger 未文档化
4. 大量前端在用端点未文档化：/accounts 子路由、/groups、/api-keys、/spam、
   /settings/*、/emails/sent/stats、/rules/apply、/translate、/probe 等
5. GET /emails、/emails/{id}、/rules、/webapi/* 等 20+ 端点响应注解为裸
   `dto.Response`（data: unknown）→ Email/Rule/Webhook 核心类型无法用生成类型替换
6. swag 不产出 required → 生成类型全可选，直接替换破坏组件
   `x.id.toString()` 等用法（多数响应类型保留手写的根本原因）

## 替换/保留决策

- **删除 9 个手写 interface**：6 替换（OAuth2ClientCreate/UpdateRequest、
  Create/UpdateWebAPIProviderRequest、TestConnectionRequest、SendResult）
  + 2 重名合并（AdapterResponse=Adapter、OAuth2ClientSmartSelectResponse=OAuth2Client）
  + 1 零引用死代码（index.ts 旧 ApiResponse）
- **保留**：Account/Email/SentEmail/SendEmailRequest 等（A 类漂移已记录，
  修复需改组件层，属后续任务）；Rule 族（前端 union 是应用层收窄）

## 建议后续任务

漂移 #1/#2/#3 是用户可见的功能缺陷，建议单独开任务修组件层 +
后端 handler 补字段绑定（发附件、附件数、适配器多选）。
