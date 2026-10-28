# P20 实施顺序与验证

## 执行顺序

1. **扩 `utils/cachePolicy.ts`** 为唯一策略源（TTL / LRU 上限 / React Query 语义 /
   持久化白名单）。纯新增，不改既有导出语义。
2. **新增 `utils/cacheInvalidation.ts`**，提供 `invalidateAccounts` / `invalidateGroups`
   / `invalidateEmails` / `invalidateSettings`。
3. **修 mutation 续命 bug**：`stores/accountStore.ts`、`stores/groupStore.ts` 的
   mutation 去掉 `cacheTimestamp: Date.now()`；结构性变更改调 `invalidate*()`。
4. **替换 `setCacheTimestamp(0)` hack** 3 处：`hooks/useEmails.ts:292`、
   `pages/AccountsPage.tsx:113`、`:215`。
5. **重写 `utils/settingsCache.ts`** 为按 key 白名单 + TTL 取自 cachePolicy。
6. **改调用方**：`stores/emailStore.ts`（`getCachedPageSize`）、
   `components/settings/SettingsCategory.tsx`（不再整分类落盘）。
7. **收敛 `stores/emailCacheStore.ts`** 的 TTL / 容量常量到 cachePolicy。
8. **收敛 `hooks/useSettings.ts`** 的 staleTime/gcTime 到 cachePolicy（数值不变）。
9. **收敛 `pages/LogsPage.tsx`** 的 `STATS_CACHE_TTL` 到 cachePolicy。
10. **修 stale 注释**：`stores/emailStore.ts` 关于"groupStore 持久化状态"的描述。
11. **grep 复核**每类数据读写路径唯一。
12. **补测试**：为 cachePolicy / cacheInvalidation / settingsCache 白名单加单测。

## 验证

```bash
cd frontend
npx tsc --noEmit
npx vitest run      # 基线 5 文件 24 测试，本次新增后应 >24 且全绿
npx vite build
```

纯前端任务，不需要 PG/Redis，后端不动。

### grep 复核清单（AC 第 2 条）

```bash
# 每类数据只应有一条读写路径
grep -rn "localStorage" src --include=*.ts --include=*.tsx   # 仅凭证/UI偏好/设置白名单
grep -rn "staleTime\|gcTime" src --include=*.ts --include=*.tsx  # 仅 cachePolicy 定义 + useSettings 引用
grep -rn "setCacheTimestamp(0)" src                          # 应为 0 结果
grep -rn "invalidate" src --include=*.ts --include=*.tsx     # 应只有 cacheInvalidation + 调用点
```

### 手工冒烟（AC 第 4 条）

1. 账户页：改账户名 → 刷新 → 名称一致
2. 分组页：建分组 → 未分组计数立即正确（回归本 bug）→ 刷新 → 一致
3. 设置页：改 `email_page_size` → 刷新 → 邮件列表首屏即按新值请求（不闪变）
4. 设置页：改非白名单设置 → 刷新 → 不落盘（localStorage 无该 key）
5. 主题/侧栏折叠 → 刷新 → 保持

## 回滚

改动集中在 `utils/` + 2 个 store + 1 个 hook，无 schema/接口变更，`git revert` 即可。
