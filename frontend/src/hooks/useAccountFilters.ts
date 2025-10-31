import { useState, useMemo } from 'react';
import { Account } from '../types';
import type { AccountStatus, AccountProvider } from '../components/account/AccountToolbar';

interface UseAccountFiltersOptions {
  accounts: Account[];
}

export const useAccountFilters = ({ accounts }: UseAccountFiltersOptions) => {
  const [searchQuery, setSearchQuery] = useState('');
  const [statusFilter, setStatusFilter] = useState<AccountStatus>('all');
  const [providerFilter, setProviderFilter] = useState<AccountProvider>('all');

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

  // 重置筛选
  const resetFilters = () => {
    setSearchQuery('');
    setStatusFilter('all');
    setProviderFilter('all');
  };

  // 检查是否有活动筛选
  const hasActiveFilters = searchQuery !== '' || statusFilter !== 'all' || providerFilter !== 'all';

  return {
    // 筛选状态
    searchQuery,
    setSearchQuery,
    statusFilter,
    setStatusFilter,
    providerFilter,
    setProviderFilter,
    
    // 筛选结果
    filteredAccounts,
    
    // 统计信息
    stats,
    
    // 工具方法
    resetFilters,
    hasActiveFilters,
  };
};