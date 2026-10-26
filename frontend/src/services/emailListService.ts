// 统一使用具名导出的 api 封装：它已经解包了一层 axios 响应，
// 返回值就是后端响应体本身（如 { success, data, total, page, size }），
// 因此这里不能再多取一次 `.data`。
import { api } from './api';

export interface EmailList {
  id: number;
  user_uid: string;
  type: 'whitelist' | 'blacklist';
  target: string;
  target_type: 'email' | 'domain';
  reason?: string;
  created_at: string;
  updated_at: string;
}

export interface AddEmailListRequest {
  target: string;
  target_type?: 'email' | 'domain';
  reason?: string;
}

export interface EmailListResponse {
  success: boolean;
  data: EmailList[];
  total: number;
  page: number;
  size: number;
}

// 获取白名单
export const getWhitelist = async (page = 1, pageSize = 20): Promise<EmailListResponse> => {
  // api.get 返回的已经是响应体，直接返回即可
  return api.get<EmailListResponse>('/emaillist/whitelist', {
    params: { page, page_size: pageSize },
  });
};

// 添加到白名单
export const addToWhitelist = async (data: AddEmailListRequest): Promise<EmailList> => {
  const response = await api.post<{ success: boolean; data: EmailList }>(
    '/emaillist/whitelist',
    data
  );
  return response.data;
};

// 从白名单删除
export const deleteFromWhitelist = async (id: number): Promise<void> => {
  await api.delete<void>(`/emaillist/whitelist/${id}`);
};

// 获取黑名单
export const getBlacklist = async (page = 1, pageSize = 20): Promise<EmailListResponse> => {
  // api.get 返回的已经是响应体，直接返回即可
  return api.get<EmailListResponse>('/emaillist/blacklist', {
    params: { page, page_size: pageSize },
  });
};

// 添加到黑名单
export const addToBlacklist = async (data: AddEmailListRequest): Promise<EmailList> => {
  const response = await api.post<{ success: boolean; data: EmailList }>(
    '/emaillist/blacklist',
    data
  );
  return response.data;
};

// 从黑名单删除
export const deleteFromBlacklist = async (id: number): Promise<void> => {
  await api.delete<void>(`/emaillist/blacklist/${id}`);
};
