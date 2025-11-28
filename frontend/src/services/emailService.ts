import { api } from './api';
import type { EmailDetail, EmailFilter, EmailListResponse, PaginationParams } from '../types';
import { useEmailCacheStore, getOrSetEmailCache } from '../stores/emailCacheStore';

export const emailService = {
  /**
   * 获取邮件列表
   * 使用缓存减少 API 调用
   */
  getList: async (
    filter?: EmailFilter,
    pagination?: PaginationParams
  ): Promise<EmailListResponse> => {
    // 生成缓存键
    const cacheKey = `emails:${JSON.stringify(filter || {})}:${JSON.stringify(pagination || {})}`;
    const { getEmailCache, setEmailCache } = useEmailCacheStore.getState();

    return getOrSetEmailCache<EmailListResponse>(
      getEmailCache,
      setEmailCache,
      cacheKey,
      async () => {
        const params = {
          ...filter,
          ...pagination,
        };
        const response = await api.get<{ success: boolean; data: EmailListResponse }>('/emails', { params });
        return response.data;
      },
      5 * 60 * 1000 // 5分钟缓存
    );
  },

  /**
   * 获取邮件详情
   * 使用独立缓存，缓存时间较长
   */
  getById: async (id: number, options?: { includeDeleted?: boolean }): Promise<EmailDetail> => {
    const includeDeleted = !!options?.includeDeleted;
    const cacheKey = `email-detail:${id}:includeDeleted:${includeDeleted ? '1' : '0'}`;
    const { getEmailDetailCache, setEmailDetailCache } = useEmailCacheStore.getState();

    return getOrSetEmailCache<EmailDetail>(
      getEmailDetailCache,
      setEmailDetailCache,
      cacheKey,
      async () => {
        const params = includeDeleted ? { include_deleted: true } : undefined;
        const response = await api.get<{ success: boolean; data: EmailDetail }>(`/emails/${id}`, { params });
        return response.data;
      },
      10 * 60 * 1000 // 10分钟缓存
    );
  },

  /**
   * 搜索邮件
   * 使用专门的搜索缓存
   */
  search: async (
    query: string,
    accountUid?: string,
    pagination?: PaginationParams
  ): Promise<EmailListResponse> => {
    const cacheKey = `search:${query}:${accountUid || 'all'}:${JSON.stringify(pagination || {})}`;
    const { getSearchCache, setSearchCache } = useEmailCacheStore.getState();

    return getOrSetEmailCache<EmailListResponse>(
      getSearchCache,
      setSearchCache,
      cacheKey,
      async () => {
        const params = {
          q: query,
          account_uid: accountUid,
          ...pagination,
        };
        const response = await api.get<{ success: boolean; data: EmailListResponse }>('/emails/search', { params });
        return response.data;
      },
      2 * 60 * 1000 // 2分钟缓存
    );
  },

  /**
   * 获取未读邮件数
   * 短时间缓存，频繁调用
   */
  getUnreadCount: async (accountUid?: string): Promise<number> => {
    const cacheKey = `unread-count:${accountUid || 'all'}`;
    const { getEmailCache, setEmailCache } = useEmailCacheStore.getState();

    return getOrSetEmailCache<number>(
      getEmailCache,
      setEmailCache,
      cacheKey,
      async () => {
        const params = accountUid ? { account_uid: accountUid } : {};
        const response = await api.get<{ success: boolean; unread_count: number }>('/emails/unread-count', { params });
        return response.unread_count;
      },
      30 * 1000 // 30秒缓存
    );
  },

  /**
   * 获取账户邮件统计
   */
  getAccountStats: async (accountUid: string) => {
    const response = await api.get<{ success: boolean; data: any }>(`/emails/stats/${accountUid}`);
    return response.data;
  },

  /**
   * 获取全局邮件统计（聚合接口，一次请求）
   */
  getGlobalStats: async (): Promise<{
    total_count: number;
    unread_count: number;
    starred_count: number;
    archived_count: number;
    deleted_count: number;
    spam_count?: number;
  }> => {
    const response = await api.get<{ success: boolean; data: {
      total_count: number;
      unread_count: number;
      starred_count: number;
      archived_count: number;
      deleted_count: number;
      spam_count?: number;
    } }>('\/emails\/stats');
    return response.data;
  },

  /**
   * 批量标记为已读
   */
  markAsRead: async (ids: number[]): Promise<void> => {
    await api.post('/emails/mark-read', { ids });
    // 清除相关缓存，确保返回列表时状态最新
    try {
      const cache = useEmailCacheStore.getState();
      // 清理这些邮件的详情缓存
      const pattern = `^email-detail:(${ids.join('|')}):`;
      cache.clearEmailDetailCache(pattern);
      // 列表与搜索缓存都会受影响
      cache.clearEmailCache();
      cache.clearSearchCache();
    } catch (e) {
      // 忽略缓存清理中的异常
      console.warn('Failed to clear cache after markAsRead:', e);
    }
  },

  /**
   * 批量标记为未读
   */
  markAsUnread: async (ids: number[]): Promise<void> => {
    await api.post('/emails/mark-unread', { ids });
    // 清除相关缓存，避免未读状态被旧缓存覆盖
    try {
      const cache = useEmailCacheStore.getState();
      const pattern = `^email-detail:(${ids.join('|')}):`;
      cache.clearEmailDetailCache(pattern);
      cache.clearEmailCache();
      cache.clearSearchCache();
    } catch (e) {
      console.warn('Failed to clear cache after markAsUnread:', e);
    }
  },

  /**
   * 全部标记为已读
   */
  markAllAsRead: async (accountUid?: string): Promise<{ count: number }> => {
    const response = await api.post<{ success: boolean; data: { message: string; count: number } }>(
      '/emails/mark-all-read',
      { account_uid: accountUid }
    );
    // 清除所有相关缓存，确保列表与统计正确刷新
    try {
      const cache = useEmailCacheStore.getState();
      cache.clearEmailDetailCache('^email-detail:');
      cache.clearEmailCache();
      cache.clearSearchCache();
    } catch (e) {
      console.warn('Failed to clear cache after markAllAsRead:', e);
    }
    return { count: response.data.count };
  },

  /**
   * 切换星标状态
   */
  toggleStar: async (id: number): Promise<void> => {
    await api.post(`/emails/${id}/toggle-star`);
    // 清除相关缓存，确保列表/搜索/详情状态刷新
    useEmailCacheStore.getState().clearEmailDetailCache(`email-detail:${id}`);
    useEmailCacheStore.getState().clearEmailCache();
    useEmailCacheStore.getState().clearSearchCache();
  },

  /**
   * 归档邮件
   */
  archive: async (id: number): Promise<void> => {
    await api.post(`/emails/${id}/archive`);

    // 清除相关缓存，因为归档会影响搜索结果
    useEmailCacheStore.getState().clearEmailCache();
    useEmailCacheStore.getState().clearSearchCache();
  },

  /**
   * 删除邮件
   */
  delete: async (id: number): Promise<void> => {
    await api.delete(`/emails/${id}`);

    // 清除相关缓存
    useEmailCacheStore.getState().clearEmailDetailCache(`email-detail:${id}`);
    useEmailCacheStore.getState().clearEmailCache();
    // 清除搜索缓存，因为删除邮件会影响搜索结果
    useEmailCacheStore.getState().clearSearchCache();
  },

  /**
   * 恢复已删除邮件
   */
  restore: async (id: number): Promise<void> => {
    await api.post(`/emails/${id}/restore`);

    // 清除相关缓存
    useEmailCacheStore.getState().clearEmailDetailCache(`email-detail:${id}`);
    useEmailCacheStore.getState().clearEmailCache();
    useEmailCacheStore.getState().clearSearchCache();
  },

  /**
   * 永久删除邮件（物理删除）
   */
  permanentDelete: async (id: number): Promise<void> => {
    await api.delete(`/emails/${id}/permanent`);

    // 清除相关缓存
    useEmailCacheStore.getState().clearEmailDetailCache(`email-detail:${id}`);
    useEmailCacheStore.getState().clearEmailCache();
    useEmailCacheStore.getState().clearSearchCache();
  },

  /**
   * 批量永久删除邮件（物理删除）
   */
  batchPermanentDelete: async (ids: number[]): Promise<{ deleted_count: number }> => {
    const response = await api.post<{ success: boolean; data: { deleted_count: number } }>(
      '/emails/permanent-delete',
      { ids }
    );

    // 清除相关缓存
    useEmailCacheStore.getState().clearEmailCache();
    useEmailCacheStore.getState().clearSearchCache();
    ids.forEach(id => {
      useEmailCacheStore.getState().clearEmailDetailCache(`email-detail:${id}`);
    });

    return response.data;
  },

  /**
   * 清空回收站（永久删除所有已删除邮件）
   */
  emptyTrash: async (): Promise<{ deleted_count: number }> => {
    const response = await api.post<{ success: boolean; data: { deleted_count: number } }>(
      '/emails/empty-trash'
    );

    // 清除所有缓存
    useEmailCacheStore.getState().clearAllCache();

    return response.data;
  },

  /**
   * 清除所有缓存
   */
  clearAllCache: () => {
    useEmailCacheStore.getState().clearAllCache();
  },

  /**
   * 获取缓存统计
   */
  getCacheStats: () => {
    return useEmailCacheStore.getState().getCacheStats();
  },
};
