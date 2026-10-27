import { Plus } from 'lucide-react';
import { Button } from '../ui/button';
import { Input } from '../ui/input';
import { Label } from '../ui/label';
import { Textarea } from '../ui/textarea';
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
  DialogTrigger,
} from '../ui/dialog';
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '../ui/select';
import type { OAuth2ClientCreateRequest } from '../../types';
import type { Provider } from '../../services/systemService';
import { getOAuth2CallbackUri, getOAuth2ProviderType } from './oauth2ClientMeta';

interface CreateClientDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  providers: Provider[];
  form: OAuth2ClientCreateRequest;
  onFormChange: (form: OAuth2ClientCreateRequest) => void;
  onSubmit: () => void;
  loading: boolean;
}

/** 创建 OAuth2 客户端对话框（自 OAuth2ClientsPage 同位搬移）。
 * 纯展示：不请求、不碰 store；提供商过滤/回调推导随唯一消费迁入。 */
export function CreateClientDialog({
  open,
  onOpenChange,
  providers,
  form,
  onFormChange,
  onSubmit,
  loading,
}: CreateClientDialogProps) {
  return (
        <Dialog open={open} onOpenChange={onOpenChange}>
          <DialogTrigger asChild>
            <Button onClick={() => onOpenChange(true)} disabled={loading}>
              <Plus className="h-4 w-4 mr-2" />
              新增配置
            </Button>
          </DialogTrigger>
          <DialogContent className="sm:max-w-[500px]">
            <DialogHeader>
              <DialogTitle>创建 OAuth2 客户端配置</DialogTitle>
              <DialogDescription>
                为指定邮箱提供商创建新的 OAuth2 客户端配置
              </DialogDescription>
            </DialogHeader>
            <div className="space-y-4 py-4">
              <div className="space-y-2">
                <Label>邮箱提供商 *</Label>
                <Select
                  value={form.provider_id?.toString() || ''}
                  onValueChange={(value: string) => {
                    const providerId = parseInt(value, 10);
                    const provider = providers.find(p => p.id === providerId);
                    // 使用 supported_adapters 推导 OAuth2 提供商类型
                    const oauth2Type = getOAuth2ProviderType(provider);
                    onFormChange({
                      ...form,
                      provider_id: providerId,
                      redirect_uri: getOAuth2CallbackUri(oauth2Type),
                    });
                  }}
                >
                  <SelectTrigger>
                    <SelectValue placeholder="选择提供商" />
                  </SelectTrigger>
                  <SelectContent>
                    {providers
                      .filter(p => {
                        // 只显示启用且支持 OAuth2 的提供商
                        if (!p.enabled) return false;
                        
                        // 方法1：检查 supported_adapters 中是否有 oauth2 类型的适配器
                        if (p.supported_adapters && p.supported_adapters.length > 0) {
                          return p.supported_adapters.some(a => a.auth_type === 'oauth2');
                        }
                        
                        // 方法2：检查 requires_oauth 字段（回退兼容）
                        if (p.requires_oauth) return true;
                        
                        // 方法3：通过 provider 名称判断（最终回退，兼容旧数据）
                        const name = p.name?.toLowerCase() || '';
                        return name === 'gmail' || name === 'outlook';
                      })
                      .map((provider) => (
                        <SelectItem key={provider.id} value={provider.id.toString()}>
                          {provider.display_name}
                        </SelectItem>
                      ))}
                  </SelectContent>
                </Select>
                <p className="text-xs text-muted-foreground">
                  仅显示支持 OAuth2 认证的邮箱提供商
                </p>
              </div>

              <div className="space-y-2">
                <Label>配置名称 *</Label>
                <Input
                  placeholder="例如：生产环境、测试环境等"
                  value={form.name}
                  onChange={(e) =>
                    onFormChange({ ...form, name: e.target.value })
                  }
                />
              </div>

              <div className="space-y-2">
                <Label>Client ID *</Label>
                <Input
                  placeholder="输入 Client ID"
                  value={form.client_id}
                  onChange={(e) =>
                    onFormChange({ ...form, client_id: e.target.value })
                  }
                />
              </div>

              <div className="space-y-2">
                <Label>Client Secret *</Label>
                <Input
                  type="password"
                  placeholder="输入 Client Secret"
                  value={form.client_secret}
                  onChange={(e) =>
                    onFormChange({ ...form, client_secret: e.target.value })
                  }
                />
              </div>

              <div className="space-y-2">
                <Label>重定向 URI</Label>
                <Input
                  placeholder="输入重定向 URI"
                  value={form.redirect_uri}
                  onChange={(e) =>
                    onFormChange({ ...form, redirect_uri: e.target.value })
                  }
                />
              </div>

              <div className="space-y-2">
                <Label>描述信息</Label>
                <Textarea
                  placeholder="请输入OAuth2客户端配置的描述信息（可选）"
                  value={form.metadata}
                  onChange={(e) =>
                    onFormChange({ ...form, metadata: e.target.value })
                  }
                  rows={3}
                />
                <p className="text-xs text-muted-foreground">
                  描述信息将显示在客户端配置列表中，帮助识别配置的用途
                </p>
              </div>

              <div className="grid grid-cols-2 gap-4">
                <div className="space-y-2">
                  <Label>日配额（-1为无限制）</Label>
                  <Input
                    type="number"
                    value={form.quota_daily}
                    onChange={(e) =>
                      onFormChange({
                        ...form,
                        quota_daily: parseInt(e.target.value) || -1,
                      })
                    }
                  />
                </div>
                <div className="space-y-2">
                  <Label>月配额（-1为无限制）</Label>
                  <Input
                    type="number"
                  value={form.quota_monthly}
                    onChange={(e) =>
                      onFormChange({
                        ...form,
                        quota_monthly: parseInt(e.target.value) || -1,
                      })
                    }
                  />
                </div>
              </div>
            </div>
            <div className="flex justify-end gap-2">
              <Button
                variant="outline"
                onClick={() => onOpenChange(false)}
                disabled={loading}
              >
                取消
              </Button>
              <Button onClick={onSubmit} disabled={loading}>
                {loading ? '创建中...' : '创建'}
              </Button>
            </div>
          </DialogContent>
        </Dialog>
  );
}
