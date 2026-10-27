// 自定义 WebAPI 认证字段（WebAPI 账户编辑对话框 · 纯展示）
import { Input } from '../ui/input';
import { Label } from '../ui/label';
import { Badge } from '../ui/badge';
import {
  Collapsible,
  CollapsibleContent,
  CollapsibleTrigger,
} from '../ui/collapsible';
import { Key, ChevronRight } from 'lucide-react';

// 自定义 WebAPI 认证数据接口
export interface CustomWebAPIAuthData {
  service_name: string;
  base_url: string;
  endpoint: string;
  auth_type: string;
  auth_token?: string;
  api_key?: string;
  username?: string;
  password?: string;
  data_path: string;
  target_email?: string;
}

interface CustomWebAPIAuthFieldsProps {
  /** 自定义 WebAPI 认证数据 */
  data: CustomWebAPIAuthData;
  /** 局部更新认证字段（对话框侧合并进 authData） */
  onChange: (field: string, value: any) => void;
}

/**
 * 自定义 WebAPI 认证字段（纯展示组件）：基本信息 + 按认证类型渲染凭据字段，不自己请求数据。
 */
export const CustomWebAPIAuthFields = ({
  data,
  onChange,
}: CustomWebAPIAuthFieldsProps) => {
  return (
    <div className="space-y-4">
      {/* 可编辑的基本信息 */}
      <div className="space-y-3">
        <div className="space-y-2">
          <Label htmlFor="custom_service_name">服务名称</Label>
          <Input
            id="custom_service_name"
            placeholder="自定义服务名称"
            value={data.service_name || ''}
            onChange={(e) => onChange('service_name', e.target.value)}
          />
        </div>
        <div className="space-y-2">
          <Label htmlFor="custom_base_url">API 地址</Label>
          <Input
            id="custom_base_url"
            placeholder="https://api.example.com"
            value={data.base_url || ''}
            onChange={(e) => onChange('base_url', e.target.value)}
          />
        </div>
        <div className="space-y-2">
          <Label htmlFor="custom_endpoint">端点</Label>
          <Input
            id="custom_endpoint"
            placeholder="/api/emails"
            value={data.endpoint || ''}
            onChange={(e) => onChange('endpoint', e.target.value)}
          />
        </div>
      </div>
      
      {/* 只读信息 */}
      <div className="p-3 rounded-lg bg-muted/30 space-y-2 text-sm">
        <div className="flex justify-between">
          <span className="text-muted-foreground">认证类型</span>
          <Badge variant="outline">{data.auth_type || '-'}</Badge>
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
          {data.auth_type === 'bearer_token' && (
            <div className="space-y-2">
              <Label htmlFor="auth_token">Bearer Token</Label>
              <Input
                id="auth_token"
                type="password"
                placeholder="留空则不修改"
                value={data.auth_token || ''}
                onChange={(e) => onChange('auth_token', e.target.value)}
              />
            </div>
          )}
          {data.auth_type === 'api_key' && (
            <div className="space-y-2">
              <Label htmlFor="api_key">API Key</Label>
              <Input
                id="api_key"
                type="password"
                placeholder="留空则不修改"
                value={data.api_key || ''}
                onChange={(e) => onChange('api_key', e.target.value)}
              />
            </div>
          )}
          {data.auth_type === 'basic_auth' && (
            <>
              <div className="space-y-2">
                <Label htmlFor="username">用户名</Label>
                <Input
                  id="username"
                  placeholder="留空则不修改"
                  value={data.username || ''}
                  onChange={(e) => onChange('username', e.target.value)}
                />
              </div>
              <div className="space-y-2">
                <Label htmlFor="password">密码</Label>
                <Input
                  id="password"
                  type="password"
                  placeholder="留空则不修改"
                  value={data.password || ''}
                  onChange={(e) => onChange('password', e.target.value)}
                />
              </div>
            </>
          )}
        </CollapsibleContent>
      </Collapsible>
    </div>
  );
};
