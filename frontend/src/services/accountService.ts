import { api } from './api';
import { Account, SyncProgress } from '../types';

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
  // 通用邮箱配置字段
  imap_host?: string;
  imap_port?: number;
  pop3_host?: string;
  pop3_port?: number;
  encryption?: string;
  // 删除策略
  server_delete_policy?: string; // 'off' 或 'soft'
  // 首次同步优化配置
  first_sync_days?: number;      // 首次同步天数（0 表示全量同步）
  batch_size?: number;           // 批次大小
  max_emails_per_sync?: number;  // 单次同步最大邮件数
  // 分组
  group_id?: number | null;      // 所属分组 ID
  // 发件功能
  smtp_enabled?: boolean;        // 是否启用 SMTP 发件功能
}

export interface UpdateAccountRequest {
  sync_enabled?: boolean;
  sync_interval?: number;
  proxy_enabled?: boolean;
  proxy_type?: string;
  proxy_host?: string;
  proxy_port?: number;
  // 删除策略
  server_delete_policy?: string; // 'off' 或 'soft'
  // 首次同步优化配置
  first_sync_days?: number;
  batch_size?: number;
  max_emails_per_sync?: number;
  // 分组
  group_id?: number | null;      // 所属分组 ID
}

// 账户列表筛选参数
export interface AccountListFilter {
  page?: number;
  page_size?: number;
  group_id?: number;  // -1=所有，0=未分组，>0=具体分组
  email?: string;
  provider?: string;
  status?: string;
}

// 账户列表响应
export interface AccountListResponse {
  accounts: Account[];
  total: number;
  page: number;
  page_size: number;
  total_pages: number;
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
   * 获取账户列表（支持分页和筛选）
   */
  getListWithFilter: async (filter: AccountListFilter): Promise<AccountListResponse> => {
    const params = new URLSearchParams();
    if (filter.page) params.append('page', String(filter.page));
    if (filter.page_size) params.append('page_size', String(filter.page_size));
    if (filter.group_id !== undefined) params.append('group_id', String(filter.group_id));
    if (filter.email) params.append('email', filter.email);
    if (filter.provider) params.append('provider', filter.provider);
    if (filter.status) params.append('status', filter.status);
    
    const response = await api.get<{ success: boolean; data: AccountListResponse }>(
      `/accounts/filter?${params.toString()}`
    );
    return response.data;
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

  // 注意：getSyncStatus 方法已移除，如需实时同步状态，请使用账户的 last_sync_status 字段

  /**
   * 禁用账户
   */
  disable: async (uid: string): Promise<void> => {
    await api.post(`/accounts/${uid}/disable`);
  },

  /**
   * 启用账户
   */
  enable: async (uid: string): Promise<void> => {
    await api.post(`/accounts/${uid}/enable`);
  },

  /**
   * 批量启用账户
   */
  batchEnable: async (uids: string[]): Promise<{
    success: number;
    failed: number;
    total: number;
    failed_items: Array<{
      uid: string;
      email: string;
      error: string;
    }>;
  }> => {
    const response = await api.post<{
      success: boolean;
      data: {
        success: number;
        failed: number;
        total: number;
        failed_items: Array<{
          uid: string;
          email: string;
          error: string;
        }>;
      };
    }>('/accounts/batch/enable', { uids });
    return response.data;
  },

  /**
   * 批量禁用账户
   */
  batchDisable: async (uids: string[]): Promise<{
    success: number;
    failed: number;
    total: number;
    failed_items: Array<{
      uid: string;
      email: string;
      error: string;
    }>;
  }> => {
    const response = await api.post<{
      success: boolean;
      data: {
        success: number;
        failed: number;
        total: number;
        failed_items: Array<{
          uid: string;
          email: string;
          error: string;
        }>;
      };
    }>('/accounts/batch/disable', { uids });
    return response.data;
  },

  /**
   * 清除同步错误状态
   */
  clearSyncError: async (uid: string): Promise<void> => {
    await api.post(`/accounts/${uid}/clear-error`);
  },

  /**
   * 获取回收站中的账户（仅软删除的）
   */
  getTrashList: async (): Promise<Account[]> => {
    const response = await api.get<{ success: boolean; data: Account[] }>('/accounts/trash');
    return response.data || [];
  },

  /**
   * 恢复软删除的账户
   */
  restore: async (uid: string): Promise<void> => {
    await api.post(`/accounts/${uid}/restore`);
  },

  /**
   * 永久删除账户（包括所有相关数据）
   */
  forceDelete: async (uid: string): Promise<void> => {
    await api.delete(`/accounts/${uid}/force`);
  },

  /**
   * 批量导入短效邮箱账户
   */
  batchImport: async (
    accounts: string[], 
    syncEnabled?: boolean, 
    syncInterval?: number,
    groupId?: number,
    firstSyncDays?: number
  ): Promise<{
    success: number;
    failed: number;
    results: Array<{
      email: string;
      status: 'success' | 'failed';
      error?: string;
    }>;
  }> => {
    const response = await api.post<{
      success: boolean;
      data: {
        success: number;
        failed: number;
        results: Array<{
          email: string;
          status: 'success' | 'failed';
          error?: string;
        }>;
      };
    }>('/accounts/batch-import', { 
      accounts, 
      sync_enabled: syncEnabled, 
      sync_interval: syncInterval,
      group_id: groupId,
      first_sync_days: firstSyncDays
    });
    return response.data;
  },

  /**
   * 取消账户同步
   * Requirements: 5.1 - 支持同步取消
   */
  cancelSync: async (uid: string): Promise<void> => {
    await api.post(`/accounts/${uid}/sync/cancel`);
  },

  /**
   * 获取账户同步进度
   * Requirements: 2.1-2.4 - 同步进度追踪
   */
  getSyncProgress: async (uid: string): Promise<SyncProgress | null> => {
    const response = await api.get<{ success: boolean; data: SyncProgress | { status: string; message: string } }>(`/accounts/${uid}/sync/progress`);
    // 如果返回的是 idle 状态，表示没有进行中的同步
    if (response.data && 'message' in response.data && response.data.status === 'idle') {
      return null;
    }
    return response.data as SyncProgress;
  },
};
