import { useState, useEffect, useCallback } from 'react';
import { api } from '../services/api';

export interface SyncStatusResponse {
  account_uid: string;
  account_name: string;
  provider: string;
  status: 'idle' | 'syncing' | 'failed';
  last_sync_time?: string;
  next_sync_time?: string;
  sync_interval: number;
  error_message: string;
  email_count: number;
  unread_count: number;
}

export interface SyncStatus {
  isRunning: boolean;
  currentAccount?: string;
  progress: number;
  totalAccounts: number;
  completedAccounts: number;
  lastSyncTime?: string;
  error?: string;
  accounts: SyncStatusResponse[];
}

export interface SyncLog {
  id: number;
  account_uid: string;
  account_name: string;
  provider: string;
  status: 'success' | 'failed';
  start_time: string;
  end_time?: string;
  duration: number;
  emails_new: number;
  emails_updated: number;
  error_message?: string;
}

export const useSync = () => {
  const [syncStatus, setSyncStatus] = useState<SyncStatus>({
    isRunning: false,
    progress: 0,
    totalAccounts: 0,
    completedAccounts: 0,
    accounts: [],
  });

  const [syncLogs, setSyncLogs] = useState<SyncLog[]>([]);
  const [isLoadingLogs, setIsLoadingLogs] = useState(false);

  // 获取同步状态
  const fetchSyncStatus = useCallback(async () => {
    try {
      const response = await api.get<{ success: boolean; data: SyncStatusResponse[] }>('/sync/status');
      if (response.success && response.data) {
        const accounts = response.data;
        const runningAccounts = accounts.filter(acc => acc.status === 'syncing');
        const isRunning = runningAccounts.length > 0;
        const totalAccounts = accounts.length;
        const completedAccounts = accounts.filter(acc => acc.status !== 'syncing').length;
        
        // 找到最近的同步时间
        const lastSyncTimes = accounts
          .map(acc => acc.last_sync_time)
          .filter(time => time)
          .sort((a, b) => new Date(b!).getTime() - new Date(a!).getTime());
        
        const lastSyncTime = lastSyncTimes[0];
        
        // 找到错误信息
        const errorAccount = accounts.find(acc => acc.status === 'failed' && acc.error_message);
        const error = errorAccount?.error_message;
        
        const currentAccount = runningAccounts[0]?.account_name;
        
        setSyncStatus({
          isRunning,
          currentAccount,
          progress: totalAccounts > 0 ? (completedAccounts / totalAccounts) * 100 : 0,
          totalAccounts,
          completedAccounts,
          lastSyncTime,
          error,
          accounts,
        });
      }
    } catch (error) {
      console.error('获取同步状态失败:', error);
    }
  }, []);

  // 获取同步日志
  const fetchSyncLogs = useCallback(async (limit = 20) => {
    setIsLoadingLogs(true);
    try {
      const response = await api.get<{ success: boolean; data: SyncLog[] }>('/sync/logs', {
        params: { limit }
      });
      if (response.success) {
        setSyncLogs(response.data);
      }
    } catch (error) {
      console.error('获取同步日志失败:', error);
    } finally {
      setIsLoadingLogs(false);
    }
  }, []);

  // 触发手动同步
  const triggerSync = useCallback(async (accountUid?: string) => {
    try {
      const endpoint = accountUid ? `/accounts/${accountUid}/sync` : '/sync/trigger';
      const response = await api.post<{ success: boolean; message: string }>(endpoint);
      if (response.success) {
        // 立即更新状态
        setSyncStatus(prev => ({ ...prev, isRunning: true }));
        // 延迟获取最新状态
        setTimeout(fetchSyncStatus, 1000);
      }
      return response;
    } catch (error) {
      console.error('触发同步失败:', error);
      throw error;
    }
  }, [fetchSyncStatus]);

  // 停止同步
  const stopSync = useCallback(async () => {
    try {
      const response = await api.post<{ success: boolean; message: string }>('/sync/stop');
      if (response.success) {
        setSyncStatus(prev => ({ ...prev, isRunning: false }));
      }
      return response;
    } catch (error) {
      console.error('停止同步失败:', error);
      throw error;
    }
  }, []);

  // 定期更新同步状态
  useEffect(() => {
    fetchSyncStatus();
    fetchSyncLogs();

    const interval = setInterval(() => {
      fetchSyncStatus();
    }, 5000); // 每5秒更新一次

    return () => clearInterval(interval);
  }, [fetchSyncStatus, fetchSyncLogs]);

  return {
    syncStatus,
    syncLogs,
    isLoadingLogs,
    fetchSyncStatus,
    fetchSyncLogs,
    triggerSync,
    stopSync,
  };
};