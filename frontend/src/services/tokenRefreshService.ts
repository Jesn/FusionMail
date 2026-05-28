import { api } from '@/services/api'
import { useAuthStore } from '@/stores/authStore'
import { API_ENDPOINTS } from '@/lib/constants'
import type { RefreshTokenResponse, ApiResponse } from '@/types/auth'

class TokenRefreshService {
  private refreshTimer: ReturnType<typeof setInterval> | null = null
  private isRefreshing = false
  private readonly CHECK_INTERVAL = 5 * 60 * 1000 // 每 5 分钟检查一次
  private readonly REFRESH_THRESHOLD = 10 * 60 * 1000 // 会话过期前 10 分钟刷新

  /**
   * 启动自动刷新
   */
  start(): void {
    this.stop() // 先停止之前的定时器
    
    // Session refresh service started
    
    // 每 5 分钟检查一次
    this.refreshTimer = setInterval(() => {
      this.checkAndRefresh()
    }, this.CHECK_INTERVAL)
    
    // 立即检查一次
    this.checkAndRefresh()
  }

  /**
   * 停止自动刷新
   */
  stop(): void {
    if (this.refreshTimer) {
      clearInterval(this.refreshTimer)
      this.refreshTimer = null
      // Session refresh service stopped
    }
  }

  /**
   * 检查并刷新 Cookie 会话
   */
  private async checkAndRefresh(): Promise<void> {
    if (this.isRefreshing) {
      // Already refreshing, skipping
      return
    }

    const store = useAuthStore.getState()
    const { expiresAt } = store

    if (!expiresAt) {
      // No session expiry, skipping
      return
    }

    // 计算会话剩余有效时间
    const expirationTime = new Date(expiresAt).getTime()
    const currentTime = Date.now()
    const timeUntilExpiry = expirationTime - currentTime

    // Check session expiry time

    // 如果会话在 10 分钟内过期，则刷新
    if (timeUntilExpiry < this.REFRESH_THRESHOLD && timeUntilExpiry > 0) {
      // Session expiring soon, refreshing
      await this.refresh()
    } else if (timeUntilExpiry <= 0) {
      // Session already expired
      store.logout()
    }
  }

  /**
   * 刷新 Cookie 会话
   */
  async refresh(): Promise<void> {
    if (this.isRefreshing) {
      // Already refreshing
      return
    }

    this.isRefreshing = true

    try {
      const response = await api.post<ApiResponse<RefreshTokenResponse>>(
        API_ENDPOINTS.AUTH.REFRESH
      )

      if (response.success && response.data) {
        const { expiresAt } = response.data
        const store = useAuthStore.getState()
        
        // 更新会话过期时间，保持用户信息不变
        if (store.user) {
          store.login(store.user, null, expiresAt)
        }
      }
    } catch (error) {
      console.error('[TokenRefresh] Session refresh failed:', error)
      // 刷新失败，让用户重新登录
      // 注意：不要在这里调用 logout，因为 api 拦截器会处理 401
    } finally {
      this.isRefreshing = false
    }
  }

  /**
   * 手动刷新会话
   */
  async manualRefresh(): Promise<void> {
    // Manual refresh triggered
    await this.refresh()
  }
}

export const tokenRefreshService = new TokenRefreshService()
