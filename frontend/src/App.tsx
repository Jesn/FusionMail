import { Suspense } from 'react'
import { BrowserRouter, Routes, Route, Navigate } from 'react-router-dom'
import ProtectedRoute from '@/components/auth/ProtectedRoute'
import { Toaster } from '@/components/ui/sonner'
import { ErrorBoundary } from '@/components/error/ErrorBoundary'
import { ErrorPage } from '@/components/error/ErrorPage'
import LoginPage from '@/pages/LoginPage'
import DashboardPage from '@/pages/DashboardPage'

/**
 * 加载中组件
 */
function LoadingFallback() {
  return (
    <div className="flex min-h-screen items-center justify-center bg-gray-50">
      <div className="text-center">
        <div className="inline-block h-8 w-8 animate-spin rounded-full border-4 border-solid border-current border-r-transparent align-[-0.125em] motion-reduce:animate-[spin_1.5s_linear_infinite]" />
        <p className="mt-4 text-gray-600">加载中...</p>
      </div>
    </div>
  )
}

function App() {
  return (
    <ErrorBoundary
      fallback={<ErrorPage />}
      onError={(error, errorInfo) => {
        // 这里可以添加错误上报逻辑
        // 例如：发送到 Sentry、LogRocket 等错误监控服务
        console.error('应用错误:', error)
        console.error('错误信息:', errorInfo)
      }}
    >
      <BrowserRouter>
        <Suspense fallback={<LoadingFallback />}>
          <Routes>
            {/* 公开路由 - 登录页面 */}
            <Route path="/login" element={<LoginPage />} />

            {/* 受保护路由 - 需要登录 */}
            <Route
              path="/dashboard"
              element={
                <ProtectedRoute>
                  <DashboardPage />
                </ProtectedRoute>
              }
            />

            {/* 默认路由 */}
            <Route path="/" element={<Navigate to="/dashboard" replace />} />
            <Route path="*" element={<Navigate to="/dashboard" replace />} />
          </Routes>
        </Suspense>
        <Toaster />
      </BrowserRouter>
    </ErrorBoundary>
  )
}

export default App