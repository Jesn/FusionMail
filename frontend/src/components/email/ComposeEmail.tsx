/**
 * 邮件编写组件
 * 支持新建邮件、回复、转发功能
 */

import { useState, useCallback, useRef } from 'react';
import { X, Send, Paperclip, Loader2, ChevronDown, ChevronUp } from 'lucide-react';
import { Button } from '../ui/button';
import { Input } from '../ui/input';
import { Label } from '../ui/label';
import { Textarea } from '../ui/textarea';
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
} from '../ui/dialog';
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '../ui/select';
import { Badge } from '../ui/badge';
import { useAccounts } from '../../hooks/useAccounts';
import { emailService } from '../../services/emailService';
import type { 
  SendEmailRequest, 
  AttachmentInfo, 
  EmailDetail,
  Account,
} from '../../types';
import toast from 'react-hot-toast';

// 编写模式
export type ComposeMode = 'new' | 'reply' | 'replyAll' | 'forward';

interface ComposeEmailProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  mode?: ComposeMode;
  originalEmail?: EmailDetail;
  defaultAccountUid?: string;
}

// 最大附件大小（25MB）
const MAX_ATTACHMENT_SIZE = 25 * 1024 * 1024;
// 最大附件总大小（50MB）
const MAX_TOTAL_ATTACHMENT_SIZE = 50 * 1024 * 1024;

export const ComposeEmail = ({
  open,
  onOpenChange,
  mode = 'new',
  originalEmail,
  defaultAccountUid,
}: ComposeEmailProps) => {
  const { accounts } = useAccounts();
  const activeAccounts = (accounts || []).filter(
    (account: Account) => !account.deleted_at && account.status === 'active'
  );

  // 表单状态
  const [accountUid, setAccountUid] = useState(defaultAccountUid || '');
  const [to, setTo] = useState('');
  const [cc, setCc] = useState('');
  const [bcc, setBcc] = useState('');
  const [subject, setSubject] = useState('');
  const [body, setBody] = useState('');
  const [attachments, setAttachments] = useState<AttachmentInfo[]>([]);
  const [showCcBcc, setShowCcBcc] = useState(false);
  const [isSending, setIsSending] = useState(false);
  const [isUploading, setIsUploading] = useState(false);

  const fileInputRef = useRef<HTMLInputElement>(null);

  // 解析地址 JSON 字符串
  const parseAddresses = (addressesJson: string): string[] => {
    try {
      return JSON.parse(addressesJson);
    } catch {
      return [];
    }
  };

  // 初始化表单（根据模式）
  const initializeForm = useCallback(() => {
    if (!originalEmail) {
      // 新建邮件
      setTo('');
      setCc('');
      setBcc('');
      setSubject('');
      setBody('');
      setAttachments([]);
      setShowCcBcc(false);
      return;
    }

    const fromAddress = originalEmail.from_address;
    const toAddresses = parseAddresses(originalEmail.to_addresses);
    const ccAddresses = originalEmail.cc_addresses 
      ? parseAddresses(originalEmail.cc_addresses) 
      : [];

    switch (mode) {
      case 'reply':
        setTo(fromAddress);
        setCc('');
        setBcc('');
        setSubject(`Re: ${originalEmail.subject || ''}`);
        setBody(buildQuotedContent(originalEmail));
        setAttachments([]);
        setShowCcBcc(false);
        break;

      case 'replyAll': {
        // 回复全部：收件人 = 原发件人 + 原收件人（排除自己）
        const selectedAccount = activeAccounts.find(a => a.uid === accountUid);
        const myEmail = selectedAccount?.email || '';
        const allRecipients = [fromAddress, ...toAddresses].filter(
          addr => addr.toLowerCase() !== myEmail.toLowerCase()
        );
        setTo(allRecipients.join(', '));
        setCc(ccAddresses.filter(
          addr => addr.toLowerCase() !== myEmail.toLowerCase()
        ).join(', '));
        setBcc('');
        setSubject(`Re: ${originalEmail.subject || ''}`);
        setBody(buildQuotedContent(originalEmail));
        setAttachments([]);
        setShowCcBcc(ccAddresses.length > 0);
        break;
      }

      case 'forward':
        setTo('');
        setCc('');
        setBcc('');
        setSubject(`Fwd: ${originalEmail.subject || ''}`);
        setBody(buildForwardContent(originalEmail));
        // 转发时可以选择是否包含附件
        setAttachments([]);
        setShowCcBcc(false);
        break;

      default:
        break;
    }
  }, [mode, originalEmail, accountUid, activeAccounts]);

  // 构建引用内容（回复）
  const buildQuotedContent = (email: EmailDetail): string => {
    const date = new Date(email.sent_at).toLocaleString('zh-CN');
    const header = `\n\n---------- 原始邮件 ----------\n发件人: ${email.from_name || email.from_address} <${email.from_address}>\n日期: ${date}\n主题: ${email.subject || '(无主题)'}\n\n`;
    const content = email.text_body || '(无内容)';
    return header + content;
  };

  // 构建转发内容
  const buildForwardContent = (email: EmailDetail): string => {
    const date = new Date(email.sent_at).toLocaleString('zh-CN');
    const toAddresses = parseAddresses(email.to_addresses);
    const header = `\n\n---------- 转发邮件 ----------\n发件人: ${email.from_name || email.from_address} <${email.from_address}>\n日期: ${date}\n主题: ${email.subject || '(无主题)'}\n收件人: ${toAddresses.join(', ')}\n\n`;
    const content = email.text_body || '(无内容)';
    return header + content;
  };

  // 当对话框打开时初始化表单
  useState(() => {
    if (open) {
      initializeForm();
      // 设置默认账户
      if (defaultAccountUid) {
        setAccountUid(defaultAccountUid);
      } else if (originalEmail) {
        setAccountUid(originalEmail.account_uid);
      } else if (activeAccounts.length > 0) {
        setAccountUid(activeAccounts[0].uid);
      }
    }
  });

  // 解析收件人字符串
  const parseRecipients = (input: string): string[] => {
    if (!input.trim()) return [];
    return input
      .split(/[,;，；]/)
      .map(addr => addr.trim())
      .filter(addr => addr.length > 0);
  };

  // 验证邮箱格式
  const isValidEmail = (email: string): boolean => {
    const emailRegex = /^[^\s@]+@[^\s@]+\.[^\s@]+$/;
    return emailRegex.test(email);
  };

  // 验证所有收件人
  const validateRecipients = (): { valid: boolean; errors: string[] } => {
    const errors: string[] = [];
    const toList = parseRecipients(to);
    const ccList = parseRecipients(cc);
    const bccList = parseRecipients(bcc);

    if (toList.length === 0) {
      errors.push('请输入至少一个收件人');
    }

    [...toList, ...ccList, ...bccList].forEach(addr => {
      if (!isValidEmail(addr)) {
        errors.push(`无效的邮箱地址: ${addr}`);
      }
    });

    return { valid: errors.length === 0, errors };
  };

  // 处理附件上传
  const handleFileSelect = async (event: React.ChangeEvent<HTMLInputElement>) => {
    const files = event.target.files;
    if (!files || files.length === 0) return;

    setIsUploading(true);
    try {
      for (const file of Array.from(files)) {
        // 检查单个文件大小
        if (file.size > MAX_ATTACHMENT_SIZE) {
          toast.error(`文件 "${file.name}" 超过 25MB 限制`);
          continue;
        }

        // 检查总大小
        const currentTotal = attachments.reduce((sum, att) => sum + att.size, 0);
        if (currentTotal + file.size > MAX_TOTAL_ATTACHMENT_SIZE) {
          toast.error('附件总大小超过 50MB 限制');
          break;
        }

        // 上传附件
        const result = await emailService.uploadAttachment(file);
        setAttachments(prev => [...prev, {
          filename: result.filename,
          content_type: result.content_type,
          size: result.size,
          temp_path: result.temp_path,
        }]);
      }
    } catch (error) {
      console.error('上传附件失败:', error);
      toast.error('上传附件失败');
    } finally {
      setIsUploading(false);
      // 清空 input 以便重复选择同一文件
      if (fileInputRef.current) {
        fileInputRef.current.value = '';
      }
    }
  };

  // 移除附件
  const removeAttachment = (index: number) => {
    setAttachments(prev => prev.filter((_, i) => i !== index));
  };

  // 格式化文件大小
  const formatFileSize = (bytes: number): string => {
    if (bytes < 1024) return `${bytes} B`;
    if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KB`;
    return `${(bytes / (1024 * 1024)).toFixed(1)} MB`;
  };

  // 发送邮件
  const handleSend = async () => {
    // 验证账户
    if (!accountUid) {
      toast.error('请选择发件账户');
      return;
    }

    // 验证收件人
    const validation = validateRecipients();
    if (!validation.valid) {
      validation.errors.forEach(err => toast.error(err));
      return;
    }

    // 验证主题
    if (!subject.trim()) {
      const confirmed = window.confirm('邮件没有主题，确定要发送吗？');
      if (!confirmed) return;
    }

    setIsSending(true);
    try {
      const request: SendEmailRequest = {
        account_uid: accountUid,
        to: parseRecipients(to),
        cc: parseRecipients(cc),
        bcc: parseRecipients(bcc),
        subject: subject.trim(),
        text_body: body,
        attachments: attachments.length > 0 ? attachments : undefined,
      };

      // 如果是回复，添加引用信息
      if ((mode === 'reply' || mode === 'replyAll') && originalEmail) {
        request.in_reply_to = originalEmail.message_id;
        request.references = originalEmail.message_id ? [originalEmail.message_id] : undefined;
      }

      let result;
      if (mode === 'reply' && originalEmail) {
        result = await emailService.replyEmail(originalEmail.id, {
          account_uid: accountUid,
          text_body: body,
          attachments: attachments.length > 0 ? attachments : undefined,
        });
      } else if (mode === 'replyAll' && originalEmail) {
        result = await emailService.replyAllEmail(originalEmail.id, {
          account_uid: accountUid,
          text_body: body,
          attachments: attachments.length > 0 ? attachments : undefined,
        });
      } else if (mode === 'forward' && originalEmail) {
        result = await emailService.forwardEmail(originalEmail.id, {
          account_uid: accountUid,
          to: parseRecipients(to),
          cc: parseRecipients(cc),
          bcc: parseRecipients(bcc),
          text_body: body,
          include_attachments: true,
        });
      } else {
        result = await emailService.sendEmail(request);
      }

      if (result.success) {
        toast.success('邮件发送成功');
        onOpenChange(false);
        // 重置表单
        resetForm();
      } else {
        toast.error(result.error || '发送失败');
      }
    } catch (error: any) {
      console.error('发送邮件失败:', error);
      // 解析后端错误信息，提供更友好的提示
      const errorMessage = error?.response?.data?.error || error.message || '发送邮件失败';
      if (errorMessage.includes('SMTP not enabled')) {
        toast.error('该账户未配置 SMTP 发送功能，请在账户管理中配置 SMTP 设置');
      } else if (errorMessage.includes('SMTP host not configured')) {
        toast.error('SMTP 服务器未配置，请在账户管理中完善 SMTP 设置');
      } else if (errorMessage.includes('credentials required')) {
        toast.error('账户凭证无效，请重新授权账户');
      } else {
        toast.error(errorMessage);
      }
    } finally {
      setIsSending(false);
    }
  };

  // 重置表单
  const resetForm = () => {
    setTo('');
    setCc('');
    setBcc('');
    setSubject('');
    setBody('');
    setAttachments([]);
    setShowCcBcc(false);
  };

  // 获取对话框标题
  const getDialogTitle = (): string => {
    switch (mode) {
      case 'reply':
        return '回复邮件';
      case 'replyAll':
        return '回复全部';
      case 'forward':
        return '转发邮件';
      default:
        return '写邮件';
    }
  };

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-w-3xl max-h-[90vh] flex flex-col">
        <DialogHeader>
          <DialogTitle>{getDialogTitle()}</DialogTitle>
        </DialogHeader>

        <div className="flex-1 overflow-y-auto space-y-4 py-4 px-1">
          {/* 发件账户选择 */}
          <div className="space-y-2">
            <Label>发件账户</Label>
            <Select value={accountUid} onValueChange={setAccountUid}>
              <SelectTrigger>
                <SelectValue placeholder="选择发件账户" />
              </SelectTrigger>
              <SelectContent>
                {activeAccounts.map((account: Account) => (
                  <SelectItem key={account.uid} value={account.uid}>
                    {account.email}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
          </div>

          {/* 收件人 */}
          <div className="space-y-2">
            <div className="flex items-center justify-between">
              <Label>收件人</Label>
              <Button
                variant="ghost"
                size="sm"
                className="h-6 text-xs"
                onClick={() => setShowCcBcc(!showCcBcc)}
              >
                {showCcBcc ? (
                  <>
                    <ChevronUp className="h-3 w-3 mr-1" />
                    隐藏抄送/密送
                  </>
                ) : (
                  <>
                    <ChevronDown className="h-3 w-3 mr-1" />
                    显示抄送/密送
                  </>
                )}
              </Button>
            </div>
            <Input
              placeholder="多个地址用逗号或分号分隔"
              value={to}
              onChange={(e) => setTo(e.target.value)}
            />
          </div>

          {/* 抄送/密送 */}
          {showCcBcc && (
            <>
              <div className="space-y-2">
                <Label>抄送 (Cc)</Label>
                <Input
                  placeholder="多个地址用逗号或分号分隔"
                  value={cc}
                  onChange={(e) => setCc(e.target.value)}
                />
              </div>
              <div className="space-y-2">
                <Label>密送 (Bcc)</Label>
                <Input
                  placeholder="多个地址用逗号或分号分隔"
                  value={bcc}
                  onChange={(e) => setBcc(e.target.value)}
                />
              </div>
            </>
          )}

          {/* 主题 */}
          <div className="space-y-2">
            <Label>主题</Label>
            <Input
              placeholder="邮件主题"
              value={subject}
              onChange={(e) => setSubject(e.target.value)}
            />
          </div>

          {/* 正文 */}
          <div className="space-y-2">
            <Label>正文</Label>
            <Textarea
              placeholder="输入邮件内容..."
              value={body}
              onChange={(e) => setBody(e.target.value)}
              className="min-h-[200px] resize-y"
            />
          </div>

          {/* 附件列表 */}
          {attachments.length > 0 && (
            <div className="space-y-2">
              <Label>附件</Label>
              <div className="flex flex-wrap gap-2">
                {attachments.map((att, index) => (
                  <Badge
                    key={index}
                    variant="secondary"
                    className="flex items-center gap-1 py-1 px-2"
                  >
                    <Paperclip className="h-3 w-3" />
                    <span className="max-w-[150px] truncate">{att.filename}</span>
                    <span className="text-xs text-muted-foreground">
                      ({formatFileSize(att.size)})
                    </span>
                    <button
                      onClick={() => removeAttachment(index)}
                      className="ml-1 hover:text-destructive"
                    >
                      <X className="h-3 w-3" />
                    </button>
                  </Badge>
                ))}
              </div>
            </div>
          )}
        </div>

        {/* 底部操作栏 */}
        <div className="flex items-center justify-between pt-4 border-t">
          <div className="flex items-center gap-2">
            <input
              ref={fileInputRef}
              type="file"
              multiple
              className="hidden"
              onChange={handleFileSelect}
            />
            <Button
              variant="outline"
              size="sm"
              onClick={() => fileInputRef.current?.click()}
              disabled={isUploading}
            >
              {isUploading ? (
                <Loader2 className="h-4 w-4 mr-1 animate-spin" />
              ) : (
                <Paperclip className="h-4 w-4 mr-1" />
              )}
              添加附件
            </Button>
            <span className="text-xs text-muted-foreground">
              单个文件最大 25MB，总计最大 50MB
            </span>
          </div>

          <div className="flex items-center gap-2">
            <Button
              variant="outline"
              onClick={() => onOpenChange(false)}
              disabled={isSending}
            >
              取消
            </Button>
            <Button onClick={handleSend} disabled={isSending || isUploading}>
              {isSending ? (
                <>
                  <Loader2 className="h-4 w-4 mr-1 animate-spin" />
                  发送中...
                </>
              ) : (
                <>
                  <Send className="h-4 w-4 mr-1" />
                  发送
                </>
              )}
            </Button>
          </div>
        </div>
      </DialogContent>
    </Dialog>
  );
};

export default ComposeEmail;
