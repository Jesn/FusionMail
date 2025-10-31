import { Suspense, useEffect } from 'react'
import { BrowserRouter, Routes, Route, Navigate } from 'react-router-dom'
import ProtectedRoute from '@/components/auth/ProtectedRoute'
import { Toaster } from '@/components/ui/sonner'
import { ErrorBoundary } from '@/components/error/ErrorBoundary'
import { ErrorPage } from '@/components/error/ErrorPage'
import { MainLayout } from '@/components/layout/MainLayout'
import LoginPage from '@/pages/LoginPage'
import { InboxPage } from '@/pages/InboxPage'
import { EmailDetailPage } from '@/pages/EmailDetailPage'
import { AccountsPage } from '@/pages/AccountsPage'
import { AccountTableTestPage } from '@/pages/AccountTableTestPage'
import { RulesPage } from '@/pages/RulesPage'
import { WebhooksPage } from '@/pages/WebhooksPage'
import { SearchPage } from '@/pages/SearchPage'
import { SettingsPage } from '@/pages/SettingsPage'
import { tokenRefreshService } from '@/services/tokenRefreshService'
import { useAuthStore } from '@/stores/authStore'

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
  const isAuthenticated = useAuthStore(state => state.isAuthenticated)

  // 启动/停止 token 自动刷新服务
  useEffect(() => {
    if (isAuthenticated) {
      tokenRefreshService.start()
    } else {
      tokenRefreshService.stop()
    }

    return () => {
      tokenRefreshService.stop()
    }
  }, [isAuthenticated])

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
              path="/inbox"
              element={
                <ProtectedRoute>
                  <MainLayout>
                    <InboxPage />
                  </MainLayout>
                </ProtectedRoute>
              }
            />
            <Route
              path="/email/:id"
              element={
                <ProtectedRoute>
                  <MainLayout>
                    <EmailDetailPage />
                  </MainLayout>
                </ProtectedRoute>
              }
            />
            <Route
              path="/accounts"
              element={
                <ProtectedRoute>
                  <MainLayout>
                    <AccountsPage />
                  </MainLayout>
                </ProtectedRoute>
              }
            />
            <Route
              path="/accounts/test"
              element={
                <ProtectedRoute>
                  <MainLayout>
                    <AccountTableTestPage />
                  </MainLayout>
                </ProtectedRoute>
              }
            />
            <Route
              path="/rules"
              element={
                <ProtectedRoute>
                  <MainLayout>
                    <RulesPage />
                  </MainLayout>
                </ProtectedRoute>
              }
            />
            <Route
              path="/webhooks"
              element={
                <ProtectedRoute>
                  <MainLayout>
                    <WebhooksPage />
                  </MainLayout>
                </ProtectedRoute>
              }
            />
            <Route
              path="/search"
              element={
                <ProtectedRoute>
                  <MainLayout>
                    <SearchPage />
                  </MainLayout>
                </ProtectedRoute>
              }
            />
            <Route
              path="/settings"
              element={
                <ProtectedRoute>
                  <MainLayout>
                    <SettingsPage />
                  </MainLayout>
                </ProtectedRoute>
              }
            />

            {/* 默认路由 */}
            <Route path="/" element={<Navigate to="/inbox" replace />} />
            <Route path="*" element={<Navigate to="/inbox" replace />} />
          </Routes>
        </Suspense>
        <Toaster />
      </BrowserRouter>
    </ErrorBoundary>
  )
}

export default App