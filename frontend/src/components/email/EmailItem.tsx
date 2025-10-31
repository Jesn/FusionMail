import { Star, Paperclip, Mail } from 'lucide-react';
import { Email, Account } from '../../types';
import { cn } from '../../lib/utils';
import { formatDistanceToNow } from 'date-fns';
import { zhCN } from 'date-fns/locale';
import { Badge } from '../ui/badge';

interface EmailItemProps {
  email: Email;
  isSelected: boolean;
  onClick: () => void;
  showAccountBadge?: boolean;
  accounts?: Account[];
}

export const EmailItem = ({ email, isSelected, onClick, showAccountBadge = false, accounts = [] }: EmailItemProps) => {
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

  // 获取邮箱账户信息
  const getAccountInfo = () => {
    if (!showAccountBadge) return null;
    
    const account = accounts.find(acc => acc.uid === email.account_uid);
    if (!account) return null;
    
    // 提取邮箱的用户名部分作为简短标识
    const emailParts = account.email.split('@');
    const username = emailParts[0];
    const domain = emailParts[1];
    
    return {
      email: account.email,
      shortName: username,
      domain: domain,
    };
  };

  const accountInfo = getAccountInfo();

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

      {/* 右侧：时间和邮箱标识 */}
      <div className="flex flex-col items-end gap-1 flex-shrink-0">
        {/* 邮箱标识 - 只在显示所有邮箱时显示 */}
        {accountInfo && (
          <Badge 
            variant="secondary" 
            className="text-xs px-1.5 py-0 h-4 bg-muted/30 text-muted-foreground border-0 font-normal"
            title={accountInfo.email}
          >
            <Mail className="w-2.5 h-2.5 mr-0.5" />
            {accountInfo.shortName}
          </Badge>
        )}
        
        {/* 时间 */}
        <div className="text-xs text-muted-foreground">
          {formatDate(email.sent_at)}
        </div>
      </div>
    </div>
  );
};
