import { Mail, RefreshCw, Trash2, Edit, CheckCircle2, XCircle } from 'lucide-react';
import { Button } from '../ui/button';
import { Badge } from '../ui/badge';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '../ui/card';
import { Account } from '../../types';
import { format } from 'date-fns';
import { zhCN } from 'date-fns/locale';

interface AccountCardProps {
  account: Account;
  onSync: () => void;
  onDelete: () => void;
  onTest: () => void;
  isSyncing?: boolean;
}

export const AccountCard = ({
  account,
  onSync,
  onDelete,
  onTest,
  isSyncing,
}: AccountCardProps) => {
  const formatDate = (dateString?: string) => {
    if (!dateString) return '从未同步';
    try {
      return format(new Date(dateString), 'PPP HH:mm', { locale: zhCN });
    } catch {
      return dateString;
    }
  };

  const getProviderName = (provider: string) => {
    const names: Record<string, string> = {
      qq: 'QQ 邮箱',
      '163': '163 邮箱',
      gmail: 'Gmail',
      outlook: 'Outlook',
      icloud: 'iCloud',
    };
    return names[provider] || provider;
  };

  const getSyncStatusColor = (status?: string) => {
    switch (status) {
      case 'success':
        return 'text-green-600';
      case 'failed':
        return 'text-red-600';
      case 'running':
        return 'text-blue-600';
      default:
        return 'text-muted-foreground';
    }
  };

  return (
    <Card>
      <CardHeader>
        <div className="flex items-start justify-between">
          <div className="flex items-center gap-3">
            <div className="flex h-12 w-12 items-center justify-center rounded-full bg-primary text-primary-foreground">
              <Mail className="h-6 w-6" />
            </div>
            <div>
              <CardTitle className="text-lg">{account.email}</CardTitle>
              <CardDescription>{getProviderName(account.provider)}</CardDescription>
            </div>
          </div>
          <div className="flex gap-1">
            <Button
              variant="ghost"
              size="icon"
              onClick={onSync}
              disabled={isSyncing}
              title="同步"
            >
              <RefreshCw className={`h-4 w-4 ${isSyncing ? 'animate-spin' : ''}`} />
            </Button>
            <Button variant="ghost" size="icon" onClick={onTest} title="编辑账户">
              <Edit className="h-4 w-4" />
            </Button>
            <Button
              variant="ghost"
              size="icon"
              onClick={onDelete}
              title="删除"
              className="text-red-600 hover:text-red-700"
            >
              <Trash2 className="h-4 w-4" />
            </Button>
          </div>
        </div>
      </CardHeader>

      <CardContent>
        <div className="space-y-3">
          {/* 同步状态 */}
          <div className="flex items-center justify-between text-sm">
            <span className="text-muted-foreground">同步状态</span>
            <div className="flex items-center gap-2">
              {account.last_sync_status === 'success' ? (
                <CheckCircle2 className="h-4 w-4 text-green-600" />
              ) : account.last_sync_status === 'failed' ? (
                <XCircle className="h-4 w-4 text-red-600" />
              ) : null}
              <span className={getSyncStatusColor(account.last_sync_status)}>
                {account.last_sync_status === 'success'
                  ? '成功'
                  : account.last_sync_status === 'failed'
                  ? '失败'
                  : account.last_sync_status === 'running'
                  ? '同步中'
                  : '未同步'}
              </span>
            </div>
          </div>

          {/* 最后同步时间 */}
          <div className="flex items-center justify-between text-sm">
            <span className="text-muted-foreground">最后同步</span>
            <span>{formatDate(account.last_sync_at)}</span>
          </div>

          {/* 同步设置 */}
          <div className="flex items-center justify-between text-sm">
            <span className="text-muted-foreground">自动同步</span>
            <div className="flex items-center gap-2">
              <Badge variant={account.sync_enabled ? 'default' : 'secondary'}>
                {account.sync_enabled ? '已启用' : '已禁用'}
              </Badge>
              {account.sync_enabled && (
                <span className="text-muted-foreground">
                  每 {account.sync_interval} 分钟
                </span>
              )}
            </div>
          </div>

          {/* 错误信息 */}
          {account.last_sync_error && (
            <div className="rounded-lg bg-red-50 p-3 text-sm text-red-600 dark:bg-red-950/20">
              {account.last_sync_error}
            </div>
          )}
        </div>
      </CardContent>
    </Card>
  );
};
