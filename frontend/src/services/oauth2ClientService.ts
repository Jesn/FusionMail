import { api } from './api';
import {
  OAuth2Client,
  OAuth2ClientCreateRequest,
  OAuth2ClientUpdateRequest,
  OAuth2ClientListResponse,
  OAuth2ClientSmartSelectResponse,
} from '../types';

export interface OAuth2ClientService {
  // CRUD operations
  create: (data: OAuth2ClientCreateRequest) => Promise<OAuth2Client>;
  update: (id: number, data: OAuth2ClientUpdateRequest) => Promise<OAuth2Client>;
  delete: (id: number) => Promise<void>;
  getById: (id: number) => Promise<OAuth2Client>;

  // List operations
  getList: (page?: number, pageSize?: number) => Promise<OAuth2ClientListResponse>;
  getByProvider: (providerId: number) => Promise<OAuth2Client[]>;
  getDefault: (providerId: number) => Promise<OAuth2Client>;

  // Management operations
  setDefault: (id: number, providerId: number) => Promise<void>;

  // Smart selection
  smartSelect: (providerId: number, clientId?: number) => Promise<OAuth2ClientSmartSelectResponse>;
}

export const oauth2ClientService: OAuth2ClientService = {
  /**
   * 创建新的 OAuth2 客户端配置
   */
  create: async (data: OAuth2ClientCreateRequest): Promise<OAuth2Client> => {
    const response = await api.post<{ success: boolean; data: OAuth2Client }>(
      '/oauth2/clients',
      data
    );
    return response.data;
  },

  /**
   * 更新 OAuth2 客户端配置
   */
  update: async (id: number, data: OAuth2ClientUpdateRequest): Promise<OAuth2Client> => {
    const response = await api.put<{ success: boolean; data: OAuth2Client }>(
      `/oauth2/clients/${id}`,
      data
    );
    return response.data;
  },

  /**
   * 删除 OAuth2 客户端配置
   */
  delete: async (id: number): Promise<void> => {
    await api.delete(`/oauth2/clients/${id}`);
  },

  /**
   * 获取 OAuth2 客户端详情
   */
  getById: async (id: number): Promise<OAuth2Client> => {
    const response = await api.get<{ success: boolean; data: OAuth2Client }>(
      `/oauth2/clients/${id}`
    );
    return response.data;
  },

  /**
   * 分页获取 OAuth2 客户端列表
   */
  getList: async (page: number = 1, pageSize: number = 20): Promise<OAuth2ClientListResponse> => {
    const response = await api.get<{
      success: boolean;
      data: {
        data: OAuth2Client[];
        total: number;
        page: number;
        page_size: number;
        total_page: number;
      };
    }>(`/oauth2/clients?page=${page}&page_size=${pageSize}`);
    return {
      data: response.data?.data || [],
      total: response.data?.total || 0,
      page: response.data?.page || 1,
      page_size: response.data?.page_size || 20,
      total_page: response.data?.total_page || 0,
    };
  },

  /**
   * 按提供商获取 OAuth2 客户端列表
   */
  getByProvider: async (providerId: number): Promise<OAuth2Client[]> => {
    const response = await api.get<{ success: boolean; data: OAuth2Client[] }>(
      `/oauth2/clients/provider/${providerId}`
    );
    return response.data || [];
  },

  /**
   * 获取提供商的默认 OAuth2 客户端
   */
  getDefault: async (providerId: number): Promise<OAuth2Client> => {
    const response = await api.get<{ success: boolean; data: OAuth2Client }>(
      `/oauth2/clients/provider/${providerId}/default`
    );
    return response.data;
  },

  /**
   * 设置默认 OAuth2 客户端
   */
  setDefault: async (id: number, providerId: number): Promise<void> => {
    await api.post(`/oauth2/clients/${id}/default/${providerId}`);
  },

  /**
   * 智能选择 OAuth2 客户端
   * 优先使用指定的客户端，其次使用默认客户端，最后选择第一个可用的客户端
   */
  smartSelect: async (
    providerId: number,
    clientId?: number
  ): Promise<OAuth2ClientSmartSelectResponse> => {
    const params = clientId ? `?client_id=${clientId}` : '';
    const response = await api.get<{
      success: boolean;
      data: OAuth2ClientSmartSelectResponse;
    }>(`/oauth2/clients/smart-select/${providerId}${params}`);
    return response.data;
  },
};
