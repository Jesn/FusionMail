/**
 * 认证相关类型定义
 */

export interface User {
  id: number
  email: string
  name?: string
  avatar?: string
}

export interface LoginRequest {
  password: string
}

export interface LoginResponse {
  token: string
  expiresAt: string
  user?: User
}

export interface RefreshTokenRequest {
  token: string
}

export interface RefreshTokenResponse {
  token: string
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
