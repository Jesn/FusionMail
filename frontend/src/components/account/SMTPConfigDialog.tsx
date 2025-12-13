/**
 * SMTP 配置对话框组件
 * 用于配置账户的 SMTP 发送设置
 */

import { useState, useEffect } from 'react';
import { Button } from '../ui/button';
import { Input } from '../ui/input';
import { Label } from '../ui/label';
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '../ui/select';
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
import { Loader2, CheckCircle2, XCircle, AlertCircle, RefreshCw } from 'lucide-react';
import { smtpService } from '../../services/smtpService';
import type { UpdateSMTPConfigRequest, DefaultSMTPConfig } from '../../types';
import toast from 'react-hot-toast';

interface SMTPConfigDialogProps {
  open: boolean;
  onClose: () => void;
  accountUid: string;
  accountEmail: string;
  accountProvider?: string; // 可选，用于未来扩展
}

export const SMTPConfigDialog = ({
  open,
  onClose,
  accountUid,
  accountEmail,
  accountProvider: _accountProvider, // 保留参数用于未来扩展
}: SMTPConfigDialogProps) => {
  // accountProvider 可用于未来根据服务商自动选择默认配置
  void _accountProvider;
  // 表单状态
  const [formData, setFormData] = useState<UpdateSMTPConfigRequest>({
    smtp_host: '',
    smtp_port: 465,
    smtp_encryption: 'tls',
    smtp_username: '',
    smtp_password: '',
    smtp_enabled: false,
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

  // 默认配置列表
  const [defaultConfigs, setDefaultConfigs] = useState<DefaultSMTPConfig[]>([]);

  // 加载当前配置
  useEffect(() => {
    if (open && accountUid) {
      loadConfig();
      loadDefaultConfigs();
    }
  }, [open, accountUid]);

  // 加载 SMTP 配置
  const loadConfig = async () => {
    setIsLoading(true);
    try {
      const config = await smtpService.getConfig(accountUid);
      setFormData({
        smtp_host: config.smtp_host || '',
        smtp_port: config.smtp_port || 465,
        smtp_encryption: config.smtp_encryption || 'tls',
        smtp_username: config.smtp_username || accountEmail,
        smtp_password: '', // 密码不回显
        smtp_enabled: config.smtp_enabled || false,
      });
    } catch (error) {
      console.error('加载 SMTP 配置失败:', error);
      // 如果加载失败，使用默认值
      setFormData({
        smtp_host: '',
        smtp_port: 465,
        smtp_encryption: 'tls',
        smtp_username: accountEmail,
        smtp_password: '',
        smtp_enabled: false,
      });
    } finally {
      setIsLoading(false);
    }
  };

  // 加载默认配置列表
  const loadDefaultConfigs = async () => {
    try {
      const configs = await smtpService.getDefaultConfigs();
      setDefaultConfigs(configs);
    } catch (error) {
      console.error('加载默认配置失败:', error);
    }
  };

  // 应用默认配置
  const applyDefaultConfig = (provider: string) => {
    const config = defaultConfigs.find((c) => c.provider === provider);
    if (config) {
      setFormData((prev) => ({
        ...prev,
        smtp_host: config.smtp_host,
        smtp_port: config.smtp_port,
        smtp_encryption: config.smtp_encryption,
      }));
      toast.success(`已应用 ${config.name} 的默认配置`);
    }
  };

  // 保存配置
  const handleSave = async () => {
    if (!formData.smtp_host) {
      toast.error('请输入 SMTP 服务器地址');
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
    setIsTesting(true);
    setTestResult(null);
    try {
      const result = await smtpService.testConnection(accountUid);
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

            {/* 服务商快速配置 */}
            {defaultConfigs.length > 0 && (
              <div className="space-y-2">
                <Label>快速配置</Label>
                <Select onValueChange={applyDefaultConfig}>
                  <SelectTrigger>
                    <SelectValue placeholder="选择服务商自动填充配置" />
                  </SelectTrigger>
                  <SelectContent>
                    {defaultConfigs.map((config) => (
                      <SelectItem key={config.provider} value={config.provider}>
                        {config.name}
                      </SelectItem>
                    ))}
                  </SelectContent>
                </Select>
                <p className="text-xs text-muted-foreground">
                  选择邮箱服务商可自动填充 SMTP 服务器配置
                </p>
              </div>
            )}

            {/* SMTP 服务器 */}
            <div className="grid grid-cols-2 gap-4">
              <div className="space-y-2">
                <Label htmlFor="smtp_host">SMTP 服务器 *</Label>
                <Input
                  id="smtp_host"
                  placeholder="smtp.example.com"
                  value={formData.smtp_host}
                  onChange={(e) =>
                    setFormData({ ...formData, smtp_host: e.target.value })
                  }
                />
              </div>
              <div className="space-y-2">
                <Label htmlFor="smtp_port">端口 *</Label>
                <Input
                  id="smtp_port"
                  type="number"
                  placeholder="465"
                  value={formData.smtp_port}
                  onChange={(e) =>
                    setFormData({
                      ...formData,
                      smtp_port: parseInt(e.target.value) || 465,
                    })
                  }
                />
              </div>
            </div>

            {/* 加密方式 */}
            <div className="space-y-2">
              <Label htmlFor="smtp_encryption">加密方式</Label>
              <Select
                value={formData.smtp_encryption}
                onValueChange={(value: 'none' | 'tls' | 'starttls') =>
                  setFormData({ ...formData, smtp_encryption: value })
                }
              >
                <SelectTrigger>
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="tls">SSL/TLS（推荐，端口 465）</SelectItem>
                  <SelectItem value="starttls">STARTTLS（端口 587）</SelectItem>
                  <SelectItem value="none">无加密（不推荐）</SelectItem>
                </SelectContent>
              </Select>
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
            disabled={isLoading || isTesting || !formData.smtp_host}
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
          <Button onClick={handleSave} disabled={isLoading || isSaving}>
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
