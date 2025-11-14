import { useState, useMemo } from 'react';
import { Account } from '../types';
import type { AccountStatus, AccountProvider, SyncStatus } from '../components/account/AccountToolbar';

interface UseAccountFiltersOptions {
  accounts: Account[];
}

export const useAccountFilters = ({ accounts }: UseAccountFiltersOptions) => {
  const [searchQuery, setSearchQuery] = useState('');
  const [statusFilter, setStatusFilter] = useState<AccountStatus>('all');
  const [providerFilter, setProviderFilter] = useState<AccountProvider>('all');
  const [syncStatusFilter, setSyncStatusFilter] = useState<SyncStatus>('all');

  // 计算统计信息
  const stats = useMemo(() => {
    const total = accounts.length;
    const active = accounts.filter(acc => acc.status === 'active').length;
    const disabled = accounts.filter(acc => acc.status === 'disabled').length;
    const error = accounts.filter(acc => acc.status === 'error').length;
    
    // 按服务商统计
    const providerStats = accounts.reduce((acc, account) => {
      acc[account.provider] = (acc[account.provider] || 0) + 1;
      return acc;
    }, {} as Record<string, number>);

    return {
      total,
      active,
      disabled,
      error,
      providerStats,
    };
  }, [accounts]);

  // 筛选后的账户
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

      // 状态筛选（包括软删除）
      if (statusFilter !== 'all') {
        if (statusFilter === 'deleted') {
          // 特殊处理：筛选已删除的账号（通过 deleted_at 字段）
          if (!account.deleted_at) {
            return false;
          }
        } else {
          // 筛选正常状态的账号
          if (account.status !== statusFilter || account.deleted_at) {
            return false;
          }
        }
      }

      // 服务商筛选
      if (providerFilter !== 'all' && account.provider !== providerFilter) {
        return false;
      }

      // 同步状态筛选
      if (syncStatusFilter !== 'all') {
        const syncStatus = getSyncStatus(account);
        if (syncStatus !== syncStatusFilter) {
          return false;
        }
      }

      return true;
    });
  }, [accounts, searchQuery, statusFilter, providerFilter, syncStatusFilter]);

  // 获取同步状态
  const getSyncStatus = (account: Account): SyncStatus => {
    if (!account.last_sync_at) return 'never';
    if (account.last_sync_status === 'running') return 'running';
    if (account.last_sync_status === 'success') return 'success';
    if (account.last_sync_status === 'failed') return 'failed';
    return 'never';
  };

  // 重置筛选
  const resetFilters = () => {
    setSearchQuery('');
    setStatusFilter('all');
    setProviderFilter('all');
    setSyncStatusFilter('all');
  };

  // 检查是否有活动筛选
  const hasActiveFilters = searchQuery !== '' || statusFilter !== 'all' || providerFilter !== 'all' || syncStatusFilter !== 'all';

  return {
    // 筛选状态
    searchQuery,
    setSearchQuery,
    statusFilter,
    setStatusFilter,
    providerFilter,
    setProviderFilter,
    syncStatusFilter,
    setSyncStatusFilter,
    
    // 筛选结果
    filteredAccounts,
    
    // 统计信息
    stats,
    
    // 工具方法
    resetFilters,
    hasActiveFilters,
  };
};