import { Button } from '../ui/button';
import { Input } from '../ui/input';
import { Label } from '../ui/label';
import { Textarea } from '../ui/textarea';
import { Switch } from '../ui/switch';
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
} from '../ui/dialog';
import type { Provider, ProviderUpdateRequest } from '../../types';

interface ProviderEditDialogProps {
  /** 对话框是否打开 */
  open: boolean;
  onOpenChange: (open: boolean) => void;
  /** 正在编辑的 Provider（用于标题与提示文案） */
  provider: Provider | null;
  /** 是否为 WebAPI 供应商（页面根据 provider 计算） */
  isWebAPI: boolean;
  /** 编辑表单当前值 */
  form: ProviderUpdateRequest;
  /** 更新编辑表单（页面负责与现有表单合并） */
  onFormChange: (patch: Partial<ProviderUpdateRequest>) => void;
  /** 提交更新（页面负责请求） */
  onSubmit: () => void;
  /** 全局提交 loading */
  loading: boolean;
}

/**
 * 编辑邮箱提供商对话框（纯展示组件）：
 * WebAPI 供应商仅展示基础信息字段，传统供应商展示完整配置。
 * 不自己请求数据，全部状态与回调来自页面。
 */
export const ProviderEditDialog = ({
  open,
  onOpenChange,
  provider,
  isWebAPI,
  form,
  onFormChange,
  onSubmit,
  loading,
}: ProviderEditDialogProps) => {
  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-[600px] max-h-[90vh] overflow-y-auto">
        <DialogHeader>
          <DialogTitle>编辑邮箱提供商配置</DialogTitle>
          <DialogDescription>
            修改 "{provider?.display_name}" 的配置信息
            {provider && isWebAPI && (
              <span className="block mt-1 text-primary">（WebAPI 服务 - 系统内置）</span>
            )}
          </DialogDescription>
        </DialogHeader>
        <div className="space-y-4 py-4">
          {/* 基础信息 - 所有供应商都显示 */}
          <div className="grid grid-cols-2 gap-4">
            <div className="space-y-2">
              <Label>提供商标识</Label>
              <Input
                placeholder="例如：gmail, outlook"
                value={form.name || ''}
                onChange={(e) => onFormChange({ name: e.target.value })}
                disabled={provider ? isWebAPI : false}
              />
              {provider && isWebAPI && (
                <p className="text-xs text-muted-foreground">WebAPI 供应商标识不可修改</p>
              )}
            </div>
            <div className="space-y-2">
              <Label>显示名称</Label>
              <Input
                placeholder="例如：Google Gmail"
                value={form.display_name || ''}
                onChange={(e) => onFormChange({ display_name: e.target.value })}
              />
            </div>
          </div>

          <div className="space-y-2">
            <Label>描述信息</Label>
            <Textarea
              placeholder="请输入邮箱提供商的描述信息（可选）"
              value={form.description || ''}
              onChange={(e) => onFormChange({ description: e.target.value })}
              rows={3}
            />
            <p className="text-xs text-muted-foreground">
              描述信息将显示在提供商列表中，帮助用户了解该提供商的特点
            </p>
          </div>

          {/* 排序 */}
          <div className="space-y-2">
            <Label>排序顺序</Label>
            <Input
              type="number"
              placeholder="0"
              value={form.sort_order || 0}
              onChange={(e) =>
                onFormChange({ sort_order: parseInt(e.target.value) || 0 })
              }
            />
            <p className="text-xs text-muted-foreground">
              数值越小排序越靠前
            </p>
          </div>

          {/* 以下配置仅非 WebAPI 供应商显示 */}
          {provider && !isWebAPI && (
            <>
              {/* IMAP 配置 */}
              <div className="grid grid-cols-3 gap-4">
                <div className="space-y-2">
                  <Label>IMAP 服务器</Label>
                  <Input
                    placeholder="imap.example.com"
                    value={form.imap_host || ''}
                    onChange={(e) => onFormChange({ imap_host: e.target.value })}
                  />
                </div>
                <div className="space-y-2">
                  <Label>IMAP 端口</Label>
                  <Input
                    type="number"
                    placeholder="993"
                    value={form.imap_port || ''}
                    onChange={(e) =>
                      onFormChange({ imap_port: parseInt(e.target.value) || 993 })
                    }
                  />
                </div>
                <div className="space-y-2">
                  <Label>IMAP 加密</Label>
                  <Select
                    value={form.imap_encryption || 'ssl'}
                    onValueChange={(value) =>
                      onFormChange({ imap_encryption: value })
                    }
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
                    value={form.smtp_port || ''}
                    onChange={(e) =>
                      onFormChange({ smtp_port: parseInt(e.target.value) || 587 })
                    }
                  />
                </div>
                <div className="space-y-2">
                  <Label>SMTP 加密</Label>
                  <Select
                    value={form.smtp_encryption || 'ssl'}
                    onValueChange={(value) =>
                      onFormChange({ smtp_encryption: value })
                    }
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
                    value={form.pop3_port || ''}
                    onChange={(e) =>
                      onFormChange({ pop3_port: parseInt(e.target.value) || 995 })
                    }
                  />
                </div>
                <div className="space-y-2">
                  <Label>POP3 加密</Label>
                  <Select
                    value={form.pop3_encryption || 'ssl'}
                    onValueChange={(value) =>
                      onFormChange({ pop3_encryption: value })
                    }
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
                  value={form.recommended_protocol || 'imap'}
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
            </>
          )}

          <div className="flex items-center space-x-2">
            <Switch
              id="enabled-edit"
              checked={form.enabled ?? true}
              onCheckedChange={(checked) => onFormChange({ enabled: checked })}
            />
            <Label htmlFor="enabled-edit">启用此提供商</Label>
          </div>
        </div>
        <div className="flex justify-end gap-2">
          <Button variant="outline" onClick={() => onOpenChange(false)} disabled={loading}>
            取消
          </Button>
          <Button onClick={onSubmit} disabled={loading}>
            {loading ? '保存中...' : '保存'}
          </Button>
        </div>
      </DialogContent>
    </Dialog>
  );
};
