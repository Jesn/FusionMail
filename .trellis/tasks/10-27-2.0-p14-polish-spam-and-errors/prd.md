# 打磨 P0：垃圾规则引擎 Pattern 化 + 同步错误结构化归因

## Goal

P0-1 垃圾规则引擎 content/url/attachment 三类目改为 Pattern 驱动（含内置规则向后兼容）；P0-2 同步失败归因从字符串匹配改为结构化错误分类；附带：同步路径把适配器附件元数据注入检测（内存，不落库）

## Requirements

- R1（P0-1）：垃圾规则 content/url/attachment 三类目改为 Pattern 驱动，历史内置规则行为不变或从失效变生效
- R2（P0-2）：同步失败归因三层结构化（WebAPIError → TokenError → 收窄的字符串短语），移除裸 "401"/"auth" 误判源
- R3（附带）：同步路径检测前注入适配器附件元数据（仅内存），附件类规则与预过滤层附件提取开始工作

## Acceptance Criteria

- [x] matchContent 支持 "uppercase:0.3"/"special:0.2"/"html_tags:100" 语法，裸阈值回退名字路由（测试 7 例）
- [x] matchURL 整数 Pattern=链接数阈值、其他=URL 域名匹配、空 Pattern 回退（测试 5 例）
- [x] matchAttachment 扩展名/MIME 前缀匹配 email.Attachments（测试 5 例）
- [x] isAuthError 三层：webapi.IsAuthError（errors.As 穿透 %w）→ TokenError(invalid_grant/401/403) → 收窄短语（测试 18 例）
- [x] go test ./... -race -count=1 22 包全绿
- [x] 提交：7593600（P0-1）、bb1c90b（P0-2），均已推送

## 验收记录（2026-10-27）

- P0-1 提交 `7593600`：rule_engine.go 三 matcher 重写 + sync_processor 两处注入 + 13 新用例
- P0-2 提交 `bb1c90b`：sync_error_policy.go 三层归因 + 18 新用例
- 附带发现（记录不展开）：附件元数据全系统从未落库（email_attachments 0 行，
  无 Create 调用方），收件详情页附件列表恒空 —— 持久化需 StoragePath 语义
  设计，列为后续独立任务

## Notes

- Keep `prd.md` focused on requirements, constraints, and acceptance criteria.
- Lightweight tasks can remain PRD-only.
- For complex tasks, add `design.md` for technical design and `implement.md` for execution planning before `task.py start`.
