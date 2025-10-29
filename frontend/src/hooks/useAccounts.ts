import { useEffect, useCallback } from 'react';
import { useAccountStore } from '../stores/accountStore';
import { accountService, CreateAccountRequest, UpdateAccountRequest } from '../services/accountService';
import { toast } from 'sonner';

export const useAccounts = () => {
  const {
    accounts,
    selectedAccount,
    accountStats,
    isLoading,
    error,
    setAccounts,
    setSelectedAccount,
    addAccount,
    updateAccount,
    removeAccount,
    setAccountStats,
    setLoading,
    setError,
  } = useAccountStore();

  // 加载账户列表
  const loadAccounts = useCallback(async () => {
    try {
      setLoading(true);
      setError(null);
      const data = await accountService.getList();
      setAccounts(data);
    } catch (err) {
      const message = err instanceof Error ? err.message : '加载账户列表失败';
      setError(message);
      toast.error(message);
    } finally {
      setLoading(false);
    }
  }, [setAccounts, setLoading, setError]);

  // 加载账户详情
  const loadAccountDetail = useCallback(async (uid: string) => {
    try {
      const account = await accountService.getByUid(uid);
      setSelectedAccount(account);
    } catch (err) {
      const message = err instanceof Error ? err.message : '加载账户详情失败';
      toast.error(message);
    }
  }, [setSelectedAccount]);

  // 加载账户统计
  const loadAccountStats = useCallback(async (uid: string) => {
    try {
      const stats = await accountService.getByUid(uid);
      setAccountStats(uid, stats as any);
    } catch (err) {
      console.error('Failed to load account stats:', err);
    }
  }, [setAccountStats]);

  // 创建账户
  const createAccount = useCallback(async (data: CreateAccountRequest) => {
    try {
      setLoading(true);
      const account = await accountService.create(data);
      addAccount(account);
      toast.success('账户添加成功');
      return account;
    } catch (err) {
      const message = err instanceof Error ? err.message : '添加账户失败';
      toast.error(message);
      throw err;
    } finally {
      setLoading(false);
    }
  }, [addAccount, setLoading]);

  // 更新账户
  const updateAccountData = useCallback(async (uid: string, data: UpdateAccountRequest) => {
    try {
      setLoading(true);
      const account = await accountService.update(uid, data);
      updateAccount(uid, account);
      toast.success('账户更新成功');
      return account;
    } catch (err) {
      const message = err instanceof Error ? err.message : '更新账户失败';
      toast.error(message);
      throw err;
    } finally {
      setLoading(false);
    }
  }, [updateAccount, setLoading]);

  // 删除账户
  const deleteAccount = useCallback(async (uid: string) => {
    try {
      setLoading(true);
      await accountService.delete(uid);
      removeAccount(uid);
      toast.success('账户删除成功');
    } catch (err) {
      const message = err instanceof Error ? err.message : '删除账户失败';
      toast.error(message);
      throw err;
    } finally {
      setLoading(false);
    }
  }, [removeAccount, setLoading]);

  // 测试连接
  const testConnection = useCallback(async (uid: string) => {
    try {
      const result = await accountService.testConnection(uid);
      if (result.success) {
        toast.success('连接测试成功');
      } else {
        toast.error(result.message || '连接测试失败');
      }
      return result;
    } catch (err) {
      const message = err instanceof Error ? err.message : '连接测试失败';
      toast.error(message);
      throw err;
    }
  }, []);

  // 同步账户
  const syncAccount = useCallback(async (uid: string) => {
    try {
      await accountService.sync(uid);
      toast.success('同步已开始');
    } catch (err) {
      const message = err instanceof Error ? err.message : '同步失败';
      toast.error(message);
      throw err;
    }
  }, []);

  // 同步所有账户
  const syncAllAccounts = useCallback(async () => {
    try {
      await accountService.syncAll();
      toast.success('同步已开始');
    } catch (err) {
      const message = err instanceof Error ? err.message : '同步失败';
      toast.error(message);
      throw err;
    }
  }, []);

  // 初始加载
  useEffect(() => {
    loadAccounts();
  }, [loadAccounts]);

  return {
    // 状态
    accounts,
    selectedAccount,
    accountStats,
    isLoading,
    error,

    // 操作
    loadAccounts,
    loadAccountDetail,
    loadAccountStats,
    createAccount,
    updateAccount: updateAccountData,
    deleteAccount,
    testConnection,
    syncAccount,
    syncAllAccounts,
  };
};
