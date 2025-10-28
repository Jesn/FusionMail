import { Navigate, useLocation } from 'react-router-dom'
import { authService } from '@/services/authService'

interface ProtectedRouteProps {
  children: React.ReactNode
}

/**
 * 路由守卫组件 - 保护需要认证的路由
 * 未登录用户将被重定向到登录页
 */
export default function ProtectedRoute({ children }: ProtectedRouteProps) {
  const location = useLocation()
  const isAuthenticated = authService.isAuthenticated()

  if (!isAuthenticated) {
    // 未登录，重定向到登录页并记录当前URL
    const returnUrl = encodeURIComponent(location.pathname + location.search)
    return <Navigate to={`/login?returnUrl=${returnUrl}`} replace />
  }

  // 已登录，渲染子组件
  return <>{children}</>
}