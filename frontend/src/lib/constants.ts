/**
 * 应用常量定义
 */

// 本地存储键名（保留用于兼容性，但主要使用 Zustand persist）
export const STORAGE_KEYS = {
  AUTH_TOKEN: 'auth_token',
  AUTH_EXPIRES: 'auth_expires',
  AUTH_STORAGE: 'fusionmail-auth', // Zustand persist 键名
} as const

// API 端点
// 注意：这些路径会自动添加到 baseURL (http://localhost:8080/api/v1) 后面
export const API_ENDPOINTS = {
  AUTH: {
    LOGIN: '/auth/login',
    LOGOUT: '/auth/logout',
    VERIFY: '/auth/verify',
    REFRESH: '/auth/refresh',
  },
} as const