import { formatDistanceToNow, format } from 'date-fns';
import { zhCN } from 'date-fns/locale';
import { CheckCircle2, XCircle, Clock } from 'lucide-react';

// 已发送邮件展示元数据（自 SentPage 同位搬移）：日期格式化 / 收件人解析 / 状态图标与标签
  export const formatRelativeDate = (dateString?: string) => {
  if (!dateString) return '-';
  try {
    return formatDistanceToNow(new Date(dateString), {
      addSuffix: true,
      locale: zhCN,
    });
  } catch {
    return dateString;
  }
};

// 格式化完整日期
export const formatFullDate = (dateString?: string) => {
  if (!dateString) return '-';
  try {
    return format(new Date(dateString), 'yyyy年MM月dd日 HH:mm:ss', {
      locale: zhCN,
    });
  } catch {
    return dateString;
  }
};

// 解析收件人
// 处理多种情况：null、undefined、空字符串、"null" 字符串、JSON 数组字符串
export const parseRecipients = (addressesJson: string | null | undefined): string[] => {
  // 处理 null、undefined、空字符串、"null" 字符串
  if (!addressesJson || addressesJson === 'null' || addressesJson === '[]') {
    return [];
  }
  try {
    const parsed = JSON.parse(addressesJson);
    // 确保返回的是数组，JSON.parse("null") 会返回 null
    return Array.isArray(parsed) ? parsed : [];
  } catch {
    return [];
  }
};

// 获取状态图标
export const getStatusIcon = (status: string) => {
  switch (status) {
    case 'sent':
      return <CheckCircle2 className="h-4 w-4 text-green-500" />;
    case 'failed':
      return <XCircle className="h-4 w-4 text-red-500" />;
    case 'pending':
      return <Clock className="h-4 w-4 text-yellow-500" />;
    default:
      return <Clock className="h-4 w-4 text-muted-foreground" />;
  }
};

// 获取状态信息
export const getStatusInfo = (status: string) => {
  switch (status) {
    case 'sent':
      return { label: '已发送', color: 'text-green-500' };
    case 'failed':
      return { label: '发送失败', color: 'text-red-500' };
    case 'pending':
      return { label: '发送中', color: 'text-yellow-500' };
    default:
      return { label: status, color: 'text-muted-foreground' };
  }
};
