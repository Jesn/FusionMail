import { create } from 'zustand';

export interface Account {
  uid: string;
  email: string;
  provider: string;
  protocol: string;
  auth_type: string;
  sync_enabled: boolean;
  sync_interval: number;
  last_sync_at?: string;
  last_sync_status?: string;
  last_sync_error?: string;
  created_at: string;
  updated_at: string;
}

export interface AccountStats {
  total_count: number;
  unread_count: number;
  starred_count: number;
  archived_count: number;
}

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

  setAccounts: (accounts) => set({ accounts }),

  setSelectedAccount: (account) => set({ selectedAccount: account }),

  addAccount: (account) => set((state) => ({
    accounts: [...state.accounts, account],
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
