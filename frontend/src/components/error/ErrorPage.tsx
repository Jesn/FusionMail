import { AlertTriangle, Home, RefreshCw } from 'lucide-react'
import { Button } from '@/components/ui/button'
import { Card, CardContent } from '@/components/ui/card'

interface ErrorPageProps {
  error?: Error
  errorInfo?: React.ErrorInfo
  onReset?: () => void
  onGoHome?: () => void
}

/**
 * 错误页面组件
 */
export function ErrorPage({
  error,
  errorInfo,
  onReset,
  onGoHome,
}: ErrorPageProps) {
  const handleReset = () => {
    if (onReset) {
      onReset()
    } else {
      window.location.reload()
    }
  }

  const handleGoHome = () => {
    if (onGoHome) {
      onGoHome()
    } else {
      window.location.href = '/'
    }
  }

  return (
    <div className="min-h-screen flex items-center justify-center bg-gray-50 px-4 py-12">
      <Card className="max-w-lg w-full">
        <CardContent className="pt-6">
          <div className="flex justify-center mb-6">
            <div className="w-16 h-16 bg-red-100 rounded-full flex items-center justify-center">
              <AlertTriangle className="w-8 h-8 text-red-600" />
            </div>
          </div>

          <h1 className="text-2xl font-bold text-center text-gray-900 mb-2">
            哎呀，出错了
          </h1>

          <p className="text-center text-gray-600 mb-6">
            应用程序遇到了一个意外错误。请尝试刷新页面，如果问题持续存在，请联系技术支持。
          </p>

          {import.meta.env.DEV && error && (
            <div className="mb-6">
              <details className="bg-gray-100 rounded-lg p-4">
                <summary className="cursor-pointer font-semibold text-sm text-gray-700 mb-2">
                  错误详情（仅开发环境可见）
                </summary>
                <div className="mt-3 space-y-2">
                  <div>
                    <p className="text-xs font-semibold text-red-600 mb-1">
                      错误信息：
                    </p>
                    <p className="text-xs text-gray-700 bg-white p-2 rounded">
                      {error.toString()}
                    </p>
                  </div>
                  {errorInfo && (
                    <div>
                      <p className="text-xs font-semibold text-red-600 mb-1">
                        组件堆栈：
                      </p>
                      <pre className="text-xs text-gray-700 bg-white p-2 rounded overflow-auto max-h-40">
                        {errorInfo.componentStack}
                      </pre>
                    </div>
                  )}
                </div>
              </details>
            </div>
          )}

          <div className="flex gap-3">
            <Button
              variant="outline"
              className="flex-1"
              onClick={handleReset}
            >
              <RefreshCw className="w-4 h-4 mr-2" />
              重试
            </Button>
            <Button
              className="flex-1"
              onClick={handleGoHome}
            >
              <Home className="w-4 h-4 mr-2" />
              返回首页
            </Button>
          </div>

          <div className="mt-6 pt-6 border-t border-gray-200">
            <p className="text-xs text-center text-gray-500">
              如果问题持续存在，请尝试：
            </p>
            <ul className="mt-2 text-xs text-gray-500 space-y-1">
              <li>• 清除浏览器缓存</li>
              <li>• 使用无痕模式打开</li>
              <li>• 检查网络连接</li>
              <li>• 联系技术支持</li>
            </ul>
          </div>
        </CardContent>
      </Card>
    </div>
  )
}