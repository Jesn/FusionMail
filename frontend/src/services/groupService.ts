import { api } from './api';
import type { AxiosError } from 'axios';
import type {
  AccountGroup,
  AccountGroupWithCount,
  AccountGroupWithAccounts,
  CreateGroupRequest,
  UpdateGroupRequest,
  Account,
} from '../types';

/**
 * 从 API 错误中提取用户友好的错误信息
 */
const extractErrorMessage = (error: unknown, defaultMessage: string): string => {
  if (error && typeof error === 'object') {
    const axiosError = error as AxiosError<{ error?: string; message?: string }>;
    // 优先从响应体中提取 error 字段
    if (axiosError.response?.data?.error) {
      return axiosError.response.data.error;
    }
    if (axiosError.response?.data?.message) {
      return axiosError.response.data.message;
    }
  }
  if (error instanceof Error) {
    return error.message;
  }
  return defaultMessage;
};

/**
 * 分组管理服务
 * 提供账号分组的 CRUD 操作和账号分配功能
 */
export const groupService = {
  /**
   * 获取所有分组（带账号数量）
   */
  getGroups: async (): Promise<AccountGroupWithCount[]> => {
    const response = await api.get<{ success: boolean; data: AccountGroupWithCount[] }>('/groups');
    return response.data || [];
  },

  /**
   * 根据 ID 获取分组详情（带账号列表）
   */
  getGroupById: async (id: number): Promise<AccountGroupWithAccounts> => {
    const response = await api.get<{ success: boolean; data: AccountGroupWithAccounts }>(`/groups/${id}`);
    return response.data;
  },

  /**
   * 创建分组
   */
  createGroup: async (data: CreateGroupRequest): Promise<AccountGroup> => {
    try {
      const response = await api.post<{ success: boolean; data: AccountGroup }>('/groups', data);
      return response.data;
    } catch (error) {
      throw new Error(extractErrorMessage(error, '创建分组失败'));
    }
  },

  /**
   * 更新分组
   */
  updateGroup: async (id: number, data: UpdateGroupRequest): Promise<AccountGroup> => {
    try {
      const response = await api.put<{ success: boolean; data: AccountGroup }>(`/groups/${id}`, data);
      return response.data;
    } catch (error) {
      throw new Error(extractErrorMessage(error, '更新分组失败'));
    }
  },

  /**
   * 删除分组
   * 注意：删除分组不会删除账号，只会将账号的 group_id 设为 null
   */
  deleteGroup: async (id: number): Promise<void> => {
    await api.delete(`/groups/${id}`);
  },

  /**
   * 将账号分配到分组
   * @param accountUid 账号 UID
   * @param groupId 分组 ID，传 null 表示移出分组
   */
  assignAccountToGroup: async (accountUid: string, groupId: number | null): Promise<void> => {
    await api.put(`/accounts/${accountUid}/group`, { group_id: groupId });
  },

  /**
   * 批量将账号分配到分组
   * @param accountUids 账号 UID 列表
   * @param groupId 分组 ID，传 null 表示移出分组
   */
  batchAssignAccounts: async (accountUids: string[], groupId: number | null): Promise<{ count: number }> => {
    const response = await api.post<{ success: boolean; data: { count: number } }>('/groups/batch-assign', {
      account_uids: accountUids,
      group_id: groupId,
    });
    return response.data;
  },

  /**
   * 重新排序分组
   * @param groupIds 按新顺序排列的分组 ID 列表
   */
  reorderGroups: async (groupIds: number[]): Promise<void> => {
    await api.put('/groups/reorder', { group_ids: groupIds });
  },

  /**
   * 获取未分组的账号列表
   */
  getUngroupedAccounts: async (): Promise<Account[]> => {
    const response = await api.get<{ success: boolean; data: Account[] }>('/groups/ungrouped/accounts');
    return response.data || [];
  },
};
