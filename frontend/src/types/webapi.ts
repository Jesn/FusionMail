// WebAPI 适配器相关类型定义

// ============================================
// 服务类型常量
// ============================================

// WebAPI 服务类型
export type WebAPIServiceType = 
  | 'cloudflare_temp_email'  // Cloudflare Temp Email
  | 'cloud_mail'             // Cloud Mail
  | 'custom';                // 自定义服务

// 访问模式
export type WebAPIAccessMode = 'single' | 'admin';

// 认证类型
export type WebAPIAuthType = 
  | 'bearer_token'    // Bearer Token
  | 'api_key'         // API Key (Header/Query)
  | 'basic_auth'      // Basic Auth
  | 'custom_header';  // 自定义 Header

// 分页类型
export type WebAPIPaginationType = 
  | 'offset'   // offset/limit 分页
  | 'cursor'   // 游标分页
  | 'page'     // 页码分页
  | 'id_based'; // ID 分页

// ============================================
// 认证数据结构
// ============================================

// Cloudflare Temp Email 认证数据
export interface CloudflareTempEmailAuthData {
  base_url: string;           // API 基础 URL
  access_mode: WebAPIAccessMode; // 访问模式
  // Single 模式 - 方式一：JWT Token（永不过期）
  jwt_token?: string;         // JWT Token（直接登录方式）
  email?: string;             // 目标邮箱地址
  // Single 模式 - 方式二：User Token（第三方授权登录，30天过期，支持自动刷新）
  user_token?: string;        // User Token（第三方授权登录方式）
  // Admin 模式
  admin_password?: string;    // Admin 密码
  domains?: string;           // 过滤域名列表（逗号分隔，如 "example.com, test.org"）
}

// Cloud Mail 认证数据
export interface CloudMailAuthData {
  base_url: string;           // API 基础 URL
  jwt_token?: string;         // JWT Token（可选，如果提供邮箱+密码则自动获取）
  email?: string;             // 登录邮箱（用于自动获取 Token）
  password?: string;          // 登录密码（用于自动获取 Token）
}

// 自定义 WebAPI 字段映射
export interface CustomWebAPIFieldMapping {
  id: string;                 // 邮件 ID 字段路径
  subject?: string;           // 主题字段路径
  from?: string;              // 发件人字段路径
  to?: string;                // 收件人字段路径
  date?: string;              // 日期字段路径
  body?: string;              // 正文字段路径
  html_body?: string;         // HTML 正文字段路径
  raw?: string;               // RFC822 原始内容字段路径
  target_address?: string;    // 目标地址字段路径
}

// 自定义 WebAPI 分页配置
export interface CustomWebAPIPagination {
  type: WebAPIPaginationType; // 分页类型
  page_size: number;          // 每页数量
  // offset 分页
  offset_param?: string;      // offset 参数名
  limit_param?: string;       // limit 参数名
  // cursor 分页
  cursor_param?: string;      // cursor 参数名
  cursor_field?: string;      // 响应中 cursor 字段路径
  // page 分页
  page_param?: string;        // page 参数名
  // id_based 分页
  since_id_param?: string;    // since_id 参数名
  id_field?: string;          // 响应中 ID 字段路径
}

// 自定义 WebAPI 认证数据
export interface CustomWebAPIAuthData {
  service_name: string;       // 服务名称
  base_url: string;           // API 基础 URL
  endpoint: string;           // 邮件列表端点
  method: 'GET' | 'POST';     // HTTP 方法
  
  // 认证配置
  auth_type: WebAPIAuthType;  // 认证类型
  auth_token?: string;        // Token (bearer_token)
  api_key?: string;           // API Key
  api_key_header?: string;    // API Key Header 名称
  api_key_in_query?: boolean; // API Key 是否在 Query 中
  username?: string;          // 用户名 (basic_auth)
  password?: string;          // 密码 (basic_auth)
  custom_headers?: Record<string, string>; // 自定义 Headers
  
  // 响应解析
  data_path: string;          // 数据路径 (如 "data.list")
  field_mapping: CustomWebAPIFieldMapping; // 字段映射
  
  // 分页配置
  pagination?: CustomWebAPIPagination;
  
  // 目标地址（Single 模式）
  target_email?: string;
}

// 统一的 WebAPI 认证数据类型
export type WebAPIAuthData = 
  | CloudflareTempEmailAuthData 
  | CloudMailAuthData 
  | CustomWebAPIAuthData;

// ============================================
// API 请求/响应类型
// ============================================

// 创建 WebAPI Provider 请求
export interface CreateWebAPIProviderRequest {
  name?: string;              // 显示名称（可选，如果不填则从配置中提取或自动生成）
  service_type: WebAPIServiceType; // 服务类型
  auth_data: string;          // JSON 格式的认证数据
  group_id?: number | null;   // 分组 ID（可选）
  sync_interval?: number;     // 同步间隔（分钟，可选，默认 2）
  sync_enabled?: boolean;     // 是否启用同步（可选，默认 true）
}

// 更新 WebAPI Provider 请求
export interface UpdateWebAPIProviderRequest {
  name?: string;              // 显示名称
  auth_data?: string;         // JSON 格式的认证数据
}

// 测试连接请求
export interface TestConnectionRequest {
  service_type: WebAPIServiceType;
  auth_data: string;          // JSON 格式的认证数据
}

// 测试连接结果
export interface TestConnectionResult {
  success: boolean;
  message: string;
  service_name?: string;
  email_count?: number;
  error?: string;
}

// WebAPI Provider 响应
export interface WebAPIProviderResponse {
  uid: string;
  email: string;
  provider_id: number;
  adapter_id: number;
  status: string;
  sync_enabled: boolean;
  last_sync_at?: string;
  last_sync_status?: string;
  last_sync_error?: string;
  created_at: string;
  updated_at: string;
}

// WebAPI Provider 列表响应
export interface WebAPIProviderListResponse {
  success: boolean;
  data: WebAPIProviderResponse[];
  total: number;
  page: number;
  page_size: number;
}

// 同步状态
export interface WebAPISyncStatus {
  status: string;
  last_sync_at?: string;
  last_synced_id?: string;
  email_count: number;
  error_message?: string;
}

// ============================================
// 服务模板类型
// ============================================

// 服务模板
export interface WebAPIServiceTemplate {
  service_type: WebAPIServiceType;
  name: string;               // 显示名称
  description: string;        // 描述
  icon?: string;              // 图标
  default_config: Partial<WebAPIAuthData>; // 默认配置
  config_schema: WebAPIConfigSchema; // 配置 Schema
}

// 配置字段 Schema
export interface WebAPIConfigField {
  name: string;               // 字段名
  label: string;              // 显示标签
  type: 'text' | 'password' | 'select' | 'number' | 'boolean' | 'array';
  required: boolean;
  placeholder?: string;
  default_value?: any;
  options?: { label: string; value: string }[]; // select 选项
  description?: string;
}

// 配置 Schema
export interface WebAPIConfigSchema {
  fields: WebAPIConfigField[];
  groups?: {
    name: string;
    label: string;
    fields: string[];         // 字段名列表
    condition?: {             // 显示条件
      field: string;
      value: any;
    };
  }[];
}

// 服务列表响应
export interface WebAPIServicesResponse {
  success: boolean;
  data: {
    services: WebAPIServiceTemplate[];
    supported_types: WebAPIServiceType[];
  };
}

// ============================================
// API 响应包装
// ============================================

export interface WebAPIApiResponse<T = any> {
  success: boolean;
  data: T;
  message?: string;
  error?: string;
}

// ============================================
// 常量和工具函数
// ============================================

// 服务类型显示名称
export const WebAPIServiceTypeNames: Record<WebAPIServiceType, string> = {
  cloudflare_temp_email: 'Cloudflare Temp Email',
  cloud_mail: 'Cloud Mail',
  custom: '自定义服务',
};

// 服务类型图标
export const WebAPIServiceTypeIcons: Record<WebAPIServiceType, string> = {
  cloudflare_temp_email: '☁️',
  cloud_mail: '📧',
  custom: '🔧',
};

// 认证类型显示名称
export const WebAPIAuthTypeNames: Record<WebAPIAuthType, string> = {
  bearer_token: 'Bearer Token',
  api_key: 'API Key',
  basic_auth: 'Basic Auth',
  custom_header: '自定义 Header',
};

// 分页类型显示名称
export const WebAPIPaginationTypeNames: Record<WebAPIPaginationType, string> = {
  offset: 'Offset/Limit 分页',
  cursor: '游标分页',
  page: '页码分页',
  id_based: 'ID 分页',
};

// 工具函数：获取服务类型显示名称
export const getWebAPIServiceTypeName = (type: WebAPIServiceType): string => {
  return WebAPIServiceTypeNames[type] || type;
};

// 工具函数：判断是否为预置服务
export const isPresetService = (type: WebAPIServiceType): boolean => {
  return type === 'cloudflare_temp_email' || type === 'cloud_mail';
};

// 工具函数：验证 URL 格式
export const isValidHttpsUrl = (url: string): boolean => {
  try {
    const parsed = new URL(url);
    return parsed.protocol === 'https:';
  } catch {
    return false;
  }
};
