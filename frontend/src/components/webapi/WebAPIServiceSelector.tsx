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

// 预置服务模板（仅包含已实现的服务）
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
        { name: 'admin', label: 'Admin 模式配置', fields: ['admin_password', 'domains'], condition: { field: 'access_mode', value: 'admin' }},
      ],
    },
  },
  {
    service_type: 'cloud_mail',
    name: 'Cloud Mail',
    description: '支持多账户管理的云邮箱服务（如 mail.hema.edu.kg）',
    icon: '📧',
    default_config: {
      base_url: '',
      jwt_token: '',
      email: '',
      password: '',
    },
    config_schema: {
      fields: [
        { name: 'base_url', label: 'API 地址', type: 'text', required: true, placeholder: 'https://your-domain.com' },
        { name: 'email', label: '登录邮箱', type: 'text', required: false },
        { name: 'password', label: '登录密码', type: 'password', required: false },
        { name: 'jwt_token', label: 'JWT Token', type: 'password', required: false },
      ],
      groups: [],
    },
  },
  // 注意：自定义服务（custom）暂未完整实现，暂不显示
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
