import { useState, useMemo, useEffect } from 'react';
import { useNavigate } from 'react-router-dom';
import { EmailList } from '../components/email/EmailList';
import { useEmails } from '../hooks/useEmails';
import { useAccounts } from '../hooks/useAccounts';
import { Email } from '../types';
import { Button } from '../components/ui/button';
import { ChevronLeft, ChevronRight, Mail, MailOpen, Star, Archive, Trash2, RefreshCw, MoreVertical, Undo2 } from 'lucide-react';
import { cn } from '../lib/utils';
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
import { Badge } from '../components/ui/badge';
import { Checkbox } from '../components/ui/checkbox';
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from '../components/ui/dropdown-menu';

type FilterType = 'all' | 'unread';

export const InboxPage = () => {
  const navigate = useNavigate();
  const {
    emails,
    total,
    page,
    totalPages,
    isLoading,
    filter,
    setFilter,
    setPage,
    markAsRead,
    markAsUnread,
    toggleStar,
    archiveEmail,
    deleteEmail,
    restoreEmail,
    batchPermanentDelete,
    emptyTrash,
    markAllAsRead,
    refresh,
  } = useEmails();

  const { accounts } = useAccounts();

  const [selectedEmails, setSelectedEmails] = useState<number[]>([]);
  const [selectedEmail, setSelectedEmail] = useState<Email | null>(null);
  const [filterType, setFilterType] = useState<FilterType>('all');
  const [showMarkAllReadDialog, setShowMarkAllReadDialog] = useState(false);
  const [showDeleteDialog, setShowDeleteDialog] = useState(false);
  const [showPermanentDeleteDialog, setShowPermanentDeleteDialog] = useState(false);
  const [showEmptyTrashDialog, setShowEmptyTrashDialog] = useState(false);

  // 是否在回收站视图
  const isTrashView = filter.is_deleted === true;

  // 全选/取消全选
  const isAllSelected = emails.length > 0 && selectedEmails.length === emails.length;
  const handleSelectAll = () => {
    if (isAllSelected) {
      setSelectedEmails([]);
    } else {
      setSelectedEmails(emails.map(e => e.id));
    }
  };

  // 判断是否显示邮箱标识：当选中"所有邮箱"时显示
  const showAccountBadge = !filter.account_uid;

  const handleEmailClick = (email: Email) => {
    setSelectedEmail(email);
    // 自动标记为已读
    if (!email.is_read) {
      markAsRead([email.id]);
    }
    // 跳转到详情页：垃圾箱来源时带上 include_deleted 标志
    if (filter.is_deleted) {
      navigate(`/email/${email.id}?include_deleted=true`);
    } else {
      navigate(`/email/${email.id}`);
    }
  };

  const handleMarkAsRead = () => {
    if (selectedEmails.length > 0) {
      markAsRead(selectedEmails);
      setSelectedEmails([]);
    }
  };

  const handleMarkAsUnread = () => {
    if (selectedEmails.length > 0) {
      markAsUnread(selectedEmails);
      setSelectedEmails([]);
    }
  };

  const handleToggleStar = () => {
    if (selectedEmails.length > 0) {
      selectedEmails.forEach((id) => {
        const email = emails.find((e) => e.id === id);
        if (email) {
          toggleStar(id, email.is_starred);
        }
      });
      setSelectedEmails([]);
    }
  };

  const handleArchive = () => {
    if (selectedEmails.length > 0) {
      selectedEmails.forEach((id) => archiveEmail(id));
      setSelectedEmails([]);
    }
  };

  const handleDelete = () => {
    if (selectedEmails.length > 0) {
      setShowDeleteDialog(true);
    }
  };

  const confirmDelete = () => {
    selectedEmails.forEach((id) => deleteEmail(id));
    setSelectedEmails([]);
    setShowDeleteDialog(false);
  };

  // 恢复邮件（回收站）
  const handleRestore = () => {
    if (selectedEmails.length > 0) {
      selectedEmails.forEach((id) => restoreEmail(id));
      setSelectedEmails([]);
    }
  };

  // 永久删除（回收站）
  const handlePermanentDelete = () => {
    if (selectedEmails.length > 0) {
      setShowPermanentDeleteDialog(true);
    }
  };

  const confirmPermanentDelete = async () => {
    await batchPermanentDelete(selectedEmails);
    setSelectedEmails([]);
    setShowPermanentDeleteDialog(false);
  };

  // 清空回收站
  const handleEmptyTrash = () => {
    setShowEmptyTrashDialog(true);
  };

  const confirmEmptyTrash = async () => {
    await emptyTrash();
    setShowEmptyTrashDialog(false);
  };

  // 生成删除提示文本
  const deleteMessage = useMemo(() => {
    if (selectedEmails.length === 0 || !emails.length) {
      return '确定要删除这些邮件吗？此操作仅在本地生效，不会同步到邮箱服务器。';
    }

    // 获取选中邮件所属的账号
    const selectedEmailsData = emails.filter(e => selectedEmails.includes(e.id));
    const accountUids = new Set(selectedEmailsData.map(e => e.account_uid));

    // 检查这些账号是否都启用了服务器软删除
    const selectedAccounts = accounts.filter(acc => accountUids.has(acc.uid));
    const allHaveServerDelete = selectedAccounts.length > 0 &&
                                selectedAccounts.every(acc => acc.server_delete_policy === 'soft');

    if (allHaveServerDelete) {
      return `确定要删除 ${selectedEmails.length} 封邮件吗？删除后邮件将从本地和服务器垃圾箱中移除。`;
    }

    return `确定要删除 ${selectedEmails.length} 封邮件吗？此操作仅在本地生效，不会同步到邮箱服务器。`;
  }, [selectedEmails, emails, accounts]);

  const handleMarkAllAsRead = () => {
    setShowMarkAllReadDialog(true);
  };

  const confirmMarkAllAsRead = async () => {
    const accountUid = filter.account_uid;
    await markAllAsRead(accountUid);
    setShowMarkAllReadDialog(false);
  };

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

  const handleFilterChange = (type: FilterType) => {
    setFilterType(type);
    const newFilter = { ...filter };

    if (type === 'unread') {
      newFilter.is_read = false;
    } else {
      delete newFilter.is_read;
    }

    setFilter(newFilter);
    setPage(1); // 重置到第一页
  };

  // 当全局筛选变为未读/全部时，同步更新顶部筛选按钮的选中状态
  useEffect(() => {
    if (filter.is_read === false) {
      setFilterType('unread');
    } else {
      setFilterType('all');
    }
  }, [filter.is_read]);

  return (
    <div className="flex h-full flex-col">
      {/* 合并的工具栏 */}
      <div className="flex items-center justify-between border-b bg-background px-4 py-1.5">
        {/* 左侧：全选、筛选按钮和选择信息 */}
        <div className="flex items-center gap-2">
          {/* 全选复选框 */}
          <Checkbox
            checked={isAllSelected}
            onCheckedChange={handleSelectAll}
            title={isAllSelected ? '取消全选' : '全选当前页'}
          />

          {/* 筛选按钮 */}
          <div className="flex items-center gap-1">
            <Button
              variant={filterType === 'all' ? 'secondary' : 'ghost'}
              size="sm"
              onClick={() => handleFilterChange('all')}
              className={cn(
                'h-7 px-2 text-xs',
                filterType === 'all' && 'bg-secondary'
              )}
            >
              全部
            </Button>
            <Button
              variant={filterType === 'unread' ? 'secondary' : 'ghost'}
              size="sm"
              onClick={() => handleFilterChange('unread')}
              className={cn(
                'h-7 px-2 text-xs',
                filterType === 'unread' && 'bg-secondary'
              )}
            >
              未读
            </Button>
          </div>

          {/* 分隔线 */}
          <div className="h-4 w-px bg-border" />

          {/* 选择信息和操作按钮 */}
          {selectedEmails.length > 0 ? (
            <>
              <Badge variant="secondary" className="h-6 text-xs px-2">{selectedEmails.length} 已选择</Badge>
              <div className="flex items-center gap-1">
                {isTrashView ? (
                  // 回收站视图：恢复和永久删除
                  <>
                    <Button
                      variant="ghost"
                      size="sm"
                      onClick={handleRestore}
                      title="恢复"
                      className="h-7 w-7 p-0"
                    >
                      <Undo2 className="h-3.5 w-3.5" />
                    </Button>
                    <Button
                      variant="ghost"
                      size="sm"
                      onClick={handlePermanentDelete}
                      title="永久删除"
                      className="h-7 w-7 p-0 text-destructive hover:text-destructive"
                    >
                      <Trash2 className="h-3.5 w-3.5" />
                    </Button>
                  </>
                ) : (
                  // 普通视图：标记、星标、归档、删除
                  <>
                    <Button
                      variant="ghost"
                      size="sm"
                      onClick={handleMarkAsRead}
                      title="标记为已读"
                      className="h-7 w-7 p-0"
                    >
                      <MailOpen className="h-3.5 w-3.5" />
                    </Button>
                    <Button
                      variant="ghost"
                      size="sm"
                      onClick={handleMarkAsUnread}
                      title="标记为未读"
                      className="h-7 w-7 p-0"
                    >
                      <Mail className="h-3.5 w-3.5" />
                    </Button>
                    <Button
                      variant="ghost"
                      size="sm"
                      onClick={handleToggleStar}
                      title="添加星标"
                      className="h-7 w-7 p-0"
                    >
                      <Star className="h-3.5 w-3.5" />
                    </Button>
                    <Button
                      variant="ghost"
                      size="sm"
                      onClick={handleArchive}
                      title="归档"
                      className="h-7 w-7 p-0"
                    >
                      <Archive className="h-3.5 w-3.5" />
                    </Button>
                    <Button
                      variant="ghost"
                      size="sm"
                      onClick={handleDelete}
                      title="删除"
                      className="h-7 w-7 p-0"
                    >
                      <Trash2 className="h-3.5 w-3.5" />
                    </Button>
                  </>
                )}
              </div>
            </>
          ) : (
            <span className="text-xs text-muted-foreground">
              共 {total} 封{isTrashView ? '已删除' : ''}邮件
            </span>
          )}
        </div>

        {/* 右侧：刷新和更多操作 */}
        <div className="flex items-center gap-1">
          <Button
            variant="ghost"
            size="sm"
            onClick={refresh}
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
              {isTrashView ? (
                // 回收站视图菜单
                <DropdownMenuItem
                  onClick={handleEmptyTrash}
                  className="text-destructive focus:text-destructive"
                >
                  <Trash2 className="mr-2 h-4 w-4" />
                  清空回收站
                </DropdownMenuItem>
              ) : (
                // 普通视图菜单
                <>
                  <DropdownMenuItem onClick={handleMarkAllAsRead}>
                    全部标记为已读
                  </DropdownMenuItem>
                  <DropdownMenuItem>选择全部</DropdownMenuItem>
                  <DropdownMenuItem>取消选择</DropdownMenuItem>
                </>
              )}
            </DropdownMenuContent>
          </DropdownMenu>
        </div>
      </div>

      {/* 邮件列表 */}
      <div className="flex-1 overflow-hidden">
        <EmailList
          emails={emails}
          selectedEmailId={selectedEmail?.id}
          selectedEmailIds={selectedEmails}
          onEmailClick={handleEmailClick}
          onSelectionChange={setSelectedEmails}
          isLoading={isLoading}
          showAccountBadge={showAccountBadge}
          accounts={accounts}
          onToggleStar={(email) => toggleStar(email.id, email.is_starred)}
          enableMultiSelect={true}
        />
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

      {/* 全部标记为已读确认对话框 */}
      <AlertDialog open={showMarkAllReadDialog} onOpenChange={setShowMarkAllReadDialog}>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>确认标记为已读</AlertDialogTitle>
            <AlertDialogDescription>
              {filter.account_uid ? (
                <>
                  将 <strong>当前账号</strong> 的所有未读邮件标记为已读。
                </>
              ) : (
                <>
                  将 <strong>所有账号</strong> 的所有未读邮件标记为已读。
                </>
              )}
              <br />
              <br />
              此操作仅在本地生效，不会同步到邮箱服务器。
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel>取消</AlertDialogCancel>
            <AlertDialogAction onClick={confirmMarkAllAsRead}>
              确认标记
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>

      {/* 删除邮件确认对话框 */}
      <AlertDialog open={showDeleteDialog} onOpenChange={setShowDeleteDialog}>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>确认删除</AlertDialogTitle>
            <AlertDialogDescription>
              {deleteMessage}
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel>取消</AlertDialogCancel>
            <AlertDialogAction onClick={confirmDelete} className="bg-destructive text-destructive-foreground hover:bg-destructive/90">
              确认删除
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>

      {/* 永久删除确认对话框 */}
      <AlertDialog open={showPermanentDeleteDialog} onOpenChange={setShowPermanentDeleteDialog}>
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
              onClick={confirmPermanentDelete}
              className="bg-destructive text-destructive-foreground hover:bg-destructive/90"
            >
              永久删除
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>

      {/* 清空回收站确认对话框 */}
      <AlertDialog open={showEmptyTrashDialog} onOpenChange={setShowEmptyTrashDialog}>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle className="text-destructive">清空回收站</AlertDialogTitle>
            <AlertDialogDescription>
              确定要清空回收站吗？
              <br />
              <br />
              <span className="text-destructive font-medium">
                这将永久删除所有 {total} 封已删除邮件，此操作无法撤销！
              </span>
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel>取消</AlertDialogCancel>
            <AlertDialogAction
              onClick={confirmEmptyTrash}
              className="bg-destructive text-destructive-foreground hover:bg-destructive/90"
            >
              清空回收站
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </div>
  );
};
