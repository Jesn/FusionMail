import { cn } from '../../lib/utils';

/**
 * Skeleton component - 基础骨架屏元素
 *
 * @param className - 自定义CSS类名
 * @param props - 其他HTML属性
 *
 * @example
 * <Skeleton className="h-4 w-32" />
 */
interface SkeletonProps extends React.HTMLAttributes<HTMLDivElement> {}

export function Skeleton({ className, ...props }: SkeletonProps) {
  return (
    <div
      className={cn('animate-pulse rounded-md bg-muted', className)}
      {...props}
    />
  );
}

/**
 * SkeletonRuleCard - 规则卡片骨架屏
 *
 * 用于规则列表页面首次加载时显示占位符
 * 模拟RuleCard的布局和结构
 *
 * @example
 * <div className="space-y-4">
 *   {Array.from({ length: 3 }).map((_, i) => (
 *     <SkeletonRuleCard key={i} />
 *   ))}
 * </div>
 */
export function SkeletonRuleCard() {
  return (
    <div className="space-y-3 rounded-lg border p-4">
      <div className="flex items-start justify-between">
        <div className="space-y-2 flex-1">
          <Skeleton className="h-5 w-48" />
          <div className="flex gap-2">
            <Skeleton className="h-5 w-16" />
            <Skeleton className="h-5 w-16" />
          </div>
        </div>
        <div className="flex gap-2">
          <Skeleton className="h-8 w-8 rounded" />
          <Skeleton className="h-8 w-8 rounded" />
          <Skeleton className="h-8 w-8 rounded" />
        </div>
      </div>
      {Array.from({ length: 3 }).map((_, i) => (
        <div key={i} className="space-y-2">
          <Skeleton className="h-4 w-24" />
          <Skeleton className="h-6 w-full" />
        </div>
      ))}
    </div>
  );
}

// Webhook卡片骨架屏
export function SkeletonWebhookCard() {
  return (
    <div className="rounded-lg border p-4 space-y-3">
      <div className="flex items-start justify-between">
        <div className="space-y-2 flex-1">
          <Skeleton className="h-6 w-40" />
          <Skeleton className="h-4 w-24" />
        </div>
        <Skeleton className="h-8 w-8 rounded" />
      </div>
      <div className="flex items-center gap-2">
        <Skeleton className="h-5 w-12 rounded" />
        <Skeleton className="h-4 w-48" />
      </div>
      <div className="flex items-center gap-2">
        <Skeleton className="h-4 w-12" />
        <Skeleton className="h-5 w-20 rounded" />
        <Skeleton className="h-5 w-16 rounded" />
      </div>
      <div className="grid grid-cols-3 gap-4 pt-2 border-t">
        <div className="text-center space-y-1">
          <Skeleton className="h-6 w-8 mx-auto" />
          <Skeleton className="h-3 w-12 mx-auto" />
        </div>
        <div className="text-center space-y-1">
          <Skeleton className="h-6 w-8 mx-auto" />
          <Skeleton className="h-3 w-12 mx-auto" />
        </div>
        <div className="text-center space-y-1">
          <Skeleton className="h-4 w-16 mx-auto" />
          <Skeleton className="h-3 w-12 mx-auto" />
        </div>
      </div>
    </div>
  );
}

// 账户卡片骨架屏（详细模式）
export function SkeletonAccountCardDetailed() {
  return (
    <div className="rounded-lg border p-4 space-y-4">
      <div className="flex items-start justify-between">
        <div className="flex items-center gap-3">
          <Skeleton className="h-12 w-12 rounded-full" />
          <div className="space-y-2">
            <Skeleton className="h-5 w-48" />
            <Skeleton className="h-4 w-32" />
          </div>
        </div>
        <div className="flex gap-1">
          <Skeleton className="h-8 w-8 rounded" />
          <Skeleton className="h-8 w-8 rounded" />
          <Skeleton className="h-8 w-8 rounded" />
          <Skeleton className="h-8 w-8 rounded" />
        </div>
      </div>
      {Array.from({ length: 5 }).map((_, i) => (
        <div key={i} className="flex items-center justify-between text-sm">
          <Skeleton className="h-4 w-24" />
          <Skeleton className="h-4 w-32" />
        </div>
      ))}
    </div>
  );
}

// 账户卡片骨架屏（紧凑模式）
export function SkeletonAccountCardCompact() {
  return (
    <div className="rounded-lg border p-4">
      <div className="flex items-center justify-between">
        <div className="flex items-center gap-3 flex-1">
          <Skeleton className="h-10 w-10 rounded-full" />
          <div className="flex-1 space-y-2">
            <Skeleton className="h-4 w-48" />
            <Skeleton className="h-3 w-64" />
          </div>
        </div>
        <div className="flex gap-1">
          <Skeleton className="h-8 w-8 rounded" />
          <Skeleton className="h-8 w-8 rounded" />
          <Skeleton className="h-8 w-8 rounded" />
          <Skeleton className="h-8 w-8 rounded" />
        </div>
      </div>
    </div>
  );
}

// 账户卡片骨架屏（极简模式）
export function SkeletonAccountCardMinimal() {
  return (
    <div className="rounded-lg border p-3">
      <div className="flex items-center justify-between">
        <div className="flex items-center gap-3 flex-1 min-w-0">
          <Skeleton className="h-4 w-4 rounded" />
          <Skeleton className="h-4 w-4 rounded" />
          <Skeleton className="h-4 w-48" />
          <Skeleton className="h-5 w-16 rounded" />
        </div>
        <div className="flex gap-1">
          <Skeleton className="h-7 w-7 rounded" />
          <Skeleton className="h-7 w-7 rounded" />
          <Skeleton className="h-7 w-7 rounded" />
        </div>
      </div>
    </div>
  );
}

// 邮件详情骨架屏
export function SkeletonEmailDetail() {
  return (
    <div data-testid="email-detail-skeleton" className="space-y-6">
      {/* 头部工具栏 */}
      <div className="flex items-center justify-between">
        <Skeleton className="h-8 w-20" />
        <div className="flex gap-2">
          <Skeleton className="h-8 w-8 rounded" />
          <Skeleton className="h-8 w-8 rounded" />
          <Skeleton className="h-8 w-8 rounded" />
        </div>
      </div>

      {/* 邮件头部信息 */}
      <div className="space-y-4 rounded-lg border p-6">
        <Skeleton className="h-6 w-3/4" />
        <div className="grid grid-cols-2 gap-4">
          <div className="space-y-2">
            <Skeleton className="h-4 w-16" />
            <Skeleton className="h-4 w-48" />
          </div>
          <div className="space-y-2">
            <Skeleton className="h-4 w-16" />
            <Skeleton className="h-4 w-48" />
          </div>
        </div>
        <Skeleton className="h-4 w-32" />
      </div>

      {/* 邮件正文 */}
      <div className="space-y-3">
        {Array.from({ length: 12 }).map((_, i) => (
          <Skeleton
            key={i}
            className="h-4"
            style={{ width: `${85 - i * 3}%` }}
          />
        ))}
      </div>

      {/* 附件 */}
      <div className="space-y-2">
        <Skeleton className="h-5 w-16" />
        <div className="flex gap-2">
          <Skeleton className="h-20 w-32 rounded" />
          <Skeleton className="h-20 w-32 rounded" />
        </div>
      </div>
    </div>
  );
}

// 通用列表骨架屏
export function SkeletonList({
  count = 3,
  itemHeight = 80,
  className,
}: {
  count?: number;
  itemHeight?: number;
  className?: string;
}) {
  return (
    <div className={cn('space-y-3', className)}>
      {Array.from({ length: count }).map((_, i) => (
        <div
          key={i}
          className="flex items-center gap-3 p-4 border rounded-lg"
        >
          <Skeleton className="h-4 w-4 rounded" />
          <div className="flex-1 space-y-2">
            <Skeleton className="h-4 w-1/3" />
            <Skeleton className="h-3 w-full" />
          </div>
          <Skeleton className="h-4 w-16" />
        </div>
      ))}
    </div>
  );
}

// 卡片网格骨架屏
export function SkeletonCardGrid({
  count = 6,
  columns = 3,
}: {
  count?: number;
  columns?: number;
}) {
  const gridClass = {
    2: 'grid-cols-1 md:grid-cols-2',
    3: 'grid-cols-1 md:grid-cols-2 lg:grid-cols-3',
    4: 'grid-cols-1 md:grid-cols-2 lg:grid-cols-4',
  }[columns] || 'grid-cols-3';

  return (
    <div className={`grid gap-4 ${gridClass}`}>
      {Array.from({ length: count }).map((_, i) => (
        <div key={i} className="rounded-lg border p-4 space-y-3">
          <div className="flex items-start justify-between">
            <Skeleton className="h-6 w-40" />
            <Skeleton className="h-8 w-8 rounded" />
          </div>
          <Skeleton className="h-4 w-full" />
          <Skeleton className="h-4 w-3/4" />
          <div className="grid grid-cols-3 gap-4 pt-2 border-t">
            <Skeleton className="h-8" />
            <Skeleton className="h-8" />
            <Skeleton className="h-8" />
          </div>
        </div>
      ))}
    </div>
  );
}
