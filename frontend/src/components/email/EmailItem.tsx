import { Star, Paperclip } from 'lucide-react';
import { Email, Account } from '../../types';
import { cn } from '../../lib/utils';
import { formatDistanceToNow } from 'date-fns';
import { zhCN } from 'date-fns/locale';
import { Badge } from '../ui/badge';
import { Checkbox } from '../ui/checkbox';

interface EmailItemProps {
  email: Email;
  isSelected: boolean;
  isChecked?: boolean;
  onClick: () => void;
  onCheckChange?: (checked: boolean) => void;
  onToggleStar?: (email: Email) => void;
  showAccountBadge?: boolean;
  accounts?: Account[];
  enableMultiSelect?: boolean;
}

export const EmailItem = ({
  email,
  isSelected,
  isChecked = false,
  onClick,
  onCheckChange,
  onToggleStar,
  showAccountBadge = false,
  accounts = [],
  enableMultiSelect = false,
}: EmailItemProps) => {
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

  // 从 snippet 提取纯文本摘要
  const getSnippet = (raw: string): string => {
    const text = raw.replace(/<[^>]*>/g, '').trim();
    if (!text) return '(邮件内容)';
    return text.length > 50 ? text.slice(0, 50) + '...' : text;
  };

  // 获取邮箱账户信息
  // 对于 WebAPI 账户，优先显示账户的邮箱地址，如果不是有效邮箱格式则尝试从邮件的 to_addresses 获取
  const getAccountInfo = () => {
    if (!showAccountBadge) return null;

    // 检查是否是有效的邮箱格式（包含 @ 符号）
    const isValidEmailFormat = (str: string) => str && str.includes('@') && str.indexOf('@') > 0;

    // 从 to_addresses 提取邮箱地址的辅助函数
    const extractEmailFromToAddresses = (): string | null => {
      if (!email.to_addresses) return null;

      try {
        // to_addresses 可能是 JSON 字符串数组或逗号分隔的字符串
        let toAddresses: string[] = [];
        if (email.to_addresses.startsWith('[')) {
          toAddresses = JSON.parse(email.to_addresses);
        } else {
          toAddresses = email.to_addresses.split(',').map(addr => addr.trim());
        }
        // 找到第一个有效的邮箱地址
        const validEmail = toAddresses.find(addr => isValidEmailFormat(addr));
        if (validEmail) {
          return validEmail;
        }
      } catch {
        // 解析失败，尝试直接使用
        if (email.to_addresses && !email.to_addresses.startsWith('[')) {
          const firstAddr = email.to_addresses.split(',')[0]?.trim();
          if (firstAddr && isValidEmailFormat(firstAddr)) {
            return firstAddr;
          }
        }
      }
      return null;
    };

    const account = accounts.find(acc => acc.uid === email.account_uid);

    // 如果找不到账户，尝试从 to_addresses 获取邮箱地址作为降级方案
    if (!account) {
      const fallbackEmail = extractEmailFromToAddresses();
      if (fallbackEmail) {
        return {
          email: fallbackEmail,
          fullEmail: fallbackEmail,
        };
      }
      return null;
    }

    // 检查账户邮箱是否是虚拟格式（自动生成的格式：service_type-uuid）
    // 例如：cloudflare_temp_email-a1b2c3d4、cloud_mail-e5f6g7h8
    const isVirtualEmail = /^(cloudflare_temp_email|cloud_mail|custom_webapi)-[a-f0-9]+$/.test(account.email);

    // 如果账户邮箱不是有效邮箱格式（可能是显示名称如 "Cloud Mail"）或是虚拟格式，
    // 尝试从邮件的 to_addresses 获取实际邮箱地址
    let displayEmail = account.email;
    const needFallback = isVirtualEmail || !isValidEmailFormat(account.email);

    if (needFallback) {
      const fallbackEmail = extractEmailFromToAddresses();
      if (fallbackEmail) {
        displayEmail = fallbackEmail;
      }
    }

    return {
      email: displayEmail,
      fullEmail: displayEmail, // 显示完整邮箱地址
    };
  };

  const accountInfo = getAccountInfo();

  return (
    <div
      data-testid="email-item"
      className={cn(
        'flex cursor-pointer items-start gap-3 border-b px-4 py-3 transition-colors hover:bg-accent',
        isSelected && 'bg-accent',
        !email.is_read && 'bg-muted/50'
      )}
      onClick={onClick}
    >
      {/* 左侧：复选框和星标 */}
      <div className="mt-1 flex-shrink-0 flex items-center gap-1">
        {enableMultiSelect && (
          <Checkbox
            checked={isChecked}
            onCheckedChange={(checked) => {
              onCheckChange?.(checked === true);
            }}
            onClick={(e) => e.stopPropagation()}
            className="mr-1"
          />
        )}

        <button
          onClick={(e) => {
            e.stopPropagation();
            onToggleStar?.(email)
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
      </div>

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

        {/* 摘要 - 强制单行显示，防止换行 */}
        <div
          className="text-xs text-muted-foreground"
          style={{
            whiteSpace: 'nowrap',
            overflow: 'hidden',
            textOverflow: 'ellipsis',
            maxWidth: '100%',
            display: 'block'
          }}
        >
          {getSnippet(email.snippet || '')}
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
            {accountInfo.fullEmail}
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