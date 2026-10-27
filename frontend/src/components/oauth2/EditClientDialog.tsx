import { Button } from '../ui/button';
import { Input } from '../ui/input';
import { Label } from '../ui/label';
import { Textarea } from '../ui/textarea';
import { Switch } from '../ui/switch';
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
} from '../ui/dialog';
import type { OAuth2Client, OAuth2ClientUpdateRequest } from '../../types';

interface EditClientDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  selectedClient: OAuth2Client | null;
  form: OAuth2ClientUpdateRequest;
  onFormChange: (form: OAuth2ClientUpdateRequest) => void;
  onSubmit: () => void;
  loading: boolean;
}

/** 编辑 OAuth2 客户端对话框（自 OAuth2ClientsPage 同位搬移）。纯展示。 */
export function EditClientDialog({
  open,
  onOpenChange,
  selectedClient,
  form,
  onFormChange,
  onSubmit,
  loading,
}: EditClientDialogProps) {
  return (
      <Dialog open={open} onOpenChange={onOpenChange}>
        <DialogContent className="sm:max-w-[500px]">
          <DialogHeader>
            <DialogTitle>编辑 OAuth2 客户端配置</DialogTitle>
            <DialogDescription>修改 "{selectedClient?.name}" 的配置信息</DialogDescription>
          </DialogHeader>
          <div className="space-y-4 py-4">
            <div className="space-y-2">
              <Label>配置名称</Label>
              <Input
                placeholder="输入配置名称"
                value={form.name || ''}
                onChange={(e) => onFormChange({ ...form, name: e.target.value })}
              />
            </div>

            <div className="space-y-2">
              <Label>Client ID</Label>
              <Input
                placeholder="输入 Client ID"
                value={form.client_id || ''}
                onChange={(e) =>
                  onFormChange({ ...form, client_id: e.target.value })
                }
              />
            </div>

            <div className="space-y-2">
              <Label>Client Secret（留空则不修改）</Label>
              <Input
                type="password"
                placeholder="输入新 Client Secret"
                value={form.client_secret || ''}
                onChange={(e) =>
                  onFormChange({ ...form, client_secret: e.target.value })
                }
              />
            </div>

            <div className="space-y-2">
              <Label>描述信息</Label>
              <Textarea
                placeholder="请输入OAuth2客户端配置的描述信息（可选）"
                value={form.metadata || ''}
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
                <Label>日配额</Label>
                <Input
                  type="number"
                  value={form.quota_daily ?? 100}
                  onChange={(e) =>
                    onFormChange({
                      ...form,
                      quota_daily: parseInt(e.target.value) || -1,
                    })
                  }
                />
              </div>
              <div className="space-y-2">
                <Label>月配额</Label>
                <Input
                  type="number"
                  value={form.quota_monthly ?? 2000}
                  onChange={(e) =>
                    onFormChange({
                      ...form,
                      quota_monthly: parseInt(e.target.value) || -1,
                    })
                  }
                />
              </div>
            </div>

            <div className="flex items-center space-x-2">
              <Switch
                id="enabled"
                checked={form.enabled ?? true}
                onCheckedChange={(checked) =>
                  onFormChange({ ...form, enabled: checked })
                }
              />
              <Label htmlFor="enabled">启用此配置</Label>
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
}
