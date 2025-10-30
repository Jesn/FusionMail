import axios from 'axios'
import type { AxiosInstance, AxiosRequestConfig, AxiosError } from 'axios'
import { toast } from 'sonner'
import { useAuthStore } from '../stores/authStore'

// API 基础 URL
const API_BASE_URL = import.meta.env.VITE_API_BASE_URL || 'http://localhost:8080/api/v1'

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
})

// 请求拦截器 - 添加认证 token
apiClient.interceptors.request.use(
  (config) => {
    const token = useAuthStore.getState().token
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
    const data = error.response?.data as any

    // 处理不同的 HTTP 状态码
    if (status === 401) {
      // 未授权 - 清除认证数据并重定向
      clearAuthData()
      useAuthStore.getState().logout()
      window.location.href = '/login'
      toast.error('登录已过期，请重新登录')
    } else if (status === 403) {
      toast.error('权限不足')
    } else if (status === 404) {
      toast.error('请求的资源不存在')
    } else if (status === 500) {
      toast.error('服务器内部错误')
    } else if (error.request) {
      // 网络错误
      toast.error('网络连接失败，请检查网络设置')
    } else {
      // 其他错误 - 显示服务器返回的错误消息
      const errorMessage = data?.error || error.message || '请求失败'
      toast.error(errorMessage)
    }

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
