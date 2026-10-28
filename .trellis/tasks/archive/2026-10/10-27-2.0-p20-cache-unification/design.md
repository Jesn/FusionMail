# P20 设计：前端缓存语义统一

## 目标

把 5 套并存的缓存语义收敛到**一个策略声明 + 一个失效入口**，且不改变页面可见行为。

## 单一策略源：`utils/cachePolicy.ts`

P12 已建立该文件，但只有 2 个 TTL 常量，且 `settingsCache.ts` 完全绕过它。
本次把它扩成唯一策略源，分三块：

```ts
// 1) 内存 TTL（业务数据，刷新即拉取）
ACCOUNT_CACHE_TTL / GROUP_CACHE_TTL / EMAIL_LIST_CACHE_TTL
EMAIL_DETAIL_CACHE_TTL / SEARCH_CACHE_TTL / LOG_STATS_CACHE_TTL

// 2) LRU 容量上限（原散落在 emailCacheStore）
MAX_EMAIL_CACHE_ENTRIES / MAX_EMAIL_DETAIL_CACHE_ENTRIES / MAX_SEARCH_CACHE_ENTRIES

// 3) React Query 语义（原散落在 useSettings）
SETTINGS_STALE_TIME / SETTINGS_GC_TIME / SETTINGS_PUBLIC_STALE_TIME
SETTINGS_CATEGORIES_STALE_TIME / SETTINGS_SEARCH_STALE_TIME / SETTINGS_STATS_STALE_TIME

// 4) 持久化白名单（按 key，不按分类）
PERSISTED_SETTING_KEYS: readonly string[]

// 5) 过期判断（已有）
isCacheExpired(ts, ttlMs)
```

## 失效语义：区分「数据变了」与「数据新鲜」

**当前 bug**：`accountStore` / `groupStore` 的 mutation 在本地 patch 后写
`cacheTimestamp: Date.now()`。这会把"我们持有完整新鲜服务端数据"的时钟**拨到最新**，
等于 mutation 反而**续命** TTL。

具体后果（以 groupStore 为例）：`createGroup` 只往 `groups` 数组里塞一条，**不更新**
`totalCount` / `ungroupedCount` / `unreadCount` 等来自服务端的聚合字段，却把
`cacheTimestamp` 刷新 → `fetchGroups` 在接下来 5 分钟内直接 return，聚合计数持续陈旧。

**修正**：本地 patch 只更新数据，**不碰 `cacheTimestamp`**；需要让后续读取拿到服务端
真相时，显式调用失效函数。即：

- 本地 patch（`updateAccount` / `updateGroup` 等）→ 只改数据
- 结构性变更（create/delete/reorder，影响聚合计数）→ 调用 `invalidate*()` 置
  `cacheTimestamp = null`，使下次 `fetch*` 必拉

`setCacheTimestamp(0)` 的 3 处 hack 全部替换为具名失效函数。

## 失效入口设计（最终实现）

新增 `utils/cacheInvalidation.ts`，集中失效动作，避免调用方各自摸 store 内部字段：

```ts
invalidateAccounts()   // accountStore.invalidate(): cacheTimestamp = null
invalidateGroups()     // groupStore.invalidate():  cacheTimestamp = null
```

**未提供** `invalidateEmails()` / `invalidateSettings()`：
- 邮件缓存的失效已挂在 `emailService` / `spamService` 的每个写操作上（既有 `clear*` 调用），
  再包一层只是重复。
- 设置的失效已由 React Query 的 `queryClient.invalidateQueries` 承担，且必须与
  mutation 同处一个 hook 才拿得到 `queryClient`。

**未提供** `invalidateAccountData()`（同时作废两者）：初版设计里有，但实现时发现
没有正确调用方——`accountStore.removeAccount` 已自行作废账户缓存，再作废一次会导致
`loadAccounts` 冗余重拉。故删除，调用方按需分别调用。

挂载点（AC 要求「不能只靠 TTL」）：

| mutation | 失效 |
|---|---|
| `useAccounts.createAccount` | `invalidateGroups()`（新账户默认未分组） |
| `useAccounts.updateAccountData` | 仅当 `group_id` 真的变了才 `invalidateGroups()` |
| `useAccounts.deleteAccount` / `restoreAccount` / `forceDeleteAccount` | `invalidateGroups()`（账户缓存由 `removeAccount` 自行作废） |
| `groupStore.createGroup` / `deleteGroup` | `cacheTimestamp = null`（聚合计数陈旧） |
| `groupStore.editGroup` / `reorderGroups` | 不失效、不推进（数据本地已正确） |
| `hooks/useEmails.markAllAsRead` | `invalidateGroups()`（替换原 `setCacheTimestamp(0)`） |
| `pages/AccountsPage` 2 处 | `invalidateGroups()`（替换原 `setCacheTimestamp(0)`） |
| `emailService` / `spamService` 各邮件变更 | 既有 `clear*` 调用，未改动 |


## 持久化：`utils/settingsCache.ts` 重写

现状：`updateCachedSettings(category, settings)` 把整个分类写盘，含 `isSensitive` 项。

改为按 key 白名单：

```ts
// 只写白名单内的 key，其余丢弃
saveSettingsCache(settings: Record<string, string>): void
loadSettingsCache(): Record<string, string> | null
getCachedSetting(key: string): string | null      // 替代 getCachedSettings(category)
updateCachedSettings(settings: Record<string, string>): void  // 只挑白名单 key
```

TTL 从 `cachePolicy.SETTINGS_CACHE_TTL` 取（原为本地 `CACHE_DURATION = 30min`）。

调用方改动：
- `stores/emailStore.ts` `getCachedPageSize()` → `getCachedSetting('email_page_size')`
- `components/settings/SettingsCategory.tsx` → 只传白名单 key（`email_page_size`），
  不再整分类落盘
- `services/authService.ts` 的 `clearSettingsCache()` 保持
- `pages/SettingsPage.tsx` → 见下（实现时发现的第 6 条旁路）

### 实现时发现的第 6 条持久化旁路

`pages/SettingsPage.tsx`（DEV-only 遗留页，生产环境 `/settings/legacy` 直接重定向）
用三个旁路键把**整个分类**写 localStorage：

```
fusionmail_sync_settings / fusionmail_notification_settings / fusionmail_ui_settings
```

`grep` 确认这 3 个键**没有任何读取方**（只写垃圾），且同样存在整分类落盘的
敏感值风险。按 AC「每类数据一条写路径」收敛为调用 `updateCachedSettings()`，
由白名单过滤。因该页状态未接入真实设置 API、键名也不在白名单内，保存实际不持久化
任何内容——与改动前行为一致（旧键本就无人读）。

> 建议后续（不在本任务范围）：该页是 DEV-only 死代码，可整体删除，
> 连同 `App.tsx` 的 `/settings/legacy` 路由。


## 不改动的部分（边界）

- 不动 `emailCacheStore` 的 LRU / inflight 去重**机制**，只把常量搬到 cachePolicy
- 不动 React Query 的 queryKey 结构
- 不动 UI 偏好（theme/侧栏/搜索历史/导入格式）与登录凭证的既有持久化
- 不改任何页面可见行为；`selectedGroupId` 刷新回 `-1` 的现状语义保留（仅修错注释）
- 不引入新依赖

## 风险

| 风险 | 缓解 |
|---|---|
| 去掉 mutation 续命后，请求变多 | 只在结构性变更失效；本地 patch 不失效，故编辑类操作请求数不变 |
| `settingsCache` 改按 key 后，旧的整分类缓存数据格式不兼容 | 读时按白名单过滤，旧数据自然被忽略；键名不变故无需迁移 |
| `useSettings` 常量收敛改变实际 staleTime | 逐值对照，保持数值不变（仅提取常量），行为零变化 |
