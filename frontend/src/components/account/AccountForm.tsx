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
import { CreateAccountRequest } from '../../services/accountService';
import { Account } from '../../types';
import { useProviders } from '../../hooks/useProviders';

interface AccountFormProps {
  open: boolean;
  onClose: () => void;
  onSubmit: (data: CreateAccountRequest | Partial<CreateAccountRequest>) => Promise<void>;
  account?: Account | null;
}

export const AccountForm = ({ open, onClose, onSubmit, account }: AccountFormProps) => {
  const isEditMode = !!account;
  const { providers, getProviderByEmail, getProviderByName } = useProviders();
  
  const [formData, setFormData] = useState<CreateAccountRequest>({
    email: '',
    provider: 'qq',
    protocol: 'imap',
    auth_type: 'password',
    password: '',
    sync_enabled: true,
    sync_interval: 5,
    // 通用邮箱配置
    imap_host: '',
    imap_port: 993,
    pop3_host: '',
    pop3_port: 995,
    encryption: 'ssl',
  });
  const [isSubmitting, setIsSubmitting] = useState(false);

  // 当 account 变化时，更新表单数据
  useEffect(() => {
    if (account) {
      setFormData({
        email: account.email,
        provider: account.provider,
        protocol: account.protocol,
        auth_type: account.auth_type,
        password: '', // 编辑时不显示密码
        sync_enabled: account.sync_enabled,
        sync_interval: account.sync_interval,
        // 通用邮箱配置 - 编辑时加载现有配置
        imap_host: account.imap_host || '',
        imap_port: account.imap_port || 993,
        pop3_host: account.pop3_host || '',
        pop3_port: account.pop3_port || 995,
        encryption: account.encryption || 'ssl',
      });
    } else {
      // 重置为默认值
      setFormData({
        email: '',
        provider: 'qq',
        protocol: 'imap',
        auth_type: 'password',
        password: '',
        sync_enabled: true,
        sync_interval: 5,
        // 通用邮箱配置
        imap_host: '',
        imap_port: 993,
        pop3_host: '',
        pop3_port: 995,
        encryption: 'ssl',
      });
    }
  }, [account]);

  // 处理邮箱地址变化，自动识别提供商
  const handleEmailChange = (email: string) => {
    setFormData(prev => ({ ...prev, email }));
    
    if (!isEditMode && email.includes('@')) {
      const recommendedProvider = getProviderByEmail(email);
      if (recommendedProvider) {
        setFormData(prev => ({
          ...prev,
          provider: recommendedProvider.name,
          protocol: recommendedProvider.recommended_protocol,
          // 如果是预设提供商，填充服务器配置
          imap_host: recommendedProvider.imap_host || '',
          imap_port: recommendedProvider.imap_port || 993,
          pop3_host: recommendedProvider.pop3_host || '',
          pop3_port: recommendedProvider.pop3_port || 995,
        }));
      }
    }
  };

  // 处理提供商变化
  const handleProviderChange = (provider: string) => {
    const providerInfo = getProviderByName(provider);
    if (providerInfo) {
      setFormData(prev => ({
        ...prev,
        provider,
        protocol: providerInfo.recommended_protocol,
        // 填充服务器配置
        imap_host: providerInfo.imap_host || '',
        imap_port: providerInfo.imap_port || 993,
        pop3_host: providerInfo.pop3_host || '',
        pop3_port: providerInfo.pop3_port || 995,
      }));
    }
  };

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setIsSubmitting(true);
    try {
      if (isEditMode) {
        // 编辑模式：只提交可修改的字段
        const updateData: Partial<CreateAccountRequest> = {
          sync_enabled: formData.sync_enabled,
          sync_interval: formData.sync_interval,
        };
        // 如果输入了新密码，则包含密码
        if (formData.password) {
          updateData.password = formData.password;
        }
        // 如果是通用邮箱，包含服务器配置
        if (formData.provider === 'generic') {
          updateData.imap_host = formData.imap_host;
          updateData.imap_port = formData.imap_port;
          updateData.pop3_host = formData.pop3_host;
          updateData.pop3_port = formData.pop3_port;
          updateData.encryption = formData.encryption;
        }
        await onSubmit(updateData);
      } else {
        // 创建模式：提交所有字段
        await onSubmit(formData);
      }
      onClose();
    } catch (error) {
      // 错误已在 Hook 中处理
    } finally {
      setIsSubmitting(false);
    }
  };

  return (
    <Dialog open={open} onOpenChange={onClose}>
      <DialogContent className="w-full max-w-[95vw] sm:max-w-[600px] max-h-[90vh] overflow-hidden flex flex-col">
        <DialogHeader className="flex-shrink-0">
          <DialogTitle>{isEditMode ? '编辑邮箱账户' : '添加邮箱账户'}</DialogTitle>
          <DialogDescription>
            {isEditMode ? '修改账户的同步设置' : '添加您的邮箱账户以开始接收邮件'}
          </DialogDescription>
        </DialogHeader>

        <div className="flex-1 overflow-y-auto">
          <form onSubmit={handleSubmit} className="flex flex-col h-full">
            <div className="space-y-4 py-4 px-1 flex-1 overflow-y-auto min-h-0">
            {/* 邮箱地址 */}
            <div className="space-y-2">
              <Label htmlFor="email">邮箱地址 *</Label>
              <Input
                id="email"
                type="email"
                placeholder="your@example.com"
                value={formData.email}
                onChange={(e) => handleEmailChange(e.target.value)}
                required
                disabled={isEditMode}
              />
            </div>

            {/* 邮箱提供商 */}
            <div className="space-y-2">
              <Label htmlFor="provider">邮箱提供商 *</Label>
              <Select
                value={formData.provider}
                onValueChange={handleProviderChange}
                disabled={isEditMode}
              >
                <SelectTrigger>
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  {providers.map((provider) => (
                    <SelectItem key={provider.name} value={provider.name}>
                      {provider.display_name}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </div>

            {/* 协议 */}
            <div className="space-y-2">
              <Label htmlFor="protocol">协议 *</Label>
              <Select
                value={formData.protocol}
                onValueChange={(value) =>
                  setFormData({ ...formData, protocol: value })
                }
                disabled={isEditMode}
              >
                <SelectTrigger>
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="imap">IMAP</SelectItem>
                  <SelectItem value="pop3">POP3</SelectItem>
                </SelectContent>
              </Select>
            </div>

            {/* 通用邮箱配置 */}
            {formData.provider === 'generic' && (
              <div className="space-y-4 p-4 border rounded-lg bg-gray-50 dark:bg-gray-800/50">
                <h4 className="font-medium text-sm text-gray-900 dark:text-white">
                  服务器配置
                </h4>
                <p className="text-xs text-gray-600 dark:text-gray-400">
                  请联系您的邮箱服务商获取正确的服务器配置信息
                </p>
                
                {formData.protocol === 'imap' && (
                  <div className="grid grid-cols-1 sm:grid-cols-2 gap-4">
                    <div>
                      <Label htmlFor="imap_host">IMAP 服务器 *</Label>
                      <Input
                        id="imap_host"
                        placeholder="imap.example.com"
                        value={formData.imap_host}
                        onChange={(e) =>
                          setFormData({ ...formData, imap_host: e.target.value })
                        }
                        className="w-full"
                        required
                      />
                    </div>
                    <div>
                      <Label htmlFor="imap_port">IMAP 端口 *</Label>
                      <Input
                        id="imap_port"
                        type="number"
                        placeholder="993"
                        value={formData.imap_port}
                        onChange={(e) =>
                          setFormData({ ...formData, imap_port: parseInt(e.target.value) || 993 })
                        }
                        required
                      />
                    </div>
                  </div>
                )}

                {formData.protocol === 'pop3' && (
                  <div className="grid grid-cols-1 sm:grid-cols-2 gap-4">
                    <div>
                      <Label htmlFor="pop3_host">POP3 服务器 *</Label>
                      <Input
                        id="pop3_host"
                        placeholder="pop3.example.com"
                        value={formData.pop3_host}
                        onChange={(e) =>
                          setFormData({ ...formData, pop3_host: e.target.value })
                        }
                        required
                      />
                    </div>
                    <div>
                      <Label htmlFor="pop3_port">POP3 端口 *</Label>
                      <Input
                        id="pop3_port"
                        type="number"
                        placeholder="995"
                        value={formData.pop3_port}
                        onChange={(e) =>
                          setFormData({ ...formData, pop3_port: parseInt(e.target.value) || 995 })
                        }
                        required
                      />
                    </div>
                  </div>
                )}

                <div>
                  <Label htmlFor="encryption">加密方式 *</Label>
                  <Select
                    value={formData.encryption}
                    onValueChange={(value) =>
                      setFormData({ ...formData, encryption: value })
                    }
                  >
                    <SelectTrigger>
                      <SelectValue />
                    </SelectTrigger>
                    <SelectContent>
                      <SelectItem value="ssl">SSL/TLS (推荐)</SelectItem>
                      <SelectItem value="starttls">STARTTLS</SelectItem>
                      <SelectItem value="none">无加密</SelectItem>
                    </SelectContent>
                  </Select>
                </div>
              </div>
            )}

            {/* 密码/授权码 */}
            <div className="space-y-2">
              <Label htmlFor="password">
                {isEditMode ? '新密码/授权码（留空则不修改）' : '密码/授权码 *'}
              </Label>
              <Input
                id="password"
                type="password"
                placeholder={isEditMode ? '留空则不修改密码' : '请输入密码或授权码'}
                value={formData.password}
                onChange={(e) =>
                  setFormData({ ...formData, password: e.target.value })
                }
                required={!isEditMode}
              />
              {!isEditMode && (
                <p className="text-xs text-muted-foreground">
                  QQ/163 邮箱请使用授权码，而非登录密码
                </p>
              )}
            </div>

            {/* 同步设置 */}
            <div className="space-y-4 rounded-lg border p-4">
              <div className="flex items-center justify-between">
                <Label htmlFor="sync_enabled">启用自动同步</Label>
                <Switch
                  id="sync_enabled"
                  checked={formData.sync_enabled}
                  onCheckedChange={(checked) =>
                    setFormData({ ...formData, sync_enabled: checked })
                  }
                />
              </div>

              {formData.sync_enabled && (
                <div className="space-y-2">
                  <Label htmlFor="sync_interval">同步频率（分钟）</Label>
                  <Input
                    id="sync_interval"
                    type="number"
                    min="1"
                    max="60"
                    value={formData.sync_interval}
                    onChange={(e) =>
                      setFormData({
                        ...formData,
                        sync_interval: parseInt(e.target.value, 10),
                      })
                    }
                  />
                </div>
              )}
            </div>
            </div>

            <DialogFooter className="flex-shrink-0 mt-4">
              <Button type="button" variant="outline" onClick={onClose}>
                取消
              </Button>
              <Button type="submit" disabled={isSubmitting}>
                {isSubmitting 
                  ? (isEditMode ? '保存中...' : '添加中...') 
                  : (isEditMode ? '保存' : '添加账户')}
              </Button>
            </DialogFooter>
          </form>
        </div>
      </DialogContent>
    </Dialog>
  );
};
