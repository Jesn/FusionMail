import { useMemo } from 'react';
import { Account } from '../types';

export type SmartGroupBy = 'domain' | 'provider' | 'status' | 'sync_status' | 'usage_frequency' | 'none';

interface AccountGroup {
  key: string;
  label: string;
  accounts: Account[];
  count: number;
  errorCount: number;
  activeCount: number;
  icon: string;
  color: string;
}

interface UseSmartGroupingOptions {
  accounts: Account[];
  groupBy: SmartGroupBy;
}

export const useSmartGrouping = ({ accounts, groupBy }: UseSmartGroupingOptions) => {
  const groups = useMemo(() => {
    if (groupBy === 'none') {
      return [{
        key: 'all',
        label: '所有账户',
        accounts,
        count: accounts.length,
        errorCount: accounts.filter(acc => acc.status === 'error').length,
        activeCount: accounts.filter(acc => acc.status === 'active').length,
        icon: '📧',
        color: 'text-blue-600',
      }];
    }

    const groupMap = new Map<string, Account[]>();

    accounts.forEach(account => {
      let groupKey: string;
      
      switch (groupBy) {
        case 'domain':
          groupKey = account.email.split('@')[1] || 'unknown';
          break;
        case 'provider':
          groupKey = account.provider;
          break;
        case 'status':
          groupKey = account.status;
          break;
        case 'sync_status':
          groupKey = getSyncStatus(account);
          break;
        case 'usage_frequency':
          groupKey = getUsageFrequency(account);
          break;
        default:
          groupKey = 'all';
      }

      if (!groupMap.has(groupKey)) {
        groupMap.set(groupKey, []);
      }
      groupMap.get(groupKey)!.push(account);
    });

    return Array.from(groupMap.entries()).map(([key, accounts]) => {
      const group: AccountGroup = {
        key,
        label: getGroupLabel(groupBy, key),
        accounts: accounts.sort((a, b) => a.email.localeCompare(b.email)),
        count: accounts.length,
        errorCount: accounts.filter(acc => acc.status === 'error').length,
        activeCount: accounts.filter(acc => acc.status === 'active').length,
        icon: getGroupIcon(groupBy, key),
        color: getGroupColor(groupBy, key),
      };
      return group;
    }).sort((a, b) => {
      // 优先显示有问题的分组
      if (a.errorCount > 0 && b.errorCount === 0) return -1;
      if (a.errorCount === 0 && b.errorCount > 0) return 1;
      
      // 然后按账户数量降序
      return b.count - a.count;
    });
  }, [accounts, groupBy]);

  // 获取同步状态
  const getSyncStatus = (account: Account): string => {
    if (!account.last_sync_at) return 'never';
    if (account.last_sync_status === 'running') return 'running';
    if (account.last_sync_status === 'success') return 'success';
    if (account.last_sync_status === 'failed') return 'failed';
    return 'never';
  };

  // 获取使用频率（基于最后同步时间和邮件数量）
  const getUsageFrequency = (account: Account): string => {
    const now = new Date();
    const lastSync = account.last_sync_at ? new Date(account.last_sync_at) : null;
    const daysSinceSync = lastSync ? Math.floor((now.getTime() - lastSync.getTime()) / (1000 * 60 * 60 * 24)) : 999;
    
    if (daysSinceSync <= 1) return 'daily';
    if (daysSinceSync <= 7) return 'weekly';
    if (daysSinceSync <= 30) return 'monthly';
    return 'rarely';
  };

  // 获取分组标签
  const getGroupLabel = (groupBy: SmartGroupBy, key: string): string => {
    switch (groupBy) {
      case 'domain':
        return getDomainLabel(key);
      case 'provider':
        return getProviderLabel(key);
      case 'status':
        return getStatusLabel(key);
      case 'sync_status':
        return getSyncStatusLabel(key);
      case 'usage_frequency':
        return getUsageFrequencyLabel(key);
      default:
        return key;
    }
  };

  // 获取分组图标
  const getGroupIcon = (groupBy: SmartGroupBy, key: string): string => {
    switch (groupBy) {
      case 'domain':
        return getDomainIcon(key);
      case 'provider':
        return getProviderIcon(key);
      case 'status':
        return getStatusIcon(key);
      case 'sync_status':
        return getSyncStatusIcon(key);
      case 'usage_frequency':
        return getUsageFrequencyIcon(key);
      default:
        return '📧';
    }
  };

  // 获取分组颜色
  const getGroupColor = (groupBy: SmartGroupBy, key: string): string => {
    switch (groupBy) {
      case 'status':
        return getStatusColor(key);
      case 'sync_status':
        return getSyncStatusColor(key);
      default:
        return 'text-blue-600';
    }
  };

  return {
    groups,
    totalGroups: groups.length,
    totalAccounts: accounts.length,
  };
};

// 辅助函数
function getDomainLabel(domain: string): string {
  const commonDomains: Record<string, string> = {
    'gmail.com': 'Gmail 邮箱',
    'outlook.com': 'Outlook 邮箱',
    'hotmail.com': 'Hotmail 邮箱',
    'qq.com': 'QQ 邮箱',
    '163.com': '网易 163 邮箱',
    '126.com': '网易 126 邮箱',
    'sina.com': '新浪邮箱',
    'icloud.com': 'iCloud 邮箱',
    'yahoo.com': 'Yahoo 邮箱',
  };
  return commonDomains[domain] || `@${domain}`;
}

function getDomainIcon(domain: string): string {
  const domainIcons: Record<string, string> = {
    'gmail.com': '🔴',
    'outlook.com': '🔵',
    'hotmail.com': '🔵',
    'qq.com': '🐧',
    '163.com': '📮',
    '126.com': '📮',
    'sina.com': '📧',
    'icloud.com': '🍎',
    'yahoo.com': '🟣',
  };
  return domainIcons[domain] || '📧';
}

function getProviderLabel(provider: string): string {
  const labels: Record<string, string> = {
    gmail: 'Gmail',
    outlook: 'Outlook',
    imap: 'IMAP',
    pop3: 'POP3',
  };
  return labels[provider] || provider.toUpperCase();
}

function getProviderIcon(provider: string): string {
  const icons: Record<string, string> = {
    gmail: '🔴',
    outlook: '🔵',
    imap: '📧',
    pop3: '📬',
  };
  return icons[provider] || '📧';
}

function getStatusLabel(status: string): string {
  const labels: Record<string, string> = {
    active: '正常账户',
    disabled: '已禁用账户',
    error: '异常账户',
  };
  return labels[status] || status;
}

function getStatusIcon(status: string): string {
  const icons: Record<string, string> = {
    active: '✅',
    disabled: '⏸️',
    error: '❌',
  };
  return icons[status] || '📧';
}

function getStatusColor(status: string): string {
  switch (status) {
    case 'active':
      return 'text-green-600';
    case 'disabled':
      return 'text-gray-600';
    case 'error':
      return 'text-red-600';
    default:
      return 'text-gray-600';
  }
}

function getSyncStatusLabel(status: string): string {
  const labels: Record<string, string> = {
    success: '同步成功',
    failed: '同步失败',
    running: '同步中',
    never: '从未同步',
  };
  return labels[status] || status;
}

function getSyncStatusIcon(status: string): string {
  const icons: Record<string, string> = {
    success: '✅',
    failed: '❌',
    running: '🔄',
    never: '⏳',
  };
  return icons[status] || '📧';
}

function getSyncStatusColor(status: string): string {
  switch (status) {
    case 'success':
      return 'text-green-600';
    case 'failed':
      return 'text-red-600';
    case 'running':
      return 'text-blue-600';
    case 'never':
      return 'text-gray-600';
    default:
      return 'text-gray-600';
  }
}

function getUsageFrequencyLabel(frequency: string): string {
  const labels: Record<string, string> = {
    daily: '每日使用',
    weekly: '每周使用',
    monthly: '每月使用',
    rarely: '很少使用',
  };
  return labels[frequency] || frequency;
}

function getUsageFrequencyIcon(frequency: string): string {
  const icons: Record<string, string> = {
    daily: '🔥',
    weekly: '📅',
    monthly: '📆',
    rarely: '💤',
  };
  return icons[frequency] || '📧';
}