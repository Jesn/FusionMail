import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import { authService } from '@/services/authService'
import { useNavigate } from 'react-router-dom'
import { toast } from 'sonner'

export default function DashboardPage() {
  const navigate = useNavigate()

  const handleLogout = async () => {
    try {
      await authService.logout()
      toast.success('已退出登录')
      navigate('/login', { replace: true })
    } catch (error) {
      toast.error('退出登录失败')
      console.error('Logout error:', error)
    }
  }

  return (
    <div className="min-h-screen bg-gray-50">
      {/* 顶部导航栏 */}
      <header className="bg-white shadow-sm border-b">
        <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
          <div className="flex justify-between items-center h-16">
            <h1 className="text-xl font-semibold text-gray-900">
              FusionMail
            </h1>
            <Button variant="outline" onClick={handleLogout}>
              退出登录
            </Button>
          </div>
        </div>
      </header>

      {/* 主要内容区域 */}
      <main className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-8">
        <div className="mb-8">
          <h2 className="text-2xl font-bold text-gray-900 mb-2">
            欢迎使用 FusionMail
          </h2>
          <p className="text-gray-600">
            这是基于模板生成的项目仪表板页面。你可以在这里开始构建你的业务功能。
          </p>
        </div>

        {/* 功能卡片网格 */}
        <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6">
          <Card>
            <CardHeader>
              <CardTitle>开始开发</CardTitle>
              <CardDescription>
                添加你的业务功能和页面
              </CardDescription>
            </CardHeader>
            <CardContent>
              <p className="text-sm text-gray-600 mb-4">
                在 src/pages 目录下创建新的页面组件，在 src/components 目录下创建业务组件。
              </p>
              <Button variant="outline" className="w-full">
                查看文档
              </Button>
            </CardContent>
          </Card>

          <Card>
            <CardHeader>
              <CardTitle>API 集成</CardTitle>
              <CardDescription>
                连接后端 API 服务
              </CardDescription>
            </CardHeader>
            <CardContent>
              <p className="text-sm text-gray-600 mb-4">
                在 src/services 目录下创建 API 服务，使用已配置的 HTTP 客户端。
              </p>
              <Button variant="outline" className="w-full">
                API 文档
              </Button>
            </CardContent>
          </Card>

          <Card>
            <CardHeader>
              <CardTitle>UI 组件</CardTitle>
              <CardDescription>
                使用内置的 UI 组件库
              </CardDescription>
            </CardHeader>
            <CardContent>
              <p className="text-sm text-gray-600 mb-4">
                项目已集成 shadcn/ui 组件库，提供丰富的现代化 UI 组件。
              </p>
              <Button variant="outline" className="w-full">
                组件库
              </Button>
            </CardContent>
          </Card>
        </div>

        {/* 快速开始指南 */}
        <div className="mt-12">
          <Card>
            <CardHeader>
              <CardTitle>快速开始指南</CardTitle>
              <CardDescription>
                按照以下步骤开始开发你的应用
              </CardDescription>
            </CardHeader>
            <CardContent>
              <div className="space-y-4">
                <div className="flex items-start space-x-3">
                  <div className="flex-shrink-0 w-6 h-6 bg-blue-100 rounded-full flex items-center justify-center">
                    <span className="text-sm font-medium text-blue-600">1</span>
                  </div>
                  <div>
                    <h4 className="font-medium text-gray-900">定义数据模型</h4>
                    <p className="text-sm text-gray-600">
                      在后端 Core 层定义你的业务实体和接口
                    </p>
                  </div>
                </div>

                <div className="flex items-start space-x-3">
                  <div className="flex-shrink-0 w-6 h-6 bg-blue-100 rounded-full flex items-center justify-center">
                    <span className="text-sm font-medium text-blue-600">2</span>
                  </div>
                  <div>
                    <h4 className="font-medium text-gray-900">创建 API 端点</h4>
                    <p className="text-sm text-gray-600">
                      在 API 层创建控制器和服务来处理业务逻辑
                    </p>
                  </div>
                </div>

                <div className="flex items-start space-x-3">
                  <div className="flex-shrink-0 w-6 h-6 bg-blue-100 rounded-full flex items-center justify-center">
                    <span className="text-sm font-medium text-blue-600">3</span>
                  </div>
                  <div>
                    <h4 className="font-medium text-gray-900">构建前端界面</h4>
                    <p className="text-sm text-gray-600">
                      创建页面组件和业务组件，调用后端 API
                    </p>
                  </div>
                </div>

                <div className="flex items-start space-x-3">
                  <div className="flex-shrink-0 w-6 h-6 bg-blue-100 rounded-full flex items-center justify-center">
                    <span className="text-sm font-medium text-blue-600">4</span>
                  </div>
                  <div>
                    <h4 className="font-medium text-gray-900">测试和部署</h4>
                    <p className="text-sm text-gray-600">
                      使用 Docker 进行容器化部署
                    </p>
                  </div>
                </div>
              </div>
            </CardContent>
          </Card>
        </div>
      </main>
    </div>
  )
}