import { api } from './api';
import { Account, AccountStats } from '../types';

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
    return api.get<Account[]>('/accounts');
  },

  /**
   * 获取账户详情
   */
  getByUid: async (uid: string): Promise<Account> => {
    return api.get<Account>(`/accounts/${uid}`);
  },

  /**
   * 创建账户
   */
  create: async (data: CreateAccountRequest): Promise<Account> => {
    return api.post<Account>('/accounts', data);
  },

  /**
   * 更新账户
   */
  update: async (uid: string, data: UpdateAccountRequest): Promise<Account> => {
    return api.put<Account>(`/accounts/${uid}`, data);
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
    return api.get('/sync/status');
  },
};
