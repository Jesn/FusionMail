# Inbox 页面状态优化

## Goal

修复 InboxPage 的状态同步 bug 和死代码，拆分 612 行上帝组件，消除与 SentPage 的代码重复，使收件箱主路径状态可靠、交互一致。

## Background

InboxPage 同时承载收件箱/星标/归档/回收站四种视图，存在已确认的状态同步缺陷（groupStore 持久化但 emailStore 不持久化导致刷新不同步）、死菜单项、摘要渲染 hack（cleanHtmlContent 移除常见英文词）、以及与 SentPage 的复制粘贴级重复。

分析依据：3 个并行代码探索 agent 的调查报告（useEmails hook 状态流、EmailList 组件渲染、Sidebar/MainLayout 布局上下文）。

## Requirements

### R1. 状态同步修复

- 页面刷新后侧边栏分组高亮与 InboxPage filter 一致
- 切换 filter 或翻页后 selectedEmails 自动清空
- toggleStar 成功后乐观更新 starredCount
- batchPermanentDelete 使用 removeEmails 批量删除而非逐个 removeEmail

### R2. 死代码清理

- 下拉菜单"选择全部"/"取消选择"绑定 onClick 或移除
- EmailItem 移除从未传入的 onMarkSpam prop 和垃圾邮件按钮
- EmailList 移除声明但未使用的 highlightQuery prop

### R3. 摘要渲染修复

- 替换 cleanHtmlContent 130 行正则 hack
- 邮件摘要不再出现因移除常见英文词导致的乱码

### R4. 组件重构

- 提取 EmailToolbar 共享组件，InboxPage 和 SentPage 复用
- 提取 EmailPagination 共享组件
- 提取 ConfirmDialog 通用确认弹窗组件
- SentPage 改用 EmailList 获得虚拟化
- InboxPage dialog 状态收敛到 useEmailActions hook

### R5. 不做

- 不重构 filter-based 文件夹系统为路由-based
- 不统一 InboxPage/SentPage 的详情查看模式
- 不添加错误状态 UI
- 不改后端 API

## Acceptance Criteria

- [ ] AC1：页面刷新后侧边栏分组高亮与列表 filter 一致
- [ ] AC2：切换 filter/翻页后 selectedEmails 清空，Badge 计数归零
- [ ] AC3：下拉菜单所有项可点击且有效
- [ ] AC4：邮件摘要不包含因正则 hack 导致的乱码
- [ ] AC5：toggleStar 后星标计数同步更新
- [ ] AC6：EmailItem/EmailList 无死代码 prop
- [ ] AC7：InboxPage 和 SentPage 共享 EmailToolbar/EmailPagination/ConfirmDialog
- [ ] AC8：SentPage 使用 EmailList 虚拟化
- [ ] AC9：`npm test` 全部通过
- [ ] AC10：`npm run build` 成功

## Out of Scope

- filter-based 文件夹系统改为路由-based
- InboxPage/SentPage 详情查看模式统一
- 错误状态 UI
- 后端 API 变更
- 两套计数系统（emailStore vs groupStore）的统一

## Risks

| 风险 | 缓解 |
|------|------|
| SentPage 适配 EmailList 类型差异大 | 通过 prop 适配层而非泛型，降低类型复杂度 |
| 重构后现有 36 个测试可能断裂 | 逐项重构，每次改动后跑 `npm test` |
| useEmailActions hook 提取后行为变化 | 保持纯状态提取，不改变业务逻辑 |