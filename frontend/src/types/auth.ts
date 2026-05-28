/**
 * 认证相关类型定义
 */

export interface User {
  id: number
  username: string
  email: string
  displayName?: string
  name?: string
  avatar?: string
  role: string
  theme?: string
}

export interface LoginRequest {
  username: string
  password: string
}

export interface LoginResponse {
  expiresAt: string
  user?: User
  // 2FA 相关字段
  requires_2fa?: boolean
  two_factor_user_id?: number
  two_factor_challenge_token?: string
  two_factor_challenge_expiry?: string
}

export interface RefreshTokenRequest {}

export interface RefreshTokenResponse {
  expiresAt: string
}

export interface AuthState {
  user: User | null
  token: string | null
  expiresAt: string | null
  isAuthenticated: boolean
}

export interface ApiResponse<T = any> {
  success: boolean
  data?: T
  error?: string
  message?: string
  timestamp?: string
}
