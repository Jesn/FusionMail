/**
 * SMTP 配置对话框组件
 * 用于配置账户的 SMTP 发送设置
 * 注意：SMTP 服务器配置（host/port/encryption）从 Provider 继承
 * Account 级别只需配置 username、password 和 enabled
 */

import { useState, useEffect } from 'react';
import { Button } from '../ui/button';
import { Input } from '../ui/input';
import { Label } from '../ui/label';
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '../ui/dialog';
import { Switch } from '../ui/switch';
import { Alert, AlertDescription } from '../ui/alert';
import { Loader2, CheckCircle2, XCircle, AlertCircle, RefreshCw, Server } from 'lucide-react';
import { smtpService } from '../../services/smtpService';
import type { UpdateSMTPConfigRequest } from '../../types';
import toast from 'react-hot-toast';

interface SMTPConfigDialogProps {
  open: boolean;
  onClose: () => void;
  accountUid: string;
  accountEmail: string;
  accountProvider?: string;
}

export const SMTPConfigDialog = ({
  open,
  onClose,
  accountUid,
  accountEmail,
  accountProvider: _accountProvider,
}: SMTPConfigDialogProps) => {
  void _accountProvider;

  // 表单状态 - 只保留用户需要配置的字段
  const [formData, setFormData] = useState<UpdateSMTPConfigRequest>({
    smtp_username: '',
    smtp_password: '',
    smtp_enabled: false,
  });

  // 从 Provider 继承的服务器配置（只读展示）
  const [serverConfig, setServerConfig] = useState<{
    host: string;
    port: number;
    encryption: string;
    fromProvider: boolean;
    providerName: string;
  }>({
    host: '',
    port: 0,
    encryption: '',
    fromProvider: false,
    providerName: '',
  });

  // 加载状态
  const [isLoading, setIsLoading] = useState(false);
  const [isSaving, setIsSaving] = useState(false);
  const [isTesting, setIsTesting] = useState(false);

  // 测试结果
  const [testResult, setTestResult] = useState<{
    success: boolean;
    message: string;
  } | null>(null);

  // 加载当前配置
  useEffect(() => {
    if (open && accountUid) {
      loadConfig();
    }
  }, [open, accountUid]);

  // 加载 SMTP 配置
  const loadConfig = async () => {
    setIsLoading(true);
    try {
      const config = await smtpService.getConfig(accountUid);
      // 设置用户可编辑的字段
      setFormData({
        smtp_username: config.smtp_username || accountEmail,
        smtp_password: '', // 密码不回显
        smtp_enabled: config.smtp_enabled || false,
      });
      // 设置从 Provider 继承的服务器配置（只读）
      setServerConfig({
        host: config.smtp_host || '',
        port: config.smtp_port || 0,
        encryption: config.smtp_encryption || '',
        fromProvider: config.from_provider || false,
        providerName: config.provider_name || '',
      });
    } catch (error) {
      console.error('加载 SMTP 配置失败:', error);
      // 如果加载失败，使用默认值
      setFormData({
        smtp_username: accountEmail,
        smtp_password: '',
        smtp_enabled: false,
      });
      setServerConfig({
        host: '',
        port: 0,
        encryption: '',
        fromProvider: false,
        providerName: '',
      });
    } finally {
      setIsLoading(false);
    }
  };

  // 保存配置
  const handleSave = async () => {
    // 检查服务器配置是否存在
    if (!serverConfig.host) {
      toast.error('SMTP 服务器未配置，请先在提供商管理中配置 SMTP 服务器');
      return;
    }

    setIsSaving(true);
    try {
      await smtpService.updateConfig(accountUid, formData);
      toast.success('SMTP 配置已保存');
      onClose();
    } catch (error) {
      console.error('保存 SMTP 配置失败:', error);
      toast.error('保存失败，请重试');
    } finally {
      setIsSaving(false);
    }
  };

  // 测试连接
  const handleTest = async () => {
    if (!serverConfig.host) {
      toast.error('SMTP 服务器未配置');
      return;
    }

    setIsTesting(true);
    setTestResult(null);
    try {
      // 传递当前表单中的临时凭证进行测试
      const result = await smtpService.testConnection(accountUid, {
        username: formData.smtp_username,
        password: formData.smtp_password,
      });
      setTestResult({
        success: result.success,
        message: result.message || (result.success ? '连接成功' : '连接失败'),
      });
      if (result.success) {
        toast.success('SMTP 连接测试成功');
      } else {
        toast.error(result.message || '连接测试失败');
      }
    } catch (error: any) {
      const errorMessage = error?.response?.data?.message || error?.message || '连接测试失败';
      setTestResult({
        success: false,
        message: errorMessage,
      });
      toast.error(errorMessage);
    } finally {
      setIsTesting(false);
    }
  };

  // 重置表单
  const handleReset = () => {
    setTestResult(null);
    loadConfig();
  };

  // 获取加密方式显示文本
  const getEncryptionLabel = (encryption: string) => {
    switch (encryption) {
      case 'tls':
      case 'ssl':
        return 'SSL/TLS';
      case 'starttls':
        return 'STARTTLS';
      case 'none':
        return '无加密';
      default:
        return encryption || '未配置';
    }
  };

  return (
    <Dialog open={open} onOpenChange={onClose}>
      <DialogContent className="w-full max-w-[500px] max-h-[90vh] overflow-hidden flex flex-col">
        <DialogHeader className="flex-shrink-0">
          <DialogTitle>SMTP 发送配置</DialogTitle>
          <DialogDescription>
            配置 {accountEmail} 的 SMTP 发送设置
          </DialogDescription>
        </DialogHeader>

        {isLoading ? (
          <div className="flex items-center justify-center py-8">
            <Loader2 className="h-8 w-8 animate-spin text-muted-foreground" />
          </div>
        ) : (
          <div className="flex-1 overflow-y-auto py-4 space-y-4">
            {/* 启用开关 */}
            <div className="flex items-center justify-between p-4 border rounded-lg">
              <div className="space-y-1">
                <Label htmlFor="smtp_enabled" className="font-medium">
                  启用 SMTP 发送
                </Label>
                <p className="text-xs text-muted-foreground">
                  启用后可通过此账户发送邮件
                </p>
              </div>
              <Switch
                id="smtp_enabled"
                checked={formData.smtp_enabled}
                onCheckedChange={(checked) =>
                  setFormData({ ...formData, smtp_enabled: checked })
                }
              />
            </div>

            {/* 服务器配置（只读，来自 Provider） */}
            <div className="p-4 border rounded-lg bg-muted/30">
              <div className="flex items-center gap-2 mb-3">
                <Server className="h-4 w-4 text-muted-foreground" />
                <Label className="font-medium">SMTP 服务器配置</Label>
                {serverConfig.fromProvider && (
                  <span className="text-xs bg-blue-100 text-blue-700 px-2 py-0.5 rounded">
                    来自 {serverConfig.providerName || '提供商'}
                  </span>
                )}
              </div>
              
              {serverConfig.host ? (
                <div className="grid grid-cols-3 gap-4 text-sm">
                  <div>
                    <p className="text-muted-foreground text-xs">服务器</p>
                    <p className="font-mono">{serverConfig.host}</p>
                  </div>
                  <div>
                    <p className="text-muted-foreground text-xs">端口</p>
                    <p className="font-mono">{serverConfig.port}</p>
                  </div>
                  <div>
                    <p className="text-muted-foreground text-xs">加密</p>
                    <p>{getEncryptionLabel(serverConfig.encryption)}</p>
                  </div>
                </div>
              ) : (
                <Alert variant="destructive" className="mt-2">
                  <AlertCircle className="h-4 w-4" />
                  <AlertDescription>
                    SMTP 服务器未配置，请在「提供商管理」中配置 SMTP 服务器
                  </AlertDescription>
                </Alert>
              )}
            </div>

            {/* 用户名 */}
            <div className="space-y-2">
              <Label htmlFor="smtp_username">用户名</Label>
              <Input
                id="smtp_username"
                placeholder="通常为邮箱地址"
                value={formData.smtp_username}
                onChange={(e) =>
                  setFormData({ ...formData, smtp_username: e.target.value })
                }
              />
              <p className="text-xs text-muted-foreground">
                通常为完整的邮箱地址
              </p>
            </div>

            {/* 密码 */}
            <div className="space-y-2">
              <Label htmlFor="smtp_password">密码/授权码</Label>
              <Input
                id="smtp_password"
                type="password"
                placeholder="留空则不修改"
                value={formData.smtp_password}
                onChange={(e) =>
                  setFormData({ ...formData, smtp_password: e.target.value })
                }
              />
              <p className="text-xs text-muted-foreground">
                QQ/163 等邮箱请使用授权码而非登录密码
              </p>
            </div>

            {/* 测试结果 */}
            {testResult && (
              <Alert variant={testResult.success ? 'default' : 'destructive'}>
                {testResult.success ? (
                  <CheckCircle2 className="h-4 w-4" />
                ) : (
                  <XCircle className="h-4 w-4" />
                )}
                <AlertDescription>{testResult.message}</AlertDescription>
              </Alert>
            )}

            {/* 提示信息 */}
            <Alert>
              <AlertCircle className="h-4 w-4" />
              <AlertDescription>
                <ul className="text-xs space-y-1 mt-1">
                  <li>• SMTP 服务器配置从提供商继承，如需修改请前往「提供商管理」</li>
                  <li>• Gmail 需要开启"允许不够安全的应用"或使用应用专用密码</li>
                  <li>• QQ/163 邮箱需要在邮箱设置中开启 SMTP 服务并获取授权码</li>
                  <li>• 建议先测试连接再保存配置</li>
                </ul>
              </AlertDescription>
            </Alert>
          </div>
        )}

        <DialogFooter className="flex-shrink-0 gap-2">
          <Button variant="outline" onClick={handleReset} disabled={isLoading}>
            <RefreshCw className="mr-2 h-4 w-4" />
            重置
          </Button>
          <Button
            variant="outline"
            onClick={handleTest}
            disabled={isLoading || isTesting || !serverConfig.host}
          >
            {isTesting ? (
              <>
                <Loader2 className="mr-2 h-4 w-4 animate-spin" />
                测试中...
              </>
            ) : (
              '测试连接'
            )}
          </Button>
          <Button onClick={handleSave} disabled={isLoading || isSaving || !serverConfig.host}>
            {isSaving ? (
              <>
                <Loader2 className="mr-2 h-4 w-4 animate-spin" />
                保存中...
              </>
            ) : (
              '保存'
            )}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
};
