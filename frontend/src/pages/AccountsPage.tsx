import { Plus } from 'lucide-react';
import { useState } from 'react';
import { Button } from '../components/ui/button';
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
} from '../components/ui/alert-dialog';
import { AccountList } from '../components/account/AccountList';
import { VirtualAccountList } from '../components/account/VirtualAccountList';
import { AccountGroups } from '../components/account/AccountGroups';
import { AccountTablePaginated } from '../components/account/AccountTablePaginated';
import { AccountToolbar } from '../components/account/AccountToolbar';
import { AccountForm } from '../components/account/AccountForm';
import { useAccounts } from '../hooks/useAccounts';
import { useAccountViewMode } from '../hooks/useAccountViewMode';
import { useAccountFilters } from '../hooks/useAccountFilters';
import { useAccountDensity } from '../hooks/useAccountDensity';
import { useUIStore } from '../stores/uiStore';
import { Account } from '../types';
import type { SyncStatus } from '../components/account/AccountToolbar';

export const AccountsPage = () => {
  const {
    accounts,
    isLoading,
    createAccount,
    updateAccount,
    deleteAccount,
    syncAccount,
    toggleAccountStatus,
    clearSyncError,
  } = useAccounts();
  
  // 临时的同步状态管理（后续可以集成到 useAccounts 中）
  const [syncingAccounts, setSyncingAccounts] = useState<string[]>([]);
  
  const { isAccountDialogOpen, setAccountDialogOpen } = useUIStore();
  const [editingAccount, setEditingAccount] = useState<Account | null>(null);
  const [deletingAccount, setDeletingAccount] = useState<{ uid: string; email: string } | null>(null);
  const [selectedAccounts, setSelectedAccounts] = useState<string[]>([]);
  const [showSelection, setShowSelection] = useState(false);

  // 确保 accounts 不为 undefined
  const safeAccounts = accounts || [];

  // 视图模式管理
  const { viewMode, setViewMode, groupBy, setGroupBy } = useAccountViewMode({
    accountCount: safeAccounts.length,
  });

  // 筛选管理
  const {
    searchQuery,
    setSearchQuery,
    statusFilter,
    setStatusFilter,
    providerFilter,
    setProviderFilter,
  } = useAccountFilters({ accounts: safeAccounts });

  // 同步状态筛选
  const [syncStatusFilter, setSyncStatusFilter] = useState<SyncStatus>('all');

  // 密度管理
  const { density, setDensity } = useAccountDensity({
    accountCount: safeAccounts.length,
  });

  // 统计信息
  const activeCount = safeAccounts.filter(acc => acc.status === 'active').length;
  const errorCount = safeAccounts.filter(acc => acc.status === 'error').length;

  // 事件处理
  const handleDeleteClick = (uid: string, email: string) => {
    setDeletingAccount({ uid, email });
  };

  const handleDeleteConfirm = async () => {
    if (deletingAccount) {
      await deleteAccount(deletingAccount.uid);
      setDeletingAccount(null);
    }
  };

  const handleDeleteCancel = () => {
    setDeletingAccount(null);
  };

  const handleEdit = (account: Account) => {
    setEditingAccount(account);
    setAccountDialogOpen(true);
  };

  const handleCloseDialog = () => {
    setAccountDialogOpen(false);
    setEditingAccount(null);
  };

  const handleSubmit = async (data: any) => {
    if (editingAccount) {
      await updateAccount(editingAccount.uid, data);
    } else {
      await createAccount(data);
    }
    handleCloseDialog();
  };

  // 账户选择处理
  const handleAccountSelect = (uid: string, selected: boolean) => {
    if (selected) {
      setSelectedAccounts(prev => [...prev, uid]);
    } else {
      setSelectedAccounts(prev => prev.filter(id => id !== uid));
    }
  };

  const handleSelectAll = () => {
    setSelectedAccounts(safeAccounts.map(acc => acc.uid));
  };

  const handleClearSelection = () => {
    setSelectedAccounts([]);
    setShowSelection(false);
  };

  // 批量操作
  const handleBatchSync = async () => {
    setSyncingAccounts(selectedAccounts);
    try {
      for (const uid of selectedAccounts) {
        await syncAccount(uid);
      }
    } finally {
      setSyncingAccounts([]);
      setSelectedAccounts([]);
    }
  };

  const handleBatchToggleStatus = async () => {
    for (const uid of selectedAccounts) {
      const account = safeAccounts.find(acc => acc.uid === uid);
      if (account) {
        await toggleAccountStatus(uid, account.status);
      }
    }
    setSelectedAccounts([]);
  };

  // 渲染视图组件
  const renderAccountView = () => {
    const commonProps = {
      accounts: safeAccounts,
      searchQuery,
      statusFilter,
      providerFilter,
      selectedAccounts,
      onAccountSelect: handleAccountSelect,
      showSelection,
      onSync: async (uid: string) => {
        setSyncingAccounts(prev => [...prev, uid]);
        try {
          await syncAccount(uid);
        } finally {
          setSyncingAccounts(prev => prev.filter(id => id !== uid));
        }
      },
      onDelete: handleDeleteClick,
      onEdit: handleEdit,
      onToggleStatus: toggleAccountStatus,
      onClearError: clearSyncError,
      syncingAccounts,
    };

    switch (viewMode) {
      case 'list':
        return <AccountList {...commonProps} density={density} />;
      case 'virtual':
        return <VirtualAccountList {...commonProps} density={density} />;
      case 'groups':
        return <AccountGroups {...commonProps} density={density} groupBy={groupBy} />;
      case 'table':
        return <AccountTablePaginated {...commonProps} />;
      default:
        return <AccountList {...commonProps} density={density} />;
    }
  };

  return (
    <div className="h-full overflow-auto">
      <div className="mx-auto max-w-7xl p-6">
        {/* 头部 */}
        <div className="mb-6 flex items-center justify-between">
          <div>
            <h1 className="text-3xl font-bold">邮箱账户</h1>
            <p className="text-muted-foreground">
              管理您的邮箱账户和同步设置
            </p>
          </div>
          <div className="flex items-center gap-2">
            {selectedAccounts.length > 0 && (
              <Button variant="outline" onClick={() => setShowSelection(!showSelection)}>
                {showSelection ? '取消选择' : '批量操作'}
              </Button>
            )}
            <Button onClick={() => setAccountDialogOpen(true)}>
              <Plus className="mr-2 h-4 w-4" />
              添加账户
            </Button>
          </div>
        </div>

        {/* 工具栏 */}
        {safeAccounts.length > 0 && (
          <div className="mb-6">
            <AccountToolbar
              searchQuery={searchQuery}
              onSearchChange={setSearchQuery}
              statusFilter={statusFilter}
              onStatusFilterChange={setStatusFilter}
              providerFilter={providerFilter}
              onProviderFilterChange={setProviderFilter}
              syncStatusFilter={syncStatusFilter}
              onSyncStatusFilterChange={setSyncStatusFilter}
              density={density}
              onDensityChange={setDensity}
              viewMode={viewMode}
              onViewModeChange={setViewMode}
              groupBy={groupBy}
              onGroupByChange={setGroupBy}
              selectedCount={selectedAccounts.length}
              totalCount={safeAccounts.length}
              onSelectAll={handleSelectAll}
              onClearSelection={handleClearSelection}
              onBatchSync={handleBatchSync}
              onBatchToggleStatus={handleBatchToggleStatus}
              activeCount={activeCount}
              errorCount={errorCount}
            />
          </div>
        )}

        {/* 账户视图 */}
        {isLoading ? (
          <div className="flex items-center justify-center py-12">
            <div className="text-muted-foreground">加载中...</div>
          </div>
        ) : safeAccounts.length === 0 ? (
          <div className="flex flex-col items-center justify-center rounded-lg border border-dashed py-12">
            <p className="mb-4 text-lg text-muted-foreground">
              还没有添加邮箱账户
            </p>
            <Button onClick={() => setAccountDialogOpen(true)}>
              <Plus className="mr-2 h-4 w-4" />
              添加第一个账户
            </Button>
          </div>
        ) : (
          renderAccountView()
        )}

        {/* 添加/编辑账户对话框 */}
        <AccountForm
          open={isAccountDialogOpen}
          onClose={handleCloseDialog}
          onSubmit={handleSubmit}
          account={editingAccount}
        />

        {/* 删除确认对话框 */}
        <AlertDialog open={!!deletingAccount} onOpenChange={(open) => !open && handleDeleteCancel()}>
          <AlertDialogContent>
            <AlertDialogHeader>
              <AlertDialogTitle>确认删除账户</AlertDialogTitle>
              <AlertDialogDescription>
                确定要删除账户 <span className="font-semibold text-foreground">{deletingAccount?.email}</span> 吗？
                <br />
                <br />
                此操作将删除该账户的所有邮件数据，且无法恢复。
              </AlertDialogDescription>
            </AlertDialogHeader>
            <AlertDialogFooter>
              <AlertDialogCancel onClick={handleDeleteCancel}>取消</AlertDialogCancel>
              <AlertDialogAction
                onClick={handleDeleteConfirm}
                className="bg-red-600 hover:bg-red-700 focus:ring-red-600"
              >
                删除
              </AlertDialogAction>
            </AlertDialogFooter>
          </AlertDialogContent>
        </AlertDialog>
      </div>
    </div>
  );
};