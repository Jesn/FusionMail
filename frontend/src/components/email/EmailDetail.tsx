import { useState } from 'react';
import { Star, Archive, Trash2, Download, Paperclip, Code, FileText, AlertTriangle } from 'lucide-react';
import { Button } from '../ui/button';
import { Badge } from '../ui/badge';
import { ScrollArea } from '../ui/scroll-area';
import { Email } from '../../types';
import { format } from 'date-fns';
import { zhCN } from 'date-fns/locale';
import { cn } from '../../lib/utils';
import './EmailDetail.css';

interface EmailDetailProps {
  email: Email;
  onToggleStar: () => void;
  onArchive: () => void;
  onDelete: () => void;
}

export const EmailDetail = ({
  email,
  onToggleStar,
  onArchive,
  onDelete,
}: EmailDetailProps) => {
  // 判断邮件是否有 HTML 和纯文本内容
  const hasHtmlContent = !!email.html_body;
  const hasTextContent = !!email.text_body;
  
  // 如果只有 HTML 没有纯文本，默认显示 HTML；否则默认显示纯文本（安全模式）
  const [showHtml, setShowHtml] = useState(!hasTextContent && hasHtmlContent);

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

  const toAddresses = parseAddresses(email.to_addresses);

  // 清理HTML内容中的内联样式，防止破坏布局
  const sanitizeHtml = (html: string) => {
    if (!html) return '';

    // 使用DOMParser清理HTML
    const parser = new DOMParser();
    const doc = parser.parseFromString(html, 'text/html');

    // 第一步：删除所有<style>标签（这些样式会覆盖我们的CSS）
    const styleTags = doc.querySelectorAll('style');
    styleTags.forEach(tag => tag.remove());

    // 第二步：删除<head>标签（如果存在）
    const heads = doc.querySelectorAll('head');
    heads.forEach(tag => tag.remove());

    // 第三步：查找所有可能影响布局的元素
    const elementsToClean = doc.querySelectorAll('table, td, th, tr, tbody, thead, tfoot, div, span, p, a, img, body, html');

    elementsToClean.forEach(el => {
      // 移除可能破坏布局的属性
      el.removeAttribute('width');
      el.removeAttribute('height');
      el.removeAttribute('align');
      el.removeAttribute('valign');
      el.removeAttribute('nowrap');

      // 处理style属性 - 保留字体、颜色、背景等，移除布局相关
      const style = el.getAttribute('style');
      if (style) {
        const cleanStyles = style
          .split(';')
          .filter(s => {
            const prop = s.toLowerCase().trim();
            // 移除所有布局相关的CSS属性
            return prop && !(
              prop.includes('width') ||
              prop.includes('max-width') ||
              prop.includes('min-width') ||
              prop.includes('height') ||
              prop.includes('max-height') ||
              prop.includes('min-height') ||
              prop.includes('position') ||
              prop.includes('left') ||
              prop.includes('right') ||
              prop.includes('top') ||
              prop.includes('bottom') ||
              prop.includes('float') ||
              prop.includes('clear') ||
              prop.includes('margin') ||
              prop.includes('padding') ||
              prop.includes('border') ||
              prop.includes('display') ||
              prop.includes('flex') ||
              prop.includes('grid') ||
              prop.includes('overflow')
            );
          })
          .join('; ');
        if (cleanStyles) {
          el.setAttribute('style', cleanStyles);
        } else {
          el.removeAttribute('style');
        }
      }

      // 特别处理表格相关元素
      if (el.tagName === 'TABLE' || el.tagName === 'TD' || el.tagName === 'TH') {
        el.style.tableLayout = 'auto';
        el.style.wordBreak = 'break-word';
        el.style.overflowWrap = 'break-word';
        el.style.maxWidth = '100%';
      }
    });

    // 返回清理后的HTML
    return doc.body.innerHTML;
  };

  return (
    <div className="flex h-full flex-col bg-background">
      {/* 头部工具栏 */}
      <div className="flex items-center justify-between border-b px-4 py-3">
        <div className="flex items-center gap-2">
          <Button
            variant="ghost"
            size="icon"
            onClick={onToggleStar}
            title={email.is_starred ? '取消星标' : '添加星标'}
          >
            <Star
              className={cn(
                'h-5 w-5',
                email.is_starred && 'fill-yellow-400 text-yellow-400'
              )}
            />
          </Button>
          <Button variant="ghost" size="icon" onClick={onArchive} title="归档">
            <Archive className="h-5 w-5" />
          </Button>
          <Button variant="ghost" size="icon" onClick={onDelete} title="删除">
            <Trash2 className="h-5 w-5" />
          </Button>
          {/* 回复和转发功能暂未实现 */}
          {/* <Separator orientation="vertical" className="h-6" />
          <Button variant="ghost" size="icon" title="回复">
            <Reply className="h-5 w-5" />
          </Button>
          <Button variant="ghost" size="icon" title="转发">
            <Forward className="h-5 w-5" />
          </Button> */}
        </div>

        {/* 更多菜单 - 暂时隐藏未实现的功能 */}
        {/* <DropdownMenu>
          <DropdownMenuTrigger asChild>
            <Button variant="ghost" size="icon">
              <MoreVertical className="h-5 w-4" />
            </Button>
          </DropdownMenuTrigger>
          <DropdownMenuContent align="end">
            <DropdownMenuItem>标记为未读</DropdownMenuItem>
            <DropdownMenuItem>移动到...</DropdownMenuItem>
            <DropdownMenuItem>打印</DropdownMenuItem>
          </DropdownMenuContent>
        </DropdownMenu> */}
      </div>

      {/* 邮件内容 */}
      <ScrollArea className="flex-1">
        <div className="p-6">
          {/* 主题 */}
          <h1 className="mb-4 text-2xl font-bold">{email.subject || '(无主题)'}</h1>

          {/* 本地状态提示 */}
          <div className="mb-4 flex flex-wrap gap-2">
            {email.is_archived && (
              <Badge variant="secondary">已归档（仅本地）</Badge>
            )}
            {email.is_starred && (
              <Badge variant="secondary">已星标（仅本地）</Badge>
            )}
          </div>

          {/* 发件人信息 */}
          <div className="mb-6 rounded-lg border bg-muted/50 p-4">
            <div className="mb-2 flex items-start justify-between">
              <div className="flex-1">
                <div className="flex items-center gap-2">
                  <div className="flex h-10 w-10 items-center justify-center rounded-full bg-primary text-primary-foreground">
                    {(email.from_name || email.from_address).charAt(0).toUpperCase()}
                  </div>
                  <div>
                    <div className="font-semibold">
                      {email.from_name || email.from_address}
                    </div>
                    <div className="text-sm text-muted-foreground">
                      {email.from_address}
                    </div>
                  </div>
                </div>
              </div>
              <div className="text-sm text-muted-foreground">
                {formatDate(email.sent_at)}
              </div>
            </div>

            {/* 收件人 */}
            {toAddresses.length > 0 && (
              <div className="mt-3 text-sm">
                <span className="text-muted-foreground">收件人：</span>
                <span className="ml-2">{toAddresses.join(', ')}</span>
              </div>
            )}
          </div>

          {/* 附件列表 */}
          {email.has_attachments && email.attachments && email.attachments.length > 0 && (
            <div className="mb-6">
              <div className="mb-2 flex items-center gap-2 text-sm font-medium">
                <Paperclip className="h-4 w-4" />
                <span>{email.attachments.length} 个附件</span>
              </div>
              <div className="space-y-2">
                {email.attachments.map((attachment) => (
                  <div
                    key={attachment.id}
                    className="flex items-center justify-between rounded-lg border bg-background p-3"
                  >
                    <div className="flex items-center gap-3">
                      <div className="flex h-10 w-10 items-center justify-center rounded bg-muted">
                        <Paperclip className="h-5 w-5 text-muted-foreground" />
                      </div>
                      <div>
                        <div className="font-medium">{attachment.filename}</div>
                        <div className="text-sm text-muted-foreground">
                          {(attachment.size_bytes / 1024).toFixed(1)} KB
                        </div>
                      </div>
                    </div>
                    <Button variant="ghost" size="icon" title="下载">
                      <Download className="h-4 w-4" />
                    </Button>
                  </div>
                ))}
              </div>
            </div>
          )}

          {/* 邮件正文 */}
          <div className="mt-6">
            {/* 内容格式切换按钮 - 只在同时有 HTML 和纯文本时显示 */}
            {hasHtmlContent && hasTextContent && (
              <div className="mb-4 flex items-center justify-between rounded-lg border bg-muted/50 p-3">
                <div className="flex items-center gap-2 text-sm">
                  <AlertTriangle className="h-4 w-4 text-yellow-600" />
                  <span className="text-muted-foreground">
                    {showHtml 
                      ? '正在显示 HTML 格式（可能包含外部内容）' 
                      : '正在显示纯文本格式（安全模式）'}
                  </span>
                </div>
                <Button
                  variant="secondary"
                  size="sm"
                  onClick={() => setShowHtml(!showHtml)}
                >
                  {showHtml ? (
                    <>
                      <FileText className="mr-1 h-4 w-4" />
                      切换到纯文本
                    </>
                  ) : (
                    <>
                      <Code className="mr-1 h-4 w-4" />
                      切换到 HTML
                    </>
                  )}
                </Button>
              </div>
            )}

            {/* 邮件内容显示 */}
            {showHtml && hasHtmlContent ? (
              <div
                className="email-content"
                dangerouslySetInnerHTML={{ __html: sanitizeHtml(email.html_body || '') }}
              />
            ) : hasTextContent ? (
              <div className="email-text-body">
                {email.text_body}
              </div>
            ) : hasHtmlContent ? (
              // 如果只有 HTML 没有纯文本，提示用户切换到 HTML 模式
              <div className="rounded-lg border border-dashed bg-muted/50 p-8 text-center">
                <Code className="mx-auto mb-3 h-12 w-12 text-muted-foreground" />
                <p className="mb-2 text-sm font-medium">此邮件仅包含 HTML 格式内容</p>
                <p className="mb-4 text-sm text-muted-foreground">
                  点击上方"HTML"按钮查看完整内容
                </p>
                <Button
                  variant="outline"
                  size="sm"
                  onClick={() => setShowHtml(true)}
                >
                  <Code className="mr-2 h-4 w-4" />
                  切换到 HTML 模式
                </Button>
              </div>
            ) : (
              <div className="text-muted-foreground italic">
                {email.snippet || '(无内容)'}
              </div>
            )}
          </div>
        </div>
      </ScrollArea>
    </div>
  );
};
