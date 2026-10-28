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
| **useSettings（PRD 原未列）** | **TanStack Query** | **staleTime 5/10/30min、2min、30s** |

复核更正（2026-10-28）：
- 实际是 **5 套**，不是 4 套。`hooks/useSettings.ts` 用 TanStack Query，散落
  staleTime 5/10/30min、2min、30s 与 gcTime 10min，与其余 4 套互不相干。
- `emailCacheStore` 并非"无 TTL"，而是 per-entry `expiresIn`：列表 5min、详情 10min、
  搜索 2min（`stores/emailCacheStore.ts:47-49`）。
- 全仓**没有任何显式失效入口**（`grep invalidate` 为 0）。现有唯一"强制失效"手段是
  `setCacheTimestamp(0)` 这种 hack，出现在 `hooks/useEmails.ts:292`、
  `pages/AccountsPage.tsx:113`、`:215` 共 3 处。
- `stores/emailStore.ts` 注释称"从 groupStore 的持久化状态同步 group_id"，但 groupStore
  已不持久化，注释是错的。

## ✅ 前置决策（2026-10-28 用户已确认）

**问：哪些数据要跨刷新持久化？**

**决策：选项 2 —— 仅设置类持久化，且设置白名单按 key 列举。**
同时确认：把 PRD 未列出的第 5 套语义（`hooks/useSettings.ts` 的 TanStack Query
staleTime/gcTime）一并纳入统一。

### 落盘（localStorage）三类

| 类别 | 具体 | 说明 |
|---|---|---|
| 登录凭证 | `auth_token` / `auth_expires` | 既有行为，不动 |
| UI 偏好 | theme/mode、侧栏折叠、搜索历史、批量导入格式 | 既有行为，不动 |
| 设置白名单 | 仅 `email_page_size`（按 key 列举，不整分类落盘） | 见下 |

设置白名单保留 `email_page_size` 的实据：`stores/emailStore.ts` 在**模块初始化时同步**
调用 `getCachedPageSize()` 填充 `pageSize` 初值，而 `hooks/useEmails.ts` 的 `loadEmails`
`useCallback` deps 含 `pageSize`。若改为纯内存，首屏会先按默认 20 发一次列表请求 →
`setEmails` 用服务端 `page_size` 覆盖 → deps 变化触发第二次请求，多一次请求且列表条数
闪变。持久化该 key 正是为消除这个抖动。

### 不落盘（内存 + TTL，刷新即拉取）

账户列表 / 账户统计 / 分组列表+统计 / 邮件列表·详情·搜索缓存 /
会话选择态（`selectedAccount`、`selectedGroupId`、`filter`）。

理由：均为服务端真相且易变，陈旧数据风险 > 省下的首屏时间；`emailCacheStore` 曾为此
维护整套配额估算+清理逻辑，P12 已删除，不走回头路。

### 决策带出的缺陷（本次一并修）

`components/settings/SettingsCategory.tsx` 调用
`updateCachedSettings(name, newItems)` 把**整个分类**写入 localStorage，而
`SettingItem` 有 `isSensitive` 字段、该组件自身也在统计 `sensitiveCount`。
即敏感设置值可能被整分类落盘。白名单改为按 key 列举后此风险消除。

## Requirements

- 按用户决策选定单一策略后，四个缓存点全部收敛到同一机制
- 缓存失效必须挂在对应 mutation 上（改账户 → 账户缓存失效），不允许只靠 TTL
- 不改变任何页面可见行为（除非用户决策明确要求）
- 大文件治理期间新建的组件（components/spam、oauth2、logview、email、webapi、provider、account）不得引入第 5 套语义

## Acceptance Criteria

- [x] 用户持久化决策已记录在本文件（见上方「前置决策」）
- [x] 全仓 `grep` 复核：缓存读写入口唯一（每类数据一条路径）
- [x] tsc / vitest / vite build 全绿（测试从基线 5 文件 24 个 → 11 文件 67 个）
- [x] 手工冒烟：账户/分组/设置改动后刷新页面数据一致
      —— 本机无浏览器自动化（无 playwright/puppeteer/chromium），
      已用 `vi.resetModules()` 模拟刷新固化为 `stores/cacheRefresh.test.ts`；
      真机浏览器确认仍建议人工走一遍（步骤见 implement.md）

## 完成记录（2026-10-28）

### 交付

| 文件 | 变更 |
|---|---|
| `utils/cachePolicy.ts` | 26 → 108 行，扩为唯一策略源（TTL / LRU 上限 / React Query 语义 / 持久化白名单） |
| `utils/cacheInvalidation.ts` | 新增 31 行，唯一失效入口 |
| `utils/settingsCache.ts` | 87 → 123 行，改为按 key 白名单 + TTL 取自 cachePolicy + 旧格式兼容 |
| `stores/accountStore.ts` / `stores/groupStore.ts` | 修 mutation 续命 bug，新增 `invalidate()` |
| `stores/emailCacheStore.ts` | 常量搬到 cachePolicy（机制未动） |
| `hooks/useSettings.ts` | 563 → 573 行（+10 为 import 块），10 处 staleTime/gcTime 收敛 |
| `hooks/useAccounts.ts` / `hooks/useEmails.ts` / `pages/AccountsPage.tsx` | 失效挂到 mutation；替换 3 处 `setCacheTimestamp(0)` |
| `pages/SettingsPage.tsx` | 消除第 6 条旁路持久化 |
| `services/emailService.ts` / `services/authService.ts` / `components/settings/SettingsCategory.tsx` / `stores/emailStore.ts` / `pages/LogsPage.tsx` | 常量收敛 / 白名单适配 / 修 stale 注释 |

### 修掉的真实缺陷

1. **mutation 给 TTL 续命**：`accountStore`/`groupStore` 的 mutation 写
   `cacheTimestamp: Date.now()`。`createGroup` 不更新 `totalCount`/`ungroupedCount`
   却把时钟拨到最新 → 聚合计数陈旧且 5 分钟内不再重拉。
2. **敏感设置整分类落盘**：`SettingsCategory.tsx` 与 `authService.ts` 按分类写盘，
   而设置项带 `isSensitive`。
3. **第 6 条旁路持久化**：`SettingsPage.tsx` 的 `fusionmail_{sync,notification,ui}_settings`
   三个只写不读的整分类键。
4. **`setCacheTimestamp(0)` hack** 3 处 → 具名失效函数。
5. **stale 注释**：`emailStore.ts` 称从「groupStore 的持久化状态」同步，实际 groupStore
   已不持久化。

### 未做（边界）

- 未删 `pages/SettingsPage.tsx`（DEV-only 死代码）——属 P3 deadcode 范围，已记入 design.md 建议
- 未动 `emailCacheStore` 的 LRU / inflight 机制
- 未动 React Query 的 queryKey 结构
- 未动 UI 偏好与登录凭证的既有持久化
- 未引入新依赖

## Notes

- 相关文件：frontend/src/stores/*、frontend/src/hooks/useSettings.ts
  （大文件治理时判定保留——相关 hooks 集合；本任务动过它，行数 563 → 573，
  仅新增 import 块，未增加逻辑复杂度，判定维持保留）
- 规格沉淀：`.trellis/spec/frontend/state-management.md` 已填充
  （缓存策略 / 失效入口 / 持久化白名单 / 5 条常见错误）
