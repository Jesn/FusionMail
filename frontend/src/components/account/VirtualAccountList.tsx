import { useRef, useMemo } from 'react';
import { useVirtualizer } from '@tanstack/react-virtual';
import { AccountCard } from './AccountCard';
import { Account } from '../../types';
import type { AccountDensity, AccountStatus, AccountProvider } from './AccountToolbar';

interface VirtualAccountListProps {
  accounts: Account[];
  density: AccountDensity;
  searchQuery: string;
  statusFilter: AccountStatus;
  providerFilter: AccountProvider;
  selectedAccounts: string[];
  onAccountSelect: (uid: string, selected: boolean) => void;
  showSelection: boolean;
  onSync: (uid: string) => void;
  onDelete: (uid: string, email: string) => void;
  onEdit: (account: Account) => void;
  onToggleStatus: (uid: string, status: string) => void;
  onClearError: (uid: string) => void;
  syncingAccounts?: string[];
  height?: number; // 容器高度，默认为 600px
}

export const VirtualAccountList = ({
  accounts,
  density,
  searchQuery,
  statusFilter,
  providerFilter,
  selectedAccounts,
  onAccountSelect,
  showSelection,
  onSync,
  onDelete,
  onEdit,
  onToggleStatus,
  onClearError,
  syncingAccounts = [],
  height = 600,
}: VirtualAccountListProps) => {
  const parentRef = useRef<HTMLDivElement>(null);

  // 筛选账户
  const filteredAccounts = useMemo(() => {
    return accounts.filter((account) => {
      // 搜索筛选
      if (searchQuery) {
        const query = searchQuery.toLowerCase();
        const emailMatch = account.email.toLowerCase().includes(query);
        const providerMatch = account.provider.toLowerCase().includes(query);
        if (!emailMatch && !providerMatch) {
          return false;
        }
      }

      // 状态筛选
      if (statusFilter !== 'all' && account.status !== statusFilter) {
        return false;
      }

      // 服务商筛选
      if (providerFilter !== 'all' && account.provider !== providerFilter) {
        return false;
      }

      return true;
    });
  }, [accounts, searchQuery, statusFilter, providerFilter]);

  // 根据密度获取项目高度
  const getItemSize = (density: AccountDensity) => {
    switch (density) {
      case 'minimal':
        return 72; // 60px + 12px margin
      case 'compact':
        return 112; // 100px + 12px margin
      case 'detailed':
      default:
        return 192; // 180px + 12px margin
    }
  };

  const itemSize = getItemSize(density);

  // 虚拟滚动配置
  const virtualizer = useVirtualizer({
    count: filteredAccounts.length,
    getScrollElement: () => parentRef.current,
    estimateSize: () => itemSize,
    overscan: 5, // 预渲染额外的项目数量
  });

  if (filteredAccounts.length === 0) {
    return (
      <div className="flex flex-col items-center justify-center rounded-lg border border-dashed py-12">
        <p className="mb-4 text-lg text-muted-foreground">
          {searchQuery || statusFilter !== 'all' || providerFilter !== 'all'
            ? '没有找到匹配的账户'
            : '还没有添加邮箱账户'}
        </p>
        {!searchQuery && statusFilter === 'all' && providerFilter === 'all' && (
          <p className="text-sm text-muted-foreground">
            点击右上角的"添加账户"按钮开始添加
          </p>
        )}
      </div>
    );
  }

  return (
    <div
      ref={parentRef}
      className="overflow-auto rounded-lg border"
      style={{ height: `${height}px` }}
    >
      <div
        style={{
          height: `${virtualizer.getTotalSize()}px`,
          width: '100%',
          position: 'relative',
        }}
      >
        {virtualizer.getVirtualItems().map((virtualItem) => {
          const account = filteredAccounts[virtualItem.index];
          
          return (
            <div
              key={virtualItem.key}
              style={{
                position: 'absolute',
                top: 0,
                left: 0,
                width: '100%',
                height: `${virtualItem.size}px`,
                transform: `translateY(${virtualItem.start}px)`,
                padding: '6px',
              }}
            >
              <AccountCard
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
            </div>
          );
        })}
      </div>
    </div>
  );
};