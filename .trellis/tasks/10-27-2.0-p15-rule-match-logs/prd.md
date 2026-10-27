# 打磨 P1：规则命中单封记录 —— 回答这封邮件为什么被自动处理

## Goal

规则命中目前只有 EmailRule.MatchedCount 计数器，单封邮件查不到命中了哪条规则。新增 rule_match_logs 表 + 记录 + 查询端点 + EmailDetail 展示 + 永久删除联动清理

## Requirements

- TBD

## Acceptance Criteria

- [x] RuleMatchLog 模型 + 迁移 000003 + AutoMigrate 双注册
- [x] ApplyRules 命中且动作成功后批量落库；StopProcessing 语义保持
- [x] GET /emails/:id/rule-matches 端点 + DTO；EmailDetail"已应用规则"提示条
- [x] CleanupService 每日孤儿清理（覆盖全部删除路径）
- [x] 集成测试 TestRuleMatchLogs 全链路 ✅；22 包 -race 全绿
- [x] 提交 61be3b8 已推送

## 验收记录（2026-10-27）

提交 `61be3b8`。附带发现：附件元数据全系统从未落库（email_attachments 0 行），
收件详情附件列表恒空 —— StoragePath 语义设计为独立后续任务（见 P14 PRD）。

## Notes

- Keep `prd.md` focused on requirements, constraints, and acceptance criteria.
- Lightweight tasks can remain PRD-only.
- For complex tasks, add `design.md` for technical design and `implement.md` for execution planning before `task.py start`.
