import { api } from './api';
import { Email, EmailFilter, EmailListResponse, PaginationParams } from '../types';

export const emailService = {
  /**
   * 获取邮件列表
   */
  getList: async (
    filter?: EmailFilter,
    pagination?: PaginationParams
  ): Promise<EmailListResponse> => {
    const params = {
      ...filter,
      ...pagination,
    };
    const response = await api.get<{ success: boolean; data: EmailListResponse }>('/emails', { params });
    return response.data;
  },

  /**
   * 获取邮件详情
   */
  getById: async (id: number): Promise<Email> => {
    const response = await api.get<{ success: boolean; data: Email }>(`/emails/${id}`);
    return response.data;
  },

  /**
   * 搜索邮件
   */
  search: async (
    query: string,
    accountUid?: string,
    pagination?: PaginationParams
  ): Promise<EmailListResponse> => {
    const params = {
      q: query,
      account_uid: accountUid,
      ...pagination,
    };
    const response = await api.get<{ success: boolean; data: EmailListResponse }>('/emails/search', { params });
    return response.data;
  },

  /**
   * 获取未读邮件数
   */
  getUnreadCount: async (accountUid?: string): Promise<number> => {
    const params = accountUid ? { account_uid: accountUid } : {};
    const response = await api.get<{ success: boolean; unread_count: number }>('/emails/unread-count', { params });
    return response.unread_count;
  },

  /**
   * 获取账户邮件统计
   */
  getAccountStats: async (accountUid: string) => {
    const response = await api.get<{ success: boolean; data: any }>(`/emails/stats/${accountUid}`);
    return response.data;
  },

  /**
   * 获取全局邮件统计
   */
  getGlobalStats: async (): Promise<{
    total_count: number;
    unread_count: number;
    starred_count: number;
    archived_count: number;
    deleted_count: number;
  }> => {
    // 使用多个请求来获取统计信息
    const [unreadResp, starredResp, archivedResp, deletedResp] = await Promise.all([
      api.get<{ success: boolean; data: EmailListResponse }>('/emails', { params: { is_read: false, is_deleted: false, page: 1, page_size: 1 } }),
      api.get<{ success: boolean; data: EmailListResponse }>('/emails', { params: { is_starred: true, is_deleted: false, page: 1, page_size: 1 } }),
      api.get<{ success: boolean; data: EmailListResponse }>('/emails', { params: { is_archived: true, is_deleted: false, page: 1, page_size: 1 } }),
      api.get<{ success: boolean; data: EmailListResponse }>('/emails', { params: { is_deleted: true, page: 1, page_size: 1 } }),
    ]);

    return {
      total_count: 0, // 暂时不计算总数
      unread_count: unreadResp.data.total,
      starred_count: starredResp.data.total,
      archived_count: archivedResp.data.total,
      deleted_count: deletedResp.data.total,
    };
  },

  /**
   * 批量标记为已读
   */
  markAsRead: async (ids: number[]): Promise<void> => {
    await api.post('/emails/mark-read', { ids });
  },

  /**
   * 批量标记为未读
   */
  markAsUnread: async (ids: number[]): Promise<void> => {
    await api.post('/emails/mark-unread', { ids });
  },

  /**
   * 切换星标状态
   */
  toggleStar: async (id: number): Promise<void> => {
    await api.post(`/emails/${id}/toggle-star`);
  },

  /**
   * 归档邮件
   */
  archive: async (id: number): Promise<void> => {
    await api.post(`/emails/${id}/archive`);
  },

  /**
   * 删除邮件
   */
  delete: async (id: number): Promise<void> => {
    await api.delete(`/emails/${id}`);
  },
};
