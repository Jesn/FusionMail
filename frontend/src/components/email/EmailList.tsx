import { useVirtualizer } from '@tanstack/react-virtual';
import { useRef } from 'react';
import { EmailItem } from './EmailItem';
import { Email } from '../../types';
import { Loader2 } from 'lucide-react';

interface EmailListProps {
  emails: Email[];
  selectedEmailId?: number;
  onEmailClick: (email: Email) => void;
  isLoading?: boolean;
}

export const EmailList = ({
  emails,
  selectedEmailId,
  onEmailClick,
  isLoading,
}: EmailListProps) => {
  const parentRef = useRef<HTMLDivElement>(null);

  // 虚拟滚动配置
  const virtualizer = useVirtualizer({
    count: emails.length,
    getScrollElement: () => parentRef.current,
    estimateSize: () => 80, // 每个邮件项的估计高度
    overscan: 5, // 预渲染的项数
  });

  if (isLoading) {
    return (
      <div className="flex h-full items-center justify-center">
        <Loader2 className="h-8 w-8 animate-spin text-muted-foreground" />
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

  return (
    <div ref={parentRef} className="h-full overflow-auto">
      <div
        style={{
          height: `${virtualizer.getTotalSize()}px`,
          width: '100%',
          position: 'relative',
        }}
      >
        {virtualizer.getVirtualItems().map((virtualItem) => {
          const email = emails[virtualItem.index];
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
              }}
            >
              <EmailItem
                email={email}
                isSelected={email.id === selectedEmailId}
                onClick={() => onEmailClick(email)}
              />
            </div>
          );
        })}
      </div>
    </div>
  );
};
