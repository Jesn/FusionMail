import { Plus } from 'lucide-react';
import { Button } from '../ui/button';
import { Input } from '../ui/input';
import { Label } from '../ui/label';
import { Textarea } from '../ui/textarea';
import { Switch } from '../ui/switch';
import { Checkbox } from '../ui/checkbox';
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
  DialogHeader,
  DialogTitle,
  DialogTrigger,
} from '../ui/dialog';
import type { AdapterResponse, ProviderCreateRequest } from '../../types';

interface ProviderCreateDialogProps {
  /** 对话框是否打开 */
  open: boolean;
  onOpenChange: (open: boolean) => void;
  /** 创建表单当前值 */
  form: ProviderCreateRequest;
  /** 更新创建表单（页面负责与现有表单合并） */
  onFormChange: (patch: Partial<ProviderCreateRequest>) => void;
  /** 可选适配器列表 */
  adapters: AdapterResponse[];
  /** 邮箱域名输入框的临时值（提交时才拆分为数组） */
  emailDomainsInput: string;
  onEmailDomainsInputChange: (value: string) => void;
  /** 提交创建（页面负责校验与请求） */
  onSubmit: () => void;
  /** 全局提交 loading */
  loading: boolean;
}

/**
 * 创建邮箱提供商对话框（纯展示组件）：
 * 含触发按钮与完整表单，不自己请求数据，全部状态与回调来自页面。
 */
export const ProviderCreateDialog = ({
  open,
  onOpenChange,
  form,
  onFormChange,
  adapters,
  emailDomainsInput,
  onEmailDomainsInputChange,
  onSubmit,
  loading,
}: ProviderCreateDialogProps) => {
  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogTrigger asChild>
        <Button onClick={() => onOpenChange(true)} disabled={loading}>
          <Plus className="h-4 w-4 mr-2" />
          新增提供商
        </Button>
      </DialogTrigger>
      <DialogContent className="sm:max-w-[600px] max-h-[90vh] overflow-y-auto">
        <DialogHeader>
          <DialogTitle>创建邮箱提供商配置</DialogTitle>
          <DialogDescription>
            为新邮箱提供商创建服务器和协议配置
          </DialogDescription>
        </DialogHeader>
        <div className="space-y-4 py-4">
          <div className="grid grid-cols-2 gap-4">
            <div className="space-y-2">
              <Label>提供商标识 *</Label>
              <Input
                placeholder="例如：gmail, outlook"
                value={form.name}
                onChange={(e) => onFormChange({ name: e.target.value })}
              />
            </div>
            <div className="space-y-2">
              <Label>显示名称 *</Label>
              <Input
                placeholder="例如：Google Gmail"
                value={form.display_name}
                onChange={(e) => onFormChange({ display_name: e.target.value })}
              />
            </div>
          </div>

          {/* 默认连接方式 */}
          {adapters.length > 0 && (
            <div className="space-y-2">
              <Label>默认连接方式</Label>
              <Select
                value={form.default_adapter_id?.toString() || ''}
                onValueChange={(value) => {
                  const adapterId = value ? parseInt(value) : undefined;
                  const adapter = adapters.find((a) => a.id === adapterId);
                  onFormChange({
                    default_adapter_id: adapterId,
                    requires_oauth: adapter?.auth_type === 'oauth2',
                    recommended_protocol:
                      adapter?.auth_type === 'oauth2' ? 'oauth2' : 'imap',
                  });
                }}
              >
                <SelectTrigger>
                  <SelectValue placeholder="选择默认连接方式" />
                </SelectTrigger>
                <SelectContent>
                  {adapters.map((adapter) => (
                    <SelectItem key={adapter.id} value={adapter.id.toString()}>
                      {adapter.display_name} ({adapter.auth_type === 'oauth2' ? 'OAuth2 认证' : '密码认证'})
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
              <p className="text-xs text-muted-foreground">
                选择该提供商默认使用的连接方式
              </p>
            </div>
          )}

          {/* 连接方式（多选） */}
          {adapters.length > 0 && (
            <div className="space-y-2">
              <Label>连接方式</Label>
              <div className="grid grid-cols-2 gap-2 p-3 border rounded-md">
                {adapters.map((adapter) => (
                  <div key={adapter.id} className="flex items-center space-x-2">
                    <Checkbox
                      id={`adapter-${adapter.id}`}
                      checked={form.adapter_ids?.includes(adapter.id) || false}
                      onCheckedChange={(checked) => {
                        const currentIds = form.adapter_ids || [];
                        const newIds = checked
                          ? [...currentIds, adapter.id]
                          : currentIds.filter((id) => id !== adapter.id);
                        onFormChange({ adapter_ids: newIds });
                      }}
                    />
                    <label
                      htmlFor={`adapter-${adapter.id}`}
                      className="text-sm font-medium leading-none peer-disabled:cursor-not-allowed peer-disabled:opacity-70"
                    >
                      {adapter.display_name}
                    </label>
                  </div>
                ))}
              </div>
              <p className="text-xs text-muted-foreground">
                选择该提供商支持的连接方式（Gmail API、Microsoft Graph 为 OAuth2 认证，IMAP 为通用协议）
              </p>
            </div>
          )}

          {/* 邮箱域名 */}
          <div className="space-y-2">
            <Label>支持的邮箱域名</Label>
            <Input
              placeholder="gmail.com, googlemail.com（逗号分隔）"
              value={emailDomainsInput}
              onChange={(e) => onEmailDomainsInputChange(e.target.value)}
            />
            <p className="text-xs text-muted-foreground">
              输入该提供商支持的邮箱域名，用于自动匹配提供商
            </p>
          </div>

          <div className="space-y-2">
            <Label>描述信息</Label>
            <Textarea
              placeholder="请输入邮箱提供商的描述信息（可选）"
              value={form.description}
              onChange={(e) => onFormChange({ description: e.target.value })}
              rows={3}
            />
            <p className="text-xs text-muted-foreground">
              描述信息将显示在提供商列表中，帮助用户了解该提供商的特点
            </p>
          </div>

          {/* IMAP 配置 */}
          <div className="grid grid-cols-3 gap-4">
            <div className="space-y-2">
              <Label>IMAP 服务器</Label>
              <Input
                placeholder="imap.example.com"
                value={form.imap_host}
                onChange={(e) => onFormChange({ imap_host: e.target.value })}
              />
            </div>
            <div className="space-y-2">
              <Label>IMAP 端口</Label>
              <Input
                type="number"
                placeholder="993"
                value={form.imap_port}
                onChange={(e) =>
                  onFormChange({ imap_port: parseInt(e.target.value) || 993 })
                }
              />
            </div>
            <div className="space-y-2">
              <Label>IMAP 加密</Label>
              <Select
                value={form.imap_encryption || 'ssl'}
                onValueChange={(value) => onFormChange({ imap_encryption: value })}
              >
                <SelectTrigger>
                  <SelectValue placeholder="选择加密方式" />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="ssl">SSL/TLS</SelectItem>
                  <SelectItem value="starttls">STARTTLS</SelectItem>
                  <SelectItem value="none">无加密</SelectItem>
                </SelectContent>
              </Select>
            </div>
          </div>

          {/* SMTP 配置 */}
          <div className="grid grid-cols-3 gap-4">
            <div className="space-y-2">
              <Label>SMTP 服务器</Label>
              <Input
                placeholder="smtp.example.com"
                value={form.smtp_host || ''}
                onChange={(e) => onFormChange({ smtp_host: e.target.value })}
              />
            </div>
            <div className="space-y-2">
              <Label>SMTP 端口</Label>
              <Input
                type="number"
                placeholder="587"
                value={form.smtp_port || 587}
                onChange={(e) =>
                  onFormChange({ smtp_port: parseInt(e.target.value) || 587 })
                }
              />
            </div>
            <div className="space-y-2">
              <Label>SMTP 加密</Label>
              <Select
                value={form.smtp_encryption || 'ssl'}
                onValueChange={(value) => onFormChange({ smtp_encryption: value })}
              >
                <SelectTrigger>
                  <SelectValue placeholder="选择加密方式" />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="ssl">SSL/TLS</SelectItem>
                  <SelectItem value="starttls">STARTTLS</SelectItem>
                  <SelectItem value="none">无加密</SelectItem>
                </SelectContent>
              </Select>
            </div>
          </div>

          {/* POP3 配置 */}
          <div className="grid grid-cols-3 gap-4">
            <div className="space-y-2">
              <Label>POP3 服务器</Label>
              <Input
                placeholder="pop.example.com（可选）"
                value={form.pop3_host || ''}
                onChange={(e) => onFormChange({ pop3_host: e.target.value })}
              />
            </div>
            <div className="space-y-2">
              <Label>POP3 端口</Label>
              <Input
                type="number"
                placeholder="995"
                value={form.pop3_port || 995}
                onChange={(e) =>
                  onFormChange({ pop3_port: parseInt(e.target.value) || 995 })
                }
              />
            </div>
            <div className="space-y-2">
              <Label>POP3 加密</Label>
              <Select
                value={form.pop3_encryption || 'ssl'}
                onValueChange={(value) => onFormChange({ pop3_encryption: value })}
              >
                <SelectTrigger>
                  <SelectValue placeholder="选择加密方式" />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="ssl">SSL/TLS</SelectItem>
                  <SelectItem value="starttls">STARTTLS</SelectItem>
                  <SelectItem value="none">无加密</SelectItem>
                </SelectContent>
              </Select>
            </div>
          </div>
          <p className="text-xs text-muted-foreground -mt-2">
            POP3 配置为可选，留空则添加账户时不显示 POP3 协议选项
          </p>

          <div className="space-y-2">
            <Label>推荐协议 *</Label>
            <Select
              value={form.recommended_protocol}
              onValueChange={(value) =>
                onFormChange({ recommended_protocol: value })
              }
            >
              <SelectTrigger>
                <SelectValue placeholder="选择推荐协议" />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="oauth2">OAuth2</SelectItem>
                <SelectItem value="imap">IMAP</SelectItem>
                <SelectItem value="pop3">POP3</SelectItem>
              </SelectContent>
            </Select>
            <p className="text-xs text-muted-foreground">
              选择推荐的认证和协议类型
            </p>
          </div>

          <div className="flex items-center space-x-2">
            <Switch
              id="enabled"
              checked={form.enabled}
              onCheckedChange={(checked) => onFormChange({ enabled: checked })}
            />
            <Label htmlFor="enabled">启用此提供商</Label>
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
};
