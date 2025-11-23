// OAuth2 客户端配置相关类型定义

// OAuth2 客户端配置
export interface OAuth2Client {
  id: number;
  provider_id: number;
  provider_name?: string; // 通过关联获取的提供商名称
  name: string;
  client_id: string;
  redirect_uri: string;
  enabled: boolean;
  is_default: boolean;
  usage_count: number;
  quota_daily: number;
  quota_monthly: number;
  last_used_at?: string;
  metadata: string;
  created_at: string;
  updated_at: string;
}

// 创建 OAuth2 客户端请求
export interface OAuth2ClientCreateRequest {
  provider_id: number;
  name: string;
  client_id: string;
  client_secret: string;
  redirect_uri: string;
  quota_daily?: number;
  quota_monthly?: number;
  metadata?: string;
}

// 更新 OAuth2 客户端请求
export interface OAuth2ClientUpdateRequest {
  provider_id?: number;
  name?: string;
  client_id?: string;
  client_secret?: string;
  redirect_uri?: string;
  enabled?: boolean;
  quota_daily?: number;
  quota_monthly?: number;
  metadata?: string;
}

// 分页响应
export interface OAuth2ClientListResponse {
  data: OAuth2Client[];
  total: number;
  page: number;
  page_size: number;
  total_page: number;
}

// API 响应包装
export interface OAuth2ClientApiResponse<T = any> {
  success: boolean;
  data: T;
  message?: string;
  error?: string;
}

// 智能选择请求参数
export interface OAuth2ClientSmartSelectParams {
  provider_name: string;
  client_id?: number;
}

// 智能选择响应
export interface OAuth2ClientSmartSelectResponse {
  id: number;
  provider_id: number;
  provider_name?: string;
  name: string;
  client_id: string;
  redirect_uri: string;
  enabled: boolean;
  is_default: boolean;
  usage_count: number;
  quota_daily: number;
  quota_monthly: number;
  last_used_at?: string;
  metadata: string;
  created_at: string;
  updated_at: string;
}
