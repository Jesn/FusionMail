import { useEffect, useCallback, useRef } from 'react';
import { useAccountStore, isAccountCacheExpired } from '../stores/accountStore';
import { useAuthStore } from '../stores/authStore';
import { useEmailStore } from '../stores/emailStore';
import { accountService, CreateAccountRequest, UpdateAccountRequest } from '../services/accountService';
import { emailService } from '../services/emailService';
import { toast } from 'sonner';

export const useAccounts = () => {
  const store = useAccountStore();
  const isAuthenticated = useAuthStore(state => state.isAuthenticated);
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

  // 加载账户列表（仅有效账户）
  const loadAccounts = useCallback(async (force = false) => {
    // 从 store 获取最新状态
    const currentStore = useAccountStore.getState();

    // 如果正在请求，直接返回
    if (currentStore.isFetching) {
      return;
    }

    // 如果不是强制刷新且缓存未过期，使用缓存
    if (!force && currentStore.hasLoaded && !isAccountCacheExpired(currentStore.cacheTimestamp)) {
      return;
    }

    const { setLoading, setFetching, setError, setAccounts } = currentStore;

    try {
      setFetching(true);
      setLoading(true);
      setError(null);
      // 只获取有效账户
      const data = await accountService.getList();

      setAccounts(data);
    } catch (err) {
      const message = err instanceof Error ? err.message : '加载账户列表失败';
      setError(message);
      toast.error(message);
    } finally {
      setLoading(false);
      setFetching(false);
    }
  }, []);

  // 加载回收站账户列表
  const loadTrashAccounts = useCallback(async () => {
    const { setLoading, setError } = storeRef.current;
    try {
      setLoading(true);
      setError(null);
      const data = await accountService.getTrashList();
      return data;
    } catch (err) {
      const message = err instanceof Error ? err.message : '加载回收站失败';
      setError(message);
      toast.error(message);
      return [];
    } finally {
      setLoading(false);
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
    const { setLoading } = storeRef.current;
    try {
      setLoading(true);
      const account = await accountService.create(data);
      // 创建成功后重新加载列表
      await loadAccounts(true);
      toast.success('账户添加成功');
      return account;
    } catch (err) {
      const message = err instanceof Error ? err.message : '添加账户失败';
      toast.error(message);
      throw err;
    } finally {
      setLoading(false);
    }
  }, [loadAccounts]);

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
    
    // 获取要删除的账号信息
    const account = accounts?.find(acc => acc.uid === uid);
    const unreadCount = account?.unread_count || 0;
    
    try {
      setLoading(true);
      await accountService.delete(uid);
      
      // 软删除后从有效账号列表中移除
      removeAccount(uid);
      
      // 从邮件列表中移除该账号的所有邮件
      const emailStore = useEmailStore.getState();
      emailStore.removeEmailsByAccount(uid);
      
      // 重新获取邮件统计数据以更新未读计数
      try {
        const stats = await emailService.getGlobalStats();
        emailStore.setUnreadCount(stats.unread_count);
      } catch (error) {
        console.error('Failed to update email stats after account deletion:', error);
      }
      
      toast.success('账户已移入回收站');
    } catch (err) {
      const message = err instanceof Error ? err.message : '删除账户失败';
      toast.error(message);
      throw err;
    } finally {
      setLoading(false);
    }
  }, [accounts]);

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
        loadAccounts(true); // 强制刷新
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
      await loadAccounts(true); // 强制刷新
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
      await loadAccounts(true); // 强制刷新
    } catch (err) {
      const message = err instanceof Error ? err.message : '状态切换失败';
      toast.error(message);
      throw err;
    } finally {
      setLoading(false);
    }
  }, [loadAccounts]);

  // 恢复账户
  const restoreAccount = useCallback(async (uid: string) => {
    const { setLoading } = storeRef.current;
    try {
      setLoading(true);
      await accountService.restore(uid);
      
      // 重新加载账号列表（只加载有效账号）
      await loadAccounts(true);
      
      // 重新获取邮件统计数据以更新未读计数
      try {
        const stats = await emailService.getGlobalStats();
        const emailStore = useEmailStore.getState();
        emailStore.setUnreadCount(stats.unread_count);
      } catch (error) {
        console.error('Failed to update email stats after account restoration:', error);
      }
      
      toast.success('账户已恢复');
    } catch (err) {
      const message = err instanceof Error ? err.message : '恢复账户失败';
      toast.error(message);
      throw err;
    } finally {
      setLoading(false);
    }
  }, [loadAccounts]);

  // 永久删除账户
  const forceDeleteAccount = useCallback(async (uid: string) => {
    const { setLoading, removeAccount } = storeRef.current;
    try {
      setLoading(true);
      await accountService.forceDelete(uid);
      // 永久删除后从 store 中移除
      removeAccount(uid);
      
      // 重新获取邮件统计数据以更新未读计数
      try {
        const stats = await emailService.getGlobalStats();
        const emailStore = useEmailStore.getState();
        emailStore.setUnreadCount(stats.unread_count);
      } catch (error) {
        console.error('Failed to update email stats after account force deletion:', error);
      }
      
      toast.success('账户已永久删除');
    } catch (err: any) {
      // 如果是 404 错误，说明后端已经没有这个账号了，前端也应该移除
      if (err.response?.status === 404 || err.message?.includes('404')) {
        removeAccount(uid);
        toast.success('账户已永久删除');
        return;
      }

      const message = err instanceof Error ? err.message : '永久删除账户失败';
      toast.error(message);
      throw err;
    } finally {
      setLoading(false);
    }
  }, []);

  // 初始加载（只在登录后）
  useEffect(() => {
    if (isAuthenticated) {
      loadAccounts();
    }
  }, [isAuthenticated, loadAccounts]);

  return {
    // 状态
    accounts,
    selectedAccount,
    accountStats,
    isLoading,
    error,

    // 操作
    loadAccounts,
    loadTrashAccounts,
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
    restoreAccount,
    forceDeleteAccount,
  };
};
