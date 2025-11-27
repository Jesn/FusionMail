import { useState, useEffect } from 'react';
import { Star, Archive, Trash2, Download, Paperclip, RotateCcw, ShieldAlert, ShieldCheck, UserPlus } from 'lucide-react';
import { Button } from '../ui/button';
import { Badge } from '../ui/badge';
import { ScrollArea } from '../ui/scroll-area';
import { Card, CardContent, CardHeader, CardTitle } from '../ui/card';
import { spamService } from '../../services/spamService';
import { addToWhitelist } from '../../services/emailListService';
import toast from 'react-hot-toast';
import type { EmailDetail as EmailDetailType } from '../../types';
import { format } from 'date-fns';
import { zhCN } from 'date-fns/locale';
import { cn } from '../../lib/utils';
import { isDangerousHtml } from '../../utils/sanitize';
import { cidProcessor } from '../../utils/cid-processor';
import ShadowHtmlComponent from './ShadowHtmlComponent';
import './EmailDetail.css';

interface EmailDetailProps {
  email: EmailDetailType;
  onToggleStar: () => void;
  onArchive: () => void;
  onDelete: () => void;
  onRestore?: () => void;
  onBack: () => void;
  onSpamStatusChange?: (isSpam: boolean) => void;
  forceDeletedView?: boolean;
}

export const EmailDetail = ({
  email,
  onToggleStar,
  onArchive,
  onDelete,
  onRestore,
  onBack,
  onSpamStatusChange,
  forceDeletedView,
}: EmailDetailProps) => {
  const hasHtmlContent = !!email.html_body;
  const hasTextContent = !!email.text_body;
  const hasDangerousContent = isDangerousHtml(email.html_body || '');
  const [processedHtml, setProcessedHtml] = useState<string>('');
  const [isMarkingSpam, setIsMarkingSpam] = useState(false);

  useEffect(() => {
    if (!email.html_body) {
      setProcessedHtml('');
      return;
    }
    try {
      let html = email.html_body;
      if (email.attachments && email.attachments.length > 0) {
        const cidMap = cidProcessor.extractCidMap(email.attachments);
        html = cidProcessor.replaceCidInHtml(html, cidMap);
      }
      setProcessedHtml(html);
    } catch (error) {
      console.error('处理邮件 CID 失败:', error);
      setProcessedHtml(email.html_body);
    }
    return () => {
      cidProcessor.cleanup();
    };
  }, [email]);

  const formatDate = (dateString: string) => {
    try {
      return format(new Date(dateString), 'PPP HH:mm', { locale: zhCN });
    } catch {
      return dateString;
    }
  };

  const parseAddresses = (addressesJson: string): string[] => {
    try {
      return JSON.parse(addressesJson);
    } catch {
      return [];
    }
  };

  const handleToggleSpam = async () => {
    setIsMarkingSpam(true);
    try {
      if (email.is_spam) {
        await spamService.unmarkAsSpam([email.id]);
        toast.success('已移出垃圾箱');
        onSpamStatusChange?.(false);
      } else {
        await spamService.markAsSpam([email.id]);
        toast.success('已标记为垃圾邮件');
        onSpamStatusChange?.(true);
      }
    } catch (error) {
      console.error('Failed to toggle spam:', error);
      toast.error('操作失败');
    } finally {
      setIsMarkingSpam(false);
    }
  };

  const handleAddToWhitelist = async () => {
    try {
      await addToWhitelist({
        target: email.from_address,
        target_type: 'email',
        reason: `从邮件详情页添加 - ${email.subject || '无主题'}`,
      });
      toast.success(`已将 ${email.from_address} 加入白名单`);
      // 同时取消垃圾邮件标记
      if (email.is_spam) {
        await spamService.unmarkAsSpam([email.id]);
        onSpamStatusChange?.(false);
      }
    } catch (error) {
      console.error('Failed to add to whitelist:', error);
      toast.error('添加白名单失败');
    }
  };

  const toAddresses = parseAddresses(email.to_addresses);

  return (
    <div className="flex h-full flex-col bg-background">
      {/* 顶部导航栏 */}
      <div className="flex items-center justify-between bg-background px-4 py-1.5">
        <div className="flex items-center gap-2">
          <Button variant="ghost" size="sm" onClick={onBack} className="text-xs h-7">
            ← 返回
          </Button>
        </div>
        <div className="flex items-center gap-1">
          {(forceDeletedView || email.is_deleted) ? (
            <Button variant="ghost" size="icon" onClick={onRestore} title="恢复" className="h-7 w-7">
              <RotateCcw className="h-3.5 w-3.5" />
            </Button>
          ) : (
            <>
              <Button
                variant="ghost"
                size="icon"
                onClick={onToggleStar}
                title={email.is_starred ? '取消星标' : '添加星标'}
                className="h-7 w-7"
              >
                <Star className={cn('h-3.5 w-3.5', email.is_starred && 'fill-yellow-400 text-yellow-400')} />
              </Button>
              <Button
                variant="ghost"
                size="icon"
                onClick={handleToggleSpam}
                disabled={isMarkingSpam}
                title={email.is_spam ? '移出垃圾箱' : '标记为垃圾邮件'}
                className="h-7 w-7"
              >
                <ShieldAlert className={cn('h-3.5 w-3.5', email.is_spam && 'text-orange-500')} />
              </Button>
              <Button variant="ghost" size="icon" onClick={onArchive} title="归档" className="h-7 w-7">
                <Archive className="h-3.5 w-3.5" />
              </Button>
              <Button variant="ghost" size="icon" onClick={onDelete} title="删除" className="h-7 w-7">
                <Trash2 className="h-3.5 w-3.5" />
              </Button>
            </>
          )}
        </div>
      </div>

      {/* 邮件内容 */}
      <ScrollArea className="flex-1">
        <div className="mx-auto max-w-5xl px-6 py-6">
          {/* 垃圾邮件检测信息面板 */}
          {email.is_spam && (
            <Card className="mb-4 border-orange-200 bg-orange-50">
              <CardHeader className="pb-2">
                <CardTitle className="flex items-center gap-2 text-orange-700 text-sm">
                  <ShieldAlert className="h-4 w-4" />
                  垃圾邮件检测信息
                </CardTitle>
              </CardHeader>
              <CardContent className="pt-0">
                <div className="grid grid-cols-2 md:grid-cols-4 gap-4 text-sm">
                  {email.spam_score !== undefined && (
                    <div>
                      <div className="text-muted-foreground text-xs">评分</div>
                      <div className="font-medium text-orange-700">{email.spam_score}</div>
                    </div>
                  )}
                  {email.spam_confidence !== undefined && (
                    <div>
                      <div className="text-muted-foreground text-xs">置信度</div>
                      <div className="font-medium text-orange-700">{Math.round(email.spam_confidence * 100)}%</div>
                    </div>
                  )}
                  {email.spam_detected_by && (
                    <div>
                      <div className="text-muted-foreground text-xs">检测层级</div>
                      <div className="font-medium text-orange-700">{email.spam_detected_by}</div>
                    </div>
                  )}
                  {email.spam_detected_at && (
                    <div>
                      <div className="text-muted-foreground text-xs">检测时间</div>
                      <div className="font-medium text-orange-700">{formatDate(email.spam_detected_at)}</div>
                    </div>
                  )}
                </div>
                {email.spam_reason && (
                  <div className="mt-3 pt-3 border-t border-orange-200">
                    <div className="text-muted-foreground text-xs mb-1">检测原因</div>
                    <div className="text-sm text-orange-700">{email.spam_reason}</div>
                  </div>
                )}
                {email.user_marked_spam && email.user_marked_at && (
                  <div className="mt-2 text-xs text-muted-foreground">
                    用户手动标记于 {formatDate(email.user_marked_at)}
                  </div>
                )}
                <div className="mt-3 flex gap-2">
                  <Button variant="outline" size="sm" onClick={handleToggleSpam} disabled={isMarkingSpam} className="text-xs">
                    <ShieldCheck className="h-3 w-3 mr-1" />
                    这不是垃圾邮件
                  </Button>
                  <Button variant="outline" size="sm" onClick={handleAddToWhitelist} className="text-xs">
                    <UserPlus className="h-3 w-3 mr-1" />
                    加入白名单
                  </Button>
                </div>
              </CardContent>
            </Card>
          )}

          {/* 邮件头部区域 */}
          <div className="mb-4 pb-4">
            <div className="mb-3">
              <h1 className="text-xl font-semibold mb-2 text-foreground leading-tight">
                {email.subject || '(无主题)'}
              </h1>
              <div className="flex flex-wrap gap-2">
                {email.is_spam && (
                  <Badge variant="outline" className="text-xs border-orange-300 text-orange-700 bg-orange-50">
                    <ShieldAlert className="h-3 w-3 mr-1" />
                    垃圾邮件
                  </Badge>
                )}
                {email.is_archived && (
                  <Badge variant="secondary" className="text-xs">已归档（仅本地）</Badge>
                )}
                {email.is_starred && (
                  <Badge variant="secondary" className="text-xs">已星标（仅本地）</Badge>
                )}
                {(forceDeletedView || email.is_deleted) && (
                  <Badge variant="destructive" className="text-xs">已删除（垃圾箱）</Badge>
                )}
              </div>
            </div>

            {/* 发件人信息卡片 */}
            <div className="bg-muted/30 rounded-lg p-3 border">
              <div className="flex items-start gap-3">
                <div className="flex h-9 w-9 items-center justify-center rounded-full bg-gradient-to-br from-blue-500 to-blue-600 text-white text-sm font-semibold shadow-sm">
                  {(email.from_name || email.from_address).charAt(0).toUpperCase()}
                </div>
                <div className="flex-1 min-w-0">
                  <div className="flex items-start justify-between">
                    <div>
                      <div className="font-medium text-sm text-foreground">
                        {email.from_name || email.from_address}
                      </div>
                      <div className="text-xs text-muted-foreground break-all">
                        {email.from_address}
                      </div>
                    </div>
                    <div className="text-xs text-muted-foreground whitespace-nowrap ml-2">
                      {formatDate(email.sent_at)}
                    </div>
                  </div>
                  {toAddresses.length > 0 && (
                    <div className="text-xs text-muted-foreground mt-1">
                      <span className="inline">收件人：</span>
                      <span className="ml-1 inline">{toAddresses.join(', ')}</span>
                    </div>
                  )}
                </div>
              </div>
            </div>
          </div>

          {/* 附件列表 */}
          {email.has_attachments && email.attachments && email.attachments.length > 0 && (
            <div className="mb-4 pb-4">
              <div className="mb-3 flex items-center gap-2 text-sm font-medium text-foreground">
                <Paperclip className="h-4 w-4" />
                <span>{email.attachments.length} 个附件</span>
              </div>
              <div className="grid grid-cols-1 md:grid-cols-2 gap-2">
                {email.attachments.map((attachment) => (
                  <div
                    key={attachment.id}
                    className="group flex items-center justify-between rounded-lg border bg-card p-3 hover:border-primary/50 hover:shadow-sm transition-all duration-200"
                  >
                    <div className="flex items-center gap-3 min-w-0 flex-1">
                      <div className="flex h-10 w-10 items-center justify-center rounded-lg bg-primary/10 group-hover:bg-primary/20 transition-colors">
                        <Paperclip className="h-5 w-5 text-primary" />
                      </div>
                      <div className="min-w-0 flex-1">
                        <div className="font-medium text-sm truncate" title={attachment.filename}>
                          {attachment.filename}
                        </div>
                        <div className="text-xs text-muted-foreground">
                          {(attachment.size_bytes / 1024).toFixed(1)} KB
                        </div>
                      </div>
                    </div>
                    <Button variant="ghost" size="icon" title="下载" className="h-8 w-8 opacity-0 group-hover:opacity-100 transition-opacity">
                      <Download className="h-4 w-4" />
                    </Button>
                  </div>
                ))}
              </div>
            </div>
          )}

          {/* 邮件正文 */}
          <div>
            {hasHtmlContent ? (
              <div className="email-content-wrapper">
                <ShadowHtmlComponent htmlContent={processedHtml} useStrictMode={hasDangerousContent} />
              </div>
            ) : hasTextContent ? (
              <div className="email-text-content">{email.text_body}</div>
            ) : (
              <div className="text-center py-8 text-muted-foreground italic text-sm">
                {email.snippet || '(无内容)'}
              </div>
            )}
          </div>
        </div>
      </ScrollArea>
    </div>
  );
};
