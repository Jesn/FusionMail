import { useEffect, useCallback, useRef } from 'react';
import { useAccountStore } from '../stores/accountStore';
import { accountService, CreateAccountRequest, UpdateAccountRequest } from '../services/accountService';
import { toast } from 'sonner';

export const useAccounts = () => {
  const store = useAccountStore();
  const {
    accounts,
    selectedAccount,
    accountStats,
    isLoading,
    error,
  } = store;

  // 使用 ref 保存 store actions，避免依赖项变化
  const storeRef = useRef(store);
  storeRef.current = store;

  // 防止重复请求
  const fetchingRef = useRef(false);

  // 加载账户列表
  const loadAccounts = useCallback(async () => {
    // 如果正在请求，直接返回
    if (fetchingRef.current) {
      return;
    }

    const { setLoading, setError, setAccounts } = storeRef.current;
    try {
      fetchingRef.current = true;
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
      fetchingRef.current = false;
    }
  }, []);

  // 加载账户详情
  const loadAccountDetail = useCallback(async (uid: string) => {
    const { setSelectedAccount } = storeRef.current;
    try {
      const account = await accountService.getByUid(uid);
      setSelectedAccount(account);
    } catch (err) {
      const message = err instanceof Error ? err.message : '加载账户详情失败';
      toast.error(message);
    }
  }, []);

  // 加载账户统计
  const loadAccountStats = useCallback(async (uid: string) => {
    const { setAccountStats } = storeRef.current;
    try {
      const stats = await accountService.getByUid(uid);
      setAccountStats(uid, stats as any);
    } catch (err) {
      console.error('Failed to load account stats:', err);
    }
  }, []);

  // 创建账户
  const createAccount = useCallback(async (data: CreateAccountRequest) => {
    const { setLoading, addAccount } = storeRef.current;
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
  }, []);

  // 更新账户
  const updateAccountData = useCallback(async (uid: string, data: UpdateAccountRequest) => {
    const { setLoading, updateAccount } = storeRef.current;
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
  }, []);

  // 删除账户
  const deleteAccount = useCallback(async (uid: string) => {
    const { setLoading, removeAccount } = storeRef.current;
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
  }, []);

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
      
      // 延迟刷新账户列表以获取最新状态
      setTimeout(() => {
        loadAccounts();
      }, 2000);
    } catch (err) {
      const message = err instanceof Error ? err.message : '同步失败';
      toast.error(message);
      throw err;
    }
  }, [loadAccounts]);

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

  // 清除同步错误
  const clearSyncError = useCallback(async (uid: string) => {
    try {
      await accountService.clearSyncError(uid);
      toast.success('错误状态已清除');
      // 刷新账户列表
      await loadAccounts();
    } catch (err) {
      const message = err instanceof Error ? err.message : '清除错误状态失败';
      toast.error(message);
      throw err;
    }
  }, [loadAccounts]);

  // 切换账户状态
  const toggleAccountStatus = useCallback(async (uid: string, currentStatus: string) => {
    const { setLoading } = storeRef.current;
    try {
      setLoading(true);
      if (currentStatus === 'active') {
        await accountService.disable(uid);
        toast.success('账户已禁用');
      } else {
        await accountService.enable(uid);
        toast.success('账户已启用');
      }
      // 重新加载账户列表以获取最新状态
      await loadAccounts();
    } catch (err) {
      const message = err instanceof Error ? err.message : '状态切换失败';
      toast.error(message);
      throw err;
    } finally {
      setLoading(false);
    }
  }, [loadAccounts]);

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
    toggleAccountStatus,
    clearSyncError,
  };
};
