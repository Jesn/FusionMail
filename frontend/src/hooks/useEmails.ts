import { useEffect, useCallback, useRef } from 'react';
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
    markAllAsRead: markAllAsReadStore,
  } = useEmailStore();

  const fetchingRef = useRef(false); // 防止重复请求
  const lastRequestRef = useRef<string>(''); // 记录上次请求的参数

  // 加载邮件列表
  const loadEmails = useCallback(async () => {
    // 生成请求参数的唯一标识
    const requestKey = JSON.stringify({ filter, searchQuery, page, pageSize });
    
    // 如果正在请求相同的数据，直接返回
    if (fetchingRef.current && lastRequestRef.current === requestKey) {
      return;
    }

    try {
      fetchingRef.current = true;
      lastRequestRef.current = requestKey;
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
      fetchingRef.current = false;
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
  const loadGlobalStats = useCallback(async (force = false) => {
    // 从 store 获取最新状态
    const currentStore = useEmailStore.getState();
    
    // 如果正在请求或已经加载过（且不是强制刷新），直接返回
    if (currentStore.isFetchingStats || (!force && currentStore.hasLoadedStats)) {
      return;
    }

    const { setFetchingStats, setUnreadCount, setStarredCount, setArchivedCount, setDeletedCount } = currentStore;
    
    try {
      setFetchingStats(true);
      const stats = await emailService.getGlobalStats();
      setUnreadCount(stats.unread_count);
      setStarredCount(stats.starred_count);
      setArchivedCount(stats.archived_count);
      setDeletedCount(stats.deleted_count);
    } catch (err) {
      console.error('Failed to load global stats:', err);
    } finally {
      setFetchingStats(false);
    }
  }, []);

  // 标记为已读
  const markAsRead = useCallback(async (ids: number[]) => {
    try {
      await emailService.markAsRead(ids);
      ids.forEach(id => updateEmailStatus(id, { is_read: true }));
      // 静默标记，不显示提示
      loadGlobalStats(true); // 强制刷新统计
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
      loadGlobalStats(true); // 强制刷新统计
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
      loadGlobalStats(true); // 强制刷新统计
    } catch (err) {
      const message = err instanceof Error ? err.message : '操作失败';
      toast.error(message);
    }
  }, [updateEmailStatus, loadGlobalStats]);

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
      loadGlobalStats(true); // 强制刷新统计
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
      loadGlobalStats(true); // 强制刷新统计
    } catch (err) {
      const message = err instanceof Error ? err.message : '删除失败';
      toast.error(message);
    }
  }, [removeEmail, loadGlobalStats]);

  // 全部标记为已读
  const markAllAsRead = useCallback(async (accountUid?: string) => {
    try {
      const result = await emailService.markAllAsRead(accountUid);
      markAllAsReadStore(accountUid);
      toast.success(`已标记 ${result.count} 封邮件为已读`);
      loadGlobalStats(true); // 强制刷新统计
      // 刷新列表以更新显示
      loadEmails();
    } catch (err) {
      const message = err instanceof Error ? err.message : '标记失败';
      toast.error(message);
    }
  }, [markAllAsReadStore, loadGlobalStats, loadEmails]);

  // 刷新列表
  const refresh = useCallback(() => {
    loadEmails();
    loadGlobalStats(true); // 强制刷新统计
  }, [loadEmails, loadGlobalStats]);

  // 初始加载
  useEffect(() => {
    loadEmails();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [filter, searchQuery, page, pageSize]); // 依赖实际数据变化

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
    markAllAsRead,
    toggleStar,
    archiveEmail,
    deleteEmail,
    refresh,
  };
};
