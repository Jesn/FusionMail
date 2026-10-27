// OAuth2 客户端配置相关类型定义
//
// API 契约类型以 openapi-typescript 生成的 api.generated.ts 为单一来源（backend/docs/swagger.json），
// 此处仅保留生成类型无法表达的运行时形态（见 P6 报告：OAuth2ClientListResponse 的 swagger 注解有误）。
import type { components } from './api.generated';

type Schemas = components['schemas'];

// OAuth2 客户端配置
// 注：后端 model.OAuth2ClientResponse 所有字段均未标 required（生成类型全可选），
// 前端消费方（OAuth2ClientsPage）依赖 id/name 等必填语义，故保留手写形态；
// 字段集合与 model.OAuth2ClientResponse 完全一致（已逐字段核对）。
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

// 创建 OAuth2 客户端请求 —— 与 model.OAuth2ClientCreateRequest 一致，改用生成类型
export type OAuth2ClientCreateRequest = Schemas['model.OAuth2ClientCreateRequest'];

// 更新 OAuth2 客户端请求 —— 与 model.OAuth2ClientUpdateRequest 一致，改用生成类型
export type OAuth2ClientUpdateRequest = Schemas['model.OAuth2ClientUpdateRequest'];

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

// 智能选择响应 —— 与 OAuth2Client 字段完全一致（swagger 中 smart-select 响应即 model.OAuth2ClientResponse），
// 合并为单一来源，消除重复定义
export type OAuth2ClientSmartSelectResponse = OAuth2Client;
