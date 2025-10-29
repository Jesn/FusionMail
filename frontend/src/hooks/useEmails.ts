import { useEffect, useCallback } from 'react';
import { useEmailStore } from '../stores/emailStore';
import { emailService } from '../services/emailService';
import { toast } from 'sonner';

export const useEmails = () => {
  const {
    emails,
    selectedEmail,
    total,
    page,
    pageSize,
    totalPages,
    filter,
    searchQuery,
    isLoading,
    isLoadingDetail,
    error,
    unreadCount,
    setEmails,
    setSelectedEmail,
    setFilter,
    setSearchQuery,
    setPage,
    setPageSize,
    setLoading,
    setLoadingDetail,
    setError,
    setUnreadCount,
    updateEmailStatus,
    removeEmail,
  } = useEmailStore();

  // 加载邮件列表
  const loadEmails = useCallback(async () => {
    try {
      setLoading(true);
      setError(null);

      const response = searchQuery
        ? await emailService.search(searchQuery, filter.account_uid, { page, page_size: pageSize })
        : await emailService.getList(filter, { page, page_size: pageSize });

      setEmails(response);
    } catch (err) {
      const message = err instanceof Error ? err.message : '加载邮件列表失败';
      setError(message);
      toast.error(message);
    } finally {
      setLoading(false);
    }
  }, [filter, searchQuery, page, pageSize, setEmails, setLoading, setError]);

  // 加载邮件详情
  const loadEmailDetail = useCallback(async (id: number) => {
    try {
      setLoadingDetail(true);
      const email = await emailService.getById(id);
      setSelectedEmail(email);
    } catch (err) {
      const message = err instanceof Error ? err.message : '加载邮件详情失败';
      toast.error(message);
    } finally {
      setLoadingDetail(false);
    }
  }, [setSelectedEmail, setLoadingDetail]);

  // 加载未读数
  const loadUnreadCount = useCallback(async (accountUid?: string) => {
    try {
      const count = await emailService.getUnreadCount(accountUid);
      setUnreadCount(count);
    } catch (err) {
      console.error('Failed to load unread count:', err);
    }
  }, [setUnreadCount]);

  // 标记为已读
  const markAsRead = useCallback(async (ids: number[]) => {
    try {
      await emailService.markAsRead(ids);
      ids.forEach(id => updateEmailStatus(id, { is_read: true }));
      toast.success('已标记为已读');
      loadUnreadCount(filter.account_uid);
    } catch (err) {
      const message = err instanceof Error ? err.message : '标记失败';
      toast.error(message);
    }
  }, [updateEmailStatus, loadUnreadCount, filter.account_uid]);

  // 标记为未读
  const markAsUnread = useCallback(async (ids: number[]) => {
    try {
      await emailService.markAsUnread(ids);
      ids.forEach(id => updateEmailStatus(id, { is_read: false }));
      toast.success('已标记为未读');
      loadUnreadCount(filter.account_uid);
    } catch (err) {
      const message = err instanceof Error ? err.message : '标记失败';
      toast.error(message);
    }
  }, [updateEmailStatus, loadUnreadCount, filter.account_uid]);

  // 切换星标
  const toggleStar = useCallback(async (id: number, currentStarred: boolean) => {
    try {
      await emailService.toggleStar(id);
      updateEmailStatus(id, { is_starred: !currentStarred });
      toast.success(currentStarred ? '已取消星标' : '已添加星标');
    } catch (err) {
      const message = err instanceof Error ? err.message : '操作失败';
      toast.error(message);
    }
  }, [updateEmailStatus]);

  // 归档邮件
  const archiveEmail = useCallback(async (id: number) => {
    try {
      await emailService.archive(id);
      updateEmailStatus(id, { is_archived: true });
      toast.success('已归档');
    } catch (err) {
      const message = err instanceof Error ? err.message : '归档失败';
      toast.error(message);
    }
  }, [updateEmailStatus]);

  // 删除邮件
  const deleteEmail = useCallback(async (id: number) => {
    try {
      await emailService.delete(id);
      removeEmail(id);
      toast.success('已删除');
      loadUnreadCount(filter.account_uid);
    } catch (err) {
      const message = err instanceof Error ? err.message : '删除失败';
      toast.error(message);
    }
  }, [removeEmail, loadUnreadCount, filter.account_uid]);

  // 刷新列表
  const refresh = useCallback(() => {
    loadEmails();
    loadUnreadCount(filter.account_uid);
  }, [loadEmails, loadUnreadCount, filter.account_uid]);

  // 初始加载
  useEffect(() => {
    loadEmails();
  }, [loadEmails]);

  // 加载未读数
  useEffect(() => {
    loadUnreadCount(filter.account_uid);
  }, [loadUnreadCount, filter.account_uid]);

  return {
    // 状态
    emails,
    selectedEmail,
    total,
    page,
    pageSize,
    totalPages,
    filter,
    searchQuery,
    isLoading,
    isLoadingDetail,
    error,
    unreadCount,

    // 操作
    setFilter,
    setSearchQuery,
    setPage,
    setPageSize,
    loadEmailDetail,
    markAsRead,
    markAsUnread,
    toggleStar,
    archiveEmail,
    deleteEmail,
    refresh,
  };
};
