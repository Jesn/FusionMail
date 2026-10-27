# P13 类型漂移修复

## Goal

修复 P6 类型代码生成时发现的字段级漂移中的**真实功能缺陷**
（见 `10-27-2.0-p6-boundary/research/frontend-type-drift-report.md`）。
每项先核实后端真实行为再修，区分"缺陷"与"设计如此"。

## 缺陷清单与修复方向

| # | 缺陷 | 根因 | 修复 |
|---|------|------|------|
| 1 | 发邮件带附件被静默丢弃 | 链路三处断：UploadAttachment 不存文件不返回 temp_path（send_handler.go:481 注释"可能需要保存到临时存储"）；handler SendEmailRequest 不绑 attachments；service 层支持完好 | 后端补临时文件生命周期 + handler 绑定 + 路径穿越防护 |
| 2 | 已发送附件数恒显示回退 `\|\| 1` | 前端 `attachments_count` vs 后端 `attachment_count`（单数） | 前端改名 |
| 3 | 适配器多选提交即丢失 | 后端有 ProviderAdapter 关联模型（model/provider_adapter.go:9）且 seed 写入，但 API 层（dto/request/provider.go 只有 default_adapter_id）没接 | 后端补全接线（dto+service+事务），前端提交逻辑对齐 |
| 4 | 已发送搜索失效 | 前端发 `keyword`，后端绑定 `search`（sent_email_service.go:31） | 前端改名 |
| 5 | User.displayName 收不到 | 后端 `display_name` | 前端改名 |
| 6 | Account.total_count 收不到 | 后端 `total_emails` | 前端改名 |
| 7 | Provider 徽章显示回退名 | 前端读 `adapter.name`，后端嵌套 `.adapter.name` | 前端对齐形状 |

## 复核后不修（设计如此）

- `in_reply_to`/`references` 被忽略 —— 后端 Reply/Forward 时自动从原邮件
  推导（send_service.go:222,298），前端发它是多余动作；前端类型加注释说明
- `UpdateSMTPConfigRequest` smtp_enabled/enabled —— smtpService.ts:51 已在边界翻译

## 明确不做（工程决策，记录于 STATUS.md）

- **cursor 分页**：个人邮箱应用分页深度浅（每页 20-50 条），索引表上
  offset+limit 性能足够；实时插入导致的重复/漏行在邮件列表场景影响小；
  契约变更波及 24 端点 + 全部列表页，收益不抵风险

## Acceptance Criteria

- [x] 带附件发信端到端可用（上传→temp_path→发送→SMTP 含附件）—— 后端链路已通（`8e940ea`），SMTP 实发需运行环境
- [x] 临时文件有生命周期（发送后 defer 清理 / 上传时清 24h 孤儿）+ 路径穿越防护（5 个测试）
- [x] 适配器多选创建 Provider 后关联真实落库（`f150e3d`，6 个关联落库测试）
- [x] #2/#4/#5/#6 前端对齐后端真实字段名（`bcd2673`）；#7 归 adapter_ids 接线任务
- [x] go build/vet/test -race 22 包绿；tsc/vitest(36)/vite build 绿
- [x] swagger.json 重新生成且与注解一致（附件 +35 行、adapter_ids +21 行）

## 完成记录（2026-10-27）

| 提交 | 内容 |
|------|------|
| `8e940ea` | 附件链路修复：AttachmentTempStore（落盘/穿越防护/生命周期）+ 三端点绑定 attachments |
| `bcd2673` | 前端 4 字段对齐后端真实命名（attachment_count/search/display_name/total_emails） |
| `f150e3d` | ProviderAdapter 关联全栈接线（dto/事务 repo/service 校验/handler ToResponse/前端形状） |

执行方式：主会话做安全敏感的附件链路；子智能体 E 做前端字段重命名
（正确识别 total_count 在 /emails/stats 是真实字段名，避免过度修复）；
子智能体 F 做 adapter_ids 全栈接线（发现并修复 container 里 adapterRepo
未接线的隐藏 bug）。
