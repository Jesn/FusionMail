// WebAPI Provider 服务层
import { api } from './api';
import type {
  CreateWebAPIProviderRequest,
  UpdateWebAPIProviderRequest,
  TestConnectionRequest,
  TestConnectionResult,
  WebAPIProviderResponse,
  WebAPIProviderListResponse,
  WebAPISyncStatus,
  WebAPIServiceTemplate,
  WebAPIServicesResponse,
  WebAPIApiResponse,
  WebAPIServiceType,
} from '../types/webapi';

// API 路径前缀
const BASE_PATH = '/webapi';

/**
 * WebAPI Provider 服务
 */
export const webapiService = {
  // ============================================
  // Provider CRUD 操作
  // ============================================

  /**
   * 创建 WebAPI Provider
   */
  async create(data: CreateWebAPIProviderRequest): Promise<WebAPIProviderResponse> {
    const response = await api.post<WebAPIApiResponse<WebAPIProviderResponse>>(
      `${BASE_PATH}/providers`,
      data
    );
    if (!response.success) {
      throw new Error(response.error || response.message || '创建失败');
    }
    return response.data;
  },

  /**
   * 获取 WebAPI Provider 列表
   */
  async list(page = 1, pageSize = 20): Promise<WebAPIProviderListResponse> {
    const response = await api.get<WebAPIProviderListResponse>(
      `${BASE_PATH}/providers`,
      { params: { page, page_size: pageSize } }
    );
    return response;
  },

  /**
   * 获取单个 WebAPI Provider
   */
  async get(uid: string): Promise<WebAPIProviderResponse> {
    const response = await api.get<WebAPIApiResponse<WebAPIProviderResponse>>(
      `${BASE_PATH}/providers/${uid}`
    );
    if (!response.success) {
      throw new Error(response.error || response.message || '获取失败');
    }
    return response.data;
  },

  /**
   * 更新 WebAPI Provider
   */
  async update(uid: string, data: UpdateWebAPIProviderRequest): Promise<WebAPIProviderResponse> {
    const response = await api.put<WebAPIApiResponse<WebAPIProviderResponse>>(
      `${BASE_PATH}/providers/${uid}`,
      data
    );
    if (!response.success) {
      throw new Error(response.error || response.message || '更新失败');
    }
    return response.data;
  },

  /**
   * 删除 WebAPI Provider
   */
  async delete(uid: string): Promise<void> {
    const response = await api.delete<WebAPIApiResponse<null>>(
      `${BASE_PATH}/providers/${uid}`
    );
    if (!response.success) {
      throw new Error(response.error || response.message || '删除失败');
    }
  },

  // ============================================
  // 连接测试
  // ============================================

  /**
   * 测试连接（使用配置）
   */
  async testConnection(data: TestConnectionRequest): Promise<TestConnectionResult> {
    const response = await api.post<WebAPIApiResponse<TestConnectionResult>>(
      `${BASE_PATH}/providers/test`,
      data
    );
    if (!response.success) {
      throw new Error(response.error || response.message || '测试失败');
    }
    return response.data;
  },

  /**
   * 测试已存在 Provider 的连接
   */
  async testConnectionByUID(uid: string): Promise<TestConnectionResult> {
    const response = await api.post<WebAPIApiResponse<TestConnectionResult>>(
      `${BASE_PATH}/providers/${uid}/test`
    );
    if (!response.success) {
      throw new Error(response.error || response.message || '测试失败');
    }
    return response.data;
  },

  // ============================================
  // 同步操作
  // ============================================

  /**
   * 手动触发同步
   */
  async triggerSync(uid: string): Promise<void> {
    const response = await api.post<WebAPIApiResponse<null>>(
      `${BASE_PATH}/providers/${uid}/sync`
    );
    if (!response.success) {
      throw new Error(response.error || response.message || '触发同步失败');
    }
  },

  /**
   * 获取同步状态
   */
  async getSyncStatus(uid: string): Promise<WebAPISyncStatus> {
    const response = await api.get<WebAPIApiResponse<WebAPISyncStatus>>(
      `${BASE_PATH}/providers/${uid}/sync-status`
    );
    if (!response.success) {
      throw new Error(response.error || response.message || '获取同步状态失败');
    }
    return response.data;
  },

  // ============================================
  // 服务模板
  // ============================================

  /**
   * 获取支持的服务列表
   */
  async getServices(): Promise<{
    services: WebAPIServiceTemplate[];
    supported_types: WebAPIServiceType[];
  }> {
    const response = await api.get<WebAPIServicesResponse>(
      `${BASE_PATH}/services`
    );
    if (!response.success) {
      throw new Error((response as any).error || '获取服务列表失败');
    }
    return response.data;
  },

  /**
   * 获取服务详情
   */
  async getServiceDetail(serviceType: WebAPIServiceType): Promise<WebAPIServiceTemplate> {
    const response = await api.get<WebAPIApiResponse<WebAPIServiceTemplate>>(
      `${BASE_PATH}/services/${serviceType}`
    );
    if (!response.success) {
      throw new Error(response.error || response.message || '获取服务详情失败');
    }
    return response.data;
  },

  /**
   * 获取支持的服务类型列表
   */
  async getSupportedTypes(): Promise<WebAPIServiceType[]> {
    const response = await api.get<WebAPIApiResponse<WebAPIServiceType[]>>(
      `${BASE_PATH}/services/types`
    );
    if (!response.success) {
      throw new Error(response.error || response.message || '获取服务类型失败');
    }
    return response.data;
  },

  /**
   * 验证配置
   */
  async validateConfig(serviceType: WebAPIServiceType, authData: string): Promise<{
    valid: boolean;
    errors?: string[];
  }> {
    const response = await api.post<WebAPIApiResponse<{ valid: boolean; errors?: string[] }>>(
      `${BASE_PATH}/services/validate`,
      { service_type: serviceType, auth_data: authData }
    );
    if (!response.success) {
      throw new Error(response.error || response.message || '验证失败');
    }
    return response.data;
  },
};

export default webapiService;
