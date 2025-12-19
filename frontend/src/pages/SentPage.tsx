/**
 * 已发送邮件页面
 * 展示通过 FusionMail 发送的邮件记录
 */

import { useState, useEffect, useCallback } from 'react';
import { emailService } from '../services/emailService';
import { useAccounts } from '../hooks/useAccounts';
import type { SentEmail, SentEmailFilter, Account } from '../types';
import { Button } from '../components/ui/button';
import { Badge } from '../components/ui/badge';
import { Input } from '../components/ui/input';
import { Checkbox } from '../components/ui/checkbox';
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '../components/ui/select';
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
} from '../components/ui/alert-dialog';
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
} from '../components/ui/dialog';
import { ScrollArea } from '../components/ui/scroll-area';
import {
  ChevronLeft,
  ChevronRight,
  RefreshCw,
  Search,
  Trash2,
  CheckCircle2,
  XCircle,
  Clock,
  Paperclip,
  Send,
  MoreVertical,
  RotateCcw,
} from 'lucide-react';
import { formatDistanceToNow, format } from 'date-fns';
import { zhCN } from 'date-fns/locale';
import { cn } from '../lib/utils';
import toast from 'react-hot-toast';
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
  DropdownMenuSeparator,
} from '../components/ui/dropdown-menu';

type StatusFilter = 'all' | 'sent' | 'failed';

export const SentPage = () => {
  const { accounts } = useAccounts();
  const activeAccounts = (accounts || []).filter(
    (account: Account) => !account.deleted_at
  );

  // 状态
  const [emails, setEmails] = useState<SentEmail[]>([]);
  const [total, setTotal] = useState(0);
  const [page, setPage] = useState(1);
  const [pageSize] = useState(20);
  const [totalPages, setTotalPages] = useState(1);
  const [isLoading, setIsLoading] = useState(false);

  // 筛选条件
  const [filter, setFilter] = useState<SentEmailFilter>({});
  const [searchKeyword, setSearchKeyword] = useState('');
  const [statusFilter, setStatusFilter] = useState<StatusFilter>('all');

  // 选中状态
  const [selectedIds, setSelectedIds] = useState<number[]>([]);
  const [showDeleteDialog, setShowDeleteDialog] = useState(false);

  // 邮件详情弹窗
  const [selectedEmail, setSelectedEmail] = useState<SentEmail | null>(null);
  const [showDetailDialog, setShowDetailDialog] = useState(false);

  // 加载已发送邮件列表
  const loadEmails = useCallback(async () => {
    setIsLoading(true);
    try {
      const response = await emailService.getSentEmails(filter, {
        page,
        page_size: pageSize,
      });
      setEmails(response.emails || []);
      setTotal(response.total);
      setTotalPages(response.total_pages);
    } catch (error) {
      console.error('加载已发送邮件失败:', error);
      toast.error('加载已发送邮件失败');
    } finally {
      setIsLoading(false);
    }
  }, [filter, page, pageSize]);

  useEffect(() => {
    loadEmails();
  }, [loadEmails]);

  // 搜索
  const handleSearch = () => {
    setFilter((prev) => ({
      ...prev,
      keyword: searchKeyword || undefined,
    }));
    setPage(1);
  };

  // 账户筛选
  const handleAccountFilter = (accountUid: string) => {
    setFilter((prev) => ({
      ...prev,
      account_uid: accountUid === 'all' ? undefined : accountUid,
    }));
    setPage(1);
  };

  // 状态筛选
  const handleStatusFilter = (status: StatusFilter) => {
    setStatusFilter(status);
    setFilter((prev) => ({
      ...prev,
      status: status === 'all' ? undefined : status,
    }));
    setPage(1);
  };

  // 删除邮件
  const handleDelete = async () => {
    try {
      for (const id of selectedIds) {
        await emailService.deleteSentEmail(id);
      }
      toast.success(`已删除 ${selectedIds.length} 封邮件记录`);
      setSelectedIds([]);
      setShowDeleteDialog(false);
      loadEmails();
    } catch (error) {
      console.error('删除失败:', error);
      toast.error('删除失败');
    }
  };

  // 重试发送失败的邮件
  const handleRetry = async (email: SentEmail) => {
    try {
      // 解析收件人地址
      const toAddresses = parseRecipients(email.to_addresses);
      const ccAddresses = parseRecipients(email.cc_addresses);
      const bccAddresses = parseRecipients(email.bcc_addresses);

      if (toAddresses.length === 0) {
        toast.error('收件人地址无效，无法重试发送');
        return;
      }

      // 构建发送请求
      const sendRequest = {
        account_uid: email.account_uid,
        to: toAddresses,
        cc: ccAddresses.length > 0 ? ccAddresses : undefined,
        bcc: bccAddresses.length > 0 ? bccAddresses : undefined,
        subject: email.subject,
        body: email.html_body || email.text_body || '',
        content_type: email.html_body ? 'text/html' : 'text/plain',
      };

      toast.loading('正在重新发送...', { id: 'retry-send' });
      
      const result = await emailService.sendEmail(sendRequest);
      
      if (result.success) {
        toast.success('邮件重新发送成功', { id: 'retry-send' });
        setShowDetailDialog(false);
        loadEmails(); // 刷新列表
      } else {
        toast.error('重新发送失败', { id: 'retry-send' });
      }
    } catch (error) {
      console.error('重试发送失败:', error);
      toast.error('重新发送失败，请稍后重试', { id: 'retry-send' });
    }
  };

  // 格式化相对时间
  const formatRelativeDate = (dateString?: string) => {
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
  const formatFullDate = (dateString?: string) => {
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
  const parseRecipients = (addressesJson: string | null | undefined): string[] => {
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
  const getStatusIcon = (status: string) => {
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
  const getStatusInfo = (status: string) => {
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

  // 切换选中
  const toggleSelect = (id: number) => {
    setSelectedIds((prev) =>
      prev.includes(id) ? prev.filter((i) => i !== id) : [...prev, id]
    );
  };

  // 全选/取消全选
  const isAllSelected =
    emails.length > 0 && selectedIds.length === emails.length;
  const handleSelectAll = () => {
    if (isAllSelected) {
      setSelectedIds([]);
    } else {
      setSelectedIds(emails.map((e) => e.id));
    }
  };

  // 查看邮件详情
  const handleViewDetail = (email: SentEmail) => {
    setSelectedEmail(email);
    setShowDetailDialog(true);
  };

  // 获取账户邮箱地址
  const getAccountEmail = (accountUid: string) => {
    const account = activeAccounts.find((a: Account) => a.uid === accountUid);
    return account?.email || accountUid;
  };

  // 是否显示账户标识
  const showAccountBadge = !filter.account_uid;

  return (
    <div className="flex h-full flex-col">
      {/* 工具栏 */}
      <div className="flex items-center justify-between border-b bg-background px-4 py-1.5">
        {/* 左侧 */}
        <div className="flex items-center gap-2">
          <Checkbox
            checked={isAllSelected}
            onCheckedChange={handleSelectAll}
            title={isAllSelected ? '取消全选' : '全选当前页'}
          />

          {/* 状态筛选 */}
          <div className="flex items-center gap-1">
            <Button
              variant={statusFilter === 'all' ? 'secondary' : 'ghost'}
              size="sm"
              onClick={() => handleStatusFilter('all')}
              className="h-7 px-2 text-xs"
            >
              全部
            </Button>
            <Button
              variant={statusFilter === 'sent' ? 'secondary' : 'ghost'}
              size="sm"
              onClick={() => handleStatusFilter('sent')}
              className="h-7 px-2 text-xs"
            >
              已发送
            </Button>
            <Button
              variant={statusFilter === 'failed' ? 'secondary' : 'ghost'}
              size="sm"
              onClick={() => handleStatusFilter('failed')}
              className="h-7 px-2 text-xs"
            >
              失败
            </Button>
          </div>

          <div className="h-4 w-px bg-border" />

          {selectedIds.length > 0 ? (
            <>
              <Badge variant="secondary" className="h-6 text-xs px-2">
                {selectedIds.length} 已选择
              </Badge>
              <Button
                variant="ghost"
                size="sm"
                onClick={() => setShowDeleteDialog(true)}
                title="删除记录"
                className="h-7 w-7 p-0 text-destructive hover:text-destructive"
              >
                <Trash2 className="h-3.5 w-3.5" />
              </Button>
            </>
          ) : (
            <span className="text-xs text-muted-foreground">
              共 {total} 封
            </span>
          )}
        </div>

        {/* 右侧 */}
        <div className="flex items-center gap-2">
          <div className="flex items-center gap-1">
            <Input
              placeholder="搜索..."
              value={searchKeyword}
              onChange={(e) => setSearchKeyword(e.target.value)}
              onKeyDown={(e) => e.key === 'Enter' && handleSearch()}
              className="h-7 w-32 text-xs"
            />
            <Button
              variant="ghost"
              size="sm"
              onClick={handleSearch}
              className="h-7 w-7 p-0"
            >
              <Search className="h-3.5 w-3.5" />
            </Button>
          </div>

          <Select
            value={filter.account_uid || 'all'}
            onValueChange={handleAccountFilter}
          >
            <SelectTrigger className="h-7 w-32 text-xs">
              <SelectValue placeholder="所有账户" />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="all">所有账户</SelectItem>
              {activeAccounts.map((account: Account) => (
                <SelectItem key={account.uid} value={account.uid}>
                  {account.email}
                </SelectItem>
              ))}
            </SelectContent>
          </Select>

          <Button
            variant="ghost"
            size="sm"
            onClick={loadEmails}
            disabled={isLoading}
            className="h-7 w-7 p-0"
          >
            <RefreshCw className={cn('h-3.5 w-3.5', isLoading && 'animate-spin')} />
          </Button>

          <DropdownMenu>
            <DropdownMenuTrigger asChild>
              <Button variant="ghost" size="sm" className="h-7 w-7 p-0">
                <MoreVertical className="h-3.5 w-3.5" />
              </Button>
            </DropdownMenuTrigger>
            <DropdownMenuContent align="end">
              <DropdownMenuItem onClick={() => setSelectedIds(emails.map((e) => e.id))}>
                选择全部
              </DropdownMenuItem>
              <DropdownMenuItem onClick={() => setSelectedIds([])}>
                取消选择
              </DropdownMenuItem>
              <DropdownMenuSeparator />
              <DropdownMenuItem
                onClick={() => setShowDeleteDialog(true)}
                disabled={selectedIds.length === 0}
                className="text-destructive"
              >
                删除选中
              </DropdownMenuItem>
            </DropdownMenuContent>
          </DropdownMenu>
        </div>
      </div>

      {/* 邮件列表 */}
      <ScrollArea className="flex-1">
        {isLoading ? (
          <div className="flex items-center justify-center h-32">
            <RefreshCw className="h-6 w-6 animate-spin text-muted-foreground" />
          </div>
        ) : emails.length === 0 ? (
          <div className="flex flex-col items-center justify-center h-64 text-muted-foreground">
            <Send className="h-12 w-12 mb-4 opacity-20" />
            <p className="text-lg font-medium">暂无已发送邮件</p>
          </div>
        ) : (
          <div>
            {emails.map((email) => {
              const recipients = parseRecipients(email.to_addresses);
              const isSelected = selectedIds.includes(email.id);

              return (
                <div
                  key={email.id}
                  className={cn(
                    'flex cursor-pointer items-start gap-3 border-b px-4 py-3 transition-colors hover:bg-accent',
                    isSelected && 'bg-accent'
                  )}
                  onClick={() => handleViewDetail(email)}
                >
                  {/* 左侧：复选框和状态图标 */}
                  <div className="mt-1 flex-shrink-0 flex items-center gap-2">
                    <Checkbox
                      checked={isSelected}
                      onCheckedChange={() => toggleSelect(email.id)}
                      onClick={(e) => e.stopPropagation()}
                    />
                    {getStatusIcon(email.status)}
                  </div>

                  {/* 中间：邮件信息 */}
                  <div className="min-w-0 flex-1">
                    {/* 收件人 */}
                    <div className="flex items-center gap-2">
                      <span className="truncate text-sm font-medium">
                        收件人: {recipients.length > 0 ? recipients[0] : '(无)'}
                        {recipients.length > 1 && ` +${recipients.length - 1}`}
                      </span>
                      {email.has_attachments && (
                        <Paperclip className="h-3 w-3 flex-shrink-0 text-muted-foreground" />
                      )}
                    </div>

                    {/* 主题 */}
                    <div className="truncate text-sm text-muted-foreground">
                      {email.subject || '(无主题)'}
                    </div>

                    {/* 预览/错误信息 */}
                    <div
                      className="text-xs text-muted-foreground truncate"
                      style={{ maxWidth: '100%' }}
                    >
                      {email.status === 'failed' && email.error_message ? (
                        <span className="text-destructive">
                          发送失败: {email.error_message}
                        </span>
                      ) : (
                        email.text_body?.slice(0, 80) || '(无内容)'
                      )}
                    </div>
                  </div>

                  {/* 右侧：账户和时间 */}
                  <div className="flex flex-col items-end gap-1 flex-shrink-0">
                    {showAccountBadge && (
                      <Badge
                        variant="secondary"
                        className="text-xs px-1.5 py-0 h-4 bg-muted/30 text-muted-foreground border-0 font-normal"
                      >
                        {getAccountEmail(email.account_uid)}
                      </Badge>
                    )}
                    <div className="text-xs text-muted-foreground">
                      {formatRelativeDate(email.sent_at || email.created_at)}
                    </div>
                  </div>
                </div>
              );
            })}
          </div>
        )}
      </ScrollArea>

      {/* 分页 */}
      {totalPages > 1 && (
        <div className="flex items-center justify-between border-t bg-background px-4 py-2">
          <div className="text-sm text-muted-foreground">
            第 {page} 页，共 {totalPages} 页
          </div>
          <div className="flex items-center gap-2">
            <Button
              variant="outline"
              size="sm"
              onClick={() => setPage((p) => Math.max(1, p - 1))}
              disabled={page === 1}
            >
              <ChevronLeft className="h-4 w-4" />
              上一页
            </Button>
            <Button
              variant="outline"
              size="sm"
              onClick={() => setPage((p) => Math.min(totalPages, p + 1))}
              disabled={page === totalPages}
            >
              下一页
              <ChevronRight className="h-4 w-4" />
            </Button>
          </div>
        </div>
      )}

      {/* 删除确认 */}
      <AlertDialog open={showDeleteDialog} onOpenChange={setShowDeleteDialog}>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>确认删除</AlertDialogTitle>
            <AlertDialogDescription>
              确定要删除 {selectedIds.length} 封已发送邮件记录吗？
              此操作仅删除本地记录，不会影响已发送到收件人的邮件。
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel>取消</AlertDialogCancel>
            <AlertDialogAction
              onClick={handleDelete}
              className="bg-destructive text-destructive-foreground hover:bg-destructive/90"
            >
              删除
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>

      {/* 邮件详情 */}
      <Dialog open={showDetailDialog} onOpenChange={setShowDetailDialog}>
        <DialogContent className="max-w-2xl max-h-[80vh] flex flex-col">
          <DialogHeader>
            <DialogTitle className="flex items-center gap-2">
              <Send className="h-5 w-5" />
              已发送邮件详情
            </DialogTitle>
          </DialogHeader>

          {selectedEmail && (
            <div className="flex-1 overflow-y-auto space-y-4">
              {/* 状态 */}
              <div className="flex items-center gap-2">
                <Badge
                  variant="outline"
                  className={cn('gap-1', getStatusInfo(selectedEmail.status).color)}
                >
                  {getStatusIcon(selectedEmail.status)}
                  {getStatusInfo(selectedEmail.status).label}
                </Badge>
                {selectedEmail.has_attachments && (
                  <Badge variant="outline" className="gap-1">
                    <Paperclip className="h-3.5 w-3.5" />
                    {selectedEmail.attachments_count || 1} 个附件
                  </Badge>
                )}
              </div>

              {/* 错误信息 */}
              {selectedEmail.status === 'failed' && selectedEmail.error_message && (
                <div className="p-3 bg-destructive/10 border border-destructive/20 rounded-md">
                  <p className="text-sm text-destructive">
                    <strong>发送失败：</strong>{selectedEmail.error_message}
                  </p>
                </div>
              )}

              {/* 邮件信息 */}
              <div className="space-y-2 text-sm">
                <div className="flex">
                  <span className="w-16 text-muted-foreground">发件人</span>
                  <span>
                    {getAccountEmail(selectedEmail.account_uid)}
                  </span>
                </div>
                <div className="flex">
                  <span className="w-16 text-muted-foreground">收件人</span>
                  <span>{parseRecipients(selectedEmail.to_addresses).join(', ') || '-'}</span>
                </div>
                {selectedEmail.cc_addresses && (
                  <div className="flex">
                    <span className="w-16 text-muted-foreground">抄送</span>
                    <span>{parseRecipients(selectedEmail.cc_addresses).join(', ')}</span>
                  </div>
                )}
                <div className="flex">
                  <span className="w-16 text-muted-foreground">主题</span>
                  <span className="font-medium">{selectedEmail.subject || '(无主题)'}</span>
                </div>
                <div className="flex">
                  <span className="w-16 text-muted-foreground">时间</span>
                  <span>{formatFullDate(selectedEmail.sent_at || selectedEmail.created_at)}</span>
                </div>
              </div>

              {/* 邮件正文 */}
              <div className="border-t pt-4">
                <div className="p-4 bg-muted/30 rounded-md">
                  {selectedEmail.html_body ? (
                    <div
                      className="prose prose-sm max-w-none dark:prose-invert"
                      dangerouslySetInnerHTML={{ __html: selectedEmail.html_body }}
                    />
                  ) : selectedEmail.text_body ? (
                    <pre className="whitespace-pre-wrap text-sm font-sans">
                      {selectedEmail.text_body}
                    </pre>
                  ) : (
                    <p className="text-muted-foreground text-sm">(无内容)</p>
                  )}
                </div>
              </div>
            </div>
          )}

          {/* 底部操作 */}
          <div className="flex items-center justify-end gap-2 pt-4 border-t">
            {selectedEmail?.status === 'failed' && (
              <Button variant="outline" onClick={() => selectedEmail && handleRetry(selectedEmail)}>
                <RotateCcw className="h-4 w-4 mr-1" />
                重试发送
              </Button>
            )}
            <Button variant="outline" onClick={() => setShowDetailDialog(false)}>
              关闭
            </Button>
          </div>
        </DialogContent>
      </Dialog>
    </div>
  );
};

export default SentPage;
