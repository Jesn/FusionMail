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
    starredCount,
    archivedCount,
    deletedCount,
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
    setStarredCount,
    setArchivedCount,
    setDeletedCount,
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

  // 加载未读数（保留以备将来使用）
  // const loadUnreadCount = useCallback(async (accountUid?: string) => {
  //   try {
  //     const count = await emailService.getUnreadCount(accountUid);
  //     setUnreadCount(count);
  //   } catch (err) {
  //     console.error('Failed to load unread count:', err);
  //   }
  // }, [setUnreadCount]);

  // 加载全局统计
  const loadGlobalStats = useCallback(async () => {
    try {
      const stats = await emailService.getGlobalStats();
      setUnreadCount(stats.unread_count);
      setStarredCount(stats.starred_count);
      setArchivedCount(stats.archived_count);
      setDeletedCount(stats.deleted_count);
    } catch (err) {
      console.error('Failed to load global stats:', err);
    }
  }, [setUnreadCount, setStarredCount, setArchivedCount, setDeletedCount]);

  // 标记为已读
  const markAsRead = useCallback(async (ids: number[]) => {
    try {
      await emailService.markAsRead(ids);
      ids.forEach(id => updateEmailStatus(id, { is_read: true }));
      // 静默标记，不显示提示
      loadGlobalStats();
    } catch (err) {
      const message = err instanceof Error ? err.message : '标记失败';
      toast.error(message);
    }
  }, [updateEmailStatus, loadGlobalStats]);

  // 标记为未读
  const markAsUnread = useCallback(async (ids: number[]) => {
    try {
      await emailService.markAsUnread(ids);
      ids.forEach(id => updateEmailStatus(id, { is_read: false }));
      toast.success('已标记为未读');
      loadGlobalStats();
    } catch (err) {
      const message = err instanceof Error ? err.message : '标记失败';
      toast.error(message);
    }
  }, [updateEmailStatus, loadGlobalStats]);

  // 切换星标
  const toggleStar = useCallback(async (id: number, currentStarred: boolean) => {
    try {
      await emailService.toggleStar(id);
      updateEmailStatus(id, { is_starred: !currentStarred });
      toast.success(currentStarred ? '已取消星标' : '已添加星标');
      loadGlobalStats();
    } catch (err) {
      const message = err instanceof Error ? err.message : '操作失败';
      toast.error(message);
    }
  }, [updateEmailStatus]);

  // 归档邮件
  const archiveEmail = useCallback(async (id: number) => {
    try {
      await emailService.archive(id);
      // 更新邮件状态：归档并取消删除状态
      updateEmailStatus(id, { is_archived: true, is_deleted: false });
      
      // 如果当前在垃圾箱视图，需要从列表中移除该邮件
      if (filter.is_deleted) {
        removeEmail(id);
      }
      
      toast.success('已归档');
      loadGlobalStats();
    } catch (err) {
      const message = err instanceof Error ? err.message : '归档失败';
      toast.error(message);
    }
  }, [updateEmailStatus, removeEmail, filter, loadGlobalStats]);

  // 删除邮件
  const deleteEmail = useCallback(async (id: number) => {
    try {
      await emailService.delete(id);
      removeEmail(id);
      toast.success('已删除');
      loadGlobalStats();
    } catch (err) {
      const message = err instanceof Error ? err.message : '删除失败';
      toast.error(message);
    }
  }, [removeEmail, loadGlobalStats]);

  // 刷新列表
  const refresh = useCallback(() => {
    loadEmails();
    loadGlobalStats();
  }, [loadEmails, loadGlobalStats]);

  // 初始加载
  useEffect(() => {
    loadEmails();
  }, [loadEmails]);

  // 加载全局统计
  useEffect(() => {
    loadGlobalStats();
  }, [loadGlobalStats]);

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
    starredCount,
    archivedCount,
    deletedCount,

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
