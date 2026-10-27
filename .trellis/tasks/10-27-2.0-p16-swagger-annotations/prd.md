# 打磨 P2a：swagger 注解质量 —— 消除 data: unknown、补 required

## Goal

20+ 端点 swagger 注解 data: unknown、无 required 标注，是 openapi 生成类型覆盖率的天花板。补具体类型与 required

## Requirements

- TBD

## Acceptance Criteria

- [x] 全库裸 `@Success {object} dto.Response` 清零（26 处 200 + 6 处非 200 同款）
- [x] 新增 9 个 response DTO 文件；service 统计结构/裸序列化 model 不进契约
- [x] swag 0 解析错误；definitions 72→110；paths 111 不变；无 unknown 残留
- [x] 22 包 -race 全绿；提交 0f7fa1d 已推送

## 验收记录（2026-10-27）

子代理两次失败（第一次零产出，第二次产出 21/113 后中断），剩余
92 端点由主会话直接完成。数据流溯源均以 dto.SuccessResponse 实参
为 ground truth，gin.H/model → DTO 替换均逐键核对 JSON 等价。

## Notes

- Keep `prd.md` focused on requirements, constraints, and acceptance criteria.
- Lightweight tasks can remain PRD-only.
- For complex tasks, add `design.md` for technical design and `implement.md` for execution planning before `task.py start`.
