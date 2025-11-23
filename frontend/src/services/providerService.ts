import { api } from './api';
import {
  Provider,
  ProviderCreateRequest,
  ProviderUpdateRequest,
  ProviderListResponse,
} from '../types';

export interface ProviderService {
  // CRUD operations
  create: (data: ProviderCreateRequest) => Promise<Provider>;
  update: (id: number, data: ProviderUpdateRequest) => Promise<Provider>;
  delete: (id: number) => Promise<void>;
  getById: (id: number) => Promise<Provider>;

  // List operations
  getList: (page?: number, pageSize?: number) => Promise<ProviderListResponse>;
  getAll: () => Promise<Provider[]>;

  // Management operations
  toggleEnabled: (id: number, enabled: boolean) => Promise<Provider>;
  reorder: (id: number, sortOrder: number) => Promise<void>;
}

export const providerService: ProviderService = {
  /**
   * 创建新的 Provider 配置
   */
  create: async (data: ProviderCreateRequest): Promise<Provider> => {
    const response = await api.post<{ success: boolean; data: Provider }>(
      '/providers',
      data
    );
    return response.data;
  },

  /**
   * 更新 Provider 配置
   */
  update: async (id: number, data: ProviderUpdateRequest): Promise<Provider> => {
    const response = await api.put<{ success: boolean; data: Provider }>(
      `/providers/${id}`,
      data
    );
    return response.data;
  },

  /**
   * 删除 Provider 配置
   */
  delete: async (id: number): Promise<void> => {
    await api.delete(`/providers/${id}`);
  },

  /**
   * 获取 Provider 详情
   */
  getById: async (id: number): Promise<Provider> => {
    const response = await api.get<{ success: boolean; data: Provider }>(
      `/providers/${id}`
    );
    return response.data;
  },

  /**
   * 分页获取 Provider 列表
   */
  getList: async (page: number = 1, pageSize: number = 20): Promise<ProviderListResponse> => {
    const response = await api.get<{
      success: boolean;
      data: {
        items: Provider[];
        total: number;
        page: number;
        page_size: number;
      };
    }>(`/providers?page=${page}&page_size=${pageSize}`);

    const items = response.data?.items || [];
    const total = response.data?.total || 0;
    const pageNum = response.data?.page || 1;
    const size = response.data?.page_size || 20;

    return {
      data: items,
      total: total,
      page: pageNum,
      page_size: size,
      total_page: Math.ceil(total / size),
    };
  },

  /**
   * 获取所有 Provider（不分页）
   */
  getAll: async (): Promise<Provider[]> => {
    const response = await api.get<{
      success: boolean;
      data: Provider[];
    }>('/providers/all');
    return response.data || [];
  },

  /**
   * 切换 Provider 启用状态
   */
  toggleEnabled: async (id: number, enabled: boolean): Promise<Provider> => {
    const response = await api.patch<{ success: boolean; data: Provider }>(
      `/providers/${id}`,
      { enabled }
    );
    return response.data;
  },

  /**
   * 重新排序
   */
  reorder: async (id: number, sortOrder: number): Promise<void> => {
    await api.patch(`/providers/${id}/reorder`, { sort_order: sortOrder });
  },
};
