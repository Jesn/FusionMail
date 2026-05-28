/**
 * 双因素认证 (2FA) 服务
 */

import apiClient from '@/services/api'

// 2FA 状态响应
export interface TwoFactorStatus {
  enabled: boolean
  enabled_at: string | null
  backup_codes_count: number
}

// 2FA 设置响应
export interface TwoFactorSetupResponse {
  secret: string
  qr_code_url: string
  backup_codes: string[]
}

// API 响应包装
interface ApiResponse<T> {
  success: boolean
  data: T
  error?: string
}

class TwoFactorService {
  /**
   * 获取 2FA 状态
   */
  async getStatus(): Promise<TwoFactorStatus> {
    const response = await apiClient.get<ApiResponse<TwoFactorStatus>>('/auth/2fa/status')
    if (response.data.success) {
      return response.data.data
    }
    throw new Error(response.data.error || '获取 2FA 状态失败')
  }

  /**
   * 设置 2FA（生成密钥和二维码）
   */
  async setup(): Promise<TwoFactorSetupResponse> {
    const response = await apiClient.post<ApiResponse<TwoFactorSetupResponse>>('/auth/2fa/setup')
    if (response.data.success) {
      return response.data.data
    }
    throw new Error(response.data.error || '设置 2FA 失败')
  }

  /**
   * 验证并启用 2FA
   */
  async verify(code: string): Promise<void> {
    const response = await apiClient.post<ApiResponse<null>>('/auth/2fa/verify', { code })
    if (!response.data.success) {
      throw new Error(response.data.error || '验证失败')
    }
  }

  /**
   * 禁用 2FA
   */
  async disable(password: string, code?: string): Promise<void> {
    const response = await apiClient.post<ApiResponse<null>>('/auth/2fa/disable', { 
      password, 
      code 
    })
    if (!response.data.success) {
      throw new Error(response.data.error || '禁用 2FA 失败')
    }
  }

  /**
   * 重新生成恢复码
   */
  async regenerateBackupCodes(code: string): Promise<string[]> {
    const response = await apiClient.post<ApiResponse<{ backup_codes: string[] }>>('/auth/2fa/backup-codes', { code })
    if (response.data.success) {
      return response.data.data.backup_codes
    }
    throw new Error(response.data.error || '重新生成恢复码失败')
  }

  /**
   * 登录时验证 2FA 并建立 Cookie 会话
   */
  async validateLogin(userId: number, code: string, challengeToken: string): Promise<TwoFactorLoginResponse> {
    const response = await apiClient.post<ApiResponse<TwoFactorLoginResponse>>('/auth/2fa/validate', {
      user_id: userId,
      code,
      two_factor_challenge_token: challengeToken
    })
    if (response.data.success) {
      return response.data.data
    }
    throw new Error(response.data.error || '验证码错误')
  }
}

// 2FA 登录验证响应（认证凭据由 HttpOnly Cookie 承载）
export interface TwoFactorLoginResponse {
  expiresAt: string
  user: {
    id: number
    username: string
    email: string
    display_name: string
    role: string
    theme: string
  }
}

export const twoFactorService = new TwoFactorService()
