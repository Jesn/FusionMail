import { api } from './api';
import { Account } from '../types';

export interface CreateAccountRequest {
  email: string;
  provider: string;
  protocol: string;
  auth_type: string;
  password: string;
  sync_enabled?: boolean;
  sync_interval?: number;
  proxy_enabled?: boolean;
  proxy_type?: string;
  proxy_host?: string;
  proxy_port?: number;
}

export interface UpdateAccountRequest {
  sync_enabled?: boolean;
  sync_interval?: number;
  proxy_enabled?: boolean;
  proxy_type?: string;
  proxy_host?: string;
  proxy_port?: number;
}

export const accountService = {
  /**
   * 获取账户列表
   */
  getList: async (): Promise<Account[]> => {
    const response = await api.get<{ success: boolean; data: Account[] }>('/accounts');
    return response.data || [];
  },

  /**
   * 获取账户详情
   */
  getByUid: async (uid: string): Promise<Account> => {
    const response = await api.get<{ success: boolean; data: Account }>(`/accounts/${uid}`);
    return response.data;
  },

  /**
   * 创建账户
   */
  create: async (data: CreateAccountRequest): Promise<Account> => {
    const response = await api.post<{ success: boolean; data: Account }>('/accounts', data);
    return response.data;
  },

  /**
   * 更新账户
   */
  update: async (uid: string, data: UpdateAccountRequest): Promise<Account> => {
    const response = await api.put<{ success: boolean; data: Account }>(`/accounts/${uid}`, data);
    return response.data;
  },

  /**
   * 删除账户
   */
  delete: async (uid: string): Promise<void> => {
    await api.delete(`/accounts/${uid}`);
  },

  /**
   * 测试账户连接
   */
  testConnection: async (uid: string): Promise<{ success: boolean; message: string }> => {
    return api.post(`/accounts/${uid}/test`);
  },

  /**
   * 手动同步账户
   */
  sync: async (uid: string): Promise<void> => {
    await api.post(`/sync/accounts/${uid}`);
  },

  /**
   * 同步所有账户
   */
  syncAll: async (): Promise<void> => {
    await api.post('/sync/all');
  },

  /**
   * 获取同步状态
   */
  getSyncStatus: async (): Promise<{ running: boolean }> => {
    const response = await api.get<{ success: boolean; data: { running: boolean } }>('/sync/status');
    return response.data;
  },
};
