import { httpClient } from '@/lib/httpClient'
import { STORAGE_KEYS, API_ENDPOINTS } from '@/lib/constants'

interface LoginResponse {
  token: string
  expiresAt: string
}

class AuthService {
  private readonly TOKEN_KEY = STORAGE_KEYS.AUTH_TOKEN
  private readonly EXPIRES_KEY = STORAGE_KEYS.AUTH_EXPIRES

  /**
   * 用户登录
   */
  async login(password: string): Promise<void> {
    const response = await httpClient.post<LoginResponse>(API_ENDPOINTS.AUTH.LOGIN, {
      password
    })

    if (response.success && response.data) {
      this.setToken(response.data.token, response.data.expiresAt)
    } else {
      throw new Error(response.error?.message || '登录失败')
    }
  }

  /**
   * 用户退出登录
   */
  logout(): void {
    localStorage.removeItem(this.TOKEN_KEY)
    localStorage.removeItem(this.EXPIRES_KEY)
  }

  /**
   * 检查用户是否已登录
   */
  isAuthenticated(): boolean {
    const token = this.getToken()
    const expiresAt = localStorage.getItem(this.EXPIRES_KEY)

    if (!token || !expiresAt) {
      return false
    }

    // 检查 token 是否过期
    const expirationTime = new Date(expiresAt).getTime()
    const currentTime = Date.now()

    if (currentTime >= expirationTime) {
      this.logout() // 清除过期的 token
      return false
    }

    return true
  }

  /**
   * 获取当前的认证 token
   */
  getToken(): string | null {
    return localStorage.getItem(this.TOKEN_KEY)
  }

  /**
   * 设置认证 token
   */
  private setToken(token: string, expiresAt: string): void {
    localStorage.setItem(this.TOKEN_KEY, token)
    localStorage.setItem(this.EXPIRES_KEY, expiresAt)
  }
}

export const authService = new AuthService()