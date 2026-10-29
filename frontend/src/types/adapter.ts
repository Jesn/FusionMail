// Adapter 适配器相关类型定义
import type { components } from './api.generated';

type Schemas = components['schemas'];

// 认证类型（扩展支持 token 类型用于 WebAPI）
export type AuthType = 'oauth2' | 'password' | 'token';

// 适配器配置
//
// 已收敛到生成类型：model.AdapterResponse 的 8 个字段已补 validate:"required"。
// 注意 response.Adapter 是「最小引用形态」（仅 id/name），用于
// response.AccountResponse 的 adapter_ref，与本类型同名不同义，故不用它。
export type Adapter = Schemas['model.AdapterResponse'];

// Provider-Adapter 关联（多对多）
//
// 已收敛到生成类型：model.ProviderAdapterResponse 的 3 个非 omitempty 字段
// 已补 validate:"required"，与手写形态逐字段一致。
export type ProviderAdapter = Schemas['model.ProviderAdapterResponse'];

// 适配器响应（API 返回格式）
// 与 Adapter 字段完全一致（已与 model.AdapterResponse 逐字段核对），
// 合并为单一来源，消除重复定义。
// 现 Adapter 本身即 model.AdapterResponse 的别名，此处直接复用。
export type AdapterResponse = Adapter;

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
  WebAPI: 'webapi',
} as const;

export type AdapterName = typeof AdapterNames[keyof typeof AdapterNames];

// 适配器显示名称映射
export const AdapterDisplayNames: Record<string, string> = {
  gmail: 'Gmail API (OAuth2)',
  graph: 'Microsoft Graph (OAuth2)',
  imap: 'IMAP (密码认证)',
  webapi: 'Web API (Token)',
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
