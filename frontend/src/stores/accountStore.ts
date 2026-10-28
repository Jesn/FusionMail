import { create } from 'zustand';
import type { Account, AccountStats } from '../types';
// 缓存语义统一入口（P12）：业务数据内存态 + TTL，不跨刷新
import { ACCOUNT_CACHE_TTL, isCacheExpired } from '../utils/cachePolicy';

// 兼容导出：既有消费方（useAccounts.ts）沿用旧名
export const isAccountCacheExpired = (ts: number | null): boolean =>
  isCacheExpired(ts, ACCOUNT_CACHE_TTL);

interface AccountState {
  // 账户列表
  accounts: Account[];
  selectedAccount: Account | null;

  // 账户统计
  accountStats: Record<string, AccountStats>;

  // 加载状态
  isLoading: boolean;
  isFetching: boolean; // 是否正在请求中
  hasLoaded: boolean; // 是否已经加载过
  error: string | null;

  // 缓存时间戳
  cacheTimestamp: number | null; // 上次加载账户列表的时间戳

  // Actions
  setAccounts: (accounts: Account[]) => void;
  setSelectedAccount: (account: Account | null) => void;
  addAccount: (account: Account) => void;
  updateAccount: (uid: string, updates: Partial<Account>) => void;
  removeAccount: (uid: string) => void;
  setAccountStats: (uid: string, stats: AccountStats) => void;
  setLoading: (loading: boolean) => void;
  setFetching: (fetching: boolean) => void;
  setHasLoaded: (hasLoaded: boolean) => void;
  setError: (error: string | null) => void;
  setCacheTimestamp: (timestamp: number) => void;
  /** 作废缓存：下次 loadAccounts 必拉（见 utils/cacheInvalidation.ts） */
  invalidate: () => void;
  reset: () => void;

  // 分组相关
  getAccountsByGroupId: (groupId: number | null) => Account[];
  getUngroupedAccounts: () => Account[];
}

const initialState = {
  accounts: [],
  selectedAccount: null,
  accountStats: {},
  isLoading: false,
  isFetching: false,
  hasLoaded: false,
  error: null,
  cacheTimestamp: null,
};

// 业务数据不跨刷新（P12 决策，见 utils/cachePolicy.ts）：
// 移除 persist/localStorage，刷新后由 useAccounts 正常拉取
export const useAccountStore = create<AccountState>()((set) => ({
  ...initialState,

  // 全量服务端列表：唯一允许推进缓存时钟的入口
  setAccounts: (accounts) => set({
    accounts: accounts.sort((a, b) => a.email.localeCompare(b.email)),
    hasLoaded: true,
    cacheTimestamp: Date.now(),
  }),

  setSelectedAccount: (account) => set({ selectedAccount: account }),

  // 本地 patch：数据本身已正确，不推进缓存时钟
  // （推进等于给 TTL 续命，会让后续 fetch 跳过本该发生的重拉）
  addAccount: (account) => set((state) => ({
    accounts: [...state.accounts, account].sort((a, b) => a.email.localeCompare(b.email)),
  })),

  updateAccount: (uid, updates) => set((state) => ({
    accounts: state.accounts.map((account) =>
      account.uid === uid ? { ...account, ...updates } : account
    ),
    selectedAccount: state.selectedAccount?.uid === uid
      ? { ...state.selectedAccount, ...updates }
      : state.selectedAccount,
  })),

  // 结构性变更：列表已本地修正，但聚合计数（分组 account_count 等）陈旧，
  // 故置空时间戳让下次读取拿到服务端真相；同时丢弃该账户的统计缓存
  removeAccount: (uid) => set((state) => {
    const { [uid]: _removedStats, ...remainingStats } = state.accountStats;
    void _removedStats;
    return {
      accounts: state.accounts.filter((account) => account.uid !== uid),
      selectedAccount: state.selectedAccount?.uid === uid ? null : state.selectedAccount,
      accountStats: remainingStats,
      cacheTimestamp: null,
    };
  }),

  setAccountStats: (uid, stats) => set((state) => ({
    accountStats: {
      ...state.accountStats,
      [uid]: stats,
    },
  })),

  setLoading: (loading) => set({ isLoading: loading }),

  setFetching: (fetching) => set({ isFetching: fetching }),

  setHasLoaded: (hasLoaded) => set({ hasLoaded }),

  setError: (error) => set({ error }),

  setCacheTimestamp: (timestamp) => set({ cacheTimestamp: timestamp }),

  invalidate: () => set({ cacheTimestamp: null }),

  reset: () => set(initialState),

  // 根据分组 ID 获取账号列表
  getAccountsByGroupId: (groupId: number | null): Account[] => {
    const { accounts } = useAccountStore.getState();
    if (groupId === null) {
      // 返回未分组的账号
      return accounts.filter((account: Account) => !account.group_id);
    }
    return accounts.filter((account: Account) => account.group_id === groupId);
  },

  // 获取未分组的账号
  getUngroupedAccounts: (): Account[] => {
    const { accounts } = useAccountStore.getState();
    return accounts.filter((account: Account) => !account.group_id);
  },
}));
