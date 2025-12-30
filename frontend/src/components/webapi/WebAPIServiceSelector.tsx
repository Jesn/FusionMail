// WebAPI 服务选择器组件
import React from 'react';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '../ui/card';
import { Badge } from '../ui/badge';
import { cn } from '../../lib/utils';
import type { WebAPIServiceType, WebAPIServiceTemplate } from '../../types/webapi';
import {
  WebAPIServiceTypeNames,
  WebAPIServiceTypeIcons,
  isPresetService,
} from '../../types/webapi';

// 预置服务模板
const PRESET_SERVICES: WebAPIServiceTemplate[] = [
  {
    service_type: 'cloudflare_temp_email',
    name: 'Cloudflare Temp Email',
    description: '支持 Single 和 Admin 两种模式，可管理临时邮箱',
    icon: '☁️',
    default_config: {
      base_url: '',
      access_mode: 'single',
    },
    config_schema: {
      fields: [
        { name: 'base_url', label: 'API 地址', type: 'text', required: true, placeholder: 'https://your-domain.com' },
        { name: 'access_mode', label: '访问模式', type: 'select', required: true, options: [
          { label: 'Single (单邮箱)', value: 'single' },
          { label: 'Admin (管理员)', value: 'admin' },
        ]},
      ],
      groups: [
        { name: 'single', label: 'Single 模式配置', fields: ['jwt_token', 'email'], condition: { field: 'access_mode', value: 'single' }},
        { name: 'admin', label: 'Admin 模式配置', fields: ['admin_password', 'domain'], condition: { field: 'access_mode', value: 'admin' }},
      ],
    },
  },
  {
    service_type: 'cloud_mail',
    name: 'Cloud Mail',
    description: '支持多账户管理的云邮箱服务',
    icon: '📧',
    default_config: {
      base_url: '',
      jwt_token: '',
      accounts: [],
    },
    config_schema: {
      fields: [
        { name: 'base_url', label: 'API 地址', type: 'text', required: true, placeholder: 'https://your-domain.com' },
        { name: 'jwt_token', label: 'JWT Token', type: 'password', required: true },
        { name: 'accounts', label: '账户列表', type: 'array', required: true },
      ],
      groups: [],
    },
  },
  {
    service_type: 'custom',
    name: '自定义服务',
    description: '配置自定义 Web API 接入任意邮件服务',
    icon: '🔧',
    default_config: {
      service_name: '',
      base_url: '',
      endpoint: '/api/emails',
      method: 'GET',
      auth_type: 'bearer_token',
      data_path: 'data',
      field_mapping: { id: 'id' },
    },
    config_schema: {
      fields: [
        { name: 'service_name', label: '服务名称', type: 'text', required: true },
        { name: 'base_url', label: 'API 地址', type: 'text', required: true },
        { name: 'endpoint', label: '邮件端点', type: 'text', required: true },
        { name: 'method', label: 'HTTP 方法', type: 'select', required: true, options: [
          { label: 'GET', value: 'GET' },
          { label: 'POST', value: 'POST' },
        ]},
        { name: 'auth_type', label: '认证类型', type: 'select', required: true, options: [
          { label: 'Bearer Token', value: 'bearer_token' },
          { label: 'API Key', value: 'api_key' },
          { label: 'Basic Auth', value: 'basic_auth' },
          { label: '自定义 Header', value: 'custom_header' },
        ]},
      ],
      groups: [],
    },
  },
];

interface WebAPIServiceSelectorProps {
  selectedType?: WebAPIServiceType;
  onSelect: (type: WebAPIServiceType, template: WebAPIServiceTemplate) => void;
  disabled?: boolean;
}

/**
 * WebAPI 服务选择器组件
 * 显示预置服务和自定义服务选项
 */
export const WebAPIServiceSelector: React.FC<WebAPIServiceSelectorProps> = ({
  selectedType,
  onSelect,
  disabled = false,
}) => {
  return (
    <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
      {PRESET_SERVICES.map((service) => (
        <Card
          key={service.service_type}
          className={cn(
            'cursor-pointer transition-all hover:shadow-md',
            selectedType === service.service_type && 'ring-2 ring-primary',
            disabled && 'opacity-50 cursor-not-allowed'
          )}
          onClick={() => !disabled && onSelect(service.service_type, service)}
        >
          <CardHeader className="pb-2">
            <div className="flex items-center justify-between">
              <span className="text-2xl">{service.icon || WebAPIServiceTypeIcons[service.service_type]}</span>
              {isPresetService(service.service_type) && (
                <Badge variant="secondary" className="text-xs">预置</Badge>
              )}
            </div>
            <CardTitle className="text-lg">
              {service.name || WebAPIServiceTypeNames[service.service_type]}
            </CardTitle>
          </CardHeader>
          <CardContent>
            <CardDescription className="text-sm">
              {service.description}
            </CardDescription>
          </CardContent>
        </Card>
      ))}
    </div>
  );
};

export default WebAPIServiceSelector;
export { PRESET_SERVICES };
