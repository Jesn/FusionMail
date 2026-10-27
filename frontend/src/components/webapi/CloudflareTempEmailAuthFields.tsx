// Cloudflare Temp Email 认证字段（WebAPI 账户编辑对话框 · 纯展示）
import { Button } from '../ui/button';
import { Input } from '../ui/input';
import { Label } from '../ui/label';
import { Badge } from '../ui/badge';
import {
  Collapsible,
  CollapsibleContent,
  CollapsibleTrigger,
} from '../ui/collapsible';
import { Copy, Key, RefreshCw, Webhook, ChevronRight } from 'lucide-react';
import { toast } from 'sonner';

// 认证数据接口
export interface CloudflareTempEmailAuthData {
  base_url: string;
  access_mode: 'single' | 'admin';
  jwt_token?: string;         // JWT Token（直接登录方式，永不过期）
  user_token?: string;        // User Token（第三方授权登录，30天过期，支持自动刷新）
  email?: string;
  admin_password?: string;
  domains?: string;  // 过滤域名列表（逗号分隔）
  sync_mode?: 'polling' | 'webhook';  // 同步模式
  webhook_secret?: string;    // Webhook Secret
}

interface CloudflareTempEmailAuthFieldsProps {
  /** Cloudflare Temp Email 认证数据 */
  data: CloudflareTempEmailAuthData;
  /** 局部更新认证字段（对话框侧合并进 authData） */
  onChange: (field: string, value: any) => void;
}

/**
 * Cloudflare Temp Email 认证字段（纯展示组件）：
 * 按同步模式渲染 Webhook 极简配置或轮询完整配置，不自己请求数据。
 */
export const CloudflareTempEmailAuthFields = ({
  data,
  onChange,
}: CloudflareTempEmailAuthFieldsProps) => {
  const isWebhookMode = data.sync_mode === 'webhook';
  
  // Webhook 模式：极简配置
  if (isWebhookMode) {
    return (
      <div className="space-y-4">
        {/* 同步模式显示 */}
        <div className="p-3 rounded-lg bg-muted/30 space-y-2 text-sm">
          <div className="flex justify-between items-center">
            <span className="text-muted-foreground">同步模式</span>
            <Badge variant="outline" className="bg-green-50 text-green-700 border-green-200">
              <Webhook className="h-3 w-3 mr-1" />
              Webhook 推送
            </Badge>
          </div>
          <p className="text-xs text-muted-foreground">
            邮件服务器主动推送新邮件，系统根据收件人地址自动创建账户
          </p>
        </div>

        {/* Webhook URL */}
        <div className="space-y-2">
          <Label>Webhook URL</Label>
          <div className="flex gap-2">
            <Input
              readOnly
              value={`${window.location.origin}/api/v1/webhook/receive/cloudflare_temp_email`}
              className="font-mono text-xs"
            />
            <Button
              type="button"
              variant="outline"
              size="icon"
              onClick={() => {
                navigator.clipboard.writeText(
                  `${window.location.origin}/api/v1/webhook/receive/cloudflare_temp_email`
                );
                toast.success('已复制 Webhook URL');
              }}
            >
              <Copy className="h-4 w-4" />
            </Button>
          </div>
          <p className="text-xs text-muted-foreground">
            在 Cloudflare Temp Email 的 Webhook 设置中配置此 URL
          </p>
        </div>

        {/* Webhook Secret */}
        <div className="space-y-2">
          <div className="flex items-center justify-between">
            <Label htmlFor="edit_webhook_secret">Webhook Secret</Label>
            <Button
              type="button"
              variant="outline"
              size="sm"
              onClick={() => {
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
          <div className="flex gap-2">
            <Input
              id="edit_webhook_secret"
              type="text"
              placeholder="输入或生成 Webhook Secret"
              value={data.webhook_secret || ''}
              onChange={(e) => onChange('webhook_secret', e.target.value)}
            />
            <Button
              type="button"
              variant="outline"
              size="icon"
              onClick={() => {
                if (data.webhook_secret) {
                  navigator.clipboard.writeText(data.webhook_secret);
                  toast.success('已复制 Secret');
                }
              }}
              disabled={!data.webhook_secret}
            >
              <Copy className="h-4 w-4" />
            </Button>
          </div>
          <p className="text-xs text-muted-foreground">
            此 Secret 需要与 Cloudflare Temp Email 中配置的一致
          </p>
        </div>

      </div>
    );
  }
  
  // 轮询模式：完整配置
  return (
    <div className="space-y-4">
      {/* 基本信息 */}
      <div className="space-y-3">
        {/* API 地址（可编辑） */}
        <div className="space-y-2">
          <Label htmlFor="cf_base_url">API 地址</Label>
          <Input
            id="cf_base_url"
            placeholder="https://api.example.com"
            value={data.base_url || ''}
            onChange={(e) => onChange('base_url', e.target.value)}
          />
          <p className="text-xs text-muted-foreground">
            Cloudflare Temp Email 的 API 地址（注意：不是前端网站地址）
          </p>
        </div>
        
        {/* 只读信息 */}
        <div className="p-3 rounded-lg bg-muted/30 space-y-2 text-sm">
          <div className="flex justify-between">
            <span className="text-muted-foreground">访问模式</span>
            <Badge variant="outline">
              {data.access_mode === 'admin' ? 'Admin 模式' : 'Single 模式'}
            </Badge>
          </div>
          {data.access_mode === 'single' && data.email && (
            <div className="flex justify-between">
              <span className="text-muted-foreground">目标邮箱</span>
              <span>{data.email}</span>
            </div>
          )}
          {data.access_mode === 'single' && (
            <div className="flex justify-between">
              <span className="text-muted-foreground">认证方式</span>
              <Badge variant="outline">
                {data.user_token ? 'User Token（自动刷新）' : 'JWT Token（永不过期）'}
              </Badge>
            </div>
          )}
          {data.access_mode === 'admin' && data.domains && (
            <div className="flex justify-between">
              <span className="text-muted-foreground">过滤域名</span>
              <span>{data.domains}</span>
            </div>
          )}
        </div>
      </div>

      {/* 可编辑的认证信息 */}
      <Collapsible>
        <CollapsibleTrigger className="flex items-center gap-2 text-sm font-medium w-full py-2">
          <Key className="h-4 w-4" />
          更新认证信息
          <ChevronRight className="h-4 w-4 ml-auto transition-transform ui-open:rotate-90" />
        </CollapsibleTrigger>
        <CollapsibleContent className="space-y-3 pt-2">
          {data.access_mode === 'single' ? (
            <div className="space-y-4">
              <div className="space-y-2">
                <Label htmlFor="jwt_token">JWT Token（永不过期）</Label>
                <Input
                  id="jwt_token"
                  type="password"
                  placeholder="留空则不修改"
                  value={data.jwt_token || ''}
                  onChange={(e) => onChange('jwt_token', e.target.value)}
                />
                <p className="text-xs text-muted-foreground">
                  直接登录获取的 JWT Token，永不过期
                </p>
              </div>
              <div className="relative">
                <div className="absolute inset-0 flex items-center">
                  <span className="w-full border-t" />
                </div>
                <div className="relative flex justify-center text-xs uppercase">
                  <span className="bg-background px-2 text-muted-foreground">或</span>
                </div>
              </div>
              <div className="space-y-2">
                <Label htmlFor="user_token">User Token（30天过期，自动刷新）</Label>
                <Input
                  id="user_token"
                  type="password"
                  placeholder="留空则不修改"
                  value={data.user_token || ''}
                  onChange={(e) => onChange('user_token', e.target.value)}
                />
                <p className="text-xs text-muted-foreground">
                  第三方授权登录获取的 Token，系统会在过期前 7 天自动刷新
                </p>
              </div>
            </div>
          ) : (
            <div className="space-y-2">
              <Label htmlFor="admin_password">管理员密码</Label>
              <Input
                id="admin_password"
                type="password"
                placeholder="留空则不修改"
                value={data.admin_password || ''}
                onChange={(e) => onChange('admin_password', e.target.value)}
              />
              <p className="text-xs text-muted-foreground">
                输入新的管理员密码以更新认证，留空保持不变
              </p>
            </div>
          )}
        </CollapsibleContent>
      </Collapsible>

    </div>
  );
};
