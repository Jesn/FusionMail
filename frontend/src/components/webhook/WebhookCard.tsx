import { useState } from 'react';
import { 
  MoreHorizontal, 
  Play, 
  Pause, 
  Edit, 
  Trash2, 
  TestTube, 
  Eye
} from 'lucide-react';
import { Button } from '../ui/button';
import { Card, CardContent, CardHeader, CardTitle } from '../ui/card';
import { Badge } from '../ui/badge';
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from '../ui/dropdown-menu';
import { Webhook } from '../../services/webhookService';
import { formatDistanceToNow } from 'date-fns';
import { zhCN } from 'date-fns/locale';

interface WebhookCardProps {
  webhook: Webhook;
  onEdit: (webhook: Webhook) => void;
  onDelete: (id: number, name: string) => void;
  onToggle: (id: number) => void;
  onTest: (id: number) => void;
  onViewLogs: (webhook: Webhook) => void;
}

export const WebhookCard = ({
  webhook,
  onEdit,
  onDelete,
  onToggle,
  onTest,
  onViewLogs,
}: WebhookCardProps) => {
  const [isLoading, setIsLoading] = useState(false);

  const handleToggle = async () => {
    setIsLoading(true);
    try {
      await onToggle(webhook.id);
    } finally {
      setIsLoading(false);
    }
  };

  const handleTest = async () => {
    setIsLoading(true);
    try {
      await onTest(webhook.id);
    } finally {
      setIsLoading(false);
    }
  };

  const getStatusBadge = () => {
    if (!webhook.enabled) {
      return <Badge variant="secondary">已禁用</Badge>;
    }
    
    if (webhook.last_status === 'success') {
      return <Badge variant="default" className="bg-green-600">正常</Badge>;
    } else if (webhook.last_status === 'failed') {
      return <Badge variant="destructive">失败</Badge>;
    } else {
      return <Badge variant="outline">未调用</Badge>;
    }
  };

  const getSuccessRate = () => {
    if (webhook.total_calls === 0) return 0;
    return Math.round((webhook.success_calls / webhook.total_calls) * 100);
  };

  const formatLastCalled = () => {
    if (!webhook.last_called_at) return '从未调用';
    try {
      const date = new Date(webhook.last_called_at);
      return formatDistanceToNow(date, { addSuffix: true, locale: zhCN });
    } catch {
      return '时间无效';
    }
  };

  return (
    <Card className="hover:shadow-md transition-shadow">
      <CardHeader className="pb-3">
        <div className="flex items-center justify-between">
          <div className="flex items-center gap-3">
            <CardTitle className="text-lg">{webhook.name}</CardTitle>
            {getStatusBadge()}
          </div>
          
          <DropdownMenu>
            <DropdownMenuTrigger asChild>
              <Button variant="ghost" size="sm">
                <MoreHorizontal className="h-4 w-4" />
              </Button>
            </DropdownMenuTrigger>
            <DropdownMenuContent align="end">
              <DropdownMenuItem onClick={() => onEdit(webhook)}>
                <Edit className="mr-2 h-4 w-4" />
                编辑
              </DropdownMenuItem>
              <DropdownMenuItem onClick={handleToggle} disabled={isLoading}>
                {webhook.enabled ? (
                  <>
                    <Pause className="mr-2 h-4 w-4" />
                    禁用
                  </>
                ) : (
                  <>
                    <Play className="mr-2 h-4 w-4" />
                    启用
                  </>
                )}
              </DropdownMenuItem>
              <DropdownMenuItem onClick={handleTest} disabled={isLoading}>
                <TestTube className="mr-2 h-4 w-4" />
                测试
              </DropdownMenuItem>
              <DropdownMenuItem onClick={() => onViewLogs(webhook)}>
                <Eye className="mr-2 h-4 w-4" />
                查看日志
              </DropdownMenuItem>
              <DropdownMenuSeparator />
              <DropdownMenuItem 
                onClick={() => onDelete(webhook.id, webhook.name)}
                className="text-red-600"
              >
                <Trash2 className="mr-2 h-4 w-4" />
                删除
              </DropdownMenuItem>
            </DropdownMenuContent>
          </DropdownMenu>
        </div>
        
        {webhook.description && (
          <p className="text-sm text-gray-600 dark:text-gray-400 mt-1">
            {webhook.description}
          </p>
        )}
      </CardHeader>
      
      <CardContent>
        <div className="space-y-3">
          {/* URL 和方法 */}
          <div className="flex items-center gap-2 text-sm">
            <Badge variant="outline" className="font-mono">
              {webhook.method}
            </Badge>
            <span className="text-gray-600 dark:text-gray-400 truncate">
              {webhook.url}
            </span>
          </div>
          
          {/* 事件类型 */}
          <div className="flex items-center gap-2 text-sm">
            <span className="text-gray-500">事件:</span>
            <div className="flex flex-wrap gap-1">
              {JSON.parse(webhook.events || '[]').map((event: string, index: number) => (
                <Badge key={index} variant="secondary" className="text-xs">
                  {event}
                </Badge>
              ))}
            </div>
          </div>
          
          {/* 统计信息 */}
          <div className="grid grid-cols-3 gap-4 pt-2 border-t">
            <div className="text-center">
              <div className="text-lg font-semibold">{webhook.total_calls}</div>
              <div className="text-xs text-gray-500">总调用</div>
            </div>
            <div className="text-center">
              <div className="text-lg font-semibold text-green-600">
                {getSuccessRate()}%
              </div>
              <div className="text-xs text-gray-500">成功率</div>
            </div>
            <div className="text-center">
              <div className="text-xs text-gray-500 mb-1">最后调用</div>
              <div className="text-xs">{formatLastCalled()}</div>
            </div>
          </div>
        </div>
      </CardContent>
    </Card>
  );
};