import { useState } from 'react'
import { useNavigate, useSearchParams } from 'react-router-dom'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { toast } from 'sonner'
import { authService } from '@/services/authService'

export default function LoginPage() {
  const [password, setPassword] = useState('')
  const [isLoading, setIsLoading] = useState(false)
  const navigate = useNavigate()
  const [searchParams] = useSearchParams()

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    
    if (!password.trim()) {
      toast.error('请输入密码')
      return
    }

    setIsLoading(true)
    
    try {
      await authService.login(password)
      
      // 登录成功，重定向到目标页面或首页
      const returnUrl = searchParams.get('returnUrl')
      const targetUrl = returnUrl ? decodeURIComponent(returnUrl) : '/inbox'
      
      toast.success('登录成功')
      navigate(targetUrl, { replace: true })
    } catch (error) {
      console.error('登录失败:', error)
      // 显示具体的错误消息，如果是401则显示"用户名或密码错误"
      const message = error instanceof Error ? error.message : '登录失败，请检查密码'
      toast.error(message)
    } finally {
      setIsLoading(false)
    }
  }

  return (
    <div className="min-h-screen flex items-center justify-center bg-background px-4">
      <Card className="w-full max-w-md">
        <CardHeader className="text-center">
          <CardTitle className="text-2xl font-bold">FusionMail</CardTitle>
          <CardDescription>
            请输入主密码以访问系统
          </CardDescription>
        </CardHeader>
        <CardContent>
          <form onSubmit={handleSubmit} className="space-y-4">
            <div className="space-y-2">
              <Label htmlFor="password">主密码</Label>
              <Input
                id="password"
                type="password"
                value={password}
                onChange={(e) => setPassword(e.target.value)}
                placeholder="请输入主密码"
                disabled={isLoading}
                autoFocus
              />
            </div>
            
            <Button 
              type="submit" 
              className="w-full" 
              disabled={isLoading}
            >
              {isLoading ? '登录中...' : '登录'}
            </Button>
          </form>
          
          <div className="mt-6 text-center text-sm text-gray-600">
            <p>首次启动时，主密码会自动生成并显示在后端日志中</p>
          </div>
        </CardContent>
      </Card>
    </div>
  )
}