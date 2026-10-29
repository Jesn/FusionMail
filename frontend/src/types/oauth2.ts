// OAuth2 客户端配置相关类型定义
//
// API 契约类型以 openapi-typescript 生成的 api.generated.ts 为单一来源（backend/docs/swagger.json），
// 此处仅保留生成类型无法表达的运行时形态（生成类型全字段可选，无法表达前端依赖的必填语义）。
import type { components } from './api.generated';

type Schemas = components['schemas'];

// OAuth2 客户端配置
//
// 已收敛到生成类型：后端 model.OAuth2ClientResponse 已补 validate:"required"
// （14 个非 omitempty 字段），生成类型不再全可选。
export type OAuth2Client = Schemas['model.OAuth2ClientResponse'];

// 创建 OAuth2 客户端请求 —— 与 model.OAuth2ClientCreateRequest 一致，改用生成类型
export type OAuth2ClientCreateRequest = Schemas['model.OAuth2ClientCreateRequest'];

// 更新 OAuth2 客户端请求 —— 与 model.OAuth2ClientUpdateRequest 一致，改用生成类型
export type OAuth2ClientUpdateRequest = Schemas['model.OAuth2ClientUpdateRequest'];

// 分页响应
//
// 已收敛到生成类型：response.OAuth2ClientListResponse 的 5 个字段已全部标 required。
// 已核实（2026-10-29）：swagger 的字段与后端
// backend/internal/dto/response/oauth2_client.go 的结构体完全一致，
// 原先「swagger 注解有误」的说法已不成立，故移除该说明。
export type OAuth2ClientListResponse = Schemas['response.OAuth2ClientListResponse'];

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
