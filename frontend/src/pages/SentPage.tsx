/**
 * 已发送邮件页面
 */

import { useState, useEffect, useCallback } from 'react';
import { emailService } from '../services/emailService';
import { useAccounts } from '../hooks/useAccounts';
import type { SentEmail, SentEmailFilter, Account } from '../types';
import { Button } from '../components/ui/button';
import { Badge } from '../components/ui/badge';
import { Input } from '../components/ui/input';
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
import { ScrollArea } from '../components/ui/scroll-area';
import { 
  ChevronLeft, 
  ChevronRight, 
  RefreshCw, 
  Search, 
  Trash2,
  CheckCircle,
  XCircle,
  Clock,
  Paperclip,
} from 'lucide-react';
import { format } from 'date-fns';
import { zhCN } from 'date-fns/locale';
import { cn } from '../lib/utils';
import toast from 'react-hot-toast';

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

  // 选中状态
  const [selectedIds, setSelectedIds] = useState<number[]>([]);
  const [showDeleteDialog, setShowDeleteDialog] = useState(false);

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
    setFilter(prev => ({
      ...prev,
      keyword: searchKeyword || undefined,
    }));
    setPage(1);
  };

  // 账户筛选
  const handleAccountFilter = (accountUid: string) => {
    setFilter(prev => ({
      ...prev,
      account_uid: accountUid === 'all' ? undefined : accountUid,
    }));
    setPage(1);
  };

  // 状态筛选
  const handleStatusFilter = (status: string) => {
    setFilter(prev => ({
      ...prev,
      status: status === 'all' ? undefined : status as 'sent' | 'failed' | 'pending',
    }));
    setPage(1);
  };

  // 删除邮件
  const handleDelete = async () => {
    try {
      for (const id of selectedIds) {
        await emailService.deleteSentEmail(id);
      }
      toast.success(`已删除 ${selectedIds.length} 封邮件`);
      setSelectedIds([]);
      setShowDeleteDialog(false);
      loadEmails();
    } catch (error) {
      console.error('删除失败:', error);
      toast.error('删除失败');
    }
  };

  // 格式化日期
  const formatDate = (dateString?: string) => {
    if (!dateString) return '-';
    try {
      return format(new Date(dateString), 'MM-dd HH:mm', { locale: zhCN });
    } catch {
      return dateString;
    }
  };

  // 解析收件人
  const parseRecipients = (addressesJson: string): string[] => {
    try {
      return JSON.parse(addressesJson);
    } catch {
      return [];
    }
  };

  // 获取状态图标和颜色
  const getStatusInfo = (status: string) => {
    switch (status) {
      case 'sent':
        return { icon: CheckCircle, color: 'text-green-500', label: '已发送' };
      case 'failed':
        return { icon: XCircle, color: 'text-red-500', label: '发送失败' };
      case 'pending':
        return { icon: Clock, color: 'text-yellow-500', label: '发送中' };
      default:
        return { icon: Clock, color: 'text-muted-foreground', label: status };
    }
  };

  // 切换选中
  const toggleSelect = (id: number) => {
    setSelectedIds(prev => 
      prev.includes(id) 
        ? prev.filter(i => i !== id)
        : [...prev, id]
    );
  };

  return (
    <div className="flex h-full flex-col">
      {/* 工具栏 */}
      <div className="flex items-center justify-between border-b bg-background px-4 py-2">
        <div className="flex items-center gap-2">
          <h1 className="text-lg font-medium">已发送邮件</h1>
          <Badge variant="secondary">{total}</Badge>
        </div>

        <div className="flex items-center gap-2">
          {/* 搜索 */}
          <div className="flex items-center gap-1">
            <Input
              placeholder="搜索主题或收件人..."
              value={searchKeyword}
              onChange={(e) => setSearchKeyword(e.target.value)}
              onKeyDown={(e) => e.key === 'Enter' && handleSearch()}
              className="h-8 w-48"
            />
            <Button variant="ghost" size="sm" onClick={handleSearch} className="h-8 w-8 p-0">
              <Search className="h-4 w-4" />
            </Button>
          </div>

          {/* 账户筛选 */}
          <Select
            value={filter.account_uid || 'all'}
            onValueChange={handleAccountFilter}
          >
            <SelectTrigger className="h-8 w-40">
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

          {/* 状态筛选 */}
          <Select
            value={filter.status || 'all'}
            onValueChange={handleStatusFilter}
          >
            <SelectTrigger className="h-8 w-28">
              <SelectValue placeholder="所有状态" />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="all">所有状态</SelectItem>
              <SelectItem value="sent">已发送</SelectItem>
              <SelectItem value="failed">发送失败</SelectItem>
              <SelectItem value="pending">发送中</SelectItem>
            </SelectContent>
          </Select>

          {/* 刷新 */}
          <Button
            variant="ghost"
            size="sm"
            onClick={loadEmails}
            disabled={isLoading}
            className="h-8 w-8 p-0"
          >
            <RefreshCw className={cn('h-4 w-4', isLoading && 'animate-spin')} />
          </Button>

          {/* 删除选中 */}
          {selectedIds.length > 0 && (
            <Button
              variant="ghost"
              size="sm"
              onClick={() => setShowDeleteDialog(true)}
              className="h-8 text-destructive hover:text-destructive"
            >
              <Trash2 className="h-4 w-4 mr-1" />
              删除 ({selectedIds.length})
            </Button>
          )}
        </div>
      </div>

      {/* 邮件列表 */}
      <ScrollArea className="flex-1">
        {isLoading ? (
          <div className="flex items-center justify-center h-32">
            <RefreshCw className="h-6 w-6 animate-spin text-muted-foreground" />
          </div>
        ) : emails.length === 0 ? (
          <div className="flex flex-col items-center justify-center h-32 text-muted-foreground">
            <p>暂无已发送邮件</p>
          </div>
        ) : (
          <div className="divide-y">
            {emails.map((email) => {
              const statusInfo = getStatusInfo(email.status);
              const StatusIcon = statusInfo.icon;
              const recipients = parseRecipients(email.to_addresses);

              return (
                <div
                  key={email.id}
                  className={cn(
                    'flex items-center gap-3 px-4 py-3 hover:bg-muted/50 cursor-pointer',
                    selectedIds.includes(email.id) && 'bg-muted/30'
                  )}
                  onClick={() => toggleSelect(email.id)}
                >
                  {/* 状态图标 */}
                  <StatusIcon className={cn('h-4 w-4 flex-shrink-0', statusInfo.color)} />

                  {/* 收件人 */}
                  <div className="w-48 flex-shrink-0 truncate">
                    <span className="text-sm">
                      {recipients.slice(0, 2).join(', ')}
                      {recipients.length > 2 && ` +${recipients.length - 2}`}
                    </span>
                  </div>

                  {/* 主题和预览 */}
                  <div className="flex-1 min-w-0">
                    <div className="flex items-center gap-2">
                      <span className="text-sm font-medium truncate">
                        {email.subject || '(无主题)'}
                      </span>
                      {email.has_attachments && (
                        <Paperclip className="h-3 w-3 text-muted-foreground flex-shrink-0" />
                      )}
                    </div>
                    {email.status === 'failed' && email.error_message && (
                      <p className="text-xs text-destructive truncate mt-0.5">
                        {email.error_message}
                      </p>
                    )}
                  </div>

                  {/* 发送时间 */}
                  <div className="text-xs text-muted-foreground flex-shrink-0">
                    {formatDate(email.sent_at || email.created_at)}
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
              onClick={() => setPage(p => Math.max(1, p - 1))}
              disabled={page === 1}
            >
              <ChevronLeft className="h-4 w-4" />
              上一页
            </Button>
            <Button
              variant="outline"
              size="sm"
              onClick={() => setPage(p => Math.min(totalPages, p + 1))}
              disabled={page === totalPages}
            >
              下一页
              <ChevronRight className="h-4 w-4" />
            </Button>
          </div>
        </div>
      )}

      {/* 删除确认对话框 */}
      <AlertDialog open={showDeleteDialog} onOpenChange={setShowDeleteDialog}>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>确认删除</AlertDialogTitle>
            <AlertDialogDescription>
              确定要删除 {selectedIds.length} 封已发送邮件记录吗？
              <br />
              此操作仅删除本地记录，不会影响已发送到收件人的邮件。
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel>取消</AlertDialogCancel>
            <AlertDialogAction onClick={handleDelete}>
              删除
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </div>
  );
};

export default SentPage;
