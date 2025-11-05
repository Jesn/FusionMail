# 筛选条件变化导致的重复请求修复

## 🔍 问题描述

在 `/inbox` 页面，第一次点击左侧菜单的"文件夹"或"邮箱账户"时，会触发重复的 API 请求。

### 复现步骤
1. 访问 `/inbox` 页面
2. 点击左侧"收件箱"、"星标邮件"等文件夹
3. 或点击某个邮箱账户
4. 观察网络请求：`GET /api/v1/emails` 被请求 2 次

## 🎯 根本原因

这个问题与 StrictMode 无关，而是由于 **useEffect 依赖链导致的重复触发**：

### 问题代码

```typescript
// useEmails.ts
const loadEmails = useCallback(async () => {
  // 加载邮件...
}, [filter, searchQuery, page, pageSize, ...]); // ← loadEmails 依赖 filter

useEffect(() => {
  loadEmails(); // ← useEffect 依赖 loadEmails
}, [loadEmails]);
```

### 触发链路

```
用户点击菜单
  ↓
setFilter(newFilter)  // 更新 filter 状态
  ↓
filter 变化
  ↓
loadEmails 重新创建（因为依赖 filter）
  ↓
useEffect 检测到 loadEmails 变化
  ↓
执行 loadEmails()  ← 第 1 次请求
  ↓
同时，loadEmails 内部也检测到 filter 变化
  ↓
再次执行请求  ← 第 2 次请求
```

## ✅ 解决方案

### 方案：增强请求去重逻辑

不仅检查是否正在请求，还要检查请求参数是否相同。

**修改前**：
```typescript
const fetchingRef = useRef(false);

const loadEmails = useCallback(async () => {
  if (fetchingRef.current) return; // 只检查是否正在请求
  
  try {
    fetchingRef.current = true;
    // 发送请求...
  } finally {
    fetchingRef.current = false;
  }
}, [filter, searchQuery, page, pageSize]);
```

**修改后**：
```typescript
const fetchingRef = useRef(false);
const lastRequestRef = useRef<string>(''); // ← 记录上次请求参数

const loadEmails = useCallback(async () => {
  // 生成请求参数的唯一标识
  const requestKey = JSON.stringify({ filter, searchQuery, page, pageSize });
  
  // 如果正在请求相同的数据，直接返回
  if (fetchingRef.current && lastRequestRef.current === requestKey) {
    return;
  }

  try {
    fetchingRef.current = true;
    lastRequestRef.current = requestKey; // ← 保存请求参数
    // 发送请求...
  } finally {
    fetchingRef.current = false;
  }
}, [filter, searchQuery, page, pageSize]);
```

### useEffect 依赖优化

**修改前**：
```typescript
useEffect(() => {
  loadEmails();
}, [loadEmails]); // 依赖函数，函数变化就触发
```

**修改后**：
```typescript
useEffect(() => {
  loadEmails();
  // eslint-disable-next-line react-hooks/exhaustive-deps
}, [filter, searchQuery, page, pageSize]); // 依赖实际数据
```

## 📊 修复效果

### 修复前
```
点击"星标邮件"
  ↓
filter 变化
  ↓
loadEmails 重新创建
  ↓
useEffect 触发 → GET /api/v1/emails (第 1 次)
  ↓
loadEmails 内部检测到 filter 变化 → GET /api/v1/emails (第 2 次)
```

### 修复后
```
点击"星标邮件"
  ↓
filter 变化
  ↓
loadEmails 重新创建
  ↓
useEffect 触发 → 检查请求参数
  ↓
生成 requestKey: {"filter":{"is_starred":true},"page":1,...}
  ↓
检查：fetchingRef.current = false，lastRequestRef.current = ""
  ↓
开始请求 → GET /api/v1/emails (第 1 次)
  ↓
loadEmails 再次被调用
  ↓
检查：fetchingRef.current = true，lastRequestRef.current = 相同
  ↓
直接返回，不发送请求 ✅
```

## 🎓 技术要点

### 1. 请求去重的两个维度

**时间维度**：
- 使用 `fetchingRef` 防止并发请求
- 确保同一时间只有一个请求在进行

**参数维度**：
- 使用 `lastRequestRef` 记录请求参数
- 防止相同参数的重复请求

### 2. useEffect 依赖最佳实践

**❌ 不好的做法**：
```typescript
useEffect(() => {
  fetchData();
}, [fetchData]); // 依赖函数
```

**✅ 好的做法**：
```typescript
useEffect(() => {
  fetchData();
  // eslint-disable-next-line react-hooks/exhaustive-deps
}, [actualData1, actualData2]); // 依赖实际数据
```

### 3. 为什么需要 eslint-disable-next-line？

React Hooks 的 ESLint 规则要求 useEffect 包含所有使用的依赖。但在某些情况下，我们明确知道：
- 函数是稳定的（使用 useCallback）
- 函数内部已经处理了数据变化
- 我们只想在特定数据变化时触发

这时可以安全地禁用 ESLint 警告。

## 🔍 其他可能的解决方案

### 方案 B：使用防抖

```typescript
import { useMemo } from 'react';
import { debounce } from 'lodash';

const debouncedLoadEmails = useMemo(
  () => debounce(loadEmails, 300),
  [loadEmails]
);

useEffect(() => {
  debouncedLoadEmails();
}, [debouncedLoadEmails]);
```

**优点**：
- 可以处理快速连续的点击
- 减少不必要的请求

**缺点**：
- 增加了延迟（300ms）
- 需要额外的依赖（lodash）
- 用户体验可能不如立即响应

### 方案 C：使用 React Query

```typescript
import { useQuery } from '@tanstack/react-query';

const { data: emails } = useQuery({
  queryKey: ['emails', filter, searchQuery, page, pageSize],
  queryFn: () => emailService.getList(filter, { page, page_size: pageSize }),
  staleTime: 5 * 60 * 1000, // 5 分钟内不重新请求
});
```

**优点**：
- 自动处理请求去重
- 自动缓存
- 自动重试和刷新

**缺点**：
- 需要引入新依赖
- 需要重构现有代码

## 📝 总结

### 问题类型
- **StrictMode 重复请求**：开发环境特性，useEffect 执行两次
- **筛选变化重复请求**：useEffect 依赖链导致的重复触发

### 解决策略
- **StrictMode 问题**：使用 `fetchingRef` 防止并发请求
- **筛选变化问题**：使用 `lastRequestRef` 防止相同参数的重复请求

### 最佳实践
1. ✅ 请求去重要考虑时间和参数两个维度
2. ✅ useEffect 应该依赖实际数据，而不是函数
3. ✅ 使用 useCallback 稳定函数引用
4. ✅ 合理使用 eslint-disable 注释

---

**修复时间**：2025-11-05  
**问题类型**：useEffect 依赖链导致的重复触发  
**解决方案**：增强请求去重逻辑 + 优化 useEffect 依赖
