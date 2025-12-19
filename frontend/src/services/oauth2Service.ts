import { api } from './api';

// OAuth2 相关类型定义
export interface OAuth2AuthResponse {
  auth_url: string;
  state: string;
}

export interface OAuth2CallbackResponse {
  account_uid: string;
  email: string;
  access_token: string;
  refresh_token: string;
  expires_at: string;
}

export interface OAuth2TokenRefreshResponse {
  access_token: string;
  expires_at: string;
}

export type OAuth2Provider = 'google' | 'microsoft';

export const oauth2Service = {
  /**
   * 生成 Google OAuth2 授权 URL
   */
  generateGoogleAuthUrl: async (email?: string, groupId?: number): Promise<OAuth2AuthResponse> => {
    const params = new URLSearchParams();
    if (email) params.append('email', email);
    if (groupId) params.append('group_id', groupId.toString());
    const queryString = params.toString() ? `?${params.toString()}` : '';
    const response = await api.get<{ success: boolean; data: OAuth2AuthResponse }>(
      `/auth/google/authorize${queryString}`
    );
    return response.data;
  },

  /**
   * 处理 Google OAuth2 授权回调
   */
  handleGoogleCallback: async (code: string, state: string): Promise<OAuth2CallbackResponse> => {
    const response = await api.post<{ success: boolean; data: OAuth2CallbackResponse }>(
      `/auth/google/callback?code=${encodeURIComponent(code)}&state=${encodeURIComponent(state)}`
    );
    return response.data;
  },

  /**
   * 刷新 Google OAuth2 访问令牌
   */
  refreshGoogleToken: async (accountUid: string): Promise<OAuth2TokenRefreshResponse> => {
    const response = await api.post<{ success: boolean; data: OAuth2TokenRefreshResponse }>(
      `/auth/google/refresh/${accountUid}`
    );
    return response.data;
  },

  /**
   * 撤销 Google OAuth2 授权
   */
  revokeGoogleToken: async (accountUid: string): Promise<void> => {
    await api.post(`/auth/google/revoke/${accountUid}`);
  },

  /**
   * 生成 Microsoft OAuth2 授权 URL
   */
  generateMicrosoftAuthUrl: async (email?: string, groupId?: number): Promise<OAuth2AuthResponse> => {
    const params = new URLSearchParams();
    if (email) params.append('email', email);
    if (groupId) params.append('group_id', groupId.toString());
    const queryString = params.toString() ? `?${params.toString()}` : '';
    const response = await api.get<{ success: boolean; data: OAuth2AuthResponse }>(
      `/auth/microsoft/authorize${queryString}`
    );
    return response.data;
  },

  /**
   * 处理 Microsoft OAuth2 授权回调
   */
  handleMicrosoftCallback: async (code: string, state: string): Promise<OAuth2CallbackResponse> => {
    const response = await api.post<{ success: boolean; data: OAuth2CallbackResponse }>(
      `/auth/microsoft/callback?code=${encodeURIComponent(code)}&state=${encodeURIComponent(state)}`
    );
    return response.data;
  },

  /**
   * 刷新 Microsoft OAuth2 访问令牌
   */
  refreshMicrosoftToken: async (accountUid: string): Promise<OAuth2TokenRefreshResponse> => {
    const response = await api.post<{ success: boolean; data: OAuth2TokenRefreshResponse }>(
      `/auth/microsoft/refresh/${accountUid}`
    );
    return response.data;
  },

  /**
   * 撤销 Microsoft OAuth2 授权
   */
  revokeMicrosoftToken: async (accountUid: string): Promise<void> => {
    await api.post(`/auth/microsoft/revoke/${accountUid}`);
  },

  /**
   * 通用方法：根据提供商生成授权 URL
   */
  generateAuthUrl: async (provider: OAuth2Provider, email?: string, groupId?: number): Promise<OAuth2AuthResponse> => {
    switch (provider) {
      case 'google':
        return oauth2Service.generateGoogleAuthUrl(email, groupId);
      case 'microsoft':
        return oauth2Service.generateMicrosoftAuthUrl(email, groupId);
      default:
        throw new Error(`Unsupported OAuth2 provider: ${provider}`);
    }
  },

  /**
   * 通用方法：处理授权回调
   */
  handleCallback: async (
    provider: OAuth2Provider,
    code: string,
    state: string
  ): Promise<OAuth2CallbackResponse> => {
    switch (provider) {
      case 'google':
        return oauth2Service.handleGoogleCallback(code, state);
      case 'microsoft':
        return oauth2Service.handleMicrosoftCallback(code, state);
      default:
        throw new Error(`Unsupported OAuth2 provider: ${provider}`);
    }
  },

  /**
   * 通用方法：刷新访问令牌
   */
  refreshToken: async (
    provider: OAuth2Provider,
    accountUid: string
  ): Promise<OAuth2TokenRefreshResponse> => {
    switch (provider) {
      case 'google':
        return oauth2Service.refreshGoogleToken(accountUid);
      case 'microsoft':
        return oauth2Service.refreshMicrosoftToken(accountUid);
      default:
        throw new Error(`Unsupported OAuth2 provider: ${provider}`);
    }
  },

  /**
   * 通用方法：撤销授权
   */
  revokeToken: async (provider: OAuth2Provider, accountUid: string): Promise<void> => {
    switch (provider) {
      case 'google':
        return oauth2Service.revokeGoogleToken(accountUid);
      case 'microsoft':
        return oauth2Service.revokeMicrosoftToken(accountUid);
      default:
        throw new Error(`Unsupported OAuth2 provider: ${provider}`);
    }
  },
};