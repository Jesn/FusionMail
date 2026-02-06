import { useEffect, useCallback, useRef } from 'react';
import { useEmailStore } from '../stores/emailStore';
import { useGroupStore } from '../stores/groupStore';
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

  // 加载邮件详情（可选：允许加载已删除邮件，用于垃圾箱来源）
  const loadEmailDetail = useCallback(async (id: number, includeDeleted = false) => {
    try {
      setLoadingDetail(true);
      const email = await emailService.getById(id, { includeDeleted });
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

    const { setFetchingStats, setUnreadCount, setStarredCount, setArchivedCount, setDeletedCount, setSpamCount } = currentStore;
    
    try {
      setFetchingStats(true);
      const stats = await emailService.getGlobalStats();
      setUnreadCount(stats.unread_count);
      setStarredCount(stats.starred_count);
      setArchivedCount(stats.archived_count);
      setDeletedCount(stats.deleted_count);
      // 设置垃圾邮件数量（如果 API 返回）
      if (stats.spam_count !== undefined) {
        setSpamCount(stats.spam_count);
      }
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
      // 乐观更新未读计数，最终由 SSE 对齐
      const st = useEmailStore.getState();
      st.setUnreadCount(Math.max(0, st.unreadCount - ids.length));
    } catch (err) {
      const message = err instanceof Error ? err.message : '标记失败';
      toast.error(message);
    }
  }, [updateEmailStatus]);

  // 标记为未读
  const markAsUnread = useCallback(async (ids: number[]) => {
    try {
      await emailService.markAsUnread(ids);
      ids.forEach(id => updateEmailStatus(id, { is_read: false }));
      toast.success('已标记为未读');
      // 乐观更新未读数，最终由 SSE 对齐
      const st = useEmailStore.getState();
      st.setUnreadCount(st.unreadCount + ids.length);
    } catch (err) {
      const message = err instanceof Error ? err.message : '标记失败';
      toast.error(message);
    }
  }, [updateEmailStatus]);

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
      // 更新邮件状态：归档并取消删除状态
      updateEmailStatus(id, { is_archived: true, is_deleted: false });
      
      // 如果当前在垃圾箱视图，需要从列表中移除该邮件
      if (filter.is_deleted) {
        removeEmail(id);
      }
      
      toast.success('已归档');

    } catch (err) {
      const message = err instanceof Error ? err.message : '归档失败';
      toast.error(message);
    }
  }, [updateEmailStatus, removeEmail, filter]);

  // 删除邮件
  const deleteEmail = useCallback(async (id: number) => {
    try {
      await emailService.delete(id);
      removeEmail(id);
      toast.success('已删除');

    } catch (err) {
      const message = err instanceof Error ? err.message : '删除失败';
      toast.error(message);
    }
  }, [removeEmail]);

  // 恢复已删除邮件
  const restoreEmail = useCallback(async (id: number) => {
    try {
      await emailService.restore(id);
      // 更新本地状态：取消删除，并确保不是归档状态
      updateEmailStatus(id, { is_deleted: false, is_archived: false });
      // 如果当前在垃圾箱视图，从当前列表移除
      if (filter.is_deleted) {
        removeEmail(id);
      }
      toast.success('已恢复');

    } catch (err) {
      const message = err instanceof Error ? err.message : '恢复失败';
      toast.error(message);
    }
  }, [updateEmailStatus, removeEmail, filter]);

  // 永久删除邮件（物理删除）
  const permanentDeleteEmail = useCallback(async (id: number) => {
    try {
      await emailService.permanentDelete(id);
      removeEmail(id);
      toast.success('已永久删除');
    } catch (err) {
      const message = err instanceof Error ? err.message : '永久删除失败';
      toast.error(message);
    }
  }, [removeEmail]);

  // 批量永久删除邮件
  const batchPermanentDelete = useCallback(async (ids: number[]) => {
    try {
      const result = await emailService.batchPermanentDelete(ids);
      // 从列表中移除已删除的邮件
      ids.forEach(id => removeEmail(id));
      toast.success(`已永久删除 ${result.deleted_count} 封邮件`);
      return result.deleted_count;
    } catch (err) {
      const message = err instanceof Error ? err.message : '批量删除失败';
      toast.error(message);
      return 0;
    }
  }, [removeEmail]);

  // 清空回收站
  const emptyTrash = useCallback(async () => {
    try {
      const result = await emailService.emptyTrash();
      toast.success(`已清空回收站，删除 ${result.deleted_count} 封邮件`);
      // 刷新列表
      loadEmails();
      return result.deleted_count;
    } catch (err) {
      const message = err instanceof Error ? err.message : '清空回收站失败';
      toast.error(message);
      return 0;
    }
  }, [loadEmails]);

  // 全部标记为已读
  const markAllAsRead = useCallback(async (accountUid?: string) => {
    try {
      const result = await emailService.markAllAsRead(accountUid);
      markAllAsReadStore(accountUid);
      toast.success(`已标记 ${result.count} 封邮件为已读`);
      // 乐观更新未读数，最终由 SSE 对齐
      const st = useEmailStore.getState();
      st.setUnreadCount(Math.max(0, st.unreadCount - result.count));
      // 刷新列表以更新显示
      loadEmails();
      // 刷新分组数据以更新左侧分组的未读数
      useGroupStore.getState().setCacheTimestamp(0); // 强制缓存过期
      useGroupStore.getState().fetchGroups();
    } catch (err) {
      const message = err instanceof Error ? err.message : '标记失败';
      toast.error(message);
    }
  }, [markAllAsReadStore, loadEmails]);

  // 刷新列表
  const refresh = useCallback(() => {
    loadEmails();
  }, [loadEmails]);

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
    restoreEmail,
    permanentDeleteEmail,
    batchPermanentDelete,
    emptyTrash,
    refresh,
  };
};
