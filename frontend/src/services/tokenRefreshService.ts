import { api } from '@/services/api'
import { useAuthStore } from '@/stores/authStore'
import { API_ENDPOINTS } from '@/lib/constants'
import type { RefreshTokenResponse, ApiResponse } from '@/types/auth'

class TokenRefreshService {
  private refreshTimer: ReturnType<typeof setInterval> | null = null
  private isRefreshing = false
  private readonly CHECK_INTERVAL = 5 * 60 * 1000 // 每 5 分钟检查一次
  private readonly REFRESH_THRESHOLD = 10 * 60 * 1000 // token 过期前 10 分钟刷新

  /**
   * 启动自动刷新
   */
  start(): void {
    this.stop() // 先停止之前的定时器
    
    console.log('[TokenRefresh] Service started')
    
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
      console.log('[TokenRefresh] Service stopped')
    }
  }

  /**
   * 检查并刷新 token
   */
  private async checkAndRefresh(): Promise<void> {
    if (this.isRefreshing) {
      console.log('[TokenRefresh] Already refreshing, skipping')
      return
    }

    const store = useAuthStore.getState()
    const { token, expiresAt } = store

    if (!token || !expiresAt) {
      console.log('[TokenRefresh] No token or expiresAt, skipping')
      return
    }

    // 计算 token 剩余有效时间
    const expirationTime = new Date(expiresAt).getTime()
    const currentTime = Date.now()
    const timeUntilExpiry = expirationTime - currentTime

    console.log('[TokenRefresh] Time until expiry:', Math.floor(timeUntilExpiry / 1000 / 60), 'minutes')

    // 如果 token 在 10 分钟内过期，则刷新
    if (timeUntilExpiry < this.REFRESH_THRESHOLD && timeUntilExpiry > 0) {
      console.log('[TokenRefresh] Token expiring soon, refreshing...')
      await this.refresh()
    } else if (timeUntilExpiry <= 0) {
      console.log('[TokenRefresh] Token already expired')
      store.logout()
    }
  }

  /**
   * 刷新 token
   */
  async refresh(): Promise<void> {
    if (this.isRefreshing) {
      console.log('[TokenRefresh] Already refreshing')
      return
    }

    this.isRefreshing = true

    try {
      const currentToken = useAuthStore.getState().token
      
      if (!currentToken) {
        console.log('[TokenRefresh] No token to refresh')
        return
      }

      console.log('[TokenRefresh] Sending refresh request...')
      
      const response = await api.post<ApiResponse<RefreshTokenResponse>>(
        API_ENDPOINTS.AUTH.REFRESH, 
        { token: currentToken }
      )

      if (response.success && response.data) {
        const { token, expiresAt } = response.data
        const store = useAuthStore.getState()
        
        // 更新 token，保持用户信息不变
        if (store.user) {
          store.login(store.user, token, expiresAt)
          console.log('[TokenRefresh] Token refreshed successfully')
        }
      }
    } catch (error) {
      console.error('[TokenRefresh] Token refresh failed:', error)
      // 刷新失败，让用户重新登录
      // 注意：不要在这里调用 logout，因为 api 拦截器会处理 401
    } finally {
      this.isRefreshing = false
    }
  }

  /**
   * 手动刷新 token
   */
  async manualRefresh(): Promise<void> {
    console.log('[TokenRefresh] Manual refresh triggered')
    await this.refresh()
  }
}

export const tokenRefreshService = new TokenRefreshService()
