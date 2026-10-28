# State Management

> How state is managed in this project.

---

## Overview

- **客户端状态**：zustand（`frontend/src/stores/*`），无 middleware、无 persist。
- **服务端状态（设置类）**：TanStack React Query（`frontend/src/hooks/useSettings.ts`）。
- **缓存策略**：全部集中在 `frontend/src/utils/cachePolicy.ts`，是**唯一策略源**。
- **缓存失效**：全部集中在 `frontend/src/utils/cacheInvalidation.ts`，是**唯一失效入口**。

---

## State Categories

| 类别 | 载体 | 跨刷新 |
|---|---|---|
| 登录凭证 | `authStore` + localStorage | 是 |
| UI 偏好（主题/侧栏折叠/搜索历史/导入格式） | 各组件 + localStorage | 是 |
| 设置白名单（见下） | `utils/settingsCache.ts` + localStorage | 是 |
| 账户 / 分组 / 邮件 / 搜索 / 统计 | zustand store（内存） | 否 |
| 设置类查询 | React Query cache（内存） | 否 |

---

## Server State

### 铁律

1. **业务数据不跨刷新**。账户、分组、邮件列表/详情/搜索、统计一律只存内存，
   带 TTL，刷新后重新拉取。理由：邮件客户端数据易变，陈旧数据风险 > 首屏收益。
2. **TTL / 容量上限 / React Query 语义禁止散落魔法数字**，一律从
   `utils/cachePolicy.ts` 取。
3. **设置持久化必须按 key 登记**。`PERSISTED_SETTING_KEYS` 是白名单，
   新增一项需在该常量显式登记，并说明为何需要跨刷新（通常理由：模块初始化时
   被同步读取）。**禁止按分类整块落盘**——设置项带 `isSensitive` 字段，
   整分类落盘会把敏感值写进 localStorage。
4. **缓存失效挂在 mutation 上，不能只靠 TTL**。

### 缓存时钟语义（易错点）

`cacheTimestamp` 的含义是「**上一次拿到全量服务端数据**的时间」，不是
「最后一次改动数据的时间」。因此：

- **全量服务端响应**（`setAccounts` / `setGroups` / `fetchGroups` 成功）→
  推进 `cacheTimestamp = Date.now()`
- **本地 patch**（改名、改描述、改状态）→ **不碰** `cacheTimestamp`
- **结构性变更**（增删、影响服务端聚合计数）→ 置 `cacheTimestamp = null`

> ⚠️ **历史 bug（P20 修复）**：mutation 里写 `cacheTimestamp: Date.now()` 等于给
> TTL **续命**。`groupStore.createGroup` 只往数组塞一条、不更新
> `totalCount` / `ungroupedCount`，却把时钟拨到最新，导致 `fetchGroups` 在接下来
> 5 分钟内直接 return，聚合计数持续陈旧。
>
> 同理，`setCacheTimestamp(0)` 这种「强制过期」hack 已全部替换为
> `cacheInvalidation.ts` 的具名函数。

### 失效入口

```ts
import { invalidateAccounts, invalidateGroups } from '../utils/cacheInvalidation';

invalidateAccounts();  // 账户列表作废
invalidateGroups();    // 分组列表作废（含 account_count 等聚合计数）
```

账户增删会同时影响分组聚合计数，故需按需同时调用两者。注意
`accountStore.removeAccount` 已自行作废账户缓存，此时只需补 `invalidateGroups()`。

邮件缓存的失效走 `services/emailService.ts` / `services/spamService.ts` 里既有的
`useEmailCacheStore.getState().clear*()`，已挂在各写操作上。

### localStorage 允许的写入方

只有三类，其余一律禁止：

| 键 | 写入方 |
|---|---|
| `auth_token` / `auth_expires` / `fusionmail-auth` | `services/api.ts`、`stores/authStore.ts` |
| `fusionmail_color_theme` / `fusionmail_theme_mode` | `lib/theme.ts`、`components/theme/ThemeProvider.tsx` |
| `fusionmail_search_history` | `hooks/useSearch.ts` |
| `fusionmail_batch_import_format` | `components/account/BatchImportDialog.tsx` |
| `sidebar-mail-open-sections` / `sidebar-settings-advanced-open` | `components/layout/MailMenu.tsx`、`AdminMenu.tsx` |
| `user_settings_cache` / `user_settings_cache_timestamp` | **仅** `utils/settingsCache.ts` |

---

## Common Mistakes

1. **在 mutation 里推进 `cacheTimestamp`** → 给 TTL 续命，聚合计数长期陈旧。
2. **用 `setCacheTimestamp(0)` 强制过期** → 语义不明的 hack，已废弃，改用
   `cacheInvalidation.ts`。
3. **按分类把设置整块写 localStorage** → 敏感值泄漏风险。
   `pages/SettingsPage.tsx` 曾用 `fusionmail_{sync,notification,ui}_settings` 三个
   旁路键整分类落盘（且无任何读取方），P20 已收敛到 `settingsCache`。
4. **在组件里直接 `localStorage.setItem` 存业务数据** → 绕过白名单与 TTL。
5. **改 TTL 时只改一处** → 现在只有 `cachePolicy.ts` 一处可改；若发现第二处，
   说明有人在散落魔法数字。

---

## 验证

```bash
cd frontend
npx tsc --noEmit && npx vitest run && npx vite build
```

缓存语义的回归测试位于：
- `utils/cachePolicy.test.ts` — 策略常量与白名单
- `utils/settingsCache.test.ts` — 白名单过滤、旧格式兼容、TTL 过期
- `utils/cacheInvalidation.test.ts` — 失效入口
- `stores/accountStore.test.ts` / `stores/groupStore.test.ts` — 缓存时钟语义
- `stores/cacheRefresh.test.ts` — 模拟刷新（`vi.resetModules()`）验证持久化口径
