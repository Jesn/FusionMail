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
import { TrashPage } from '@/pages/TrashPage'
import { RulesPage } from '@/pages/RulesPage'
import { WebhooksPage } from '@/pages/WebhooksPage'
import { SearchPage } from '@/pages/SearchPage'
import { EmailListPage } from '@/pages/EmailListPage'
import { SpamPage } from '@/pages/SpamPage'
import { SpamRulesPage } from '@/pages/SpamRulesPage'
import { SpamSettingsPage } from '@/pages/SpamSettingsPage'
import { SettingsPage } from '@/pages/SettingsPage'
import { SystemSettingsPage } from '@/pages/SystemSettingsPage'
import { APIKeysPage } from '@/pages/APIKeysPage'
import { APIDocPage } from '@/pages/APIDocPage'
import { OAuth2ClientsPage } from '@/pages/OAuth2ClientsPage'
import { ProvidersPage } from '@/pages/ProvidersPage'
import { OAuth2CallbackPage } from '@/pages/OAuth2CallbackPage'
import { OAuth2TestPage } from '@/pages/OAuth2TestPage'
import { SSEDebugPage } from '@/pages/SSEDebugPage'
// 新增设置相关页面
import UserSettings from '@/pages/UserSettings'
import AdminSettings from '@/pages/AdminSettings'
import PublicSettings from '@/pages/PublicSettings'
import SettingsDashboard from '@/pages/SettingsDashboard'
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
            
            {/* OAuth2 回调路由 - 无需登录 */}
            <Route path="/auth/google/callback" element={<OAuth2CallbackPage />} />
            <Route path="/auth/microsoft/callback" element={<OAuth2CallbackPage />} />
            
            {/* OAuth2 测试页面 - 无需登录 */}
            <Route path="/oauth2-test" element={<OAuth2TestPage />} />

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
              path="/trash"
              element={
                <ProtectedRoute>
                  <MainLayout>
                    <TrashPage />
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
              path="/email-list"
              element={
                <ProtectedRoute>
                  <MainLayout>
                    <EmailListPage />
                  </MainLayout>
                </ProtectedRoute>
              }
            />
            <Route
              path="/spam"
              element={
                <ProtectedRoute>
                  <MainLayout>
                    <SpamPage />
                  </MainLayout>
                </ProtectedRoute>
              }
            />
            <Route
              path="/spam/rules"
              element={
                <ProtectedRoute>
                  <MainLayout>
                    <SpamRulesPage />
                  </MainLayout>
                </ProtectedRoute>
              }
            />
            <Route
              path="/spam/settings"
              element={
                <ProtectedRoute>
                  <MainLayout>
                    <SpamSettingsPage />
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
                    <UserSettings />
                  </MainLayout>
                </ProtectedRoute>
              }
            />
            <Route
              path="/settings/dashboard"
              element={
                <ProtectedRoute>
                  <MainLayout>
                    <SettingsDashboard />
                  </MainLayout>
                </ProtectedRoute>
              }
            />
            <Route
              path="/admin/settings"
              element={
                <ProtectedRoute>
                  <MainLayout>
                    <AdminSettings />
                  </MainLayout>
                </ProtectedRoute>
              }
            />
            <Route
              path="/public-settings"
              element={
                <ProtectedRoute>
                  <MainLayout>
                    <PublicSettings />
                  </MainLayout>
                </ProtectedRoute>
              }
            />
            {/* 保留旧的设置页面路由 */}
            <Route
              path="/settings/legacy"
              element={
                <ProtectedRoute>
                  <MainLayout>
                    <SettingsPage />
                  </MainLayout>
                </ProtectedRoute>
              }
            />
            <Route
              path="/settings/system"
              element={
                <ProtectedRoute>
                  <MainLayout>
                    <SystemSettingsPage />
                  </MainLayout>
                </ProtectedRoute>
              }
            />
            <Route
              path="/debug/sse"
              element={
                <ProtectedRoute>
                  <MainLayout>
                    <SSEDebugPage />
                  </MainLayout>
                </ProtectedRoute>
              }
            />
            <Route
              path="/api-keys"
              element={
                <ProtectedRoute>
                  <MainLayout>
                    <APIKeysPage />
                  </MainLayout>
                </ProtectedRoute>
              }
            />
            <Route
              path="/oauth2-clients"
              element={
                <ProtectedRoute>
                  <MainLayout>
                    <OAuth2ClientsPage />
                  </MainLayout>
                </ProtectedRoute>
              }
            />
            <Route
              path="/providers"
              element={
                <ProtectedRoute>
                  <MainLayout>
                    <ProvidersPage />
                  </MainLayout>
                </ProtectedRoute>
              }
            />
            <Route
              path="/api-docs"
              element={
                <ProtectedRoute>
                  <MainLayout>
                    <APIDocPage />
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