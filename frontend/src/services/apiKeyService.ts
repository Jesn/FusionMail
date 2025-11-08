import { api } from './api';

/**
 * API Key 类型定义
 */
export interface APIKey {
  id: number;
  name: string;
  description: string;
  rate_limit: number;
  enabled: boolean;
  total_requests: number;
  last_used_at: string | null;
  created_at: string;
  expires_at: string | null;
}

/**
 * 创建 API Key 请求
 */
export interface CreateAPIKeyRequest {
  name: string;
  description: string;
  rate_limit: number;
  expires_at?: string | null;
}

/**
 * 创建 API Key 响应
 */
export interface CreateAPIKeyResponse {
  api_key: string; // 明文 Key，仅此一次返回
  key_info: APIKey;
}

/**
 * 更新 API Key 请求
 */
export interface UpdateAPIKeyRequest {
  name: string;
  description: string;
  rate_limit: number;
}

/**
 * API Key 服务
 */
export const apiKeyService = {
  /**
   * 创建 API Key
   */
  create: async (data: CreateAPIKeyRequest): Promise<CreateAPIKeyResponse> => {
    const response = await api.post<{ success: boolean; data: CreateAPIKeyResponse }>(
      '/api-keys',
      data
    );
    return response.data;
  },

  /**
   * 获取 API Key 列表
   */
  list: async (): Promise<APIKey[]> => {
    const response = await api.get<{ success: boolean; data: APIKey[] }>('/api-keys');
    return response.data || [];
  },

  /**
   * 获取 API Key 详情
   */
  getById: async (id: number): Promise<APIKey> => {
    const response = await api.get<{ success: boolean; data: APIKey }>(`/api-keys/${id}`);
    return response.data;
  },

  /**
   * 更新 API Key
   */
  update: async (id: number, data: UpdateAPIKeyRequest): Promise<APIKey> => {
    const response = await api.put<{ success: boolean; data: APIKey }>(
      `/api-keys/${id}`,
      data
    );
    return response.data;
  },

  /**
   * 删除 API Key
   */
  delete: async (id: number): Promise<void> => {
    await api.delete(`/api-keys/${id}`);
  },

  /**
   * 启用 API Key
   */
  enable: async (id: number): Promise<APIKey> => {
    const response = await api.post<{ success: boolean; data: APIKey }>(
      `/api-keys/${id}/enable`
    );
    return response.data;
  },

  /**
   * 禁用 API Key
   */
  disable: async (id: number): Promise<APIKey> => {
    const response = await api.post<{ success: boolean; data: APIKey }>(
      `/api-keys/${id}/disable`
    );
    return response.data;
  },
};

