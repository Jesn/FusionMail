# 打磨 P2d：前端架构收敛 —— 缓存语义统一 + AccountsPage 拆分

## Goal

4 套缓存语义统一为内存态+TTL；AccountsPage 1471→989 行

## Requirements

- TBD

## Acceptance Criteria

- [x] 缓存语义统一：utils/cachePolicy.ts 唯一声明；accountStore/groupStore/emailCacheStore 去 persist（业务数据内存态+TTL，不跨刷新）；emailCacheStore 配额逻辑整删
- [x] AccountsPage 拆分：1471→989 行，GroupSidebar(192)/AccountTableSection(551) 纯展示组件
- [x] tsc / vitest 24 / vite build 全绿
- [x] 提交 ef8ccdc + a7f62a7 已推送

## 验收记录（2026-10-27）

缓存决策（无人值守）：业务数据不跨刷新持久化，只有 UI 偏好与凭证进
localStorage —— 正确性优先，刷新后正常拉取。989 行不再往下拆：需抽
hooks 才能到 700，与纯展示约束冲突且无测试覆盖，收益递减。

## Notes

- Keep `prd.md` focused on requirements, constraints, and acceptance criteria.
- Lightweight tasks can remain PRD-only.
- For complex tasks, add `design.md` for technical design and `implement.md` for execution planning before `task.py start`.
