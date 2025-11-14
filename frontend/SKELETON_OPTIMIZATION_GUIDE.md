# 骨架屏优化指南

## 📋 概述

骨架屏（Skeleton Screen）是一种提升加载体验的优化技术，通过显示内容占位符而非空白屏幕，让用户感知页面正在加载，减少等待焦虑。

## ✅ 已完成的优化

### 1. 邮件列表 (`EmailList.tsx`)
- ✅ 添加了骨架屏占位符 (`SkeletonItem`)
- ✅ 优化了加载状态处理
- ✅ 添加平滑过渡动画
- ✅ 添加 CSS `contain: strict` 限制重绘
- ✅ 翻页时显示半透明加载指示器

### 2. 规则列表 (`RuleList.tsx`)
- ✅ 创建了 `SkeletonRuleCard` 组件
- ✅ 首次加载时显示3个骨架卡
- ✅ 避免布局跳跃

### 3. Webhook列表 (`WebhookList.tsx`)
- ✅ 使用 `SkeletonCardGrid` 组件
- ✅ 网格布局骨架屏
- ✅ 支持自定义列数

### 4. 账户列表 (`AccountList.tsx`)
- ✅ 创建了 `SkeletonAccountCardDetailed/Compact/Minimal` 组件
- ✅ 支持三种密度模式（minimal/compact/detailed）
- ✅ 根据密度动态选择骨架屏
- ✅ 首次加载时显示5个骨架卡

## 🎯 待优化的组件

### 高优先级

#### 1. 邮件详情页 (`EmailDetailPage`)
**文件**: `src/pages/EmailDetailPage.tsx`

**优化方案**:
```tsx
import { SkeletonEmailDetail } from '../components/ui/skeleton';

if (isLoadingDetail) {
  return <SkeletonEmailDetail />;
}
```

#### 3. 搜索结果页 (`SearchPage`)
**文件**: `src/pages/SearchPage.tsx`

**优化方案**:
```tsx
import { SkeletonList } from '../components/ui/skeleton';

if (isLoading) {
  return (
    <div className="flex-1 overflow-hidden">
      <EmailList
        emails={emails}
        isLoading={true}
        // ... 其他props
      />
    </div>
  );
}
```

### 中优先级

#### 4. 仪表盘页面 (`DashboardPage`)
**文件**: `src/pages/DashboardPage.tsx`

**优化方案**:
```tsx
import { Skeleton } from '../components/ui/skeleton';

const SkeletonStatCard = () => (
  <Card>
    <CardHeader className="pb-2">
      <Skeleton className="h-4 w-24" />
    </CardHeader>
    <CardContent>
      <Skeleton className="h-8 w-16 mb-2" />
      <Skeleton className="h-3 w-32" />
    </CardContent>
  </Card>
);
```

#### 5. 设置页面 (`SettingsPage`)
**文件**: `src/pages/SettingsPage.tsx`

**优化方案**:
```tsx
import { Skeleton } from '../components/ui/skeleton';

const SkeletonFormField = () => (
  <div className="space-y-2">
    <Skeleton className="h-4 w-24" />
    <Skeleton className="h-10 w-full" />
  </div>
);
```

## 🛠️ 通用骨架屏组件

### 已创建的组件 (`src/components/ui/skeleton.tsx`)

1. **Skeleton** - 基础骨架元素
2. **SkeletonRuleCard** - 规则卡片骨架屏
3. **SkeletonWebhookCard** - Webhook卡片骨架屏
4. **SkeletonAccountCardDetailed** - 账户卡片骨架屏（详细）
5. **SkeletonAccountCardCompact** - 账户卡片骨架屏（紧凑）
6. **SkeletonEmailDetail** - 邮件详情骨架屏
7. **SkeletonList** - 通用列表骨架屏
8. **SkeletonCardGrid** - 卡片网格骨架屏

## 📐 使用模式

### 模式1: 首次加载显示骨架屏
```tsx
if (isLoading && items.length === 0) {
  return <SkeletonComponent count={3} />;
}
```

### 模式2: 翻页时覆盖层
```tsx
{isLoading && items.length > 0 && (
  <div className="absolute top-2 right-2 z-10">
    <LoadingIndicator />
  </div>
)}
```

### 模式3: 表单加载骨架
```tsx
{isLoading ? (
  <SkeletonForm />
) : (
  <ActualForm />
)}
```

## 🎨 样式规范

骨架屏使用 `bg-muted` 类，支持明暗主题自动切换：

```css
/* Tailwind CSS */
.bg-muted {
  background-color: hsl(var(--muted));
}

/* CSS 变量定义 */
:root {
  --muted: 210 40% 96.1%;
  --muted-foreground: 215.4 16.3% 46.9%;
}

.dark {
  --muted: 222.2 84% 4.9%;
  --muted-foreground: 215 20.2% 65.1%;
}
```

动画效果:
```css
.animate-pulse {
  animation: pulse 2s cubic-bezier(0.4, 0, 0.6, 1) infinite;
}

@keyframes pulse {
  0%, 100% {
    opacity: 1;
  }
  50% {
    opacity: .5;
  }
}
```

## 📊 性能优化建议

1. **避免动画卡顿**
   - 使用 `transform` 而非改变 `width/height`
   - 限制 `will-change` 属性的使用

2. **合理设置骨架数量**
   - 首屏显示 3-6 个骨架元素
   - 避免过多骨架影响性能

3. **条件渲染**
   ```tsx
   // 好
   {isLoading && <Skeleton />}

   // 避免 - 会导致DOM频繁创建销毁
   {isLoading ? <Skeleton /> : <ActualContent />}
   ```

4. **使用 React.memo**
   ```tsx
   const SkeletonItem = React.memo(() => (
     <div className="animate-pulse">...</div>
   ));
   ```

## 🚀 实施步骤

### 步骤1: 添加基础组件 ✅
- [x] 创建 `src/components/ui/skeleton.tsx`
- [x] 定义通用骨架屏组件
- [x] 支持多种骨架屏类型（卡片、列表、详情等）

### 步骤2: 优化列表页面 ✅
- [x] 邮件列表 (`EmailList.tsx`) - 虚拟滚动优化 + 骨架屏
- [x] 规则列表 (`RuleList.tsx`) - 卡片骨架屏
- [x] Webhook列表 (`WebhookList.tsx`) - 网格骨架屏
- [x] 账户列表 (`AccountList.tsx`) - 多密度模式骨架屏

### 步骤3: 优化详情页面
- [ ] 邮件详情页 (`EmailDetailPage.tsx`)
- [ ] 账户详情页

### 步骤4: 优化设置页面
- [ ] 设置页面 (`SettingsPage.tsx`)
- [ ] API密钥页面
- [ ] 账户设置页面

### 步骤5: 优化其他页面
- [ ] 仪表盘 (`DashboardPage.tsx`)
- [ ] 搜索结果页 (`SearchPage.tsx`)

## 📈 预期效果

1. **提升感知性能** - 用户看到骨架屏而非空白页面
2. **减少布局跳跃** - 骨架屏高度与实际内容一致
3. **降低跳出率** - 改善加载体验，减少用户等待焦虑
4. **一致性** - 全局统一的加载体验

## 🔍 测试要点

1. 首次加载是否显示骨架屏
2. 翻页时是否有加载指示器
3. 骨架屏高度是否与实际内容匹配
4. 明暗主题下骨架屏颜色是否正确
5. 动画是否流畅，无卡顿

## 📝 注意事项

1. 骨架屏应尽可能模拟真实内容布局
2. 避免骨架屏与实际内容差异过大
3. 加载完成后骨架屏应平滑过渡到实际内容
4. 骨架屏动画不宜过于复杂，避免消耗性能
5. 移动端注意骨架屏的响应式适配

---

**更新日期**: 2025-11-13
**版本**: v1.0
**负责人**: 前端团队
