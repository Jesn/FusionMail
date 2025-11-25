import { api, clearAuthData } from '@/services/api'
import apiClient from '@/services/api'
import { API_ENDPOINTS } from '@/lib/constants'
import { useAuthStore, type User } from '@/stores/authStore'
import type { LoginResponse, ApiResponse } from '@/types/auth'
import axios from 'axios'
import { saveSettingsCache, clearSettingsCache } from '@/utils/settingsCache'

class AuthService {
  /**
   * 用户登录
   */
  async login(username: string, password: string): Promise<void> {
    try {
      // 使用 apiClient 直接调用，避免被全局响应拦截器处理
      const response = await apiClient.post<ApiResponse<LoginResponse>>(
        API_ENDPOINTS.AUTH.LOGIN,
        { username, password }
      )

      if (response.data.success && response.data.data) {
        const { token, expiresAt, user } = response.data.data

        // 使用后端返回的用户信息，或使用默认值
        const userInfo = user || {
          id: 1,
          username: username,
          email: 'admin@localhost',
          name: username,
          role: 'admin'
        }

        // 更新 Zustand store（会自动持久化）
        useAuthStore.getState().login(userInfo, token, expiresAt)

        // 登录成功后，立即加载用户设置并缓存
        this.loadAndCacheSettings(token).catch(error => {
          console.error('加载设置失败:', error)
          // 不影响登录流程，静默失败
        })
      } else {
        throw new Error(response.data.error || '登录失败')
      }
    } catch (error) {
      // 处理登录时的401错误
      if (axios.isAxiosError(error) && error.response?.status === 401) {
        throw new Error('用户名或密码错误')
      }
      // 其他错误直接抛出
      throw error
    }
  }

  /**
   * 加载并缓存用户设置
   */
  private async loadAndCacheSettings(token: string): Promise<void> {
    try {
      // 使用 apiClient 而不是 fetch，确保使用正确的 baseURL
      const [uiResponse, syncResponse, notificationResponse] = await Promise.all([
        apiClient.get('/settings/ui', {
          headers: { 'Authorization': `Bearer ${token}` }
        }),
        apiClient.get('/settings/sync', {
          headers: { 'Authorization': `Bearer ${token}` }
        }),
        apiClient.get('/settings/notification', {
          headers: { 'Authorization': `Bearer ${token}` }
        })
      ]);

      // apiClient 返回的是 AxiosResponse，直接使用 data
      const uiData = uiResponse.data;
      const syncData = syncResponse.data;
      const notificationData = notificationResponse.data;

      // 保存到缓存
      saveSettingsCache({
        ui: uiData?.data?.settings || {},
        sync: syncData?.data?.settings || {},
        notification: notificationData?.data?.settings || {}
      })

      console.log('用户设置已加载并缓存')
    } catch (error) {
      console.error('加载设置失败:', error)
      throw error
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
      // 清除所有认证数据和设置缓存
      clearAuthData()
      clearSettingsCache()
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