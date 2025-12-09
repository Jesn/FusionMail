import axios from 'axios'
import type { AxiosInstance, AxiosRequestConfig, AxiosError } from 'axios'
import { toast } from 'sonner'
import { useAuthStore } from '../stores/authStore'

// API 基础 URL
// 在开发模式下使用相对路径（通过 Vite 代理），生产模式下使用完整 URL；
// 同时自动补全 /api/v1，避免遗漏导致请求落到错误路径（例如直接落到 3333 根路径返回 HTML）
const API_BASE_URL = (() => {
  const envBase = import.meta.env.VITE_API_BASE_URL as string | undefined

  if (envBase) {
    const trimmed = envBase.replace(/\/$/, '')
    if (trimmed.endsWith('/api/v1')) {
      return trimmed
    }
    return `${trimmed}/api/v1`
  }

  // 始终使用相对路径，避免跨域问题
  // 前端和后端部署在同一域名下时，相对路径会自动使用当前域名
  return '/api/v1'
})()

/**
 * 清除所有认证数据
 */
export function clearAuthData(): void {
  localStorage.removeItem('auth_token')
  localStorage.removeItem('auth_expires')
  localStorage.removeItem('fusionmail-auth')
}

// 创建 axios 实例
const apiClient: AxiosInstance = axios.create({
  baseURL: API_BASE_URL,
  timeout: 30000,
  headers: {
    'Content-Type': 'application/json',
  },
  withCredentials: true, // 支持 Cookie（SSE 鉴权需要）
})

// 请求拦截器 - 添加认证 token
apiClient.interceptors.request.use(
  (config) => {
    // 尝试从多个来源获取 token
    let token = useAuthStore.getState().token

    // 如果 store 中没有，尝试从 localStorage 读取（用于处理 store 初始化时序问题）
    if (!token) {
      try {
        const authData = localStorage.getItem('fusionmail-auth')
        if (authData) {
          const parsed = JSON.parse(authData)
          token = parsed?.state?.token
        }
      } catch (e) {
        console.error('Failed to read token from localStorage:', e)
      }
    }

    if (token) {
      config.headers.Authorization = `Bearer ${token}`
    }
    return config
  },
  (error) => {
    return Promise.reject(error)
  }
)

// 响应拦截器 - 统一错误处理
apiClient.interceptors.response.use(
  (response) => {
    return response
  },
  (error: AxiosError) => {
    const status = error.response?.status

    // 处理不同的 HTTP 状态码
    if (status === 401) {
      // 未授权 - 清除认证数据
      clearAuthData()
      useAuthStore.getState().logout()

      // 如果当前不是登录页面，才重定向
      const currentPath = window.location.pathname
      if (currentPath !== '/login') {
        window.location.href = '/login'
        toast.error('登录已过期，请重新登录')
      }
      // 如果当前是登录页面，不重定向也不显示 toast（由登录逻辑自行处理错误）
    } else if (status === 400) {
      // 业务错误（如验证失败、参数错误等）- 不在拦截器中显示 toast
      // 由调用方自行处理错误消息，避免重复提示
      // 错误信息已包含在 error.response.data 中
    } else if (status === 403) {
      toast.error('权限不足')
    } else if (status === 404) {
      toast.error('请求的资源不存在')
    } else if (status === 500) {
      toast.error('服务器内部错误')
    } else if (!error.response && error.request) {
      // 真正的网络错误（没有收到响应）
      toast.error('网络连接失败，请检查网络设置')
    } else if (!error.response && !error.request) {
      // 请求配置错误
      const errorMessage = error.message || '请求配置错误'
      toast.error(errorMessage)
    }
    // 其他有响应的错误（如 502、503 等）不在这里处理，由调用方处理

    return Promise.reject(error)
  }
)

// 通用请求方法
export const api = {
  get: <T = any>(url: string, config?: AxiosRequestConfig) => 
    apiClient.get<T>(url, config).then(res => res.data),
  
  post: <T = any>(url: string, data?: any, config?: AxiosRequestConfig) => 
    apiClient.post<T>(url, data, config).then(res => res.data),
  
  put: <T = any>(url: string, data?: any, config?: AxiosRequestConfig) => 
    apiClient.put<T>(url, data, config).then(res => res.data),
  
  delete: <T = any>(url: string, config?: AxiosRequestConfig) => 
    apiClient.delete<T>(url, config).then(res => res.data),
  
  patch: <T = any>(url: string, data?: any, config?: AxiosRequestConfig) => 
    apiClient.patch<T>(url, data, config).then(res => res.data),
}

export default apiClient
