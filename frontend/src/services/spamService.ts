import api from './api';
import { Email } from '../types';
import { useEmailCacheStore } from '../stores/emailCacheStore';

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
    // 清除相关邮件的详情缓存
    const cache = useEmailCacheStore.getState();
    emailIds.forEach(id => {
      cache.clearEmailDetailCache(`email-detail:${id}`);
    });
    cache.clearEmailCache();
    cache.clearSearchCache();
    return response.data.data;
  },

  // 取消垃圾邮件标记
  unmarkAsSpam: async (emailIds: number[]): Promise<{ unmarked_count: number }> => {
    const response = await api.post('/spam/unmark', { email_ids: emailIds });
    // 清除相关邮件的详情缓存
    const cache = useEmailCacheStore.getState();
    emailIds.forEach(id => {
      cache.clearEmailDetailCache(`email-detail:${id}`);
    });
    cache.clearEmailCache();
    cache.clearSearchCache();
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

// 垃圾邮件规则
export interface SpamRule {
  id: number;
  name: string;
  description: string;
  category: 'keyword' | 'pattern' | 'header' | 'content' | 'url' | 'attachment';
  pattern: string;
  score: number;
  enabled: boolean;
  is_builtin: boolean;
  hit_count: number;
  created_at: string;
  updated_at: string;
}

// 规则列表响应
export interface SpamRulesResponse {
  success: boolean;
  data: SpamRule[];
  total: number;
  page: number;
  size: number;
}

// 创建/更新规则请求
export interface SpamRuleRequest {
  name: string;
  description?: string;
  category: string;
  pattern: string;
  score?: number;
  enabled?: boolean;
}

// 规则测试请求
export interface RuleTestRequest {
  pattern: string;
  category: string;
  content: string;
}

// 规则测试响应
export interface RuleTestResponse {
  matched: boolean;
  matches: string[];
  duration: string;
  error?: string;
}

// 规则统计
export interface RuleStats {
  total_count: number;
  enabled_count: number;
  disabled_count: number;
  builtin_count: number;
  custom_count: number;
  total_hits: number;
}

// 规则管理服务
export const ruleService = {
  // 获取规则列表
  getRules: async (params: {
    category?: string;
    page?: number;
    page_size?: number;
  }): Promise<SpamRulesResponse> => {
    const response = await api.get('/spam/rules', { params });
    return response.data;
  },

  // 获取单个规则
  getRule: async (id: number): Promise<SpamRule> => {
    const response = await api.get(`/spam/rules/${id}`);
    return response.data.data;
  },

  // 创建规则
  createRule: async (rule: SpamRuleRequest): Promise<SpamRule> => {
    const response = await api.post('/spam/rules', rule);
    return response.data.data;
  },

  // 更新规则
  updateRule: async (id: number, rule: SpamRuleRequest): Promise<SpamRule> => {
    const response = await api.put(`/spam/rules/${id}`, rule);
    return response.data.data;
  },

  // 删除规则
  deleteRule: async (id: number): Promise<void> => {
    await api.delete(`/spam/rules/${id}`);
  },

  // 切换规则状态
  toggleRule: async (id: number): Promise<{ id: number; enabled: boolean }> => {
    const response = await api.put(`/spam/rules/${id}/toggle`);
    return response.data.data;
  },

  // 测试规则
  testRule: async (request: RuleTestRequest): Promise<RuleTestResponse> => {
    const response = await api.post('/spam/rules/test', request);
    return response.data.data;
  },

  // 获取规则统计
  getRuleStats: async (): Promise<RuleStats> => {
    const response = await api.get('/spam/rules/stats');
    return response.data.data;
  },
};

export default spamService;
