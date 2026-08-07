import { useState, useMemo, useEffect } from 'react';
import { useNavigate } from 'react-router-dom';
import { EmailList } from '../components/email/EmailList';
import { useEmails } from '../hooks/useEmails';
import { useEmailActions } from '../hooks/useEmailActions';
import { useAccounts } from '../hooks/useAccounts';
import { useGroupStore, ALL_ACCOUNTS_GROUP_ID, UNGROUPED_GROUP_ID } from '../stores/groupStore';
import { Email } from '../types';
import { Button } from '../components/ui/button';
import { Mail, MailOpen, Star, Archive, Trash2, RefreshCw, MoreVertical, Undo2, X, Folder } from 'lucide-react';
import { cn } from '../lib/utils';
import { Badge } from '../components/ui/badge';
import { Checkbox } from '../components/ui/checkbox';
import { EmailPagination } from '../components/email/EmailPagination';
import { ConfirmDialog } from '../components/email/ConfirmDialog';
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
    batchDeleteEmails,
    restoreEmail,
    batchPermanentDelete,
    emptyTrash,
    markAllAsRead,
    refresh,
  } = useEmails();

  const { accounts } = useAccounts();
  const { groups, setSelectedGroupId } = useGroupStore();

  const [selectedEmails, setSelectedEmails] = useState<number[]>([]);
  const [selectedEmail, setSelectedEmail] = useState<Email | null>(null);
  const [filterType, setFilterType] = useState<FilterType>('all');
  const { dialogs, isDeleting, setIsDeleting, openDialog, closeDialog } = useEmailActions();

  // 是否在回收站视图
  const isTrashView = filter.is_deleted === true;

  // 获取当前分组筛选信息
  const activeGroupFilter = useMemo(() => {
    if (filter.group_id === undefined) return null;
    if (filter.group_id === UNGROUPED_GROUP_ID) {
      return { id: UNGROUPED_GROUP_ID, name: '未分组' };
    }
    const group = groups.find(g => g.id === filter.group_id);
    return group ? { id: group.id, name: group.name } : null;
  }, [filter.group_id, groups]);

  // 清除分组筛选
  const clearGroupFilter = () => {
    const newFilter = { ...filter };
    delete newFilter.group_id;
    setFilter(newFilter);
    setSelectedGroupId(ALL_ACCOUNTS_GROUP_ID);
  };

  // 全选/取消全选
  const isAllSelected = emails.length > 0 && selectedEmails.length === emails.length;
  const handleSelectAll = () => {
    if (isAllSelected) {
      setSelectedEmails([]);
    } else {
      setSelectedEmails(emails.map(e => e.id));
    }
  };

  // 判断是否显示邮箱标识：当没有选中特定账户时显示（包括选中分组的情况）
  // 因为分组内可能包含多个邮箱账号，需要显示每封邮件属于哪个账号
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
      openDialog('delete');
    }
  };

  const confirmDelete = async () => {
    if (selectedEmails.length === 0 || isDeleting) return;

    const ids = [...selectedEmails];
    setIsDeleting(true);
    try {
      const deletedCount = await batchDeleteEmails(ids);
      if (deletedCount > 0) {
        setSelectedEmails([]);
      }
      closeDialog('delete');
    } finally {
      setIsDeleting(false);
    }
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
      openDialog('permanentDelete');
    }
  };

  const confirmPermanentDelete = async () => {
    await batchPermanentDelete(selectedEmails);
    setSelectedEmails([]);
    closeDialog('permanentDelete');
  };

  // 清空回收站
  const handleEmptyTrash = () => {
    openDialog('emptyTrash');
  };

  const confirmEmptyTrash = async () => {
    await emptyTrash();
    closeDialog('emptyTrash');
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
    openDialog('markAllRead');
  };

  const confirmMarkAllAsRead = async () => {
    const accountUid = filter.account_uid;
    await markAllAsRead(accountUid);
    closeDialog('markAllRead');
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

  // 切换 filter 或翻页时清空多选
  useEffect(() => {
    setSelectedEmails([]);
  }, [filter, page]);

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
                      disabled={isDeleting}
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
            <div className="flex items-center gap-2">
              <span className="text-xs text-muted-foreground">
                共 {total} 封{isTrashView ? '已删除' : ''}邮件
              </span>
              {/* 分组筛选指示器 */}
              {activeGroupFilter && (
                <Badge 
                  variant="outline" 
                  className="h-5 text-xs px-1.5 gap-1 cursor-pointer hover:bg-muted"
                  onClick={clearGroupFilter}
                  title="点击清除分组筛选"
                >
                  <Folder className="h-3 w-3" />
                  {activeGroupFilter.name}
                  <X className="h-3 w-3" />
                </Badge>
              )}
            </div>
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
                  <DropdownMenuItem onClick={handleSelectAll}>
                    选择全部
                  </DropdownMenuItem>
                  <DropdownMenuItem onClick={() => setSelectedEmails([])}>
                    取消选择
                  </DropdownMenuItem>
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
      <EmailPagination
        page={page}
        totalPages={totalPages}
        total={total}
        onPrev={handlePreviousPage}
        onNext={handleNextPage}
      />

      {/* 全部标记为已读确认对话框 */}
      <ConfirmDialog
        open={dialogs.markAllRead}
        onOpenChange={(open) => open ? openDialog('markAllRead') : closeDialog('markAllRead')}
        title="确认标记为已读"
        description={
          filter.account_uid
            ? '将当前账号的所有未读邮件标记为已读。此操作仅在本地生效，不会同步到邮箱服务器。'
            : '将所有账号的所有未读邮件标记为已读。此操作仅在本地生效，不会同步到邮箱服务器。'
        }
        confirmText="确认标记"
        onConfirm={confirmMarkAllAsRead}
      />

      {/* 删除邮件确认对话框 */}
      <ConfirmDialog
        open={dialogs.delete}
        onOpenChange={(open) => !isDeleting && (open ? openDialog('delete') : closeDialog('delete'))}
        title="确认删除"
        description={deleteMessage}
        confirmText={isDeleting ? '删除中...' : '确认删除'}
        variant="destructive"
        isLoading={isDeleting}
        onConfirm={confirmDelete}
      />

      {/* 永久删除确认对话框 */}
      <ConfirmDialog
        open={dialogs.permanentDelete}
        onOpenChange={(open) => open ? openDialog('permanentDelete') : closeDialog('permanentDelete')}
        title="确认永久删除"
        description={`确定要永久删除 ${selectedEmails.length} 封邮件吗？此操作无法撤销！`}
        confirmText="永久删除"
        variant="destructive"
        onConfirm={confirmPermanentDelete}
      />

      {/* 清空回收站确认对话框 */}
      <ConfirmDialog
        open={dialogs.emptyTrash}
        onOpenChange={(open) => open ? openDialog('emptyTrash') : closeDialog('emptyTrash')}
        title="清空回收站"
        description={`确定要清空回收站吗？这将永久删除所有 ${total} 封已删除邮件，此操作无法撤销！`}
        confirmText="清空回收站"
        variant="destructive"
        onConfirm={confirmEmptyTrash}
      />
    </div>
  );
};
