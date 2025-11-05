import { create } from 'zustand';
import type { Account, AccountStats } from '../types';

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
  reset: () => void;
}

const initialState = {
  accounts: [],
  selectedAccount: null,
  accountStats: {},
  isLoading: false,
  isFetching: false,
  hasLoaded: false,
  error: null,
};

export const useAccountStore = create<AccountState>((set) => ({
  ...initialState,

  setAccounts: (accounts) => set({ 
    accounts: accounts.sort((a, b) => a.email.localeCompare(b.email)),
    hasLoaded: true,
  }),

  setSelectedAccount: (account) => set({ selectedAccount: account }),

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

  removeAccount: (uid) => set((state) => ({
    accounts: state.accounts.filter((account) => account.uid !== uid),
    selectedAccount: state.selectedAccount?.uid === uid ? null : state.selectedAccount,
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

  reset: () => set(initialState),
}));
