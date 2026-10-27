# 2.0 后续 B：前端缓存语义统一

## Goal

统一前端 4 套并存的缓存语义为一套一致策略。P12 前端收敛的最后一项残余
（大文件治理已完成，见 `10-27-2.0-p12-frontend-convergence` 与 spec 修订 `613d337`）。

## 现状（4 套语义并存）

| 缓存 | 机制 | TTL/失效 |
|---|---|---|
| accountStore | 内存 store | TTL 10min |
| groupStore | 内存 store | TTL 5min |
| emailCacheStore | getOrSet 惰性 | 无 TTL（引用失效） |
| settingsCache | localStorage | 跨刷新持久化 |

## ⚠️ 前置决策（开工前必须先问用户）

**哪些数据要跨刷新持久化？** 这决定统一方向：
- 全部内存化（刷新即拉取）→ 最简单，settingsCache 也废弃
- 仅设置类持久化（现状语义）→ 统一为"内存 TTL + 设置白名单 localStorage"
- 引入统一缓存层（如 TanStack Query 的 staleTime/gcTime）→ 改动最大但语义最清晰

## Requirements

- 按用户决策选定单一策略后，四个缓存点全部收敛到同一机制
- 缓存失效必须挂在对应 mutation 上（改账户 → 账户缓存失效），不允许只靠 TTL
- 不改变任何页面可见行为（除非用户决策明确要求）
- 大文件治理期间新建的组件（components/spam、oauth2、logview、email、webapi、provider、account）不得引入第 5 套语义

## Acceptance Criteria

- [ ] 用户持久化决策已记录在本文件
- [ ] 全仓 `grep` 复核：缓存读写入口唯一（每类数据一条路径）
- [ ] tsc / vitest（5 文件 24 测试基线）/ vite build 全绿
- [ ] 手工冒烟：账户/分组/设置改动后刷新页面数据一致

## Notes

- 相关文件：frontend/src/stores/*、frontend/src/hooks/useSettings.ts（563 行，
  大文件治理时判定保留——相关 hooks 集合，本任务若动它需重新评估行数）
