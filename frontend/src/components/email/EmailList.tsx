import { useVirtualizer } from '@tanstack/react-virtual';
import { useRef } from 'react';
import { EmailItem } from './EmailItem';
import { Email, Account } from '../../types';
import { Loader2 } from 'lucide-react';

interface EmailListProps {
  emails: Email[];
  selectedEmailId?: number;
  onEmailClick: (email: Email) => void;
  isLoading?: boolean;
  highlightQuery?: string;
  showAccountBadge?: boolean;
  accounts?: Account[];
  onToggleStar?: (email: Email) => void;
}

export const EmailList = ({
  emails,
  selectedEmailId,
  onEmailClick,
  isLoading,
  highlightQuery: _highlightQuery,
  showAccountBadge = false,
  accounts = [],
  onToggleStar,

}: EmailListProps) => {
  const parentRef = useRef<HTMLDivElement>(null);

  // 虚拟滚动配置
  const virtualizer = useVirtualizer({
    count: emails.length,
    getScrollElement: () => parentRef.current,
    estimateSize: () => 80, // 每个邮件项的估计高度
    overscan: 5, // 预渲染的项数
  });

  // 骨架屏组件
  const SkeletonItem = () => (
    <div className="flex items-start gap-3 border-b px-4 py-3 animate-pulse">
      <div className="mt-1 h-4 w-4 rounded bg-muted" />
      <div className="min-w-0 flex-1 space-y-2">
        <div className="flex items-center gap-2">
          <div className="h-4 w-32 rounded bg-muted" />
          <div className="h-3 w-3 rounded bg-muted" />
        </div>
        <div className="h-4 w-3/4 rounded bg-muted" />
        <div className="h-3 w-full rounded bg-muted" />
      </div>
      <div className="flex flex-col items-end gap-1">
        <div className="h-4 w-16 rounded bg-muted" />
        <div className="h-3 w-12 rounded bg-muted" />
      </div>
    </div>
  );

  if (isLoading && emails.length === 0) {
    return (
      <div className="flex h-full flex-col items-center justify-center text-muted-foreground">
        <Loader2 className="h-8 w-8 animate-spin text-muted-foreground" />
        <p className="mt-2 text-sm">正在加载邮件...</p>
      </div>
    );
  }

  if (emails.length === 0) {
    return (
      <div className="flex h-full flex-col items-center justify-center text-muted-foreground">
        <p className="text-lg font-medium">没有邮件</p>
        <p className="text-sm">此文件夹中暂无邮件</p>
      </div>
    );
  }

  // 如果正在加载且已有数据，显示加载指示器和骨架屏
  const showLoadingOverlay = isLoading && emails.length > 0;

  return (
    <div
      ref={parentRef}
      data-testid="email-list"
      className="h-full overflow-auto relative"
      style={{
        contain: 'strict', // 限制重绘范围，减少抖动
      }}
    >
      {/* 加载覆盖层 - 在翻页时显示，不影响布局 */}
      {showLoadingOverlay && (
        <div className="absolute top-2 right-2 z-10">
          <div className="flex items-center gap-2 rounded-md bg-background/80 px-3 py-1.5 backdrop-blur-sm border shadow-sm">
            <Loader2 className="h-3 w-3 animate-spin text-muted-foreground" />
            <span className="text-xs text-muted-foreground">加载中...</span>
          </div>
        </div>
      )}
      <div
        style={{
          height: `${virtualizer.getTotalSize()}px`,
          width: '100%',
          position: 'relative',
          transition: 'height 0.2s ease-out', // 添加高度变化的平滑过渡
        }}
      >
        {virtualizer.getVirtualItems().map((virtualItem) => {
          const email = emails[virtualItem.index];

          // 如果正在加载新数据，显示骨架屏而不是实际内容
          const isLoadingItem = isLoading && !email;

          return (
            <div
              key={virtualItem.key}
              style={{
                position: 'absolute',
                top: 0,
                left: 0,
                width: '100%',
                height: `${virtualItem.size}px`,
                transform: `translateY(${virtualItem.start}px)`,
                transition: 'transform 0.15s ease-out',
              }}
            >
              {isLoadingItem ? (
                <SkeletonItem />
              ) : (
                <EmailItem
                  email={email}
                  isSelected={email.id === selectedEmailId}
                  onClick={() => onEmailClick(email)}
                  showAccountBadge={showAccountBadge}
                  accounts={accounts}
                  onToggleStar={onToggleStar}

                />
              )}
            </div>
          );
        })}
      </div>
    </div>
  );
};
