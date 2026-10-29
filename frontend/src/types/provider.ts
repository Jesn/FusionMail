// Provider 邮箱提供商相关类型定义
import type { components } from './api.generated';

type Schemas = components['schemas'];

// 邮箱提供商配置
//
// 已收敛到生成类型：model.ProviderResponse 的 20 个非 omitempty 字段已补
// validate:"required"。注意 response.Provider 是「最小引用形态」（仅
// display_name/id/name），用于 response.AccountResponse 的 provider_ref，
// 与本类型同名不同义，故不用它。
export type Provider = Schemas['model.ProviderResponse'];

// 创建 Provider 请求
export interface ProviderCreateRequest {
  name: string;
  display_name: string;
  
  // 适配器关联字段
  default_adapter_id?: number;      // 默认适配器 ID
  email_domains?: string[];         // 支持的邮箱域名列表
  adapter_ids?: number[];           // 支持的适配器 ID 列表
  
  supported_protocols?: string[];
  recommended_protocol?: string;
  requires_oauth?: boolean;
  imap_host?: string;
  imap_port?: number;
  pop3_host?: string;
  pop3_port?: number;
  smtp_host?: string;
  smtp_port?: number;
  // 加密配置
  imap_encryption?: string; // ssl/starttls/none
  pop3_encryption?: string; // ssl/starttls/none
  smtp_encryption?: string; // ssl/starttls/none
  enabled?: boolean;
  sort_order?: number;
  description?: string;
  metadata?: string;
}

// 更新 Provider 请求
export interface ProviderUpdateRequest {
  name?: string;
  display_name?: string;
  
  // 适配器关联字段
  default_adapter_id?: number;      // 默认适配器 ID
  email_domains?: string[];         // 支持的邮箱域名列表
  adapter_ids?: number[];           // 支持的适配器 ID 列表
  
  supported_protocols?: string[];
  recommended_protocol?: string;
  requires_oauth?: boolean;
  imap_host?: string;
  imap_port?: number;
  pop3_host?: string;
  pop3_port?: number;
  smtp_host?: string;
  smtp_port?: number;
  // 加密配置
  imap_encryption?: string; // ssl/starttls/none
  pop3_encryption?: string; // ssl/starttls/none
  smtp_encryption?: string; // ssl/starttls/none
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
