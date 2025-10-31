import { useState, useMemo, useCallback } from 'react';
import { Account } from '../types';

interface UseEnhancedSearchOptions {
  accounts: Account[];
  debounceMs?: number;
}

interface SearchFilters {
  query: string;
  statusFilter: string;
  providerFilter: string;
  syncStatusFilter: string;
}

export const useEnhancedSearch = ({ 
  accounts, 
  debounceMs = 300 
}: UseEnhancedSearchOptions) => {
  const [filters, setFilters] = useState<SearchFilters>({
    query: '',
    statusFilter: 'all',
    providerFilter: 'all',
    syncStatusFilter: 'all',
  });

  const [searchHistory, setSearchHistory] = useState<string[]>([]);

  // 实时搜索和筛选
  const filteredAccounts = useMemo(() => {
    return accounts.filter((account) => {
      // 搜索筛选 - 支持多字段匹配
      if (filters.query) {
        const query = filters.query.toLowerCase();
        const emailMatch = account.email.toLowerCase().includes(query);
        const domainMatch = account.email.split('@')[1]?.toLowerCase().includes(query);
        const providerMatch = account.provider.toLowerCase().includes(query);
        
        // 支持备注搜索（如果有备注字段）
        // const noteMatch = account.note?.toLowerCase().includes(query);
        
        if (!emailMatch && !domainMatch && !providerMatch) {
          return false;
        }
      }

      // 状态筛选
      if (filters.statusFilter !== 'all' && account.status !== filters.statusFilter) {
        return false;
      }

      // 服务商筛选
      if (filters.providerFilter !== 'all' && account.provider !== filters.providerFilter) {
        return false;
      }

      // 同步状态筛选
      if (filters.syncStatusFilter !== 'all') {
        const syncStatus = getSyncStatus(account);
        if (syncStatus !== filters.syncStatusFilter) {
          return false;
        }
      }

      return true;
    });
  }, [accounts, filters]);

  // 获取同步状态
  const getSyncStatus = (account: Account): string => {
    if (!account.last_sync_at) return 'never';
    if (account.last_sync_status === 'running') return 'running';
    if (account.last_sync_status === 'success') return 'success';
    if (account.last_sync_status === 'failed') return 'failed';
    return 'never';
  };

  // 搜索建议
  const searchSuggestions = useMemo(() => {
    if (!filters.query || filters.query.length < 2) return [];

    const suggestions = new Set<string>();
    const query = filters.query.toLowerCase();

    accounts.forEach(account => {
      // 邮箱地址建议
      if (account.email.toLowerCase().includes(query)) {
        suggestions.add(account.email);
      }

      // 域名建议
      const domain = account.email.split('@')[1];
      if (domain?.toLowerCase().includes(query)) {
        suggestions.add(`@${domain}`);
      }

      // 服务商建议
      if (account.provider.toLowerCase().includes(query)) {
        suggestions.add(account.provider);
      }
    });

    return Array.from(suggestions).slice(0, 5);
  }, [accounts, filters.query]);

  // 更新搜索查询
  const updateQuery = useCallback((query: string) => {
    setFilters(prev => ({ ...prev, query }));
    
    // 添加到搜索历史
    if (query && query.length > 2 && !searchHistory.includes(query)) {
      setSearchHistory(prev => [query, ...prev.slice(0, 9)]); // 保留最近10条
    }
  }, [searchHistory]);

  // 更新筛选器
  const updateStatusFilter = useCallback((status: string) => {
    setFilters(prev => ({ ...prev, statusFilter: status }));
  }, []);

  const updateProviderFilter = useCallback((provider: string) => {
    setFilters(prev => ({ ...prev, providerFilter: provider }));
  }, []);

  const updateSyncStatusFilter = useCallback((syncStatus: string) => {
    setFilters(prev => ({ ...prev, syncStatusFilter: syncStatus }));
  }, []);

  // 清除所有筛选
  const clearAllFilters = useCallback(() => {
    setFilters({
      query: '',
      statusFilter: 'all',
      providerFilter: 'all',
      syncStatusFilter: 'all',
    });
  }, []);

  // 获取筛选统计
  const filterStats = useMemo(() => {
    const total = accounts.length;
    const filtered = filteredAccounts.length;
    const activeFilters = [
      filters.query && 'search',
      filters.statusFilter !== 'all' && 'status',
      filters.providerFilter !== 'all' && 'provider',
      filters.syncStatusFilter !== 'all' && 'sync',
    ].filter(Boolean);

    return {
      total,
      filtered,
      activeFiltersCount: activeFilters.length,
      hasActiveFilters: activeFilters.length > 0,
    };
  }, [accounts.length, filteredAccounts.length, filters]);

  return {
    // 筛选结果
    filteredAccounts,
    
    // 筛选器状态
    filters,
    
    // 更新方法
    updateQuery,
    updateStatusFilter,
    updateProviderFilter,
    updateSyncStatusFilter,
    clearAllFilters,
    
    // 搜索功能
    searchSuggestions,
    searchHistory,
    
    // 统计信息
    filterStats,
  };
};