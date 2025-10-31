import { useState, useCallback, useMemo } from 'react';
import { Account } from '../types';

interface UseAccountSelectionOptions {
  accounts: Account[];
  filteredAccounts: Account[];
}

export const useAccountSelection = ({ accounts, filteredAccounts }: UseAccountSelectionOptions) => {
  const [selectedAccountUids, setSelectedAccountUids] = useState<string[]>([]);

  // 选择/取消选择单个账户
  const toggleAccountSelection = useCallback((uid: string, selected: boolean) => {
    setSelectedAccountUids(prev => {
      if (selected) {
        return prev.includes(uid) ? prev : [...prev, uid];
      } else {
        return prev.filter(id => id !== uid);
      }
    });
  }, []);

  // 全选当前筛选结果
  const selectAll = useCallback(() => {
    const allFilteredUids = filteredAccounts.map(acc => acc.uid);
    setSelectedAccountUids(allFilteredUids);
  }, [filteredAccounts]);

  // 取消全选
  const clearSelection = useCallback(() => {
    setSelectedAccountUids([]);
  }, []);

  // 反选当前筛选结果
  const invertSelection = useCallback(() => {
    const filteredUids = filteredAccounts.map(acc => acc.uid);
    const newSelection = filteredUids.filter(uid => !selectedAccountUids.includes(uid));
    setSelectedAccountUids(newSelection);
  }, [filteredAccounts, selectedAccountUids]);

  // 选择特定状态的账户
  const selectByStatus = useCallback((status: string) => {
    const statusAccountUids = filteredAccounts
      .filter(acc => acc.status === status)
      .map(acc => acc.uid);
    setSelectedAccountUids(statusAccountUids);
  }, [filteredAccounts]);

  // 选择特定服务商的账户
  const selectByProvider = useCallback((provider: string) => {
    const providerAccountUids = filteredAccounts
      .filter(acc => acc.provider === provider)
      .map(acc => acc.uid);
    setSelectedAccountUids(providerAccountUids);
  }, [filteredAccounts]);

  // 获取选中的账户对象
  const selectedAccounts = useMemo(() => {
    return accounts.filter(acc => selectedAccountUids.includes(acc.uid));
  }, [accounts, selectedAccountUids]);

  // 统计信息
  const selectionStats = useMemo(() => {
    const total = selectedAccountUids.length;
    const active = selectedAccounts.filter(acc => acc.status === 'active').length;
    const disabled = selectedAccounts.filter(acc => acc.status === 'disabled').length;
    const error = selectedAccounts.filter(acc => acc.status === 'error').length;
    
    return {
      total,
      active,
      disabled,
      error,
    };
  }, [selectedAccounts, selectedAccountUids.length]);

  // 检查是否全选
  const isAllSelected = useMemo(() => {
    if (filteredAccounts.length === 0) return false;
    return filteredAccounts.every(acc => selectedAccountUids.includes(acc.uid));
  }, [filteredAccounts, selectedAccountUids]);

  // 检查是否部分选择
  const isPartiallySelected = useMemo(() => {
    if (selectedAccountUids.length === 0) return false;
    return filteredAccounts.some(acc => selectedAccountUids.includes(acc.uid)) && !isAllSelected;
  }, [filteredAccounts, selectedAccountUids, isAllSelected]);

  return {
    // 选择状态
    selectedAccountUids,
    selectedAccounts,
    selectionStats,
    
    // 选择操作
    toggleAccountSelection,
    selectAll,
    clearSelection,
    invertSelection,
    selectByStatus,
    selectByProvider,
    
    // 状态检查
    isAllSelected,
    isPartiallySelected,
    hasSelection: selectedAccountUids.length > 0,
  };
};