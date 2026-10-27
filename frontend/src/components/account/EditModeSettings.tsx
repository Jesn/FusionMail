import { CheckCircle2, XCircle, Loader2, ChevronDown, ChevronRight, Send } from 'lucide-react';
import { Button } from '../ui/button';
import { Input } from '../ui/input';
import { Label } from '../ui/label';
import { Badge } from '../ui/badge';
import { Switch } from '../ui/switch';
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '../ui/select';
import {
  Collapsible,
  CollapsibleContent,
  CollapsibleTrigger,
} from '../ui/collapsible';
import { GroupSelector } from '../group';
import type { CreateAccountRequest } from '../../services/accountService';

/** SMTP 配置（编辑模式加载的发送设置，与页面 state 形状一致） */
export interface EditModeSmtpConfig {
  smtp_enabled: boolean;
}

/** SMTP 服务器参数（继承自提供商或手填，展示与测试用） */
export interface EditModeSmtpServerConfig {
  host: string;
  port: number | string;
  encryption: string;
  fromProvider?: boolean;
  providerName?: string;
}

interface EditModeSettingsProps {
  formData: CreateAccountRequest;
  onFormChange: (patch: Partial<CreateAccountRequest>) => void;
  smtpConfig: EditModeSmtpConfig;
  onSmtpConfigChange: (config: EditModeSmtpConfig) => void;
  smtpServerConfig: EditModeSmtpServerConfig;
  isLoadingSmtp: boolean;
  isTestingSmtp: boolean;
  smtpTestResult: { success: boolean; message: string } | null;
  onTestSmtp: () => void;
  /** 受控：发件设置折叠区展开状态（页面 loadSmtpConfig 完成后可能置 true） */
  smtpSectionOpen: boolean;
  onSmtpSectionOpenChange: (open: boolean) => void;
}

/** 编辑模式的四组折叠设置：凭证 / 分组 / 同步 / SMTP 发件。
 * 纯展示：不请求、不碰 store，所有状态由 AccountForm 下传。 */
export function EditModeSettings({
  formData,
  onFormChange,
  smtpConfig,
  onSmtpConfigChange,
  smtpServerConfig,
  isLoadingSmtp,
  isTestingSmtp,
  smtpTestResult,
  onTestSmtp,
  smtpSectionOpen,
  onSmtpSectionOpenChange,
}: EditModeSettingsProps) {

  const getEncryptionLabel = (enc: string) => {
    switch (enc) {
      case 'tls': case 'ssl': return 'SSL/TLS';
      case 'starttls': return 'STARTTLS';
      case 'none': return '无加密';
      default: return enc || '未配置';
    }
  };

  return (
    <>
      {/* 更新凭证 */}
      <Collapsible>
        <CollapsibleTrigger className="flex items-center gap-2 text-sm font-medium w-full py-2">
          <ChevronRight className="h-4 w-4 transition-transform ui-open:rotate-90" />
          更新凭证
        </CollapsibleTrigger>
        <CollapsibleContent className="space-y-2 pt-2">
          <Label htmlFor="edit_password">新密码 / 授权码</Label>
          <Input
            id="edit_password"
            type="password"
            placeholder="留空则不修改"
            value={formData.password}
            onChange={(e) => onFormChange({ password: e.target.value })}
          />
        </CollapsibleContent>
      </Collapsible>

      {/* 分组设置 */}
      <div className="space-y-2">
        <Label>分组</Label>
        <GroupSelector
          value={formData.group_id}
          onChange={(id) => onFormChange({ group_id: id })}
        />
      </div>

      {/* 同步设置 */}
      <Collapsible defaultOpen>
        <CollapsibleTrigger className="flex items-center gap-2 text-sm font-medium w-full py-2">
          <ChevronDown className="h-4 w-4 transition-transform ui-closed:rotate-[-90deg]" />
          同步设置
        </CollapsibleTrigger>
        <CollapsibleContent className="space-y-4 pt-2">
          <div className="flex items-center justify-between">
            <Label htmlFor="edit_sync_enabled">启用自动同步</Label>
            <Switch
              id="edit_sync_enabled"
              checked={formData.sync_enabled}
              onCheckedChange={(checked) => onFormChange({ sync_enabled: checked })}
            />
          </div>
          {formData.sync_enabled && (
            <div className="space-y-2">
              <Label>同步频率（分钟）</Label>
              <Select
                value={String(formData.sync_interval)}
                onValueChange={(v) => onFormChange({ sync_interval: Number(v) })}
              >
                <SelectTrigger><SelectValue /></SelectTrigger>
                <SelectContent>
                  {[1, 2, 5, 10, 15, 30, 60].map((m) => (
                    <SelectItem key={m} value={String(m)}>{m} 分钟</SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </div>
          )}
          <div className="space-y-2">
            <Label>删除策略</Label>
            <Select
              value={formData.server_delete_policy}
              onValueChange={(v) => onFormChange({ server_delete_policy: v })}
            >
              <SelectTrigger><SelectValue /></SelectTrigger>
              <SelectContent>
                <SelectItem value="off">仅本地删除</SelectItem>
                <SelectItem value="soft">同步删除到服务器</SelectItem>
              </SelectContent>
            </Select>
          </div>
        </CollapsibleContent>
      </Collapsible>

      {/* SMTP 发件设置（编辑模式） */}
      <Collapsible open={smtpSectionOpen} onOpenChange={onSmtpSectionOpenChange}>
        <CollapsibleTrigger className="flex items-center gap-2 text-sm font-medium w-full py-2">
          <Send className="h-4 w-4" />
          发件设置
          {smtpConfig.smtp_enabled && <Badge variant="secondary" className="ml-2">已启用</Badge>}
        </CollapsibleTrigger>
        <CollapsibleContent className="space-y-4 pt-2">
          {isLoadingSmtp ? (
            <div className="flex items-center gap-2 text-sm text-muted-foreground">
              <Loader2 className="h-4 w-4 animate-spin" />
              加载 SMTP 配置...
            </div>
          ) : (
            <>
              <div className="flex items-center justify-between">
                <div>
                  <Label htmlFor="edit_smtp_enabled">启用发件功能</Label>
                  <p className="text-xs text-muted-foreground">使用 SMTP 发送邮件</p>
                </div>
                <Switch
                  id="edit_smtp_enabled"
                  checked={smtpConfig.smtp_enabled}
                  onCheckedChange={(checked) => onSmtpConfigChange({ smtp_enabled: checked })}
                />
              </div>
              {smtpServerConfig.host && (
                <div className="p-3 rounded-lg bg-muted/30 text-sm space-y-1">
                  <div className="flex justify-between">
                    <span className="text-muted-foreground">服务器</span>
                    <span>{smtpServerConfig.host}:{smtpServerConfig.port}</span>
                  </div>
                  <div className="flex justify-between">
                    <span className="text-muted-foreground">加密</span>
                    <span>{getEncryptionLabel(smtpServerConfig.encryption)}</span>
                  </div>
                  {smtpServerConfig.fromProvider && (
                    <div className="flex justify-between">
                      <span className="text-muted-foreground">来源</span>
                      <span>继承自 {smtpServerConfig.providerName}</span>
                    </div>
                  )}
                </div>
              )}
              {smtpConfig.smtp_enabled && smtpServerConfig.host && (
                <div className="flex items-center gap-2">
                  <Button
                    type="button"
                    variant="outline"
                    size="sm"
                    onClick={onTestSmtp}
                    disabled={isTestingSmtp}
                  >
                    {isTestingSmtp ? <Loader2 className="h-4 w-4 animate-spin mr-2" /> : null}
                    测试连接
                  </Button>
                  {smtpTestResult && (
                    <span className={`text-sm ${smtpTestResult.success ? 'text-green-600' : 'text-red-600'}`}>
                      {smtpTestResult.success ? <CheckCircle2 className="h-4 w-4 inline mr-1" /> : <XCircle className="h-4 w-4 inline mr-1" />}
                      {smtpTestResult.message}
                    </span>
                  )}
                </div>
              )}
            </>
          )}
        </CollapsibleContent>
      </Collapsible>
    </>
  );
}
