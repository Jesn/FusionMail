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
  getAllWithAdapters: () => Promise<Provider[]>;

  // Lookup operations
  findByDomain: (domain: string) => Promise<Provider | null>;
  findByEmail: (email: string) => Promise<Provider | null>;
  getWithAdapters: (id: number) => Promise<Provider | null>;

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

  /**
   * 获取所有 Provider（带适配器信息）
   */
  getAllWithAdapters: async (): Promise<Provider[]> => {
    const response = await api.get<{
      success: boolean;
      data: Provider[];
    }>('/providers/with-adapters');
    return response.data || [];
  },

  /**
   * 根据邮箱域名查找 Provider
   * @param domain 邮箱域名，如 gmail.com
   */
  findByDomain: async (domain: string): Promise<Provider | null> => {
    try {
      const response = await api.get<{
        success: boolean;
        data: Provider;
      }>(`/providers/by-domain?domain=${encodeURIComponent(domain)}`);
      return response.data || null;
    } catch {
      return null;
    }
  },

  /**
   * 根据邮箱地址查找 Provider
   * @param email 完整邮箱地址，如 user@gmail.com
   */
  findByEmail: async (email: string): Promise<Provider | null> => {
    try {
      const response = await api.get<{
        success: boolean;
        data: Provider;
      }>(`/providers/by-email?email=${encodeURIComponent(email)}`);
      return response.data || null;
    } catch {
      return null;
    }
  },

  /**
   * 获取 Provider 详情（带适配器信息）
   */
  getWithAdapters: async (id: number): Promise<Provider | null> => {
    try {
      const response = await api.get<{
        success: boolean;
        data: Provider;
      }>(`/providers/${id}/adapters`);
      return response.data || null;
    } catch {
      return null;
    }
  },
};
