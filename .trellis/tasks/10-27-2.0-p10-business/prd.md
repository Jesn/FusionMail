# P10 业务逻辑重构与功能缺陷

## Goal

修复业务逻辑层的功能缺陷与性能问题。每项修复必须先核实问题属实
（本项目多次出现"分析说有问题、实际早已正确"的误报），再动手。

## Acceptance Criteria 与完成状态

- [x] 中文全文搜索失效 —— trigram 索引 + 按字符集选查询路径（`6860013`）
- [x] 邮件主列表缺复合索引 —— 去排序节点（`d5199cd`）
- [x] 规则动作 `add_label`/`remove_label`/`webhook` 静默 no-op —— 实现之，
      含 SSRF 防护/幂等/失败留痕（`54fc96e`）
- [x] 前后端动作名不匹配 —— 前端提交 `trigger_webhook` 被后端校验拒绝，
      用户无法创建 Webhook 规则（`54fc96e`）
- [x] 分组列表 N+1（1+2N→3）与子账户列表 N+1（3+N→4）—— GROUP BY 聚合，
      附调用次数回归测试（`d71de47`）
- [x] 复核排除两个误报：`ReconcileChildAccounts`（条件写非读 N+1）、
      `BatchDeleteEmails`（自 bf3672b 起就是批量 SQL）

## 遗留（含决策项）

| 项 | 状态 | 说明 |
|----|------|------|
| ~~`BatchPermanentDeleteEmails` / `EmptyTrash` 逐封删除~~ | ✅ 已修（`74aa0d7`） | 2N→2 / 恒 1 条，附调用次数回归测试 |
| offset → cursor 分页 | **需用户决策** | 24 个 repository 端点 + 前端全部列表页用 `page/page_size`，契约变更影响面大，不宜无人值守单方面执行 |

## Notes

- 教训沉淀：本项目"疑似 N+1"三次复核两次是误报 —— 性能修复必须先
  grep 调用链 + 读 SQL 实现，确认循环体内真有逐条查询
- 测试约定见 `.trellis/spec/backend/testing.md`
