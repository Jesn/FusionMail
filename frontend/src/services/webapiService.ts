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

  // ============================================
  // 账户配置操作
  // ============================================

  /**
   * 获取账户的 WebAPI 配置
   * @param accountUid 账户 UID
   */
  async getAccountConfig(accountUid: string): Promise<{
    service_type: string;
    auth_data: any;
  } | null> {
    try {
      const response = await api.get<WebAPIApiResponse<{
        service_type: string;
        auth_data: any;
      }>>(`${BASE_PATH}/accounts/${accountUid}/config`);
      if (!response.success) {
        return null;
      }
      return response.data;
    } catch {
      return null;
    }
  },

  /**
   * 更新账户的 WebAPI 配置
   * @param accountUid 账户 UID
   * @param data 配置数据
   */
  async updateAccountConfig(accountUid: string, data: {
    service_type: string;
    auth_data: string;
  }): Promise<void> {
    const response = await api.put<WebAPIApiResponse<null>>(
      `${BASE_PATH}/accounts/${accountUid}/config`,
      data
    );
    if (!response.success) {
      throw new Error(response.error || response.message || '更新配置失败');
    }
  },

  // ============================================
  // Cloudflare Temp Email 专用接口
  // ============================================

  /**
   * 获取 Cloudflare Temp Email 设置信息
   * 通过 JWT Token 获取邮箱地址和可用域名
   * @param baseUrl API 基础地址
   * @param jwtToken JWT Token
   */
  async fetchCloudflareTempEmailSettings(baseUrl: string, jwtToken: string): Promise<{
    email: string;
    domains?: string[];
  }> {
    const response = await api.post<WebAPIApiResponse<{
      email: string;
      domains?: string[];
    }>>(`${BASE_PATH}/cloudflare/settings`, {
      base_url: baseUrl,
      jwt_token: jwtToken,
    });
    if (!response.success) {
      throw new Error(response.error || response.message || '获取设置失败');
    }
    return response.data;
  },

  // ============================================
  // 子邮箱账户查询
  // ============================================

  /**
   * 获取 WebAPI 账户关联的子邮箱列表（FusionMail 本地子账户，非远端实时列表）
   * @param parentUid 父账户 UID
   * @param include active | orphaned | all
   */
  async getChildAccounts(parentUid: string, include: 'active' | 'orphaned' | 'all' = 'active'): Promise<{
    uid: string;
    email: string;
    status: string;
    disable_reason: string;
    total_emails: number;
    unread_count: number;
    last_sync_at: string | null;
    created_at: string;
    orphaned: boolean;
  }[]> {
    const response = await api.get<WebAPIApiResponse<{
      uid: string;
      email: string;
      status: string;
      disable_reason: string;
      total_emails: number;
      unread_count: number;
      last_sync_at: string | null;
      created_at: string;
      orphaned: boolean;
    }[]>>(`${BASE_PATH}/providers/${parentUid}/children`, {
      params: { include },
    });
    if (!response.success) {
      throw new Error(response.error || response.message || '获取子邮箱列表失败');
    }
    return response.data || [];
  },

  /**
   * 将本地子邮箱与远端有效地址对账：
   * 远端已不存在 → 标记为孤儿（禁用，保留邮件）；远端又存在 → 恢复 active
   */
  async reconcileChildAccounts(parentUid: string): Promise<{
    remote_count: number;
    local_count: number;
    marked_orphaned: number;
    reactivated: number;
    unchanged: number;
    orphaned_emails: string[];
    reactivated_emails: string[];
    skipped_remote: boolean;
    message?: string;
  }> {
    const response = await api.post<WebAPIApiResponse<{
      remote_count: number;
      local_count: number;
      marked_orphaned: number;
      reactivated: number;
      unchanged: number;
      orphaned_emails: string[];
      reactivated_emails: string[];
      skipped_remote: boolean;
      message?: string;
    }>>(`${BASE_PATH}/providers/${parentUid}/children/reconcile`);
    if (!response.success) {
      throw new Error(response.error || response.message || '对账失败');
    }
    if (!response.data) {
      throw new Error('对账结果为空');
    }
    return response.data;
  },

  /**
   * 获取 Cloud Mail 服务端的账户列表
   * 通过调用 Cloud Mail API 获取所有邮箱账户
   * @param accountUid 账户 UID
   */
  async getCloudMailAccounts(accountUid: string): Promise<{
    account_id: number;
    email: string;
    name: string;
  }[]> {
    const response = await api.get<WebAPIApiResponse<{
      account_id: number;
      email: string;
      name: string;
    }[]>>(`${BASE_PATH}/providers/${accountUid}/cloudmail-accounts`);
    if (!response.success) {
      throw new Error(response.error || response.message || '获取 Cloud Mail 账户列表失败');
    }
    return response.data || [];
  },
};


export default webapiService;
