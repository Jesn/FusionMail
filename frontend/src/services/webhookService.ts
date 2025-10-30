import { api } from './api';

export interface Webhook {
  id: number;
  name: string;
  description?: string;
  url: string;
  method: string;
  headers?: string;
  events: string;
  filters?: string;
  retry_enabled: boolean;
  max_retries: number;
  retry_intervals: string;
  enabled: boolean;
  total_calls: number;
  success_calls: number;
  failed_calls: number;
  last_called_at?: string;
  last_status?: string;
  created_at: string;
  updated_at: string;
}

export interface WebhookLog {
  id: number;
  webhook_id: number;
  request_url: string;
  request_method: string;
  request_headers?: string;
  request_body?: string;
  response_status: number;
  response_body?: string;
  response_time_ms: number;
  success: boolean;
  error_message?: string;
  retry_count: number;
  created_at: string;
}

export interface CreateWebhookRequest {
  name: string;
  description?: string;
  url: string;
  method?: string;
  headers?: string;
  events: string[];
  filters?: Record<string, any>;
  retry_enabled?: boolean;
  max_retries?: number;
  retry_intervals?: number[];
}

export interface UpdateWebhookRequest extends CreateWebhookRequest {
  enabled: boolean;
}

export interface TestWebhookRequest {
  test_data?: Record<string, any>;
}

export const webhookService = {
  /**
   * 获取 Webhook 列表
   */
  getList: async (page = 1, pageSize = 20): Promise<{ data: Webhook[]; total: number }> => {
    const response = await api.get<{
      success: boolean;
      data: Webhook[];
      total: number;
      page: number;
      size: number;
    }>('/webhooks', {
      params: { page, page_size: pageSize }
    });
    return {
      data: response.data || [],
      total: response.total || 0,
    };
  },

  /**
   * 获取 Webhook 详情
   */
  getById: async (id: number): Promise<Webhook> => {
    const response = await api.get<{ success: boolean; data: Webhook }>(`/webhooks/${id}`);
    return response.data;
  },

  /**
   * 创建 Webhook
   */
  create: async (data: CreateWebhookRequest): Promise<Webhook> => {
    // 转换数据格式
    const payload = {
      ...data,
      events: JSON.stringify(data.events),
      filters: data.filters ? JSON.stringify(data.filters) : '',
      headers: data.headers || '',
      method: data.method || 'POST',
      retry_enabled: data.retry_enabled ?? true,
      max_retries: data.max_retries || 3,
      retry_intervals: JSON.stringify(data.retry_intervals || [10, 30, 60]),
    };

    const response = await api.post<{ success: boolean; data: Webhook }>('/webhooks', payload);
    return response.data;
  },

  /**
   * 更新 Webhook
   */
  update: async (id: number, data: UpdateWebhookRequest): Promise<Webhook> => {
    // 转换数据格式
    const payload = {
      ...data,
      events: JSON.stringify(data.events),
      filters: data.filters ? JSON.stringify(data.filters) : '',
      headers: data.headers || '',
      method: data.method || 'POST',
      retry_enabled: data.retry_enabled ?? true,
      max_retries: data.max_retries || 3,
      retry_intervals: JSON.stringify(data.retry_intervals || [10, 30, 60]),
    };

    const response = await api.put<{ success: boolean; data: Webhook }>(`/webhooks/${id}`, payload);
    return response.data;
  },

  /**
   * 删除 Webhook
   */
  delete: async (id: number): Promise<void> => {
    await api.delete(`/webhooks/${id}`);
  },

  /**
   * 切换 Webhook 启用状态
   */
  toggle: async (id: number): Promise<{ enabled: boolean }> => {
    const response = await api.post<{ success: boolean; data: { enabled: boolean } }>(`/webhooks/${id}/toggle`);
    return response.data;
  },

  /**
   * 测试 Webhook
   */
  test: async (id: number, testData?: Record<string, any>): Promise<void> => {
    const payload: TestWebhookRequest = {};
    if (testData) {
      payload.test_data = testData;
    }
    await api.post(`/webhooks/${id}/test`, payload);
  },

  /**
   * 获取 Webhook 调用日志
   */
  getLogs: async (id: number, page = 1, pageSize = 20): Promise<{ data: WebhookLog[]; total: number }> => {
    const response = await api.get<{
      success: boolean;
      data: WebhookLog[];
      total: number;
      page: number;
      size: number;
    }>(`/webhooks/${id}/logs`, {
      params: { page, page_size: pageSize }
    });
    return {
      data: response.data || [],
      total: response.total || 0,
    };
  },

  /**
   * 解析事件列表（从 JSON 字符串）
   */
  parseEvents: (eventsJson: string): string[] => {
    try {
      return JSON.parse(eventsJson);
    } catch {
      return [];
    }
  },

  /**
   * 解析过滤条件（从 JSON 字符串）
   */
  parseFilters: (filtersJson: string): Record<string, any> => {
    try {
      return JSON.parse(filtersJson);
    } catch {
      return {};
    }
  },

  /**
   * 解析重试间隔（从 JSON 字符串）
   */
  parseRetryIntervals: (intervalsJson: string): number[] => {
    try {
      return JSON.parse(intervalsJson);
    } catch {
      return [10, 30, 60];
    }
  },

  /**
   * 解析请求头（从 JSON 字符串）
   */
  parseHeaders: (headersJson: string): Record<string, string> => {
    try {
      return JSON.parse(headersJson);
    } catch {
      return {};
    }
  },
};