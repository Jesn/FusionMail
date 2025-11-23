// Provider 邮箱提供商相关类型定义
import { ProviderType } from './providerType';

// 邮箱提供商配置
export interface Provider {
  id: number;
  name: string;
  display_name: string;
  provider_type: ProviderType; // 邮箱提供商类型
  supported_protocols: string[];
  recommended_protocol: string;
  requires_oauth: boolean;
  imap_host: string;
  imap_port: number;
  pop3_host?: string;
  pop3_port?: number;
  smtp_host?: string;
  smtp_port?: number;
  enabled: boolean;
  sort_order: number;
  description?: string;
  metadata?: string;
  created_at: string;
  updated_at: string;
}

// 创建 Provider 请求
export interface ProviderCreateRequest {
  name: string;
  display_name: string;
  provider_type: ProviderType; // 邮箱提供商类型
  supported_protocols: string[];
  recommended_protocol: string;
  requires_oauth?: boolean;
  imap_host?: string;
  imap_port?: number;
  pop3_host?: string;
  pop3_port?: number;
  smtp_host?: string;
  smtp_port?: number;
  enabled?: boolean;
  sort_order?: number;
  description?: string;
  metadata?: string;
}

// 更新 Provider 请求
export interface ProviderUpdateRequest {
  name?: string;
  display_name?: string;
  provider_type?: ProviderType; // 邮箱提供商类型
  supported_protocols?: string[];
  recommended_protocol?: string;
  requires_oauth?: boolean;
  imap_host?: string;
  imap_port?: number;
  pop3_host?: string;
  pop3_port?: number;
  smtp_host?: string;
  smtp_port?: number;
  enabled?: boolean;
  sort_order?: number;
  description?: string;
  metadata?: string;
}

// 分页响应
export interface ProviderListResponse {
  data: Provider[];
  total: number;
  page: number;
  page_size: number;
  total_page: number;
}

// API 响应包装
export interface ProviderApiResponse<T = any> {
  success: boolean;
  data: T;
  message?: string;
  error?: string;
}
