// WebAPI 认证配置组件
import React, { useState } from 'react';
import { Input } from '../ui/input';
import { Label } from '../ui/label';
import { Button } from '../ui/button';
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '../ui/select';
import { Loader2, Wand2 } from 'lucide-react';
import { webapiService } from '../../services/webapiService';
import type {
  WebAPIServiceType,
  WebAPIAccessMode,
  WebAPIAuthType,
  CloudflareTempEmailAuthData,
  CloudMailAuthData,
  CustomWebAPIAuthData,
} from '../../types/webapi';

interface WebAPIAuthConfigProps {
  serviceType: WebAPIServiceType;
  authData: any;
  onChange: (data: any) => void;
  errors?: Record<string, string>;
}

/**
 * WebAPI 认证配置组件
 * 根据服务类型动态渲染不同的配置表单
 */
export const WebAPIAuthConfig: React.FC<WebAPIAuthConfigProps> = ({
  serviceType,
  authData,
  onChange,
  errors = {},
}) => {
  // 更新字段值
  const updateField = (field: string, value: any) => {
    onChange({ ...authData, [field]: value });
  };

  // 渲染 Cloudflare Temp Email 配置
  const renderCloudflareTempEmailConfig = () => {
    const data = authData as CloudflareTempEmailAuthData;
    const [isFetchingSettings, setIsFetchingSettings] = useState(false);
    const [fetchError, setFetchError] = useState<string | null>(null);

    // 自动获取设置信息
    const handleFetchSettings = async () => {
      if (!data.base_url || !data.jwt_token) {
        setFetchError('请先填写 API 地址和 JWT Token');
        return;
      }

      setIsFetchingSettings(true);
      setFetchError(null);

      try {
        const settings = await webapiService.fetchCloudflareTempEmailSettings(
          data.base_url,
          data.jwt_token
        );
        // 自动填充邮箱地址
        if (settings.email) {
          onChange({ ...authData, email: settings.email });
        }
      } catch (error) {
        setFetchError(error instanceof Error ? error.message : '获取设置失败');
      } finally {
        setIsFetchingSettings(false);
      }
    };

    return (
      <div className="space-y-4">
        {/* 基础配置 */}
        <div className="space-y-2">
          <Label htmlFor="base_url">API 地址 *</Label>
          <Input
            id="base_url"
            type="url"
            placeholder="https://your-temp-email-domain.com"
            value={data.base_url || ''}
            onChange={(e) => updateField('base_url', e.target.value)}
            className={errors.base_url ? 'border-red-500' : ''}
          />
          {errors.base_url && <p className="text-sm text-red-500">{errors.base_url}</p>}
        </div>

        {/* 访问模式 */}
        <div className="space-y-2">
          <Label htmlFor="access_mode">访问模式 *</Label>
          <Select
            value={data.access_mode || 'single'}
            onValueChange={(value) => updateField('access_mode', value as WebAPIAccessMode)}
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
                    onChange={(e) => updateField('jwt_token', e.target.value)}
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
                    onChange={(e) => updateField('user_token', e.target.value)}
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
                  onClick={handleFetchSettings}
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
                onChange={(e) => updateField('email', e.target.value)}
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
                onChange={(e) => updateField('admin_password', e.target.value)}
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
                onChange={(e) => updateField('domains', e.target.value)}
              />
              <p className="text-xs text-muted-foreground">
                只同步指定域名的邮件，多个域名用逗号分隔。留空则同步所有邮件。
              </p>
            </div>
          </>
        )}
      </div>
    );
  };

  // 渲染 Cloud Mail 配置
  const renderCloudMailConfig = () => {
    const data = authData as CloudMailAuthData;
    return (
      <div className="space-y-4">
        <div className="space-y-2">
          <Label htmlFor="base_url">API 地址 *</Label>
          <Input
            id="base_url"
            type="url"
            placeholder="https://your-cloud-mail-domain.com"
            value={data.base_url || ''}
            onChange={(e) => updateField('base_url', e.target.value)}
            className={errors.base_url ? 'border-red-500' : ''}
          />
          {errors.base_url && <p className="text-sm text-red-500">{errors.base_url}</p>}
        </div>

        <div className="rounded-lg border p-4 space-y-4">
          <div className="text-sm font-medium">认证方式（二选一）</div>
          
          <div className="space-y-3">
            <div className="text-xs text-muted-foreground">方式一：邮箱+密码登录（推荐）</div>
            <div className="grid grid-cols-2 gap-4">
              <div className="space-y-2">
                <Label htmlFor="email">登录邮箱</Label>
                <Input
                  id="email"
                  type="email"
                  placeholder="user@example.com"
                  value={data.email || ''}
                  onChange={(e) => updateField('email', e.target.value)}
                  className={errors.email ? 'border-red-500' : ''}
                />
                {errors.email && <p className="text-sm text-red-500">{errors.email}</p>}
              </div>
              <div className="space-y-2">
                <Label htmlFor="password">登录密码</Label>
                <Input
                  id="password"
                  type="password"
                  placeholder="输入密码"
                  value={data.password || ''}
                  onChange={(e) => updateField('password', e.target.value)}
                  className={errors.password ? 'border-red-500' : ''}
                />
                {errors.password && <p className="text-sm text-red-500">{errors.password}</p>}
              </div>
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

          <div className="space-y-3">
            <div className="text-xs text-muted-foreground">方式二：直接输入 JWT Token</div>
            <div className="space-y-2">
              <Label htmlFor="jwt_token">JWT Token</Label>
              <Input
                id="jwt_token"
                type="password"
                placeholder="从浏览器获取的 authorization token"
                value={data.jwt_token || ''}
                onChange={(e) => updateField('jwt_token', e.target.value)}
                className={errors.jwt_token ? 'border-red-500' : ''}
              />
              {errors.jwt_token && <p className="text-sm text-red-500">{errors.jwt_token}</p>}
            </div>
          </div>
        </div>
      </div>
    );
  };

  // 渲染自定义服务配置
  const renderCustomConfig = () => {
    const data = authData as CustomWebAPIAuthData;
    return (
      <div className="space-y-4">
        {/* 基础配置 */}
        <div className="grid grid-cols-2 gap-4">
          <div className="space-y-2">
            <Label htmlFor="service_name">服务名称 *</Label>
            <Input
              id="service_name"
              placeholder="我的邮件服务"
              value={data.service_name || ''}
              onChange={(e) => updateField('service_name', e.target.value)}
              className={errors.service_name ? 'border-red-500' : ''}
            />
            {errors.service_name && <p className="text-sm text-red-500">{errors.service_name}</p>}
          </div>
          <div className="space-y-2">
            <Label htmlFor="target_email">目标邮箱</Label>
            <Input
              id="target_email"
              type="email"
              placeholder="user@example.com (可选)"
              value={data.target_email || ''}
              onChange={(e) => updateField('target_email', e.target.value)}
            />
          </div>
        </div>

        <div className="space-y-2">
          <Label htmlFor="base_url">API 基础地址 *</Label>
          <Input
            id="base_url"
            type="url"
            placeholder="https://api.example.com"
            value={data.base_url || ''}
            onChange={(e) => updateField('base_url', e.target.value)}
            className={errors.base_url ? 'border-red-500' : ''}
          />
          {errors.base_url && <p className="text-sm text-red-500">{errors.base_url}</p>}
        </div>

        <div className="grid grid-cols-2 gap-4">
          <div className="space-y-2">
            <Label htmlFor="endpoint">邮件端点 *</Label>
            <Input
              id="endpoint"
              placeholder="/api/emails"
              value={data.endpoint || ''}
              onChange={(e) => updateField('endpoint', e.target.value)}
              className={errors.endpoint ? 'border-red-500' : ''}
            />
            {errors.endpoint && <p className="text-sm text-red-500">{errors.endpoint}</p>}
          </div>
          <div className="space-y-2">
            <Label htmlFor="method">HTTP 方法</Label>
            <Select
              value={data.method || 'GET'}
              onValueChange={(value) => updateField('method', value)}
            >
              <SelectTrigger>
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="GET">GET</SelectItem>
                <SelectItem value="POST">POST</SelectItem>
              </SelectContent>
            </Select>
          </div>
        </div>

        {/* 认证配置 */}
        <div className="space-y-2">
          <Label htmlFor="auth_type">认证类型 *</Label>
          <Select
            value={data.auth_type || 'bearer_token'}
            onValueChange={(value) => updateField('auth_type', value as WebAPIAuthType)}
          >
            <SelectTrigger>
              <SelectValue />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="bearer_token">Bearer Token</SelectItem>
              <SelectItem value="api_key">API Key</SelectItem>
              <SelectItem value="basic_auth">Basic Auth</SelectItem>
              <SelectItem value="custom_header">自定义 Header</SelectItem>
            </SelectContent>
          </Select>
        </div>

        {/* 根据认证类型显示不同字段 */}
        {data.auth_type === 'bearer_token' && (
          <div className="space-y-2">
            <Label htmlFor="auth_token">Bearer Token *</Label>
            <Input
              id="auth_token"
              type="password"
              placeholder="输入 Token"
              value={data.auth_token || ''}
              onChange={(e) => updateField('auth_token', e.target.value)}
              className={errors.auth_token ? 'border-red-500' : ''}
            />
            {errors.auth_token && <p className="text-sm text-red-500">{errors.auth_token}</p>}
          </div>
        )}

        {data.auth_type === 'api_key' && (
          <div className="grid grid-cols-2 gap-4">
            <div className="space-y-2">
              <Label htmlFor="api_key">API Key *</Label>
              <Input
                id="api_key"
                type="password"
                placeholder="输入 API Key"
                value={data.api_key || ''}
                onChange={(e) => updateField('api_key', e.target.value)}
                className={errors.api_key ? 'border-red-500' : ''}
              />
              {errors.api_key && <p className="text-sm text-red-500">{errors.api_key}</p>}
            </div>
            <div className="space-y-2">
              <Label htmlFor="api_key_header">Header 名称</Label>
              <Input
                id="api_key_header"
                placeholder="X-API-Key"
                value={data.api_key_header || ''}
                onChange={(e) => updateField('api_key_header', e.target.value)}
              />
            </div>
          </div>
        )}

        {data.auth_type === 'basic_auth' && (
          <div className="grid grid-cols-2 gap-4">
            <div className="space-y-2">
              <Label htmlFor="username">用户名 *</Label>
              <Input
                id="username"
                placeholder="用户名"
                value={data.username || ''}
                onChange={(e) => updateField('username', e.target.value)}
                className={errors.username ? 'border-red-500' : ''}
              />
              {errors.username && <p className="text-sm text-red-500">{errors.username}</p>}
            </div>
            <div className="space-y-2">
              <Label htmlFor="password">密码 *</Label>
              <Input
                id="password"
                type="password"
                placeholder="密码"
                value={data.password || ''}
                onChange={(e) => updateField('password', e.target.value)}
                className={errors.password ? 'border-red-500' : ''}
              />
              {errors.password && <p className="text-sm text-red-500">{errors.password}</p>}
            </div>
          </div>
        )}

        {/* 响应解析配置 */}
        <div className="space-y-2">
          <Label htmlFor="data_path">数据路径 *</Label>
          <Input
            id="data_path"
            placeholder="data.list 或 emails"
            value={data.data_path || ''}
            onChange={(e) => updateField('data_path', e.target.value)}
            className={errors.data_path ? 'border-red-500' : ''}
          />
          <p className="text-xs text-muted-foreground">
            JSON 响应中邮件列表的路径，如 "data.list" 或 "emails"
          </p>
          {errors.data_path && <p className="text-sm text-red-500">{errors.data_path}</p>}
        </div>

        {/* 字段映射 */}
        <div className="space-y-2">
          <Label>字段映射</Label>
          <div className="grid grid-cols-2 gap-2">
            <Input
              placeholder="ID 字段 (必填)"
              value={data.field_mapping?.id || ''}
              onChange={(e) => updateField('field_mapping', { ...data.field_mapping, id: e.target.value })}
            />
            <Input
              placeholder="主题字段"
              value={data.field_mapping?.subject || ''}
              onChange={(e) => updateField('field_mapping', { ...data.field_mapping, subject: e.target.value })}
            />
            <Input
              placeholder="发件人字段"
              value={data.field_mapping?.from || ''}
              onChange={(e) => updateField('field_mapping', { ...data.field_mapping, from: e.target.value })}
            />
            <Input
              placeholder="日期字段"
              value={data.field_mapping?.date || ''}
              onChange={(e) => updateField('field_mapping', { ...data.field_mapping, date: e.target.value })}
            />
          </div>
          <p className="text-xs text-muted-foreground">
            映射 API 响应字段到邮件属性，支持嵌套路径如 "mail.subject"
          </p>
        </div>
      </div>
    );
  };

  // 根据服务类型渲染对应配置
  switch (serviceType) {
    case 'cloudflare_temp_email':
      return renderCloudflareTempEmailConfig();
    case 'cloud_mail':
      return renderCloudMailConfig();
    case 'custom':
      return renderCustomConfig();
    default:
      return <div className="text-muted-foreground">请先选择服务类型</div>;
  }
};

export default WebAPIAuthConfig;
