# 设计：Inbox 页面状态优化

## 1. 架构边界

本任务仅改前端，不涉及后端 API、数据库或部署拓扑。

| 层 | 是否改动 | 内容 |
|----|----------|------|
| 前端 stores | 是 | emailStore 初始化时同步 groupStore 持久化状态 |
| 前端 hooks | 是 | useEmails 修复 starredCount/batchDelete；新建 useEmailActions |
| 前端组件 | 是 | 新建 EmailToolbar/EmailPagination/ConfirmDialog；修改 EmailList/EmailItem |
| 前端页面 | 是 | InboxPage 拆分；SentPage 改用 EmailList |
| 后端 | 否 | — |
| DB / migration | 否 | — |

## 2. 状态同步设计

### 问题

```
页面刷新
  → groupStore 从 localStorage 恢复 selectedGroupId = 5
  → emailStore filter 重置为 { is_archived: false, is_deleted: false, is_spam: false }
  → 侧边栏高亮"分组5"，但 InboxPage 请求的是全部邮件
```

### 修复方案

在 `emailStore` 的 `initialState` 中，通过 `groupStore.getState().selectedGroupId` 读取持久化值，若不为 `ALL_ACCOUNTS_GROUP_ID(-1)` 则写入 `filter.group_id`。

```typescript
// emailStore.ts initialState
const persistedGroupId = useGroupStore.getState().selectedGroupId
const initialFilter = {
  is_archived: false,
  is_deleted: false,
  is_spam: false,
  ...(persistedGroupId !== ALL_ACCOUNTS_GROUP_ID ? { group_id: persistedGroupId } : {}),
}
```

不持久化整个 emailStore.filter——只同步这一个字段，避免引入新的持久化复杂度。

## 3. selectedEmails 清空时机

在 InboxPage 增加一个 effect：

```typescript
useEffect(() => {
  setSelectedEmails([])
}, [filter, page])
```

`filter` 是对象引用，`setFilter` 总是创建新对象，因此任何 filter 变更都会触发清空。

## 4. cleanHtmlContent 替换

### 当前问题

130 行正则 hack，移除 "row"、"text"、"color"、"content"、"header" 等常见英文词。

### 替换方案

```typescript
function getSnippet(email: Email): string {
  // 后端 snippet 已是纯文本，直接使用；fallback 剥离 HTML 标签
  const raw = email.snippet || ''
  return raw.replace(/<[^>]*>/g, '').trim().slice(0, 50)
}
```

先确认后端 `snippet` 字段是否已是纯文本（如果是则只需截断），再决定是否需要 `replace`。

## 5. 组件提取设计

### EmailToolbar

```typescript
interface EmailToolbarProps {
  selectedCount: number
  total: number
  filterType: 'all' | 'unread'
  isTrashView: boolean
  activeGroupFilter: { id: number; name: string } | null
  onSelectAll: () => void
  onFilterChange: (type: 'all' | 'unread') => void
  onMarkAsRead: () => void
  onMarkAsUnread: () => void
  onToggleStar: () => void
  onArchive: () => void
  onDelete: () => void
  onRestore: () => void
  onPermanentDelete: () => void
  onEmptyTrash: () => void
  onMarkAllAsRead: () => void
  onRefresh: () => void
  onClearGroupFilter: () => void
  isLoading: boolean
}
```

### EmailPagination

```typescript
interface EmailPaginationProps {
  page: number
  totalPages: number
  total: number
  onPrev: () => void
  onNext: () => void
}
```

### ConfirmDialog

```typescript
interface ConfirmDialogProps {
  open: boolean
  title: string
  description: string
  confirmText: string
  cancelText?: string
  variant?: 'default' | 'destructive'
  isLoading?: boolean
  onConfirm: () => void
  onCancel: () => void
}
```

## 6. SentPage 适配 EmailList

SentEmail 与 Email 类型不同（有 status/error_message，无 is_read/is_starred/snippet）。通过 `emailType` prop 区分：

```typescript
interface EmailListProps<T = Email> {
  emails: T[]
  emailType?: 'inbox' | 'sent'
  // ...其他 props 不变
}
```

EmailItem 内部根据 `emailType` 决定渲染哪些字段：
- inbox：发件人 + 主题 + 摘要 + 星标 + 已读状态
- sent：收件人 + 主题 + 状态徽标 + 发送时间

## 7. useEmailActions hook

```typescript
function useEmailActions() {
  const [dialogs, setDialogs] = useState({
    markAllRead: false,
    delete: false,
    permanentDelete: false,
    emptyTrash: false,
  })
  const [isDeleting, setIsDeleting] = useState(false)

  const openDialog = (key: keyof typeof dialogs) =>
    setDialogs(prev => ({ ...prev, [key]: true }))
  const closeDialog = (key: keyof typeof dialogs) =>
    setDialogs(prev => ({ ...prev, [key]: false }))

  return { dialogs, openDialog, closeDialog, isDeleting, setIsDeleting }
}
```

## 8. 回滚

- 阶段一（bug fix）：按提交粒度回滚
- 阶段二（重构）：还原 InboxPage/SentPage 到提取前版本；新组件文件可直接删除