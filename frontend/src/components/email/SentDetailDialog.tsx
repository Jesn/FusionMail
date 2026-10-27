import { Send, Paperclip, RotateCcw } from 'lucide-react';
import { Button } from '../ui/button';
import { Badge } from '../ui/badge';
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
} from '../ui/dialog';
import { cn } from '../../lib/utils';
import type { SentEmail } from '../../types';
import SentEmailBody from './SentEmailBody';
import { getStatusInfo, getStatusIcon, parseRecipients, formatFullDate } from './sentMeta';

interface SentDetailDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  email: SentEmail | null;
  onRetry: (email: SentEmail) => void;
  getAccountEmail: (accountUid: string) => string;
}

/** 已发送邮件详情对话框（自 SentPage 同位搬移）。
 * 纯展示：状态/收件人/日期展示复用 sentMeta 纯函数。 */
export function SentDetailDialog({
  open,
  onOpenChange,
  email,
  onRetry,
  getAccountEmail,
}: SentDetailDialogProps) {
  return (
      <Dialog open={open} onOpenChange={onOpenChange}>
        <DialogContent className="max-w-2xl max-h-[80vh] flex flex-col">
          <DialogHeader>
            <DialogTitle className="flex items-center gap-2">
              <Send className="h-5 w-5" />
              已发送邮件详情
            </DialogTitle>
          </DialogHeader>

          {email && (
            <div className="flex-1 overflow-y-auto space-y-4">
              {/* 状态 */}
              <div className="flex items-center gap-2">
                <Badge
                  variant="outline"
                  className={cn('gap-1', getStatusInfo(email.status).color)}
                >
                  {getStatusIcon(email.status)}
                  {getStatusInfo(email.status).label}
                </Badge>
                {email.has_attachments && (
                  <Badge variant="outline" className="gap-1">
                    <Paperclip className="h-3.5 w-3.5" />
                    {email.attachment_count || 1} 个附件
                  </Badge>
                )}
              </div>

              {/* 错误信息 */}
              {email.status === 'failed' && email.error_message && (
                <div className="p-3 bg-destructive/10 border border-destructive/20 rounded-md">
                  <p className="text-sm text-destructive">
                    <strong>发送失败：</strong>{email.error_message}
                  </p>
                </div>
              )}

              {/* 邮件信息 */}
              <div className="space-y-2 text-sm">
                <div className="flex">
                  <span className="w-16 text-muted-foreground">发件人</span>
                  <span>
                    {getAccountEmail(email.account_uid)}
                  </span>
                </div>
                <div className="flex">
                  <span className="w-16 text-muted-foreground">收件人</span>
                  <span>{parseRecipients(email.to_addresses).join(', ') || '-'}</span>
                </div>
                {email.cc_addresses && (
                  <div className="flex">
                    <span className="w-16 text-muted-foreground">抄送</span>
                    <span>{parseRecipients(email.cc_addresses).join(', ')}</span>
                  </div>
                )}
                <div className="flex">
                  <span className="w-16 text-muted-foreground">主题</span>
                  <span className="font-medium">{email.subject || '(无主题)'}</span>
                </div>
                <div className="flex">
                  <span className="w-16 text-muted-foreground">时间</span>
                  <span>{formatFullDate(email.sent_at || email.created_at)}</span>
                </div>
              </div>

              {/* 邮件正文 */}
              <div className="border-t pt-4">
                <div className="p-4 bg-muted/30 rounded-md">
                  <SentEmailBody email={email} />
                </div>
              </div>
            </div>
          )}

          {/* 底部操作 */}
          <div className="flex items-center justify-end gap-2 pt-4 border-t">
            {email?.status === 'failed' && (
              <Button variant="outline" onClick={() => email && onRetry(email)}>
                <RotateCcw className="h-4 w-4 mr-1" />
                重试发送
              </Button>
            )}
            <Button variant="outline" onClick={() => onOpenChange(false)}>
              关闭
            </Button>
          </div>
        </DialogContent>
      </Dialog>
  );
}
