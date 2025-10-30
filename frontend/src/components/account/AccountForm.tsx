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

interface AccountFormProps {
  open: boolean;
  onClose: () => void;
  onSubmit: (data: CreateAccountRequest | Partial<CreateAccountRequest>) => Promise<void>;
  account?: Account | null;
}

export const AccountForm = ({ open, onClose, onSubmit, account }: AccountFormProps) => {
  const isEditMode = !!account;
  
  const [formData, setFormData] = useState<CreateAccountRequest>({
    email: '',
    provider: 'qq',
    protocol: 'imap',
    auth_type: 'password',
    password: '',
    sync_enabled: true,
    sync_interval: 5,
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
      });
    }
  }, [account]);

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
      <DialogContent className="sm:max-w-[500px]">
        <DialogHeader>
          <DialogTitle>{isEditMode ? '编辑邮箱账户' : '添加邮箱账户'}</DialogTitle>
          <DialogDescription>
            {isEditMode ? '修改账户的同步设置' : '添加您的邮箱账户以开始接收邮件'}
          </DialogDescription>
        </DialogHeader>

        <form onSubmit={handleSubmit}>
          <div className="space-y-4 py-4">
            {/* 邮箱地址 */}
            <div className="space-y-2">
              <Label htmlFor="email">邮箱地址 *</Label>
              <Input
                id="email"
                type="email"
                placeholder="your@example.com"
                value={formData.email}
                onChange={(e) =>
                  setFormData({ ...formData, email: e.target.value })
                }
                required
                disabled={isEditMode}
              />
            </div>

            {/* 邮箱提供商 */}
            <div className="space-y-2">
              <Label htmlFor="provider">邮箱提供商 *</Label>
              <Select
                value={formData.provider}
                onValueChange={(value) =>
                  setFormData({ ...formData, provider: value })
                }
                disabled={isEditMode}
              >
                <SelectTrigger>
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="qq">QQ 邮箱</SelectItem>
                  <SelectItem value="163">163 邮箱</SelectItem>
                  <SelectItem value="gmail">Gmail</SelectItem>
                  <SelectItem value="outlook">Outlook</SelectItem>
                  <SelectItem value="icloud">iCloud</SelectItem>
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

          <DialogFooter>
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
      </DialogContent>
    </Dialog>
  );
};
