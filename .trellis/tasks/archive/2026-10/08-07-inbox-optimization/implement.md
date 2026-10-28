# 实现清单：Inbox 页面状态优化

> 分两阶段执行：先修 bug 再重构。每次改动后跑 `npm test` 确保不回归。

## 阶段一：快速修复

### 1.1 修复 groupStore/emailStore 刷新不同步

**文件**：`frontend/src/stores/emailStore.ts`

- [ ] 在 store initialState 中读取 `useGroupStore.getState().selectedGroupId`
- [ ] 若不为 `ALL_ACCOUNTS_GROUP_ID(-1)` 则写入 `filter.group_id`
- [ ] 验证：刷新页面后侧边栏高亮与列表一致

### 1.2 修复 selectedEmails 不清空

**文件**：`frontend/src/pages/InboxPage.tsx`

- [ ] 添加 `useEffect(() => { setSelectedEmails([]) }, [filter, page])`
- [ ] 验证：切换 filter/翻页后 Badge 计数归零

### 1.3 修复下拉菜单死项

**文件**：`frontend/src/pages/InboxPage.tsx`

- [ ] "选择全部"绑定 `handleSelectAll`
- [ ] "取消选择"绑定 `() => setSelectedEmails([])`
- [ ] 或移除这两个菜单项（全选复选框已存在）

### 1.4 替换 cleanHtmlContent

**文件**：`frontend/src/components/email/EmailItem.tsx`

- [ ] 确认后端 `snippet` 字段是否已是纯文本
- [ ] 替换为 `raw.replace(/<[^>]*>/g, '').trim().slice(0, 50)`
- [ ] 删除 130 行 cleanHtmlContent 函数
- [ ] 验证：邮件摘要不含乱码

### 1.5 修复 toggleStar 不更新 starredCount

**文件**：`frontend/src/hooks/useEmails.ts`

- [ ] `toggleStar` 成功后根据新状态增减 `starredCount`

### 1.6 修复 batchPermanentDelete

**文件**：`frontend/src/hooks/useEmails.ts`

- [ ] 替换 `ids.forEach(id => removeEmail(id))` 为 `removeEmails(ids)`

### 1.7 移除 EmailItem/EmailList 死代码

**文件**：`frontend/src/components/email/EmailItem.tsx`、`frontend/src/components/email/EmailList.tsx`

- [ ] 移除 `onMarkSpam` prop 和垃圾邮件按钮
- [ ] 移除 `highlightQuery` prop

**阶段一验证**：

```bash
cd frontend && npm test
cd frontend && npm run build
```

## 阶段二：组件重构

### 2.1 提取 EmailToolbar

**文件**：新建 `frontend/src/components/email/EmailToolbar.tsx`

- [ ] 提取工具栏 UI（全选、筛选、批量操作、刷新、更多菜单）
- [ ] InboxPage 引用 EmailToolbar 替换内联工具栏
- [ ] SentPage 引用 EmailToolbar 替换内联工具栏

### 2.2 提取 EmailPagination

**文件**：新建 `frontend/src/components/email/EmailPagination.tsx`

- [ ] 提取分页 UI（上一页/下一页 + 页码信息）
- [ ] InboxPage 和 SentPage 引用

### 2.3 提取 ConfirmDialog

**文件**：新建 `frontend/src/components/email/ConfirmDialog.tsx`

- [ ] 提取通用确认弹窗（open/title/description/confirmText/variant/onConfirm/onCancel）
- [ ] InboxPage 4 个 AlertDialog 替换为 ConfirmDialog
- [ ] SentPage 删除确认替换为 ConfirmDialog

### 2.4 SentPage 复用 EmailList

**文件**：`frontend/src/components/email/EmailList.tsx`、`frontend/src/components/email/EmailItem.tsx`、`frontend/src/pages/SentPage.tsx`

- [ ] EmailList 增加 `emailType?: 'inbox' | 'sent'` prop
- [ ] EmailItem 根据 emailType 区分字段渲染
- [ ] SentPage 移除内联邮件行，改用 EmailList
- [ ] 验证：SentPage 虚拟化生效，加载态不再整列表闪烁

### 2.5 拆分 InboxPage 状态

**文件**：新建 `frontend/src/hooks/useEmailActions.ts`，修改 `frontend/src/pages/InboxPage.tsx`

- [ ] 提取 dialog 状态到 useEmailActions hook
- [ ] InboxPage 引用 useEmailActions，移除内联 dialog useState
- [ ] 验证：所有弹窗行为不变

**阶段二验证**：

```bash
cd frontend && npm test
cd frontend && npm run build
```

## 风险文件

| 文件 | 风险 | 回滚 |
|------|------|------|
| emailStore.ts | 初始 filter 变化影响所有页面 | 还原 initialState |
| useEmails.ts | action 行为变化 | 按提交回滚 |
| EmailItem.tsx | 摘要渲染变化 | 还原 cleanHtmlContent |
| InboxPage.tsx | 重构后组件行为变化 | 还原到提取前版本 |
| SentPage.tsx | EmailList 适配不完整 | 还原内联渲染 |

## 完成定义

满足 prd.md AC1–AC10，且 implement.md 全部勾选。