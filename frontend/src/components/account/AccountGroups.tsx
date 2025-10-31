import { useState, useMemo } from 'react';
import { ChevronDown, ChevronRight, Users, Server, AlertCircle, CheckCircle } from 'lucide-react';
import { Button } from '../ui/button';
import { Badge } from '../ui/badge';
import { Card, CardContent, CardHeader } from '../ui/card';
import { Collapsible, CollapsibleContent, CollapsibleTrigger } from '../ui/collapsible';
import { AccountCard } from './AccountCard';
import { Account } from '../../types';
import type { AccountDensity } from './AccountToolbar';

export type GroupBy = 'provider' | 'status' | 'sync_status' | 'none';

// interface AccountGroup {
//   key: string;
//   label: string;
//   accounts: Account[];
//   icon: React.ReactNode;
//   color: string;
// }

interface AccountGroupsProps {
  accounts: Account[];
  groupBy: GroupBy;
  density: AccountDensity;
  selectedAccounts: string[];
  onAccountSelect: (uid: string, selected: boolean) => void;
  showSelection: boolean;
  onSync: (uid: string) => void;
  onDelete: (uid: string, email: string) => void;
  onEdit: (account: Account) => void;
  onToggleStatus: (uid: string, status: string) => void;
  onClearError: (uid: string) => void;
  syncingAccounts?: string[];
}

export const AccountGroups = ({
  accounts,
  groupBy,
  density,
  selectedAccounts,
  onAccountSelect,
  showSelection,
  onSync,
  onDelete,
  onEdit,
  onToggleStatus,
  onClearError,
  syncingAccounts = [],
}: AccountGroupsProps) => {
  const [expandedGroups, setExpandedGroups] = useState<Set<string>>(new Set());

  // 分组逻辑
  const groups = useMemo(() => {
    if (groupBy === 'none') {
      return [{
        key: 'all',
        label: '所有账户',
        accounts,
        icon: <Users className="h-4 w-4" />,
        color: 'text-blue-600',
      }];
    }

    const groupMap = new Map<string, Account[]>();

    accounts.forEach(account => {
      let groupKey: string;
      
      switch (groupBy) {
        case 'provider':
          groupKey = account.provider;
          break;
        case 'status':
          groupKey = account.status;
          break;
        case 'sync_status':
          groupKey = account.last_sync_status || 'never';
          break;
        default:
          groupKey = 'all';
      }

      if (!groupMap.has(groupKey)) {
        groupMap.set(groupKey, []);
      }
      groupMap.get(groupKey)!.push(account);
    });

    return Array.from(groupMap.entries()).map(([key, accounts]) => {
      let label: string;
      let icon: React.ReactNode;
      let color: string;

      switch (groupBy) {
        case 'provider':
          label = getProviderLabel(key);
          icon = <Server className="h-4 w-4" />;
          color = 'text-blue-600';
          break;
        case 'status':
          label = getStatusLabel(key);
          icon = getStatusIcon(key);
          color = getStatusColor(key);
          break;
        case 'sync_status':
          label = getSyncStatusLabel(key);
          icon = getSyncStatusIcon(key);
          color = getSyncStatusColor(key);
          break;
        default:
          label = key;
          icon = <Users className="h-4 w-4" />;
          color = 'text-gray-600';
      }

      return {
        key,
        label,
        accounts: accounts.sort((a, b) => a.email.localeCompare(b.email)),
        icon,
        color,
      };
    }).sort((a, b) => {
      // 按账户数量降序排列
      return b.accounts.length - a.accounts.length;
    });
  }, [accounts, groupBy]);

  // 切换分组展开状态
  const toggleGroup = (groupKey: string) => {
    setExpandedGroups(prev => {
      const newSet = new Set(prev);
      if (newSet.has(groupKey)) {
        newSet.delete(groupKey);
      } else {
        newSet.add(groupKey);
      }
      return newSet;
    });
  };

  // 全部展开
  const expandAll = () => {
    setExpandedGroups(new Set(groups.map(g => g.key)));
  };

  // 全部折叠
  const collapseAll = () => {
    setExpandedGroups(new Set());
  };

  // 根据密度获取容器样式
  const getContainerClass = () => {
    switch (density) {
      case 'minimal':
        return 'space-y-1';
      case 'compact':
        return 'space-y-2';
      case 'detailed':
      default:
        return 'space-y-3';
    }
  };

  return (
    <div className="space-y-4">
      {/* 分组控制 */}
      {groups.length > 1 && (
        <div className="flex items-center justify-between">
          <div className="text-sm text-muted-foreground">
            共 {groups.length} 个分组，{accounts.length} 个账户
          </div>
          <div className="flex gap-2">
            <Button variant="outline" size="sm" onClick={expandAll}>
              全部展开
            </Button>
            <Button variant="outline" size="sm" onClick={collapseAll}>
              全部折叠
            </Button>
          </div>
        </div>
      )}

      {/* 分组列表 */}
      <div className="space-y-3">
        {groups.map((group) => {
          const isExpanded = expandedGroups.has(group.key);
          
          return (
            <Card key={group.key}>
              <Collapsible open={isExpanded} onOpenChange={() => toggleGroup(group.key)}>
                <CollapsibleTrigger asChild>
                  <CardHeader className="cursor-pointer hover:bg-muted/50 transition-colors">
                    <div className="flex items-center justify-between">
                      <div className="flex items-center gap-3">
                        <div className="flex items-center gap-2">
                          {isExpanded ? (
                            <ChevronDown className="h-4 w-4" />
                          ) : (
                            <ChevronRight className="h-4 w-4" />
                          )}
                          <div className={group.color}>
                            {group.icon}
                          </div>
                        </div>
                        <div>
                          <h3 className="font-medium">{group.label}</h3>
                          <p className="text-sm text-muted-foreground">
                            {group.accounts.length} 个账户
                          </p>
                        </div>
                      </div>
                      <div className="flex items-center gap-2">
                        <Badge variant="secondary">
                          {group.accounts.length}
                        </Badge>
                        {group.accounts.some(acc => acc.status === 'error') && (
                          <Badge variant="destructive" className="text-xs">
                            {group.accounts.filter(acc => acc.status === 'error').length} 异常
                          </Badge>
                        )}
                      </div>
                    </div>
                  </CardHeader>
                </CollapsibleTrigger>
                
                <CollapsibleContent>
                  <CardContent className="pt-0">
                    <div className={getContainerClass()}>
                      {group.accounts.map((account) => (
                        <AccountCard
                          key={account.uid}
                          account={account}
                          density={density}
                          isSelected={selectedAccounts.includes(account.uid)}
                          onSelect={(selected) => onAccountSelect(account.uid, selected)}
                          showSelection={showSelection}
                          onSync={() => onSync(account.uid)}
                          onDelete={() => onDelete(account.uid, account.email)}
                          onEdit={() => onEdit(account)}
                          onToggleStatus={() => onToggleStatus(account.uid, account.status)}
                          onClearError={() => onClearError(account.uid)}
                          isSyncing={syncingAccounts.includes(account.uid)}
                        />
                      ))}
                    </div>
                  </CardContent>
                </CollapsibleContent>
              </Collapsible>
            </Card>
          );
        })}
      </div>
    </div>
  );
};

// 辅助函数
function getProviderLabel(provider: string): string {
  const labels: Record<string, string> = {
    gmail: 'Gmail',
    outlook: 'Outlook',
    imap: 'IMAP',
    pop3: 'POP3',
  };
  return labels[provider] || provider.toUpperCase();
}

function getStatusLabel(status: string): string {
  const labels: Record<string, string> = {
    active: '正常账户',
    disabled: '已禁用账户',
    error: '异常账户',
  };
  return labels[status] || status;
}

function getStatusIcon(status: string): React.ReactNode {
  switch (status) {
    case 'active':
      return <CheckCircle className="h-4 w-4" />;
    case 'disabled':
    case 'error':
      return <AlertCircle className="h-4 w-4" />;
    default:
      return <Users className="h-4 w-4" />;
  }
}

function getStatusColor(status: string): string {
  switch (status) {
    case 'active':
      return 'text-green-600';
    case 'disabled':
      return 'text-gray-600';
    case 'error':
      return 'text-red-600';
    default:
      return 'text-gray-600';
  }
}

function getSyncStatusLabel(status: string): string {
  const labels: Record<string, string> = {
    success: '同步成功',
    failed: '同步失败',
    running: '同步中',
    never: '从未同步',
  };
  return labels[status] || status;
}

function getSyncStatusIcon(status: string): React.ReactNode {
  switch (status) {
    case 'success':
      return <CheckCircle className="h-4 w-4" />;
    case 'failed':
    case 'never':
      return <AlertCircle className="h-4 w-4" />;
    default:
      return <Users className="h-4 w-4" />;
  }
}

function getSyncStatusColor(status: string): string {
  switch (status) {
    case 'success':
      return 'text-green-600';
    case 'failed':
      return 'text-red-600';
    case 'running':
      return 'text-blue-600';
    case 'never':
      return 'text-gray-600';
    default:
      return 'text-gray-600';
  }
}