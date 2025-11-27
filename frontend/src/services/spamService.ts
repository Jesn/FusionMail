import api from './api';
import { Email } from '../types';

// 垃圾邮件统计
export interface SpamStats {
  total_count: number;
  unread_count: number;
  today_count: number;
  week_count: number;
  month_count: number;
  blocked_count: number;
}

// 垃圾邮件列表响应
export interface SpamEmailsResponse {
  success: boolean;
  data: Email[];
  total: number;
  page: number;
  size: number;
}

// 标记垃圾邮件请求
export interface MarkSpamRequest {
  email_ids: number[];
}

// 批量删除请求
export interface BatchDeleteRequest {
  email_ids: number[];
}

// 垃圾邮件服务
export const spamService = {
  // 获取垃圾邮件列表
  getSpamEmails: async (params: {
    account_uid?: string;
    page?: number;
    page_size?: number;
  }): Promise<SpamEmailsResponse> => {
    const response = await api.get('/spam/emails', { params });
    return response.data;
  },

  // 获取垃圾邮件统计
  getSpamStats: async (accountUid?: string): Promise<SpamStats> => {
    const params = accountUid ? { account_uid: accountUid } : {};
    const response = await api.get('/spam/stats', { params });
    return response.data.data;
  },

  // 标记为垃圾邮件
  markAsSpam: async (emailIds: number[]): Promise<{ marked_count: number }> => {
    const response = await api.post('/spam/mark', { email_ids: emailIds });
    return response.data.data;
  },

  // 取消垃圾邮件标记
  unmarkAsSpam: async (emailIds: number[]): Promise<{ unmarked_count: number }> => {
    const response = await api.post('/spam/unmark', { email_ids: emailIds });
    return response.data.data;
  },

  // 批量删除垃圾邮件
  batchDeleteSpam: async (emailIds: number[]): Promise<{ deleted_count: number }> => {
    const response = await api.delete('/spam/batch', { data: { email_ids: emailIds } });
    return response.data.data;
  },

  // 清空垃圾箱
  emptySpamFolder: async (accountUid?: string): Promise<{ deleted_count: number }> => {
    const params = accountUid ? { account_uid: accountUid } : {};
    const response = await api.post('/spam/empty', null, { params });
    return response.data.data;
  },
};

export default spamService;
