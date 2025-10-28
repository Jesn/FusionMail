import axios from 'axios'
import { toast } from 'sonner'
import { eventBus, AUTH_EVENTS } from './eventBus'
import { STORAGE_KEYS } from './constants'

// API 响应接口
interface ApiResponse<T = any> {
  success: boolean
  data?: T
  error?: {
    errorCode: string
    message: string
    details?: any
  }
  timestamp: string
}

class HttpClient {
  private instance

  constructor() {
    this.instance = axios.create({
      baseURL: import.meta.env.VITE_API_BASE_URL || '',
      timeout: 30000,
      headers: {
        'Content-Type': 'application/json',
      },
    })

    this.setupInterceptors()
  }

  private setupInterceptors() {
    // 请求拦截器 - 添加认证 token
    this.instance.interceptors.request.use(
      (config) => {
        const token = localStorage.getItem(STORAGE_KEYS.AUTH_TOKEN)
        if (token) {
          config.headers.Authorization = `Bearer ${token}`
        }
        return config
      },
      (error) => {
        return Promise.reject(error)
      }
    )

    // 响应拦截器 - 统一处理响应和错误
    this.instance.interceptors.response.use(
      (response) => {
        return response
      },
      (error) => {
        this.handleError(error)
        return Promise.reject(error)
      }
    )
  }

  private handleError(error: any) {
    if (error.response) {
      const { status, data } = error.response

      switch (status) {
        case 401:
          // 未授权 - 触发登出事件
          eventBus.emit(AUTH_EVENTS.UNAUTHORIZED)
          break
        case 403:
          toast.error('权限不足')
          break
        case 404:
          toast.error('请求的资源不存在')
          break
        case 500:
          toast.error('服务器内部错误')
          break
        default:
          // 显示服务器返回的错误消息
          if (data?.error?.message) {
            toast.error(data.error.message)
          } else {
            toast.error(`请求失败 (${status})`)
          }
      }
    } else if (error.request) {
      // 网络错误
      toast.error('网络连接失败，请检查网络设置')
    } else {
      // 其他错误
      toast.error('请求配置错误')
    }
  }

  // GET 请求
  async get<T = any>(url: string, params?: any): Promise<ApiResponse<T>> {
    const response = await this.instance.get(url, { params })
    return response.data
  }

  // POST 请求
  async post<T = any>(url: string, data?: any): Promise<ApiResponse<T>> {
    const response = await this.instance.post(url, data)
    return response.data
  }

  // PUT 请求
  async put<T = any>(url: string, data?: any): Promise<ApiResponse<T>> {
    const response = await this.instance.put(url, data)
    return response.data
  }

  // DELETE 请求
  async delete<T = any>(url: string): Promise<ApiResponse<T>> {
    const response = await this.instance.delete(url)
    return response.data
  }

  // PATCH 请求
  async patch<T = any>(url: string, data?: any): Promise<ApiResponse<T>> {
    const response = await this.instance.patch(url, data)
    return response.data
  }
}

export const httpClient = new HttpClient()