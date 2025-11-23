import { useState } from 'react';
import { useQuery } from '@tanstack/react-query';
import { AlertCircle, Loader2 } from 'lucide-react';
import { Alert, AlertDescription } from '../ui/alert';
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '../ui/select';
import { oauth2ClientService } from '@/services/oauth2ClientService';
import { OAuth2Client } from '@/types';
import { cn } from '@/lib/utils';

interface OAuth2ClientSelectorProps {
  providerId: number;
  value?: number;
  onChange: (clientId: number | undefined) => void;
  className?: string;
  placeholder?: string;
  disabled?: boolean;
  showDefaultOption?: boolean;
  defaultOptionText?: string;
}

export function OAuth2ClientSelector({
  providerId,
  value,
  onChange,
  className,
  placeholder = '选择 OAuth2 客户端配置',
  disabled = false,
  showDefaultOption = true,
  defaultOptionText = '使用智能选择（自动）',
}: OAuth2ClientSelectorProps) {
  const [selectedClientId, setSelectedClientId] = useState<number | undefined>(value);

  // 获取提供商的 OAuth2 客户端列表
  const {
    data: clients = [],
    isLoading,
    error,
  } = useQuery({
    queryKey: ['oauth2-clients', providerId],
    queryFn: () => oauth2ClientService.getByProvider(providerId),
    enabled: !!providerId,
  });

  // 过滤启用的客户端
  const enabledClients = clients.filter(client => client.enabled);

  // 处理选择变化
  const handleValueChange = (newValue: string) => {
    const clientId = newValue === 'auto' ? undefined : parseInt(newValue, 10);
    setSelectedClientId(clientId);
    onChange(clientId);
  };

  // 格式化客户端显示文本
  const formatClientDisplay = (client: OAuth2Client) => {
    const parts = [client.name];
    if (client.is_default) {
      parts.push('(默认)');
    }
    if (client.usage_count > 0) {
      parts.push(`已使用 ${client.usage_count} 次`);
    }
    if (client.quota_daily > 0) {
      parts.push(`日配额 ${client.quota_daily}`);
    }
    return parts.join(' ');
  };

  if (isLoading) {
    return (
      <div className={cn('flex items-center gap-2', className)}>
        <Loader2 className="h-4 w-4 animate-spin" />
        <span className="text-sm text-muted-foreground">加载中...</span>
      </div>
    );
  }

  if (error) {
    return (
      <Alert className={className} variant="destructive">
        <AlertCircle className="h-4 w-4" />
        <AlertDescription>
          加载 OAuth2 客户端配置失败: {(error as Error).message}
        </AlertDescription>
      </Alert>
    );
  }

  if (enabledClients.length === 0) {
    return (
      <Alert className={className}>
        <AlertCircle className="h-4 w-4" />
        <AlertDescription>
          暂无可用的 OAuth2 客户端配置，请联系管理员配置。
        </AlertDescription>
      </Alert>
    );
  }

  return (
    <Select
      value={selectedClientId?.toString() || 'auto'}
      onValueChange={handleValueChange}
      disabled={disabled}
    >
      <SelectTrigger className={className}>
        <SelectValue placeholder={placeholder} />
      </SelectTrigger>
      <SelectContent>
        {showDefaultOption && (
          <SelectItem value="auto">
            <div className="flex items-center gap-2">
              <span className="font-medium">{defaultOptionText}</span>
              <span className="text-xs text-muted-foreground">
                (系统将自动选择最佳配置)
              </span>
            </div>
          </SelectItem>
        )}
        {enabledClients.map(client => (
          <SelectItem key={client.id} value={client.id.toString()}>
            <div className="flex flex-col">
              <span className="font-medium">{formatClientDisplay(client)}</span>
              <span className="text-xs text-muted-foreground">
                Client ID: {client.client_id}
              </span>
            </div>
          </SelectItem>
        ))}
      </SelectContent>
    </Select>
  );
}
