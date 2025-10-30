import { api, clearAuthData } from '@/services/api'
import { API_ENDPOINTS } from '@/lib/constants'
import { useAuthStore, type User } from '@/stores/authStore'
import type { LoginResponse, ApiResponse } from '@/types/auth'

class AuthService {
  /**
   * 用户登录
   */
  async login(password: string): Promise<void> {
    const response = await api.post<ApiResponse<LoginResponse>>(
      API_ENDPOINTS.AUTH.LOGIN, 
      { password }
    )

    if (response.success && response.data) {
      const { token, expiresAt, user } = response.data
      
      // 使用后端返回的用户信息，或使用默认值
      const userInfo = user || { id: 1, email: 'admin', name: 'Admin' }
      
      // 更新 Zustand store（会自动持久化）
      useAuthStore.getState().login(userInfo, token, expiresAt)
    } else {
      throw new Error(response.error || '登录失败')
    }
  }

  /**
   * 用户退出登录
   */
  async logout(): Promise<void> {
    try {
      // 调用后端登出接口（可选）
      await api.post(API_ENDPOINTS.AUTH.LOGOUT)
    } catch (error) {
      // 即使后端登出失败，也要清除本地数据
      console.error('Logout API call failed:', error)
    } finally {
      // 清除所有认证数据
      clearAuthData()
      useAuthStore.getState().logout()
    }
  }

  /**
   * 检查用户是否已登录
   */
  isAuthenticated(): boolean {
    const store = useAuthStore.getState()
    
    // 检查是否已认证且 token 有效
    if (!store.isAuthenticated) {
      return false
    }

    // 检查 token 是否过期
    if (!store.isTokenValid()) {
      // Token 已过期，清除数据
      clearAuthData()
      store.logout()
      return false
    }

    return true
  }

  /**
   * 获取当前的认证 token
   */
  getToken(): string | null {
    return useAuthStore.getState().token
  }

  /**
   * 获取当前用户信息
   */
  getUser(): User | null {
    return useAuthStore.getState().user
  }
}

export const authService = new AuthService()