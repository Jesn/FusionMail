import { useState } from 'react';
import { useNavigate } from 'react-router-dom';
import { EmailList } from '../components/email/EmailList';
import { EmailToolbar } from '../components/email/EmailToolbar';
import { useEmails } from '../hooks/useEmails';
import { Email } from '../types';
import { Button } from '../components/ui/button';
import { ChevronLeft, ChevronRight, Mail, MailOpen } from 'lucide-react';
import { cn } from '../lib/utils';

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
    refresh,
  } = useEmails();

  const [selectedEmails, setSelectedEmails] = useState<number[]>([]);
  const [selectedEmail, setSelectedEmail] = useState<Email | null>(null);
  const [filterType, setFilterType] = useState<FilterType>('all');

  const handleEmailClick = (email: Email) => {
    setSelectedEmail(email);
    // 自动标记为已读
    if (!email.is_read) {
      markAsRead([email.id]);
    }
    // 跳转到详情页
    navigate(`/email/${email.id}`);
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
      if (confirm(`确定要删除 ${selectedEmails.length} 封邮件吗？`)) {
        selectedEmails.forEach((id) => deleteEmail(id));
        setSelectedEmails([]);
      }
    }
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

  return (
    <div className="flex h-full flex-col">
      {/* 筛选按钮 */}
      <div className="flex items-center gap-2 border-b bg-background px-4 py-2">
        <Button
          variant={filterType === 'all' ? 'secondary' : 'ghost'}
          size="sm"
          onClick={() => handleFilterChange('all')}
          className={cn(filterType === 'all' && 'bg-secondary')}
        >
          <Mail className="mr-2 h-4 w-4" />
          全部
        </Button>
        <Button
          variant={filterType === 'unread' ? 'secondary' : 'ghost'}
          size="sm"
          onClick={() => handleFilterChange('unread')}
          className={cn(filterType === 'unread' && 'bg-secondary')}
        >
          <MailOpen className="mr-2 h-4 w-4" />
          未读
        </Button>
      </div>

      {/* 工具栏 */}
      <EmailToolbar
        selectedCount={selectedEmails.length}
        totalCount={total}
        onMarkAsRead={handleMarkAsRead}
        onMarkAsUnread={handleMarkAsUnread}
        onToggleStar={handleToggleStar}
        onArchive={handleArchive}
        onDelete={handleDelete}
        onRefresh={refresh}
        isRefreshing={isLoading}
      />

      {/* 邮件列表 */}
      <div className="flex-1 overflow-hidden">
        <EmailList
          emails={emails}
          selectedEmailId={selectedEmail?.id}
          onEmailClick={handleEmailClick}
          isLoading={isLoading}
        />
      </div>

      {/* 分页控制 */}
      {totalPages > 1 && (
        <div className="flex items-center justify-between border-t bg-background px-4 py-2">
          <div className="text-sm text-muted-foreground">
            第 {page} 页，共 {totalPages} 页
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
    </div>
  );
};
