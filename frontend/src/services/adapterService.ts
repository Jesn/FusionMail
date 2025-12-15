// Adapter 适配器服务
import api from './api';
import type { AdapterResponse, AdapterListResponse, AdapterApiResponse } from '@/types';

// 适配器服务
const adapterService = {
  /**
   * 获取所有适配器列表
   */
  async list(): Promise<AdapterResponse[]> {
    const response = await api.get<AdapterListResponse>('/adapters');
    return response.data.data || [];
  },

  /**
   * 获取启用的适配器列表
   */
  async listEnabled(): Promise<AdapterResponse[]> {
    const response = await api.get<AdapterListResponse>('/adapters/enabled');
    return response.data.data || [];
  },

  /**
   * 根据 ID 获取适配器
   */
  async getById(id: number): Promise<AdapterResponse | null> {
    try {
      const response = await api.get<AdapterApiResponse>(`/adapters/${id}`);
      return response.data.data || null;
    } catch {
      return null;
    }
  },

  /**
   * 根据名称获取适配器
   */
  async getByName(name: string): Promise<AdapterResponse | null> {
    try {
      const response = await api.get<AdapterApiResponse>(`/adapters/name/${name}`);
      return response.data.data || null;
    } catch {
      return null;
    }
  },
};

export default adapterService;
