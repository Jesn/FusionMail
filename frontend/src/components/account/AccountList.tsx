import { useMemo } from 'react';
import { AccountCard } from './AccountCard';
import { Account } from '../../types';
import type { AccountDensity, AccountStatus, AccountProvider } from './AccountToolbar';

interface AccountListProps {
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
}

export const AccountList = ({
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
}: AccountListProps) => {
  // 筛选和搜索逻辑
  const filteredAccounts = useMemo(() => {
    return accounts.filter((account) => {
      // 搜索筛选
      if (searchQuery) {
        const query = searchQuery.toLowerCase();
        if (!account.email.toLowerCase().includes(query)) {
          return false;
        }
      }

      // 状态筛选
      if (statusFilter !== 'all') {
        if (account.status !== statusFilter) {
          return false;
        }
      }

      // 服务商筛选
      if (providerFilter !== 'all') {
        if (account.provider !== providerFilter) {
          return false;
        }
      }

      return true;
    });
  }, [accounts, searchQuery, statusFilter, providerFilter]);

  // 根据密度获取容器样式
  const getContainerClass = () => {
    switch (density) {
      case 'minimal':
        return 'space-y-2';
      case 'compact':
        return 'space-y-3';
      case 'detailed':
      default:
        return 'space-y-4';
    }
  };

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
    <div className={getContainerClass()}>
      {filteredAccounts.map((account) => (
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
  );
};