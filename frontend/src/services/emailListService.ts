import api from './api';

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
  const response = await api.get('/emaillist/whitelist', {
    params: { page, page_size: pageSize },
  });
  return response.data;
};

// 添加到白名单
export const addToWhitelist = async (data: AddEmailListRequest): Promise<EmailList> => {
  const response = await api.post('/emaillist/whitelist', data);
  return response.data.data;
};

// 从白名单删除
export const deleteFromWhitelist = async (id: number): Promise<void> => {
  await api.delete(`/emaillist/whitelist/${id}`);
};

// 获取黑名单
export const getBlacklist = async (page = 1, pageSize = 20): Promise<EmailListResponse> => {
  const response = await api.get('/emaillist/blacklist', {
    params: { page, page_size: pageSize },
  });
  return response.data;
};

// 添加到黑名单
export const addToBlacklist = async (data: AddEmailListRequest): Promise<EmailList> => {
  const response = await api.post('/emaillist/blacklist', data);
  return response.data.data;
};

// 从黑名单删除
export const deleteFromBlacklist = async (id: number): Promise<void> => {
  await api.delete(`/emaillist/blacklist/${id}`);
};
