import { create } from 'zustand';
import { persist } from 'zustand/middleware';

interface UIState {
  // 侧边栏
  sidebarCollapsed: boolean;
  
  // 主题
  theme: 'light' | 'dark' | 'system';
  
  // 邮件列表视图
  emailListView: 'comfortable' | 'compact';
  
  // 对话框状态
  isAccountDialogOpen: boolean;
  isRuleDialogOpen: boolean;
  isWebhookDialogOpen: boolean;
  
  // 同步状态
  isSyncing: boolean;
  syncProgress: number;
  
  // Actions
  toggleSidebar: () => void;
  setSidebarCollapsed: (collapsed: boolean) => void;
  setTheme: (theme: 'light' | 'dark' | 'system') => void;
  setEmailListView: (view: 'comfortable' | 'compact') => void;
  setAccountDialogOpen: (open: boolean) => void;
  setRuleDialogOpen: (open: boolean) => void;
  setWebhookDialogOpen: (open: boolean) => void;
  setSyncing: (syncing: boolean) => void;
  setSyncProgress: (progress: number) => void;
}

export const useUIStore = create<UIState>()(
  persist(
    (set) => ({
      sidebarCollapsed: false,
      theme: 'system',
      emailListView: 'comfortable',
      isAccountDialogOpen: false,
      isRuleDialogOpen: false,
      isWebhookDialogOpen: false,
      isSyncing: false,
      syncProgress: 0,

      toggleSidebar: () => set((state) => ({ 
        sidebarCollapsed: !state.sidebarCollapsed 
      })),

      setSidebarCollapsed: (collapsed) => set({ sidebarCollapsed: collapsed }),

      setTheme: (theme) => set({ theme }),

      setEmailListView: (view) => set({ emailListView: view }),

      setAccountDialogOpen: (open) => set({ isAccountDialogOpen: open }),

      setRuleDialogOpen: (open) => set({ isRuleDialogOpen: open }),

      setWebhookDialogOpen: (open) => set({ isWebhookDialogOpen: open }),

      setSyncing: (syncing) => set({ isSyncing: syncing }),

      setSyncProgress: (progress) => set({ syncProgress: progress }),
    }),
    {
      name: 'ui-storage',
      partialize: (state) => ({
        sidebarCollapsed: state.sidebarCollapsed,
        theme: state.theme,
        emailListView: state.emailListView,
      }),
    }
  )
);
