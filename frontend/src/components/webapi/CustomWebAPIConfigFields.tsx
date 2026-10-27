// 自定义 Web API 配置字段（WebAPI 认证配置 · 纯展示）
// 注意：与 CustomWebAPIAuthFields（WebAPI 账户编辑对话框）是两个组件，本组件用于新建/配置流程。
import { Input } from '../ui/input';
import { Label } from '../ui/label';
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '../ui/select';
import type { WebAPIAuthType, CustomWebAPIAuthData } from '../../types/webapi';

interface CustomWebAPIConfigFieldsProps {
  /** 自定义服务认证数据 */
  data: CustomWebAPIAuthData;
  /** 局部更新认证字段（容器侧合并进 authData） */
  onChange: (field: string, value: any) => void;
  /** 字段校验错误 */
  errors: Record<string, string>;
}

/**
 * 自定义 Web API 配置字段（纯展示组件）：基础配置、认证配置（按认证类型切换字段）、
 * 响应解析与字段映射表单，不自行请求数据。
 */
export const CustomWebAPIConfigFields = ({
  data,
  onChange,
  errors,
}: CustomWebAPIConfigFieldsProps) => {
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
            onChange={(e) => onChange('service_name', e.target.value)}
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
            onChange={(e) => onChange('target_email', e.target.value)}
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
          onChange={(e) => onChange('base_url', e.target.value)}
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
            onChange={(e) => onChange('endpoint', e.target.value)}
            className={errors.endpoint ? 'border-red-500' : ''}
          />
          {errors.endpoint && <p className="text-sm text-red-500">{errors.endpoint}</p>}
        </div>
        <div className="space-y-2">
          <Label htmlFor="method">HTTP 方法</Label>
          <Select
            value={data.method || 'GET'}
            onValueChange={(value) => onChange('method', value)}
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
          onValueChange={(value) => onChange('auth_type', value as WebAPIAuthType)}
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
            onChange={(e) => onChange('auth_token', e.target.value)}
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
              onChange={(e) => onChange('api_key', e.target.value)}
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
              onChange={(e) => onChange('api_key_header', e.target.value)}
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
              onChange={(e) => onChange('username', e.target.value)}
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
              onChange={(e) => onChange('password', e.target.value)}
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
          onChange={(e) => onChange('data_path', e.target.value)}
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
            onChange={(e) => onChange('field_mapping', { ...data.field_mapping, id: e.target.value })}
          />
          <Input
            placeholder="主题字段"
            value={data.field_mapping?.subject || ''}
            onChange={(e) => onChange('field_mapping', { ...data.field_mapping, subject: e.target.value })}
          />
          <Input
            placeholder="发件人字段"
            value={data.field_mapping?.from || ''}
            onChange={(e) => onChange('field_mapping', { ...data.field_mapping, from: e.target.value })}
          />
          <Input
            placeholder="日期字段"
            value={data.field_mapping?.date || ''}
            onChange={(e) => onChange('field_mapping', { ...data.field_mapping, date: e.target.value })}
          />
        </div>
        <p className="text-xs text-muted-foreground">
          映射 API 响应字段到邮件属性，支持嵌套路径如 "mail.subject"
        </p>
      </div>
    </div>
  );
};
