/**
 * 双因素认证 (2FA) 设置组件
 */

import { useState, useEffect } from 'react'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Alert, AlertDescription } from '@/components/ui/alert'
import { Badge } from '@/components/ui/badge'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'
import { Shield, ShieldCheck, ShieldOff, Copy, RefreshCw, Loader2, AlertTriangle, CheckCircle } from 'lucide-react'
import { twoFactorService, type TwoFactorStatus, type TwoFactorSetupResponse } from '@/services/twoFactorService'
import { toast } from 'sonner'
import QRCode from 'qrcode'

export function TwoFactorSettings() {
  const [status, setStatus] = useState<TwoFactorStatus | null>(null)
  const [isLoading, setIsLoading] = useState(true)
  const [setupData, setSetupData] = useState<TwoFactorSetupResponse | null>(null)
  const [qrCodeDataUrl, setQrCodeDataUrl] = useState<string>('')
  
  // 对话框状态
  const [showSetupDialog, setShowSetupDialog] = useState(false)
  const [showDisableDialog, setShowDisableDialog] = useState(false)
  const [showBackupCodesDialog, setShowBackupCodesDialog] = useState(false)
  const [showRegenerateDialog, setShowRegenerateDialog] = useState(false)
  
  // 表单状态
  const [verifyCode, setVerifyCode] = useState('')
  const [disablePassword, setDisablePassword] = useState('')
  const [disableCode, setDisableCode] = useState('')
  const [regenerateCode, setRegenerateCode] = useState('')
  const [newBackupCodes, setNewBackupCodes] = useState<string[]>([])
  
  const [isSubmitting, setIsSubmitting] = useState(false)

  // 加载 2FA 状态
  useEffect(() => {
    loadStatus()
  }, [])

  const loadStatus = async () => {
    try {
      setIsLoading(true)
      const data = await twoFactorService.getStatus()
      setStatus(data)
    } catch (error) {
      console.error('加载 2FA 状态失败:', error)
      toast.error('加载 2FA 状态失败')
    } finally {
      setIsLoading(false)
    }
  }

  // 开始设置 2FA
  const handleStartSetup = async () => {
    try {
      setIsSubmitting(true)
      const data = await twoFactorService.setup()
      setSetupData(data)
      
      // 生成二维码
      const qrDataUrl = await QRCode.toDataURL(data.qr_code_url, {
        width: 200,
        margin: 2,
      })
      setQrCodeDataUrl(qrDataUrl)
      
      setShowSetupDialog(true)
    } catch (error: any) {
      toast.error(error.message || '设置 2FA 失败')
    } finally {
      setIsSubmitting(false)
    }
  }

  // 验证并启用 2FA
  const handleVerifyAndEnable = async () => {
    if (!verifyCode || verifyCode.length !== 6) {
      toast.error('请输入 6 位验证码')
      return
    }

    try {
      setIsSubmitting(true)
      await twoFactorService.verify(verifyCode)
      toast.success('双因素认证已启用')
      setShowSetupDialog(false)
      setShowBackupCodesDialog(true)
      setVerifyCode('')
      await loadStatus()
    } catch (error: any) {
      toast.error(error.message || '验证失败')
    } finally {
      setIsSubmitting(false)
    }
  }

  // 禁用 2FA
  const handleDisable = async () => {
    if (!disablePassword) {
      toast.error('请输入密码')
      return
    }
    
    if (!disableCode || disableCode.length !== 6) {
      toast.error('请输入 6 位验证码或恢复码')
      return
    }

    try {
      setIsSubmitting(true)
      await twoFactorService.disable(disablePassword, disableCode)
      toast.success('双因素认证已禁用')
      setShowDisableDialog(false)
      setDisablePassword('')
      setDisableCode('')
      await loadStatus()
    } catch (error: any) {
      toast.error(error.message || '禁用失败')
    } finally {
      setIsSubmitting(false)
    }
  }

  // 重新生成恢复码
  const handleRegenerateBackupCodes = async () => {
    if (!regenerateCode || regenerateCode.length !== 6) {
      toast.error('请输入 6 位验证码')
      return
    }

    try {
      setIsSubmitting(true)
      const codes = await twoFactorService.regenerateBackupCodes(regenerateCode)
      setNewBackupCodes(codes)
      setShowRegenerateDialog(false)
      setShowBackupCodesDialog(true)
      setRegenerateCode('')
      await loadStatus()
      toast.success('恢复码已重新生成')
    } catch (error: any) {
      toast.error(error.message || '重新生成失败')
    } finally {
      setIsSubmitting(false)
    }
  }

  // 复制到剪贴板
  const copyToClipboard = (text: string) => {
    navigator.clipboard.writeText(text)
    toast.success('已复制到剪贴板')
  }

  // 复制所有恢复码
  const copyAllBackupCodes = () => {
    const codes = (setupData?.backup_codes || newBackupCodes).join('\n')
    navigator.clipboard.writeText(codes)
    toast.success('所有恢复码已复制到剪贴板')
  }

  if (isLoading) {
    return (
      <Card>
        <CardContent className="flex items-center justify-center py-12">
          <Loader2 className="h-8 w-8 animate-spin text-muted-foreground" />
        </CardContent>
      </Card>
    )
  }

  return (
    <>
      <Card>
        <CardHeader>
          <div className="flex items-center justify-between">
            <div className="flex items-center gap-3">
              <Shield className="h-6 w-6 text-blue-500" />
              <div>
                <CardTitle>双因素认证 (2FA)</CardTitle>
                <CardDescription>
                  使用身份验证器应用增强账户安全性
                </CardDescription>
              </div>
            </div>
            {status?.enabled ? (
              <Badge variant="default" className="bg-green-500">
                <ShieldCheck className="h-3 w-3 mr-1" />
                已启用
              </Badge>
            ) : (
              <Badge variant="secondary">
                <ShieldOff className="h-3 w-3 mr-1" />
                未启用
              </Badge>
            )}
          </div>
        </CardHeader>
        <CardContent className="space-y-4">
          {status?.enabled ? (
            <>
              <Alert>
                <CheckCircle className="h-4 w-4" />
                <AlertDescription>
                  双因素认证已启用，您的账户受到额外保护。
                  {status.enabled_at && (
                    <span className="block text-xs text-muted-foreground mt-1">
                      启用时间: {new Date(status.enabled_at).toLocaleString()}
                    </span>
                  )}
                </AlertDescription>
              </Alert>
              
              <div className="flex items-center justify-between p-3 bg-muted rounded-lg">
                <div>
                  <p className="text-sm font-medium">恢复码</p>
                  <p className="text-xs text-muted-foreground">
                    剩余 {status.backup_codes_count} 个恢复码
                  </p>
                </div>
                <Button
                  variant="outline"
                  size="sm"
                  onClick={() => setShowRegenerateDialog(true)}
                >
                  <RefreshCw className="h-4 w-4 mr-2" />
                  重新生成
                </Button>
              </div>

              <div className="flex gap-2">
                <Button
                  variant="destructive"
                  onClick={() => setShowDisableDialog(true)}
                >
                  <ShieldOff className="h-4 w-4 mr-2" />
                  禁用 2FA
                </Button>
              </div>
            </>
          ) : (
            <>
              <Alert>
                <AlertTriangle className="h-4 w-4" />
                <AlertDescription>
                  启用双因素认证可以大大提高账户安全性。即使密码泄露，攻击者也无法登录您的账户。
                </AlertDescription>
              </Alert>
              
              <Button onClick={handleStartSetup} disabled={isSubmitting}>
                {isSubmitting ? (
                  <Loader2 className="h-4 w-4 mr-2 animate-spin" />
                ) : (
                  <ShieldCheck className="h-4 w-4 mr-2" />
                )}
                启用双因素认证
              </Button>
            </>
          )}
        </CardContent>
      </Card>

      {/* 设置 2FA 对话框 */}
      <Dialog open={showSetupDialog} onOpenChange={setShowSetupDialog}>
        <DialogContent className="max-w-md">
          <DialogHeader>
            <DialogTitle>设置双因素认证</DialogTitle>
            <DialogDescription>
              使用身份验证器应用（如 Google Authenticator、Microsoft Authenticator）扫描二维码
            </DialogDescription>
          </DialogHeader>
          
          <div className="space-y-4">
            {/* 二维码 */}
            <div className="flex justify-center">
              {qrCodeDataUrl && (
                <img src={qrCodeDataUrl} alt="2FA QR Code" className="border rounded-lg" />
              )}
            </div>
            
            {/* 手动输入密钥 */}
            <div className="space-y-2">
              <Label className="text-xs text-muted-foreground">无法扫描？手动输入密钥：</Label>
              <div className="flex items-center gap-2">
                <code className="flex-1 p-2 bg-muted rounded text-xs font-mono break-all">
                  {setupData?.secret}
                </code>
                <Button
                  variant="ghost"
                  size="icon"
                  className="cursor-pointer"
                  onClick={() => copyToClipboard(setupData?.secret || '')}
                >
                  <Copy className="h-4 w-4 cursor-pointer" />
                </Button>
              </div>
            </div>
            
            {/* 验证码输入 */}
            <div className="space-y-2">
              <Label>输入验证码</Label>
              <Input
                type="text"
                placeholder="000000"
                maxLength={6}
                value={verifyCode}
                onChange={(e) => setVerifyCode(e.target.value.replace(/\D/g, ''))}
                className="text-center text-2xl tracking-widest"
              />
            </div>
          </div>
          
          <DialogFooter>
            <Button variant="outline" onClick={() => setShowSetupDialog(false)}>
              取消
            </Button>
            <Button onClick={handleVerifyAndEnable} disabled={isSubmitting || verifyCode.length !== 6}>
              {isSubmitting && <Loader2 className="h-4 w-4 mr-2 animate-spin" />}
              验证并启用
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      {/* 恢复码对话框 */}
      <Dialog open={showBackupCodesDialog} onOpenChange={setShowBackupCodesDialog}>
        <DialogContent className="max-w-md">
          <DialogHeader>
            <DialogTitle>保存恢复码</DialogTitle>
            <DialogDescription>
              请妥善保存这些恢复码。如果您无法访问身份验证器应用，可以使用恢复码登录。
            </DialogDescription>
          </DialogHeader>
          
          <Alert variant="destructive">
            <AlertTriangle className="h-4 w-4" />
            <AlertDescription>
              每个恢复码只能使用一次。请将它们保存在安全的地方。
            </AlertDescription>
          </Alert>
          
          <div className="grid grid-cols-2 gap-2 p-4 bg-muted rounded-lg">
            {(setupData?.backup_codes || newBackupCodes).map((code, index) => (
              <code key={index} className="text-sm font-mono p-1 bg-background rounded text-center">
                {code}
              </code>
            ))}
          </div>
          
          <DialogFooter>
            <Button variant="outline" onClick={copyAllBackupCodes}>
              <Copy className="h-4 w-4 mr-2" />
              复制全部
            </Button>
            <Button onClick={() => {
              setShowBackupCodesDialog(false)
              setSetupData(null)
              setNewBackupCodes([])
            }}>
              我已保存
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      {/* 禁用 2FA 对话框 */}
      <Dialog open={showDisableDialog} onOpenChange={setShowDisableDialog}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>禁用双因素认证</DialogTitle>
            <DialogDescription>
              禁用后，您的账户将只使用密码保护。请确认您要禁用双因素认证。
            </DialogDescription>
          </DialogHeader>
          
          <Alert variant="destructive">
            <AlertTriangle className="h-4 w-4" />
            <AlertDescription>
              为了安全起见，禁用双因素认证需要同时验证密码和 2FA 验证码。
            </AlertDescription>
          </Alert>
          
          <div className="space-y-4">
            <div className="space-y-2">
              <Label>账户密码 *</Label>
              <Input
                type="password"
                placeholder="输入您的密码"
                value={disablePassword}
                onChange={(e) => setDisablePassword(e.target.value)}
              />
            </div>
            
            <div className="space-y-2">
              <Label>验证码 *</Label>
              <Input
                type="text"
                placeholder="输入 6 位验证码或恢复码"
                maxLength={6}
                value={disableCode}
                onChange={(e) => setDisableCode(e.target.value.replace(/\D/g, ''))}
                className="text-center text-xl tracking-widest"
              />
              <p className="text-xs text-muted-foreground">
                请输入身份验证器中的 6 位验证码，或使用恢复码
              </p>
            </div>
          </div>
          
          <DialogFooter>
            <Button variant="outline" onClick={() => setShowDisableDialog(false)}>
              取消
            </Button>
            <Button 
              variant="destructive" 
              onClick={handleDisable} 
              disabled={isSubmitting || !disablePassword || disableCode.length !== 6}
            >
              {isSubmitting && <Loader2 className="h-4 w-4 mr-2 animate-spin" />}
              禁用 2FA
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      {/* 重新生成恢复码对话框 */}
      <Dialog open={showRegenerateDialog} onOpenChange={setShowRegenerateDialog}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>重新生成恢复码</DialogTitle>
            <DialogDescription>
              重新生成后，旧的恢复码将失效。请输入验证码确认。
            </DialogDescription>
          </DialogHeader>
          
          <div className="space-y-2">
            <Label>验证码</Label>
            <Input
              type="text"
              placeholder="输入 6 位验证码"
              maxLength={6}
              value={regenerateCode}
              onChange={(e) => setRegenerateCode(e.target.value.replace(/\D/g, ''))}
              className="text-center text-xl tracking-widest"
            />
          </div>
          
          <DialogFooter>
            <Button variant="outline" onClick={() => setShowRegenerateDialog(false)}>
              取消
            </Button>
            <Button onClick={handleRegenerateBackupCodes} disabled={isSubmitting || regenerateCode.length !== 6}>
              {isSubmitting && <Loader2 className="h-4 w-4 mr-2 animate-spin" />}
              重新生成
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </>
  )
}

export default TwoFactorSettings
