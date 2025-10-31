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

  // 清理HTML标签和CSS代码，提取纯文本内容
  const cleanHtmlContent = (htmlString: string) => {
    if (!htmlString) return '';
    
    let cleanText = htmlString;
    
    // 1. 先移除HTML标签
    const tempDiv = document.createElement('div');
    tempDiv.innerHTML = cleanText;
    cleanText = tempDiv.textContent || tempDiv.innerText || cleanText;
    
    // 2. 移除CSS样式块
    cleanText = cleanText.replace(/\{[^}]*\}/g, '');
    
    // 3. 移除CSS属性
    cleanText = cleanText.replace(/[\w-]+\s*:\s*[^;]+;/g, '');
    
    // 4. 彻底清理CSS类名和选择器 - 特别针对 .x-row-helper
    // 首先移除所有以点开头的CSS类名
    cleanText = cleanText.replace(/\.[a-zA-Z][\w-]*\b/g, '');
    cleanText = cleanText.replace(/#[a-zA-Z][\w-]*\b/g, '');
    
    // 特别处理顽固的CSS类名 - 必须彻底移除 .x-row-helper
    cleanText = cleanText.replace(/\.x-row-helper/gi, '');
    cleanText = cleanText.replace(/x-row-helper/gi, '');
    cleanText = cleanText.replace(/\.mmsgLetter/gi, '');
    cleanText = cleanText.replace(/mmsgLetter/gi, '');
    cleanText = cleanText.replace(/row-helper/gi, '');
    cleanText = cleanText.replace(/\.row-helper/gi, '');
    
    // 移除任何包含连字符的技术术语
    cleanText = cleanText.replace(/\b\w*-\w*\b/g, '');
    
    // 额外清理：移除任何剩余的CSS相关内容
    cleanText = cleanText.replace(/\.\w+/g, '');
    cleanText = cleanText.replace(/helper/gi, '');
    cleanText = cleanText.replace(/row/gi, '');
    cleanText = cleanText.replace(/msg/gi, '');
    cleanText = cleanText.replace(/Letter/gi, '');
    
    // 5. 移除CSS vendor前缀
    cleanText = cleanText.replace(/-webkit-[\w-]+/g, '');
    cleanText = cleanText.replace(/-moz-[\w-]+/g, '');
    cleanText = cleanText.replace(/-ms-[\w-]+/g, '');
    cleanText = cleanText.replace(/-o-[\w-]+/g, '');
    
    // 6. 移除CSS单位
    cleanText = cleanText.replace(/\d+(\.\d+)?(px|em|rem|%|vh|vw|pt|pc|in|cm|mm|ex|ch|vmin|vmax|fr)/g, '');
    
    // 7. 移除CSS颜色值
    cleanText = cleanText.replace(/#[0-9a-fA-F]{3,6}/g, '');
    
    // 8. 移除HTML标签名称列表
    const htmlTags = /\b(html|body|head|title|meta|link|script|style|div|span|p|a|img|ul|ol|li|table|tr|td|th|form|input|button|select|textarea|textare|h1|h2|h3|h4|h5|h6|br|hr|strong|em|b|i|u|small|big|sub|sup|pre|code|blockquote|cite|abbr|address|article|aside|footer|header|main|nav|section|details|summary|dialog|figure|figcaption|mark|time|canvas|svg|audio|video|source|track|embed|object|param|iframe|fieldset|legend|label|datalist|optgroup|option|output|progress|meter|dd|dl|dt)\b/gi;
    cleanText = cleanText.replace(htmlTags, '');
    
    // 9. 移除常见CSS关键词和技术术语
    const cssKeywords = /\b(width|height|margin|padding|font-size|color|background|border|display|position|top|left|right|bottom|auto|none|inherit|initial|block|inline|flex|grid|absolute|relative|fixed|static|hidden|visible|solid|dashed|dotted|important|antialiased|smoothing|adjust|zoom|helper|Letter|row|half|mmsg|wrapper|container|content|header|footer|sidebar|main|nav|menu|item|list|card|box|panel|modal|dialog|overlay|backdrop|shadow|gradient|transition|animation|transform|rotate|scale|translate|opacity|z-index|overflow|scroll|clip|ellipsis|nowrap|break|word|text|font|line|vertical|horizontal|center|middle|baseline|stretch|space|between|around|evenly|start|end|first|last|odd|even|nth|child|before|after|hover|focus|active|visited|disabled|enabled|checked|selected|required|optional|valid|invalid|empty|full|loading|error|success|warning|info|primary|secondary|tertiary|accent|muted|foreground|background)\b/gi;
    cleanText = cleanText.replace(cssKeywords, '');
    
    // 10. 清理符号和多余空白
    cleanText = cleanText.replace(/[{}();:,.-]/g, ' ');
    cleanText = cleanText.replace(/\s+/g, ' ');
    cleanText = cleanText.trim();
    
    // 11. 最终清理：移除剩余的技术术语和无意义内容
    cleanText = cleanText.replace(/\b(htmlbodybody|url|http|https|www|com|org|net|css|js|php|html|xml|json|api|src|href|class|id|div|span|after|before|Date|From|To|CST|GMT|UTC)\b/gi, '');
    cleanText = cleanText.replace(/\s+/g, ' ').trim();
    
    // 12. 强力去重：移除重复的句子和短语
    const sentences = cleanText.split(/[。！？.!?]/);
    const uniqueSentences = [];
    const seenSentences = new Set();
    
    for (const sentence of sentences) {
      const trimmed = sentence.trim();
      if (trimmed && !seenSentences.has(trimmed.toLowerCase())) {
        seenSentences.add(trimmed.toLowerCase());
        uniqueSentences.push(trimmed);
      }
    }
    
    cleanText = uniqueSentences.join('，');
    
    // 13. 单词级去重（处理重复的词汇）
    const words = cleanText.split(/\s+/);
    const uniqueWords = [];
    const seenWords = new Set();
    
    for (const word of words) {
      if (word && word.length > 1 && !seenWords.has(word.toLowerCase())) {
        seenWords.add(word.toLowerCase());
        uniqueWords.push(word);
      }
    }
    
    cleanText = uniqueWords.join(' ');
    
    // 14. 最后一次检查：确保 .x-row-helper 完全被移除
    cleanText = cleanText.replace(/x-row-helper/gi, '');
    cleanText = cleanText.replace(/\.x-row-helper/gi, '');
    cleanText = cleanText.replace(/row-helper/gi, '');
    cleanText = cleanText.replace(/\.row-helper/gi, '');
    cleanText = cleanText.replace(/helper/gi, '');
    cleanText = cleanText.replace(/\./g, ''); // 移除所有剩余的点
    
    // 15. 如果清理后内容太短、为空或只包含空白，返回友好提示
    if (cleanText.length < 3 || /^\s*$/.test(cleanText)) {
      const meaningfulStart = htmlString.match(/^([^{<]*?)(?:[{<]|$)/);
      if (meaningfulStart && meaningfulStart[1].trim().length > 5) {
        return meaningfulStart[1].trim();
      }
      return '(邮件内容)';
    }
    
    // 16. 限制长度，确保单行显示
    if (cleanText.length > 50) {
      cleanText = cleanText.substring(0, 50) + '...';
    }
    
    return cleanText;
  };

  // 获取邮箱账户信息
  const getAccountInfo = () => {
    if (!showAccountBadge) return null;
    
    const account = accounts.find(acc => acc.uid === email.account_uid);
    if (!account) return null;
    
    return {
      email: account.email,
      fullEmail: account.email, // 显示完整邮箱地址
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
          {cleanHtmlContent(email.snippet || '')}
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