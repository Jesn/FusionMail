import { create } from 'zustand';
import { persist } from 'zustand/middleware';
import type { Account, AccountStats } from '../types';

// 缓存过期时间：10分钟（增加到10分钟，减少重复请求）
export const ACCOUNT_CACHE_TTL = 10 * 60 * 1000;

// 辅助函数：检查缓存是否过期
export const isAccountCacheExpired = (cacheTimestamp: number | null): boolean => {
  if (!cacheTimestamp) return true;
  return Date.now() - cacheTimestamp > ACCOUNT_CACHE_TTL;
};

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

export const useAccountStore = create<AccountState>()(
  persist(
    (set) => ({
      ...initialState,

      setAccounts: (accounts) => set({
        accounts: accounts.sort((a, b) => a.email.localeCompare(b.email)),
        hasLoaded: true,
        cacheTimestamp: Date.now(), // 设置缓存时间戳
      }),

      setSelectedAccount: (account) => set({ selectedAccount: account }),

      addAccount: (account) => set((state) => ({
        accounts: [...state.accounts, account].sort((a, b) => a.email.localeCompare(b.email)),
        cacheTimestamp: Date.now(), // 更新缓存时间戳
      })),

      updateAccount: (uid, updates) => set((state) => ({
        accounts: state.accounts.map((account) =>
          account.uid === uid ? { ...account, ...updates } : account
        ),
        selectedAccount: state.selectedAccount?.uid === uid
          ? { ...state.selectedAccount, ...updates }
          : state.selectedAccount,
        cacheTimestamp: Date.now(), // 更新缓存时间戳
      })),

      removeAccount: (uid) => set((state) => ({
        accounts: state.accounts.filter((account) => account.uid !== uid),
        selectedAccount: state.selectedAccount?.uid === uid ? null : state.selectedAccount,
        cacheTimestamp: Date.now(), // 更新缓存时间戳
      })),

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
    }),
    {
      name: 'fusionmail-accounts', // localStorage 键名
      version: 1, // 版本号
      // 只持久化必要的数据，排除瞬时状态
      partialize: (state) => ({
        accounts: state.accounts,
        hasLoaded: state.hasLoaded,
        cacheTimestamp: state.cacheTimestamp,
        accountStats: state.accountStats,
      }),
    }
  )
);
