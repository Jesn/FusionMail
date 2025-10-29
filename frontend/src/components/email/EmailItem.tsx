import { Star, Paperclip } from 'lucide-react';
import { Email } from '../../types';
import { cn } from '../../lib/utils';
import { formatDistanceToNow } from 'date-fns';
import { zhCN } from 'date-fns/locale';

interface EmailItemProps {
  email: Email;
  isSelected: boolean;
  onClick: () => void;
}

export const EmailItem = ({ email, isSelected, onClick }: EmailItemProps) => {
  const formatDate = (dateString: string) => {
    try {
      return formatDistanceToNow(new Date(dateString), {
        addSuffix: true,
        locale: zhCN,
      });
    } catch {
      return dateString;
    }
  };

  return (
    <div
      className={cn(
        'flex cursor-pointer items-start gap-3 border-b px-4 py-3 transition-colors hover:bg-accent',
        isSelected && 'bg-accent',
        !email.is_read && 'bg-muted/50'
      )}
      onClick={onClick}
    >
      {/* 左侧：星标按钮 */}
      <button
        className="mt-1 flex-shrink-0"
        onClick={(e) => {
          e.stopPropagation();
          // TODO: 实现星标切换
        }}
      >
        <Star
          className={cn(
            'h-4 w-4',
            email.is_starred
              ? 'fill-yellow-400 text-yellow-400'
              : 'text-muted-foreground hover:text-yellow-400'
          )}
        />
      </button>

      {/* 中间：邮件信息 */}
      <div className="min-w-0 flex-1">
        {/* 发件人 */}
        <div className="flex items-center gap-2">
          <span
            className={cn(
              'truncate text-sm',
              !email.is_read ? 'font-semibold' : 'font-normal'
            )}
          >
            {email.from_name || email.from_address}
          </span>
          {email.has_attachments && (
            <Paperclip className="h-3 w-3 flex-shrink-0 text-muted-foreground" />
          )}
        </div>

        {/* 主题 */}
        <div
          className={cn(
            'truncate text-sm',
            !email.is_read ? 'font-medium' : 'text-muted-foreground'
          )}
        >
          {email.subject || '(无主题)'}
        </div>

        {/* 摘要 */}
        <div className="truncate text-xs text-muted-foreground">
          {email.snippet}
        </div>
      </div>

      {/* 右侧：时间 */}
      <div className="flex-shrink-0 text-xs text-muted-foreground">
        {formatDate(email.sent_at)}
      </div>
    </div>
  );
};
