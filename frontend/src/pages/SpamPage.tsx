import { useState, useEffect, useCallback } from 'react';
import { useNavigate } from 'react-router-dom';
import { ShieldAlert, Trash2, RefreshCw, ChevronLeft, ChevronRight, MoreVertical, Undo2 } from 'lucide-react';
import { Button } from '../components/ui/button';
import { Badge } from '../components/ui/badge';
import { EmailList } from '../components/email/EmailList';
import { useAccounts } from '../hooks/useAccounts';
import { Email } from '../types';
import { spamService, SpamStats } from '../services/spamService';
import { toast } from 'sonner';
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
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from '../components/ui/dropdown-menu';

export const SpamPage = () => {
  const navigate = useNavigate();
  const { accounts } = useAccounts();

  // 状态
  const [emails, setEmails] = useState<Email[]>([]);
  const [total, setTotal] = useState(0);
  const [page, setPage] = useState(1);
  const [pageSize] = useState(20);
  const [isLoading, setIsLoading] = useState(false);
  const [selectedEmails, setSelectedEmails] = useState<number[]>([]);
  const [selectedEmail, setSelectedEmail] = useState<Email | null>(null);
  const [stats, setStats] = useState<SpamStats | null>(null);

  // 对话框状态
  const [showDeleteDialog, setShowDeleteDialog] = useState(false);
  const [showEmptyDialog, setShowEmptyDialog] = useState(false);
  const [showUnmarkDialog, setShowUnmarkDialog] = useState(false);

  // 计算总页数
  const totalPages = Math.ceil(total / pageSize);

  // 加载垃圾邮件列表
  const loadSpamEmails = useCallback(async () => {
    setIsLoading(true);
    try {
      const response = await spamService.getSpamEmails({
        page,
        page_size: pageSize,
      });
      setEmails(response.data || []);
      setTotal(response.total || 0);
    } catch (error) {
      console.error('Failed to load spam emails:', error);
      toast.error('加载垃圾邮件失败');
    } finally {
      setIsLoading(false);
    }
  }, [page, pageSize]);

  // 加载统计信息
  const loadStats = useCallback(async () => {
    try {
      const data = await spamService.getSpamStats();
      setStats(data);
    } catch (error) {
      console.error('Failed to load spam stats:', error);
    }
  }, []);

  // 初始加载
  useEffect(() => {
    loadSpamEmails();
    loadStats();
  }, [loadSpamEmails, loadStats]);

  // 刷新
  const handleRefresh = () => {
    loadSpamEmails();
    loadStats();
  };

  // 点击邮件
  const handleEmailClick = (email: Email) => {
    setSelectedEmail(email);
    navigate(`/email/${email.id}`);
  };

  // 取消垃圾邮件标记（恢复到收件箱）
  const handleUnmark = async () => {
    if (selectedEmails.length === 0) return;

    try {
      await spamService.unmarkAsSpam(selectedEmails);
      toast.success(`已将 ${selectedEmails.length} 封邮件移出垃圾箱`);
      setSelectedEmails([]);
      setShowUnmarkDialog(false);
      handleRefresh();
    } catch (error) {
      console.error('Failed to unmark spam:', error);
      toast.error('操作失败');
    }
  };

  // 批量删除
  const handleDelete = async () => {
    if (selectedEmails.length === 0) return;

    try {
      const result = await spamService.batchDeleteSpam(selectedEmails);
      toast.success(`已删除 ${result.deleted_count} 封邮件`);
      setSelectedEmails([]);
      setShowDeleteDialog(false);
      handleRefresh();
    } catch (error) {
      console.error('Failed to delete spam:', error);
      toast.error('删除失败');
    }
  };

  // 清空垃圾箱
  const handleEmptySpam = async () => {
    try {
      const result = await spamService.emptySpamFolder();
      toast.success(`已清空垃圾箱，删除 ${result.deleted_count} 封邮件`);
      setShowEmptyDialog(false);
      handleRefresh();
    } catch (error) {
      console.error('Failed to empty spam folder:', error);
      toast.error('清空垃圾箱失败');
    }
  };

  // 分页
  const handlePreviousPage = () => {
    if (page > 1) {
      setPage(page - 1);
    }
  };

  const handleNextPage = () => {
    if (page < totalPages) {
      setPage(page + 1);
    }
  };

  return (
    <div className="flex h-full flex-col">
      {/* 工具栏 */}
      <div className="flex items-center justify-between border-b bg-background px-4 py-1.5">
        {/* 左侧：标题和选择信息 */}
        <div className="flex items-center gap-2">
          <ShieldAlert className="h-4 w-4 text-orange-500" />
          <span className="font-medium text-sm">垃圾邮件</span>

          {/* 分隔线 */}
          <div className="h-4 w-px bg-border" />

          {/* 选择信息和操作按钮 */}
          {selectedEmails.length > 0 ? (
            <>
              <Badge variant="secondary" className="h-6 text-xs px-2">
                {selectedEmails.length} 已选择
              </Badge>
              <div className="flex items-center gap-1">
                <Button
                  variant="ghost"
                  size="sm"
                  onClick={() => setShowUnmarkDialog(true)}
                  title="移出垃圾箱"
                  className="h-7 w-7 p-0"
                >
                  <Undo2 className="h-3.5 w-3.5" />
                </Button>
                <Button
                  variant="ghost"
                  size="sm"
                  onClick={() => setShowDeleteDialog(true)}
                  title="永久删除"
                  className="h-7 w-7 p-0 text-destructive hover:text-destructive"
                >
                  <Trash2 className="h-3.5 w-3.5" />
                </Button>
              </div>
            </>
          ) : (
            <span className="text-xs text-muted-foreground">
              共 {total} 封垃圾邮件
              {stats && stats.unread_count > 0 && (
                <span className="ml-1">（{stats.unread_count} 封未读）</span>
              )}
            </span>
          )}
        </div>

        {/* 右侧：刷新和更多操作 */}
        <div className="flex items-center gap-1">
          <Button
            variant="ghost"
            size="sm"
            onClick={handleRefresh}
            disabled={isLoading}
            title="刷新"
            className="h-7 w-7 p-0"
          >
            <RefreshCw
              className={`h-3.5 w-3.5 ${isLoading ? 'animate-spin' : ''}`}
            />
          </Button>

          <DropdownMenu>
            <DropdownMenuTrigger asChild>
              <Button variant="ghost" size="sm" title="更多操作" className="h-7 w-7 p-0">
                <MoreVertical className="h-3.5 w-3.5" />
              </Button>
            </DropdownMenuTrigger>
            <DropdownMenuContent align="end">
              <DropdownMenuItem
                onClick={() => setShowEmptyDialog(true)}
                className="text-destructive focus:text-destructive"
              >
                <Trash2 className="mr-2 h-4 w-4" />
                清空垃圾箱
              </DropdownMenuItem>
            </DropdownMenuContent>
          </DropdownMenu>
        </div>
      </div>

      {/* 邮件列表 */}
      <div className="flex-1 overflow-hidden">
        {emails.length === 0 && !isLoading ? (
          <div className="flex flex-col items-center justify-center h-full text-muted-foreground">
            <ShieldAlert className="h-12 w-12 mb-4 text-muted-foreground/50" />
            <p className="text-lg">垃圾箱是空的</p>
            <p className="text-sm mt-1">被标记为垃圾邮件的邮件会出现在这里</p>
          </div>
        ) : (
          <EmailList
            emails={emails}
            selectedEmailId={selectedEmail?.id}
            onEmailClick={handleEmailClick}
            isLoading={isLoading}
            showAccountBadge={true}
            accounts={accounts}
          />
        )}
      </div>

      {/* 分页控制 */}
      {totalPages > 1 && (
        <div className="flex items-center justify-between border-t bg-background px-4 py-2">
          <div className="text-sm text-muted-foreground">
            第 {page} 页，共 {totalPages} 页 · 总计 {total} 封邮件
          </div>
          <div className="flex items-center gap-2">
            <Button
              variant="outline"
              size="sm"
              onClick={handlePreviousPage}
              disabled={page === 1}
            >
              <ChevronLeft className="h-4 w-4" />
              上一页
            </Button>
            <Button
              variant="outline"
              size="sm"
              onClick={handleNextPage}
              disabled={page === totalPages}
            >
              下一页
              <ChevronRight className="h-4 w-4" />
            </Button>
          </div>
        </div>
      )}

      {/* 取消垃圾邮件标记确认对话框 */}
      <AlertDialog open={showUnmarkDialog} onOpenChange={setShowUnmarkDialog}>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>确认移出垃圾箱</AlertDialogTitle>
            <AlertDialogDescription>
              确定要将 {selectedEmails.length} 封邮件移出垃圾箱吗？
              <br />
              <br />
              这些邮件将恢复到收件箱。
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel>取消</AlertDialogCancel>
            <AlertDialogAction onClick={handleUnmark}>
              确认移出
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>

      {/* 删除确认对话框 */}
      <AlertDialog open={showDeleteDialog} onOpenChange={setShowDeleteDialog}>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>确认永久删除</AlertDialogTitle>
            <AlertDialogDescription>
              确定要永久删除 {selectedEmails.length} 封邮件吗？
              <br />
              <br />
              <span className="text-destructive font-medium">
                此操作无法撤销！
              </span>
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel>取消</AlertDialogCancel>
            <AlertDialogAction
              onClick={handleDelete}
              className="bg-destructive text-destructive-foreground hover:bg-destructive/90"
            >
              永久删除
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>

      {/* 清空垃圾箱确认对话框 */}
      <AlertDialog open={showEmptyDialog} onOpenChange={setShowEmptyDialog}>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle className="text-destructive">清空垃圾箱</AlertDialogTitle>
            <AlertDialogDescription>
              确定要清空垃圾箱吗？
              <br />
              <br />
              <span className="text-destructive font-medium">
                这将永久删除所有 {total} 封垃圾邮件，此操作无法撤销！
              </span>
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel>取消</AlertDialogCancel>
            <AlertDialogAction
              onClick={handleEmptySpam}
              className="bg-destructive text-destructive-foreground hover:bg-destructive/90"
            >
              清空垃圾箱
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </div>
  );
};

export default SpamPage;
