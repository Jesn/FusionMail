import { useState } from 'react'
import { useNavigate, useSearchParams } from 'react-router-dom'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { toast } from 'sonner'
import { authService } from '@/services/authService'
import { twoFactorService } from '@/services/twoFactorService'
import { Shield, ArrowLeft } from 'lucide-react'

export default function LoginPage() {
  const [username, setUsername] = useState('')
  const [password, setPassword] = useState('')
  const [isLoading, setIsLoading] = useState(false)
  const navigate = useNavigate()
  const [searchParams] = useSearchParams()

  // 2FA 状态
  const [requires2FA, setRequires2FA] = useState(false)
  const [twoFactorUserId, setTwoFactorUserId] = useState<number | null>(null)
  const [twoFactorChallengeToken, setTwoFactorChallengeToken] = useState('')
  const [twoFactorCode, setTwoFactorCode] = useState('')

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()

    if (!username.trim()) {
      toast.error('请输入用户名')
      return
    }

    if (!password.trim()) {
      toast.error('请输入密码')
      return
    }

    setIsLoading(true)

    try {
      const result = await authService.login(username, password)

      if (result.requires2FA && result.userId && result.challengeToken) {
        // 需要 2FA 验证
        setRequires2FA(true)
        setTwoFactorUserId(result.userId)
        setTwoFactorChallengeToken(result.challengeToken)
        toast.info('请输入双因素认证验证码')
      } else {
        // 登录成功，重定向到目标页面或首页
        const returnUrl = searchParams.get('returnUrl')
        const targetUrl = returnUrl ? decodeURIComponent(returnUrl) : '/inbox'

        toast.success('登录成功')
        navigate(targetUrl, { replace: true })
      }
    } catch (error) {
      console.error('登录失败:', error)
      const message = error instanceof Error ? error.message : '登录失败，请检查用户名和密码'
      toast.error(message)
    } finally {
      setIsLoading(false)
    }
  }

  const handle2FASubmit = async (e: React.FormEvent) => {
    e.preventDefault()

    if (!twoFactorCode || twoFactorCode.length < 6) {
      toast.error('请输入 6 位验证码或恢复码')
      return
    }

    if (!twoFactorUserId || !twoFactorChallengeToken) {
      toast.error('验证状态异常，请重新登录')
      return
    }

    setIsLoading(true)

    try {
      // 验证 2FA 码并建立 Cookie 会话
      const loginResponse = await twoFactorService.validateLogin(
        twoFactorUserId,
        twoFactorCode,
        twoFactorChallengeToken
      )

      // 2FA 验证通过，使用返回的会话元数据完成登录
      await authService.complete2FALogin(
        loginResponse.expiresAt,
        loginResponse.user
      )

      const returnUrl = searchParams.get('returnUrl')
      const targetUrl = returnUrl ? decodeURIComponent(returnUrl) : '/inbox'

      toast.success('登录成功')
      navigate(targetUrl, { replace: true })
    } catch (error) {
      console.error('2FA 验证失败:', error)
      const message = error instanceof Error ? error.message : '验证码错误'
      toast.error(message)
    } finally {
      setIsLoading(false)
    }
  }

  const handleBack = () => {
    setRequires2FA(false)
    setTwoFactorUserId(null)
    setTwoFactorChallengeToken('')
    setTwoFactorCode('')
  }

  // 2FA 验证界面
  if (requires2FA) {
    return (
      <div className="min-h-screen flex items-center justify-center bg-background px-4">
        <Card className="w-full max-w-md">
          <CardHeader className="text-center">
            <div className="flex justify-center mb-4">
              <div className="p-3 bg-blue-100 rounded-full">
                <Shield className="h-8 w-8 text-blue-600" />
              </div>
            </div>
            <CardTitle className="text-2xl font-bold">双因素认证</CardTitle>
            <CardDescription>
              请输入身份验证器应用中的 6 位验证码，或使用恢复码
            </CardDescription>
          </CardHeader>
          <CardContent>
            <form onSubmit={handle2FASubmit} className="space-y-4">
              <div className="space-y-2">
                <Label htmlFor="twoFactorCode">验证码</Label>
                <Input
                  id="twoFactorCode"
                  type="text"
                  value={twoFactorCode}
                  onChange={(e) => setTwoFactorCode(e.target.value.replace(/\D/g, '').slice(0, 8))}
                  placeholder="000000"
                  disabled={isLoading}
                  autoFocus
                  className="text-center text-2xl tracking-widest"
                  maxLength={8}
                />
                <p className="text-xs text-muted-foreground text-center">
                  输入 6 位验证码或 8 位恢复码
                </p>
              </div>

              <Button
                type="submit"
                className="w-full"
                disabled={isLoading || twoFactorCode.length < 6}
              >
                {isLoading ? '验证中...' : '验证'}
              </Button>

              <Button
                type="button"
                variant="ghost"
                className="w-full"
                onClick={handleBack}
                disabled={isLoading}
              >
                <ArrowLeft className="h-4 w-4 mr-2" />
                返回登录
              </Button>
            </form>
          </CardContent>
        </Card>
      </div>
    )
  }

  // 普通登录界面
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
              <Label htmlFor="username">用户名</Label>
              <Input
                id="username"
                type="text"
                value={username}
                onChange={(e) => setUsername(e.target.value)}
                placeholder="请输入用户名"
                disabled={isLoading}
                autoFocus
              />
            </div>

            <div className="space-y-2">
              <Label htmlFor="password">密码</Label>
              <Input
                id="password"
                type="password"
                value={password}
                onChange={(e) => setPassword(e.target.value)}
                placeholder="请输入密码"
                disabled={isLoading}
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

        </CardContent>
      </Card>
    </div>
  )
}
