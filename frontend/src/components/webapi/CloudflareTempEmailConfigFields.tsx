// Cloudflare Temp Email 配置字段（WebAPI 认证配置 · 纯展示）
// 注意：与 CloudflareTempEmailAuthFields（WebAPI 账户编辑对话框）是两个组件。
// 本组件用于新建/配置流程，含同步模式选择与访问模式切换；
// 自动获取设置的 fetch 逻辑留在 WebAPIAuthConfig 容器中，通过 props 传入。
import { Input } from '../ui/input';
import { Label } from '../ui/label';
import { Button } from '../ui/button';
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '../ui/select';
import { RadioGroup, RadioGroupItem } from '../ui/radio-group';
import { Loader2, Wand2, RefreshCw, Info } from 'lucide-react';
import { toast } from 'sonner';
import type {
  WebAPIAccessMode,
  CloudflareTempEmailAuthData,
} from '../../types/webapi';

interface CloudflareTempEmailConfigFieldsProps {
  /** Cloudflare Temp Email 认证数据 */
  data: CloudflareTempEmailAuthData;
  /** 局部更新认证字段（容器侧合并进 authData） */
  onChange: (field: string, value: any) => void;
  /** 字段校验错误 */
  errors: Record<string, string>;
  /** 自动获取邮箱地址（fetch 与状态留在容器组件） */
  onFetchSettings: () => void;
  /** 是否正在获取设置 */
  isFetchingSettings: boolean;
  /** 自动获取失败信息 */
  fetchError: string | null;
}

/**
 * Cloudflare Temp Email 配置字段（纯展示组件）：
 * 按同步模式（轮询/Webhook）与访问模式（Single/Admin）渲染配置表单，不自行请求数据。
 */
export const CloudflareTempEmailConfigFields = ({
  data,
  onChange,
  errors,
  onFetchSettings,
  isFetchingSettings,
  fetchError,
}: CloudflareTempEmailConfigFieldsProps) => {
  // 当前同步模式
  const syncMode = data.sync_mode || 'polling';

  return (
    <div className="space-y-4">
      {/* 同步模式选择 - 放在最前面 */}
      <div className="rounded-lg border p-4 space-y-4">
        <div className="flex items-center gap-2">
          <div className="text-sm font-medium">同步模式</div>
          <Info className="h-4 w-4 text-muted-foreground" />
        </div>
        
        <RadioGroup
          value={syncMode}
          onValueChange={(value) => onChange('sync_mode', value as 'polling' | 'webhook')}
          className="space-y-3"
        >
          <div className="flex items-start space-x-3">
            <RadioGroupItem value="polling" id="sync_polling" className="mt-1" />
            <div className="space-y-1">
              <Label htmlFor="sync_polling" className="font-normal cursor-pointer">
                轮询模式（默认）
              </Label>
              <p className="text-xs text-muted-foreground">
                FusionMail 定时从邮件服务器拉取新邮件，需要配置 API 地址和认证信息
              </p>
            </div>
          </div>
          
          <div className="flex items-start space-x-3">
            <RadioGroupItem value="webhook" id="sync_webhook" className="mt-1" />
            <div className="space-y-1">
              <Label htmlFor="sync_webhook" className="font-normal cursor-pointer">
                Webhook 模式
              </Label>
              <p className="text-xs text-muted-foreground">
                邮件服务器主动推送新邮件到 FusionMail，实时性更好，配置更简单
              </p>
            </div>
          </div>
        </RadioGroup>
      </div>

      {/* ========== Webhook 模式配置（极简版） ========== */}
      {syncMode === 'webhook' && (
        <div className="space-y-4">
          {/* Webhook Secret */}
          <div className="space-y-2">
            <div className="flex items-center justify-between">
              <Label htmlFor="webhook_secret">Webhook Secret *</Label>
              <Button
                type="button"
                variant="outline"
                size="sm"
                onClick={() => {
                  // 生成随机 Secret（32 字符）
                  const chars = 'ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789';
                  let secret = '';
                  for (let i = 0; i < 32; i++) {
                    secret += chars.charAt(Math.floor(Math.random() * chars.length));
                  }
                  onChange('webhook_secret', secret);
                  toast.success('已生成随机 Secret');
                }}
                className="h-7 text-xs"
              >
                <RefreshCw className="h-3 w-3 mr-1" />
                生成随机
              </Button>
            </div>
            <Input
              id="webhook_secret"
              type="text"
              placeholder="输入或生成 Webhook Secret"
              value={data.webhook_secret || ''}
              onChange={(e) => onChange('webhook_secret', e.target.value)}
              className={errors.webhook_secret ? 'border-red-500' : ''}
            />
            {errors.webhook_secret && <p className="text-sm text-red-500">{errors.webhook_secret}</p>}
            <p className="text-xs text-muted-foreground">
              用于验证 Webhook 推送请求的安全密钥
            </p>
          </div>

          {/* 配置说明 */}
          <div className="rounded-md bg-muted p-3 space-y-2">
            <p className="text-xs font-medium">配置说明</p>
            <p className="text-xs text-muted-foreground">
              保存账户后，将生成 Webhook URL。请在 Cloudflare Temp Email 的 Webhook 设置中配置：
            </p>
            <ul className="text-xs text-muted-foreground list-disc list-inside space-y-1">
              <li>Webhook URL：保存后在账户详情中查看</li>
              <li>Secret：使用上方配置的 Webhook Secret</li>
            </ul>
          </div>
        </div>
      )}

      {/* ========== 轮询模式配置（完整版） ========== */}
      {syncMode === 'polling' && (
        <div className="space-y-4">
          {/* API 地址 */}
          <div className="space-y-2">
            <Label htmlFor="base_url">API 地址 *</Label>
            <Input
              id="base_url"
              type="url"
              placeholder="https://your-temp-email-domain.com"
              value={data.base_url || ''}
              onChange={(e) => onChange('base_url', e.target.value)}
              className={errors.base_url ? 'border-red-500' : ''}
            />
            {errors.base_url && <p className="text-sm text-red-500">{errors.base_url}</p>}
          </div>

          {/* 访问模式 */}
          <div className="space-y-2">
            <Label htmlFor="access_mode">访问模式 *</Label>
            <Select
              value={data.access_mode || 'single'}
              onValueChange={(value) => onChange('access_mode', value as WebAPIAccessMode)}
            >
              <SelectTrigger>
                <SelectValue placeholder="选择访问模式" />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="single">Single (单邮箱模式)</SelectItem>
                <SelectItem value="admin">Admin (管理员模式)</SelectItem>
              </SelectContent>
            </Select>
            <p className="text-xs text-muted-foreground">
              Single 模式：使用 JWT Token 访问单个邮箱；Admin 模式：使用管理员密码访问所有邮箱
            </p>
          </div>

          {/* Single 模式配置 */}
          {data.access_mode === 'single' && (
            <>
              <div className="rounded-lg border p-4 space-y-4">
                <div className="text-sm font-medium">认证方式（二选一）</div>
                
                {/* 方式一：JWT Token（永不过期） */}
                <div className="space-y-3">
                  <div className="text-xs text-muted-foreground">方式一：JWT Token（永不过期，推荐）</div>
                  <div className="space-y-2">
                    <Label htmlFor="jwt_token">JWT Token</Label>
                    <Input
                      id="jwt_token"
                      type="password"
                      placeholder="直接登录获取的 JWT Token"
                      value={data.jwt_token || ''}
                      onChange={(e) => onChange('jwt_token', e.target.value)}
                      className={errors.jwt_token ? 'border-red-500' : ''}
                    />
                    {errors.jwt_token && <p className="text-sm text-red-500">{errors.jwt_token}</p>}
                    <p className="text-xs text-muted-foreground">
                      从浏览器开发者工具获取的 authorization token，永不过期
                    </p>
                  </div>
                </div>

                <div className="relative">
                  <div className="absolute inset-0 flex items-center">
                    <span className="w-full border-t" />
                  </div>
                  <div className="relative flex justify-center text-xs uppercase">
                    <span className="bg-background px-2 text-muted-foreground">或</span>
                  </div>
                </div>

                {/* 方式二：User Token（第三方授权登录） */}
                <div className="space-y-3">
                  <div className="text-xs text-muted-foreground">方式二：User Token（第三方授权登录，30天过期）</div>
                  <div className="space-y-2">
                    <Label htmlFor="user_token">User Token</Label>
                    <Input
                      id="user_token"
                      type="password"
                      placeholder="第三方授权登录获取的 User Token"
                      value={data.user_token || ''}
                      onChange={(e) => onChange('user_token', e.target.value)}
                      className={errors.user_token ? 'border-red-500' : ''}
                    />
                    {errors.user_token && <p className="text-sm text-red-500">{errors.user_token}</p>}
                    <p className="text-xs text-muted-foreground">
                      通过 Linux.do 等第三方授权登录获取的 token，有效期 30 天，系统会在过期前 7 天自动刷新
                    </p>
                  </div>
                </div>
              </div>

              <div className="space-y-2">
                <div className="flex items-center justify-between">
                  <Label htmlFor="email">目标邮箱地址</Label>
                  <Button
                    type="button"
                    variant="outline"
                    size="sm"
                    onClick={onFetchSettings}
                    disabled={isFetchingSettings || !data.base_url || !data.jwt_token}
                    className="h-7 text-xs"
                  >
                    {isFetchingSettings ? (
                      <>
                        <Loader2 className="h-3 w-3 mr-1 animate-spin" />
                        获取中...
                      </>
                    ) : (
                      <>
                        <Wand2 className="h-3 w-3 mr-1" />
                        自动获取
                      </>
                    )}
                  </Button>
                </div>
                <Input
                  id="email"
                  type="email"
                  placeholder="user@example.com（可自动获取）"
                  value={data.email || ''}
                  onChange={(e) => onChange('email', e.target.value)}
                  className={errors.email ? 'border-red-500' : ''}
                />
                {fetchError && <p className="text-sm text-red-500">{fetchError}</p>}
                {errors.email && <p className="text-sm text-red-500">{errors.email}</p>}
                <p className="text-xs text-muted-foreground">
                  填写 API 地址和 JWT Token 后，点击"自动获取"可自动填充邮箱地址
                </p>
              </div>
            </>
          )}

          {/* Admin 模式配置 */}
          {data.access_mode === 'admin' && (
            <>
              <div className="space-y-2">
                <Label htmlFor="admin_password">管理员密码 *</Label>
                <Input
                  id="admin_password"
                  type="password"
                  placeholder="输入管理员密码"
                  value={data.admin_password || ''}
                  onChange={(e) => onChange('admin_password', e.target.value)}
                  className={errors.admin_password ? 'border-red-500' : ''}
                />
                {errors.admin_password && <p className="text-sm text-red-500">{errors.admin_password}</p>}
              </div>
              <div className="space-y-2">
                <Label htmlFor="domains">过滤域名</Label>
                <Input
                  id="domains"
                  type="text"
                  placeholder="example.com, test.org（逗号分隔，可选）"
                  value={data.domains || ''}
                  onChange={(e) => onChange('domains', e.target.value)}
                />
                <p className="text-xs text-muted-foreground">
                  只同步指定域名的邮件，多个域名用逗号分隔。留空则同步所有邮件。
                </p>
              </div>
            </>
          )}
        </div>
      )}
    </div>
  );
};
