import {
  AlertCircle,
  CheckCircle2,
  ChevronRight,
  Mail,
  Send,
  Server,
} from 'lucide-react';
import { Button } from '../ui/button';
import { Badge } from '../ui/badge';
import { Input } from '../ui/input';
import { Label } from '../ui/label';
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
import { Alert, AlertDescription } from '../ui/alert';
import { OAuth2AuthButton } from '../auth/OAuth2AuthButton';
import { OAuth2ClientSelector } from '../oauth2';
import { GroupSelector } from '../group';
import { toast } from 'sonner';
import type { Provider } from '../../services/systemService';
import type { CreateAccountRequest } from '../../services/accountService';

interface CreateModeFieldsProps {
  formData: CreateAccountRequest;
  onFormChange: (patch: Partial<CreateAccountRequest>) => void;
  matchedProvider: Provider | null;
  onClearProvider: () => void;
  onEmailChange: (email: string) => void;
  onProviderChange: (providerName: string) => void;
  onProtocolChange: (protocol: string) => void;
  selectedOAuth2ClientId: number | undefined;
  onOAuth2ClientChange: (id: number | undefined) => void;
  /** OAuth2 认证成功收尾（toast + 关闭 + 触发列表刷新），由页面实现 */
  onOAuth2AuthSuccess: () => void;
  providers: Provider[];
}

/** 新建模式「邮箱优先」流程字段区（自 AccountForm 同位搬移）。
 * 纯展示：不请求、不碰 store；协议派生随唯一消费组件迁入。 */
export function CreateModeFields({
  formData,
  onFormChange,
  matchedProvider,
  onClearProvider,
  onEmailChange,
  onProviderChange,
  onProtocolChange,
  selectedOAuth2ClientId,
  onOAuth2ClientChange,
  onOAuth2AuthSuccess,
  providers,
}: CreateModeFieldsProps) {
  const getAvailableProtocols = () => {
    if (!matchedProvider) return [];
    const protocols: Array<{ value: string; label: string; recommended?: boolean }> = [];
    const supported = matchedProvider.supported_protocols || [];
    const rec = matchedProvider.recommended_protocol;
    if (matchedProvider.requires_oauth) protocols.push({ value: 'oauth2', label: 'OAuth2 安全认证', recommended: rec === 'oauth2' });
    if (matchedProvider.imap_host?.trim()) protocols.push({ value: 'imap', label: 'IMAP 协议', recommended: rec === 'imap' });
    if (matchedProvider.pop3_host?.trim()) protocols.push({ value: 'pop3', label: 'POP3 协议', recommended: rec === 'pop3' });
    if (supported.includes('batch_import')) protocols.push({ value: 'batch_import', label: '批量导入（短效邮箱）' });
    return protocols;
  };

  const hasSmtpSupport = () => matchedProvider ? !!(matchedProvider.smtp_host?.trim()) : false;

  return (
    <>
          {/* 邮箱地址输入（非 OAuth2） */}
          {formData.protocol !== 'oauth2' && (
            <div className="space-y-2">
              <Label htmlFor="email">
                <span className="flex items-center gap-2">
                  <Mail className="h-4 w-4" />
                  邮箱地址 *
                </span>
              </Label>
              <Input
                id="email"
                type="email"
                placeholder="your@example.com"
                value={formData.email}
                onChange={(e) => onEmailChange(e.target.value)}
                required
                autoFocus
              />
              <p className="text-xs text-muted-foreground">输入邮箱地址，系统将自动识别邮箱提供商</p>
            </div>
          )}

          {/* 匹配状态显示 */}
          {matchedProvider && (
            <Alert className="border-green-200 bg-green-50 dark:border-green-800 dark:bg-green-950">
              <CheckCircle2 className="h-4 w-4 text-green-600" />
              <AlertDescription className="text-green-700 dark:text-green-300">
                <div className="flex items-center justify-between">
                  <span>
                    已识别为 <strong>{matchedProvider.display_name}</strong>
                    {matchedProvider.recommended_protocol && (
                      <span className="ml-2 text-xs">（推荐：{matchedProvider.recommended_protocol.toUpperCase()}）</span>
                    )}
                  </span>
                  <Button
                    type="button"
                    variant="ghost"
                    size="sm"
                    className="h-6 px-2 text-xs text-green-700 hover:text-green-800 hover:bg-green-100"
                    onClick={() => {
                      onClearProvider();
                      onFormChange({ provider: '', protocol: 'imap', auth_type: 'password' });
                    }}
                  >
                    重新选择
                  </Button>
                </div>
              </AlertDescription>
            </Alert>
          )}

          {/* 未匹配时显示警告提示 - 只有当邮箱格式完整时才显示 */}
          {!matchedProvider && formData.email.includes('@') && formData.email.split('@')[1]?.includes('.') && (
            <Alert className="border-amber-200 bg-amber-50 dark:border-amber-800 dark:bg-amber-950">
              <AlertCircle className="h-4 w-4 text-amber-600" />
              <AlertDescription className="text-amber-700 dark:text-amber-300">
                <div className="space-y-2">
                  <p>未识别的邮箱域名 <strong>{formData.email.split('@')[1]}</strong>，请手动选择提供商或前往「提供商管理」添加新的提供商配置。</p>
                  <Select value={formData.provider} onValueChange={onProviderChange}>
                    <SelectTrigger className="bg-white dark:bg-gray-900">
                      <SelectValue placeholder="选择邮箱提供商..." />
                    </SelectTrigger>
                    <SelectContent>
                      {providers.map((p) => (
                        <SelectItem key={p.name} value={p.name}>{p.display_name}</SelectItem>
                      ))}
                    </SelectContent>
                  </Select>
                </div>
              </AlertDescription>
            </Alert>
          )}

          {/* 协议选择 */}
          {matchedProvider && getAvailableProtocols().length > 1 && (
            <div className="space-y-2">
              <Label>认证方式 / 协议</Label>
              <Select value={formData.protocol} onValueChange={onProtocolChange}>
                <SelectTrigger>
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  {getAvailableProtocols().map((p) => (
                    <SelectItem key={p.value} value={p.value}>
                      {p.label} {p.recommended && <Badge variant="secondary" className="ml-2">推荐</Badge>}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </div>
          )}

          {/* OAuth2 认证 */}
          {formData.protocol === 'oauth2' && matchedProvider && (
            <div className="space-y-4 p-4 rounded-lg border bg-muted/20">
              <div className="flex items-center gap-2 text-sm font-medium">
                <Server className="h-4 w-4" />
                OAuth2 安全认证
              </div>
              <OAuth2ClientSelector
                providerId={matchedProvider.id}
                value={selectedOAuth2ClientId}
                onChange={onOAuth2ClientChange}
              />
              <OAuth2AuthButton
                provider={formData.provider === 'gmail' ? 'google' : 'microsoft'}
                email={formData.email}
                selectedClientId={selectedOAuth2ClientId}
                groupId={formData.group_id ?? undefined}
                onSuccess={onOAuth2AuthSuccess}
                onError={(err) => toast.error(err)}
              />
            </div>
          )}

          {/* 密码/授权码输入 */}
          {formData.protocol !== 'oauth2' && matchedProvider && (
            <div className="space-y-2">
              <Label htmlFor="password">
                {['qq', '163', '126', '139', '189'].includes(formData.provider) ? '授权码 / 应用专用密码 *' : '密码 *'}
              </Label>
              <Input
                id="password"
                type="password"
                placeholder={['qq', '163', '126', '139', '189'].includes(formData.provider) ? '请输入授权码' : '请输入密码'}
                value={formData.password}
                onChange={(e) => onFormChange({ password: e.target.value })}
                required
              />
              {['qq', '163', '126', '139', '189'].includes(formData.provider) && (
                <p className="text-xs text-muted-foreground">
                  国内邮箱通常需要使用授权码而非登录密码，请在邮箱设置中开启 IMAP/POP3 服务并获取授权码
                </p>
              )}
            </div>
          )}

          {/* 发件功能开关（新建模式） - 只要有匹配的提供商且非 OAuth2 就显示 */}
          {matchedProvider && formData.protocol !== 'oauth2' && (
            <div className="flex items-center justify-between p-3 rounded-lg border bg-muted/20">
              <div className="flex items-center gap-2">
                <Send className="h-4 w-4 text-muted-foreground" />
                <div>
                  <Label htmlFor="smtp_enabled" className="cursor-pointer">启用发件功能</Label>
                  <p className="text-xs text-muted-foreground">
                    {hasSmtpSupport() 
                      ? '使用 SMTP 发送邮件（服务器配置继承自提供商）' 
                      : '该提供商暂未配置 SMTP 服务器'}
                  </p>
                </div>
              </div>
              <Switch
                id="smtp_enabled"
                checked={formData.smtp_enabled}
                onCheckedChange={(checked) => onFormChange({ smtp_enabled: checked })}
                disabled={!hasSmtpSupport()}
              />
            </div>
          )}

          {/* 分组选择 */}
          <div className="space-y-2">
            <Label>分组</Label>
            <GroupSelector
              value={formData.group_id}
              onChange={(id) => onFormChange({ group_id: id })}
            />
          </div>

          {/* 同步设置折叠区 */}
          <Collapsible>
            <CollapsibleTrigger className="flex items-center gap-2 text-sm font-medium w-full py-2 px-3 rounded-lg border bg-muted/30 hover:bg-muted/50 transition-colors">
              <ChevronRight className="h-4 w-4 transition-transform duration-200 [[data-state=open]>&]:rotate-90" />
              高级设置
              <span className="ml-auto text-xs text-muted-foreground font-normal">同步频率、首次同步天数等</span>
            </CollapsibleTrigger>
            <CollapsibleContent className="space-y-4 pt-3 pl-3 border-l-2 border-muted ml-2 mt-2">
              {/* 同步开关 */}
              <div className="flex items-center justify-between">
                <Label htmlFor="sync_enabled">启用自动同步</Label>
                <Switch
                  id="sync_enabled"
                  checked={formData.sync_enabled}
                  onCheckedChange={(checked) => onFormChange({ sync_enabled: checked })}
                />
              </div>
              {/* 同步频率 */}
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
              {/* 首次同步天数 */}
              <div className="space-y-2">
                <Label>首次同步天数</Label>
                <Select
                  value={String(formData.first_sync_days)}
                  onValueChange={(v) => onFormChange({ first_sync_days: Number(v) })}
                >
                  <SelectTrigger><SelectValue /></SelectTrigger>
                  <SelectContent>
                    {[7, 14, 30, 60, 90, 180, 365, 0].map((d) => (
                      <SelectItem key={d} value={String(d)}>{d === 0 ? '全部邮件' : `最近 ${d} 天`}</SelectItem>
                    ))}
                  </SelectContent>
                </Select>
              </div>
              {/* 删除策略 */}
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
    </>
  );
}
