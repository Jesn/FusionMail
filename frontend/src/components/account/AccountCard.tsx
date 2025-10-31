import { Mail, RefreshCw, Trash2, Edit, CheckCircle2, XCircle, Power } from 'lucide-react';
import { Checkbox } from '../ui/checkbox';
import { Button } from '../ui/button';
import { Badge } from '../ui/badge';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '../ui/card';
import { Account } from '../../types';
import { format } from 'date-fns';
import { zhCN } from 'date-fns/locale';
import { useProviders } from '../../hooks/useProviders';

interface AccountCardProps {
  account: Account;
  onSync: () => void;
  onDelete: () => void;
  onEdit: () => void;
  onToggleStatus: () => void;
  onClearError?: () => void;
  isSyncing?: boolean;
  // 新增属性
  density?: 'detailed' | 'compact' | 'minimal';
  isSelected?: boolean;
  onSelect?: (selected: boolean) => void;
  showSelection?: boolean;
}

export const AccountCard = ({
  account,
  onSync,
  onDelete,
  onEdit,
  onToggleStatus,
  onClearError,
  isSyncing,
  density = 'detailed',
  isSelected = false,
  onSelect,
  showSelection = false,
}: AccountCardProps) => {
  const { getProviderByName } = useProviders();

  const formatDate = (dateString?: string) => {
    if (!dateString) return '从未同步';
    try {
      return format(new Date(dateString), 'PPP HH:mm', { locale: zhCN });
    } catch {
      return dateString;
    }
  };

  const getProviderName = (provider: string) => {
    const providerInfo = getProviderByName(provider);
    return providerInfo?.display_name || provider;
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

  // 根据密度渲染不同的布局
  if (density === 'minimal') {
    return (
      <Card 
        className={`hover:shadow-sm transition-all duration-200 ${isSelected ? 'ring-2 ring-primary' : ''}`}
        data-testid="account-card"
      >
        <CardContent className="p-3">
          <div className="flex items-center justify-between">
            <div className="flex items-center gap-3 flex-1 min-w-0">
              {showSelection && (
                <Checkbox 
                  checked={isSelected}
                  onCheckedChange={onSelect}
                />
              )}
              <div className="flex items-center gap-2">
                {account.status === 'active' ? (
                  <CheckCircle2 className="h-4 w-4 text-green-600" />
                ) : account.status === 'disabled' ? (
                  <XCircle className="h-4 w-4 text-red-600" />
                ) : account.status === 'error' ? (
                  <XCircle className="h-4 w-4 text-orange-600" />
                ) : null}
              </div>
              <span className="font-medium truncate">{account.email}</span>
              <Badge variant="outline" className="text-xs">
                {getProviderName(account.provider)}
              </Badge>
              {account.last_sync_error && (
                <Badge variant="destructive" className="text-xs">
                  错误
                </Badge>
              )}
            </div>
            <div className="flex gap-1">
              <Button
                variant="ghost"
                size="sm"
                onClick={onSync}
                disabled={isSyncing}
                title="同步"
              >
                <RefreshCw className={`h-3 w-3 ${isSyncing ? 'animate-spin' : ''}`} />
              </Button>
              <Button variant="ghost" size="sm" onClick={onEdit} title="编辑">
                <Edit className="h-3 w-3" />
              </Button>
              <Button
                variant="ghost"
                size="sm"
                onClick={onToggleStatus}
                title={account.status === 'active' ? '禁用' : '启用'}
                className={account.status === 'active' ? 'text-orange-600' : 'text-green-600'}
              >
                <Power className="h-3 w-3" />
              </Button>
            </div>
          </div>
        </CardContent>
      </Card>
    );
  }

  if (density === 'compact') {
    return (
      <Card 
        className={`hover:shadow-md transition-all duration-200 ${isSelected ? 'ring-2 ring-primary' : ''}`}
        data-testid="account-card"
      >
        <CardContent className="p-4">
          <div className="flex items-center justify-between">
            <div className="flex items-center gap-3 flex-1">
              {showSelection && (
                <Checkbox 
                  checked={isSelected}
                  onCheckedChange={onSelect}
                />
              )}
              <div className="flex h-10 w-10 items-center justify-center rounded-full bg-primary text-primary-foreground">
                <Mail className="h-5 w-5" />
              </div>
              <div className="min-w-0 flex-1">
                <div className="flex items-center gap-2">
                  <span className="font-medium truncate">{account.email}</span>
                  {account.status === 'active' ? (
                    <CheckCircle2 className="h-4 w-4 text-green-600" />
                  ) : account.status === 'disabled' ? (
                    <XCircle className="h-4 w-4 text-red-600" />
                  ) : account.status === 'error' ? (
                    <XCircle className="h-4 w-4 text-orange-600" />
                  ) : null}
                </div>
                <div className="flex items-center gap-2 text-sm text-muted-foreground">
                  <span>{getProviderName(account.provider)}</span>
                  <span>•</span>
                  <span className={getSyncStatusColor(account.last_sync_status)}>
                    {account.last_sync_status === 'success'
                      ? '同步成功'
                      : account.last_sync_status === 'failed'
                      ? '同步失败'
                      : account.last_sync_status === 'running'
                      ? '同步中'
                      : '未同步'}
                  </span>
                  {account.last_sync_at && (
                    <>
                      <span>•</span>
                      <span>{formatDate(account.last_sync_at)}</span>
                    </>
                  )}
                </div>
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
              <Button variant="ghost" size="icon" onClick={onEdit} title="编辑账户">
                <Edit className="h-4 w-4" />
              </Button>
              <Button
                variant="ghost"
                size="icon"
                onClick={onToggleStatus}
                title={account.status === 'active' ? '禁用账户' : '启用账户'}
                className={account.status === 'active' ? 'text-orange-600 hover:text-orange-700' : 'text-green-600 hover:text-green-700'}
              >
                <Power className="h-4 w-4" />
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
          
          {/* 错误信息 */}
          {account.last_sync_error && (
            <div className="mt-3 rounded-lg bg-red-50 p-2 text-sm text-red-600 dark:bg-red-950/20">
              <div className="flex items-start justify-between gap-2">
                <div className="flex-1 text-xs">
                  {account.last_sync_error}
                </div>
                <Button
                  variant="ghost"
                  size="sm"
                  onClick={() => onClearError?.()}
                  className="h-5 px-1 text-xs text-red-600 hover:text-red-700"
                  title="清除错误状态"
                >
                  清除
                </Button>
              </div>
            </div>
          )}
        </CardContent>
      </Card>
    );
  }

  // 详细视图（原有的完整布局）
  return (
    <Card 
      className={`transition-all duration-200 ${isSelected ? 'ring-2 ring-primary' : ''}`}
      data-testid="account-card"
    >
      <CardHeader>
        <div className="flex items-start justify-between">
          <div className="flex items-center gap-3">
            {showSelection && (
              <Checkbox 
                checked={isSelected}
                onCheckedChange={onSelect}
              />
            )}
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
            <Button variant="ghost" size="icon" onClick={onEdit} title="编辑账户">
              <Edit className="h-4 w-4" />
            </Button>
            <Button
              variant="ghost"
              size="icon"
              onClick={onToggleStatus}
              title={account.status === 'active' ? '禁用账户' : '启用账户'}
              className={account.status === 'active' ? 'text-orange-600 hover:text-orange-700' : 'text-green-600 hover:text-green-700'}
              data-testid="toggle-status-button"
            >
              <Power className="h-4 w-4" />
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
          {/* 账户状态 */}
          <div className="flex items-center justify-between text-sm">
            <span className="text-muted-foreground">账户状态</span>
            <div className="flex items-center gap-2">
              {account.status === 'active' ? (
                <CheckCircle2 className="h-4 w-4 text-green-600" data-testid="account-status-icon" />
              ) : account.status === 'disabled' ? (
                <XCircle className="h-4 w-4 text-red-600" data-testid="account-status-icon" />
              ) : account.status === 'error' ? (
                <XCircle className="h-4 w-4 text-orange-600" data-testid="account-status-icon" />
              ) : null}
              <Badge variant={account.status === 'active' ? 'default' : 'destructive'} data-testid="account-status-badge">
                {account.status === 'active'
                  ? '正常'
                  : account.status === 'disabled'
                  ? '已禁用'
                  : account.status === 'error'
                  ? '错误'
                  : '未知'}
              </Badge>
            </div>
          </div>

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
              <div className="flex items-start justify-between gap-2">
                <div className="flex-1">
                  {account.last_sync_error}
                </div>
                <Button
                  variant="ghost"
                  size="sm"
                  onClick={() => onClearError?.()}
                  className="h-6 px-2 text-xs text-red-600 hover:text-red-700"
                  title="清除错误状态"
                >
                  清除
                </Button>
              </div>
            </div>
          )}
        </div>
      </CardContent>
    </Card>
  );
};