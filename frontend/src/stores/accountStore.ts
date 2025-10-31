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
  error: string | null;
  
  // Actions
  setAccounts: (accounts: Account[]) => void;
  setSelectedAccount: (account: Account | null) => void;
  addAccount: (account: Account) => void;
  updateAccount: (uid: string, updates: Partial<Account>) => void;
  removeAccount: (uid: string) => void;
  setAccountStats: (uid: string, stats: AccountStats) => void;
  setLoading: (loading: boolean) => void;
  setError: (error: string | null) => void;
  reset: () => void;
}

const initialState = {
  accounts: [],
  selectedAccount: null,
  accountStats: {},
  isLoading: false,
  error: null,
};

export const useAccountStore = create<AccountState>((set) => ({
  ...initialState,

  setAccounts: (accounts) => set({ 
    accounts: accounts.sort((a, b) => a.email.localeCompare(b.email))
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

  setError: (error) => set({ error }),

  reset: () => set(initialState),
}));
