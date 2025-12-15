// Adapter 适配器相关类型定义

// 认证类型
export type AuthType = 'oauth2' | 'password';

// 适配器配置
export interface Adapter {
  id: number;
  name: string;           // 适配器标识：gmail, graph, imap
  display_name: string;   // 显示名称：Gmail API, Microsoft Graph, IMAP
  auth_type: AuthType;    // 认证类型：oauth2, password
  description?: string;   // 描述
  is_enabled: boolean;    // 是否启用
  created_at: string;
  updated_at: string;
}

// Provider-Adapter 关联（多对多）
export interface ProviderAdapter {
  provider_id: number;
  adapter_id: number;
  priority: number;       // 优先级，0 为最高
  adapter?: Adapter;      // 预加载的适配器信息
}

// 适配器响应（API 返回格式）
export interface AdapterResponse {
  id: number;
  name: string;
  display_name: string;
  auth_type: AuthType;
  description?: string;
  is_enabled: boolean;
  created_at: string;
  updated_at: string;
}

// 适配器列表响应
export interface AdapterListResponse {
  success: boolean;
  data: AdapterResponse[];
}

// 适配器 API 响应包装
export interface AdapterApiResponse<T = AdapterResponse> {
  success: boolean;
  data: T;
  message?: string;
  error?: string;
}

// 预定义的适配器名称常量
export const AdapterNames = {
  Gmail: 'gmail',
  Graph: 'graph',
  IMAP: 'imap',
} as const;

export type AdapterName = typeof AdapterNames[keyof typeof AdapterNames];

// 适配器显示名称映射
export const AdapterDisplayNames: Record<string, string> = {
  gmail: 'Gmail API (OAuth2)',
  graph: 'Microsoft Graph (OAuth2)',
  imap: 'IMAP (密码认证)',
};

// 工具函数：获取适配器显示名称
export const getAdapterDisplayName = (name: string): string => {
  return AdapterDisplayNames[name] || name;
};

// 工具函数：判断适配器是否为 OAuth2 类型
export const isOAuth2Adapter = (adapter: Adapter | AdapterResponse): boolean => {
  return adapter.auth_type === 'oauth2';
};

// 工具函数：判断适配器名称是否为 OAuth2 类型
export const isOAuth2AdapterName = (name: string): boolean => {
  return name === AdapterNames.Gmail || name === AdapterNames.Graph;
};
