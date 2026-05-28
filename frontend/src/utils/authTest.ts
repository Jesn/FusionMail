/**
 * 认证功能测试工具
 * 用于在浏览器控制台中测试认证功能
 */

import { authService } from '@/services/authService'
import { useAuthStore } from '@/stores/authStore'
import { tokenRefreshService } from '@/services/tokenRefreshService'
import { clearAuthData } from '@/services/api'

export const authTest = {
  /**
   * 测试登录
   */
  async testLogin(password: string = 'admin123') {
    console.log('[AuthTest] Testing login...')
    try {
      await authService.login('admin', password)
      console.log('[AuthTest] ✅ Login successful')
      console.log('[AuthTest] User:', useAuthStore.getState().user)
      console.log('[AuthTest] ExpiresAt:', useAuthStore.getState().expiresAt)
    } catch (error) {
      console.error('[AuthTest] ❌ Login failed:', error)
    }
  },

  /**
   * 测试登出
   */
  async testLogout() {
    console.log('[AuthTest] Testing logout...')
    try {
      await authService.logout()
      console.log('[AuthTest] ✅ Logout successful')
      console.log('[AuthTest] User:', useAuthStore.getState().user)
      console.log('[AuthTest] IsAuthenticated:', useAuthStore.getState().isAuthenticated)
    } catch (error) {
      console.error('[AuthTest] ❌ Logout failed:', error)
    }
  },

  /**
   * 测试认证状态检查
   */
  testIsAuthenticated() {
    console.log('[AuthTest] Testing isAuthenticated...')
    const isAuth = authService.isAuthenticated()
    const store = useAuthStore.getState()
    console.log('[AuthTest] IsAuthenticated:', isAuth)
    console.log('[AuthTest] Store state:', {
      isAuthenticated: store.isAuthenticated,
      hasUser: !!store.user,
      isSessionValid: store.isTokenValid(),
    })
  },

  /**
   * 测试会话有效性
   */
  testTokenValidity() {
    console.log('[AuthTest] Testing session validity...')
    const store = useAuthStore.getState()
    const isValid = store.isTokenValid()
    
    if (store.expiresAt) {
      const expirationTime = new Date(store.expiresAt).getTime()
      const currentTime = Date.now()
      const timeUntilExpiry = expirationTime - currentTime
      const minutesUntilExpiry = Math.floor(timeUntilExpiry / 1000 / 60)
      
      console.log('[AuthTest] Session valid:', isValid)
      console.log('[AuthTest] Expires at:', store.expiresAt)
      console.log('[AuthTest] Time until expiry:', minutesUntilExpiry, 'minutes')
    } else {
      console.log('[AuthTest] No active session found')
    }
  },

  /**
   * 测试会话刷新
   */
  async testTokenRefresh() {
    console.log('[AuthTest] Testing session refresh...')
    try {
      const oldExpiresAt = useAuthStore.getState().expiresAt
      await tokenRefreshService.manualRefresh()
      const newExpiresAt = useAuthStore.getState().expiresAt
      
      if (oldExpiresAt !== newExpiresAt) {
        console.log('[AuthTest] ✅ Session refreshed successfully')
        console.log('[AuthTest] Old expiresAt:', oldExpiresAt)
        console.log('[AuthTest] New expiresAt:', newExpiresAt)
      } else {
        console.log('[AuthTest] ⚠️  Session expiry not changed (might be too early to refresh)')
      }
    } catch (error) {
      console.error('[AuthTest] ❌ Session refresh failed:', error)
    }
  },

  /**
   * 测试自动刷新服务
   */
  testAutoRefreshService() {
    console.log('[AuthTest] Testing auto refresh service...')
    console.log('[AuthTest] Starting service...')
    tokenRefreshService.start()
    console.log('[AuthTest] ✅ Service started, check console for periodic checks')
    console.log('[AuthTest] To stop: authTest.stopAutoRefresh()')
  },

  /**
   * 停止自动刷新服务
   */
  stopAutoRefresh() {
    console.log('[AuthTest] Stopping auto refresh service...')
    tokenRefreshService.stop()
    console.log('[AuthTest] ✅ Service stopped')
  },

  /**
   * 测试清理数据
   */
  testClearData() {
    console.log('[AuthTest] Testing clear data...')
    clearAuthData()
    useAuthStore.getState().logout()
    console.log('[AuthTest] ✅ Data cleared')
    console.log('[AuthTest] localStorage:', {
      auth_token: localStorage.getItem('auth_token'),
      auth_expires: localStorage.getItem('auth_expires'),
      'fusionmail-auth': localStorage.getItem('fusionmail-auth'),
    })
  },

  /**
   * 模拟会话即将过期
   */
  simulateTokenExpiringSoon() {
    console.log('[AuthTest] Simulating session expiring soon...')
    const store = useAuthStore.getState()
    if (store.user) {
      // 设置前端会话时间在 9 分钟后过期
      const expiresAt = new Date(Date.now() + 9 * 60 * 1000).toISOString()
      store.login(store.user, null, expiresAt)
      console.log('[AuthTest] ✅ Session set to expire in 9 minutes')
      console.log('[AuthTest] ExpiresAt:', expiresAt)
      console.log('[AuthTest] Auto refresh should trigger soon...')
    } else {
      console.log('[AuthTest] ❌ No active session')
    }
  },

  /**
   * 模拟会话已过期
   */
  simulateTokenExpired() {
    console.log('[AuthTest] Simulating session expired...')
    const store = useAuthStore.getState()
    if (store.user) {
      // 设置前端会话时间在 1 分钟前过期
      const expiresAt = new Date(Date.now() - 1 * 60 * 1000).toISOString()
      store.login(store.user, null, expiresAt)
      console.log('[AuthTest] ✅ Session set to expired')
      console.log('[AuthTest] ExpiresAt:', expiresAt)
      console.log('[AuthTest] Try to access a protected page or call isAuthenticated()')
    } else {
      console.log('[AuthTest] ❌ No active session')
    }
  },

  /**
   * 显示当前状态
   */
  showCurrentState() {
    console.log('[AuthTest] Current state:')
    const store = useAuthStore.getState()
    console.log({
      user: store.user,
      expiresAt: store.expiresAt,
      isAuthenticated: store.isAuthenticated,
      isSessionValid: store.isTokenValid(),
    })
    console.log('[AuthTest] localStorage:')
    console.log({
      'fusionmail-auth': localStorage.getItem('fusionmail-auth'),
    })
  },

  /**
   * 运行所有测试
   */
  async runAllTests(password: string = 'admin123') {
    console.log('[AuthTest] ========================================')
    console.log('[AuthTest] Running all tests...')
    console.log('[AuthTest] ========================================')
    
    // 1. 测试登录
    await this.testLogin(password)
    await new Promise(resolve => setTimeout(resolve, 1000))
    
    // 2. 测试认证状态
    this.testIsAuthenticated()
    await new Promise(resolve => setTimeout(resolve, 1000))
    
    // 3. 测试会话有效性
    this.testTokenValidity()
    await new Promise(resolve => setTimeout(resolve, 1000))
    
    // 4. 显示当前状态
    this.showCurrentState()
    await new Promise(resolve => setTimeout(resolve, 1000))
    
    // 5. 测试登出
    await this.testLogout()
    
    console.log('[AuthTest] ========================================')
    console.log('[AuthTest] All tests completed!')
    console.log('[AuthTest] ========================================')
  },

  /**
   * 显示帮助信息
   */
  help() {
    console.log(`
[AuthTest] Available commands:
  
  authTest.testLogin(password)          - 测试登录
  authTest.testLogout()                 - 测试登出
  authTest.testIsAuthenticated()        - 测试认证状态检查
  authTest.testTokenValidity()          - 测试会话有效性
  authTest.testTokenRefresh()           - 测试会话刷新
  authTest.testAutoRefreshService()     - 测试自动刷新服务
  authTest.stopAutoRefresh()            - 停止自动刷新服务
  authTest.testClearData()              - 测试清理数据
  authTest.simulateTokenExpiringSoon()  - 模拟会话即将过期
  authTest.simulateTokenExpired()       - 模拟会话已过期
  authTest.showCurrentState()           - 显示当前状态
  authTest.runAllTests(password)        - 运行所有测试
  authTest.help()                       - 显示帮助信息
    `)
  }
}

// 在开发环境中暴露到全局
if (import.meta.env.DEV) {
  (window as any).authTest = authTest
  console.log('[AuthTest] Test utilities loaded. Type "authTest.help()" for available commands.')
}
