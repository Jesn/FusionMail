import { create } from 'zustand';
import type { Email } from '../types';

export interface EmailFilter {
  account_uid?: string;
  is_read?: boolean;
  is_starred?: boolean;
  is_archived?: boolean;
  is_deleted?: boolean;
  from_address?: string;
  subject?: string;
  start_date?: string;
  end_date?: string;
}

export interface EmailListResponse {
  emails: Email[];
  total: number;
  page: number;
  page_size: number;
  total_pages: number;
}

interface EmailState {
  // 邮件列表
  emails: Email[];
  selectedEmail: Email | null;
  total: number;
  page: number;
  pageSize: number;
  totalPages: number;
  
  // 筛选和搜索
  filter: EmailFilter;
  searchQuery: string;
  
  // 加载状态
  isLoading: boolean;
  isLoadingDetail: boolean;
  isFetchingStats: boolean; // 是否正在请求统计
  hasLoadedStats: boolean; // 是否已经加载过统计
  error: string | null;
  
  // 统计信息
  unreadCount: number;
  starredCount: number;
  archivedCount: number;
  deletedCount: number;
  
  // Actions
  setEmails: (response: EmailListResponse) => void;
  setSelectedEmail: (email: Email | null) => void;
  setFilter: (filter: EmailFilter) => void;
  setSearchQuery: (query: string) => void;
  setPage: (page: number) => void;
  setPageSize: (pageSize: number) => void;
  setLoading: (loading: boolean) => void;
  setLoadingDetail: (loading: boolean) => void;
  setFetchingStats: (fetching: boolean) => void;
  setHasLoadedStats: (hasLoaded: boolean) => void;
  setError: (error: string | null) => void;
  setUnreadCount: (count: number) => void;
  setStarredCount: (count: number) => void;
  setArchivedCount: (count: number) => void;
  setDeletedCount: (count: number) => void;
  
  // 邮件操作
  updateEmailStatus: (id: number, updates: Partial<Email>) => void;
  removeEmail: (id: number) => void;
  markAllAsRead: (accountUid?: string) => void;
  
  // 重置
  reset: () => void;
}

const initialState = {
  emails: [],
  selectedEmail: null,
  total: 0,
  page: 1,
  pageSize: 20,
  totalPages: 0,
  filter: {
    is_archived: false,
    is_deleted: false,
  },
  searchQuery: '',
  isLoading: false,
  isLoadingDetail: false,
  isFetchingStats: false,
  hasLoadedStats: false,
  error: null,
  unreadCount: 0,
  starredCount: 0,
  archivedCount: 0,
  deletedCount: 0,
};

export const useEmailStore = create<EmailState>((set) => ({
  ...initialState,

  setEmails: (response) => set({
    emails: response.emails,
    total: response.total,
    page: response.page,
    pageSize: response.page_size,
    totalPages: response.total_pages,
  }),

  setSelectedEmail: (email) => set({ selectedEmail: email }),

  setFilter: (filter) => set({ filter, page: 1 }),

  setSearchQuery: (query) => set({ searchQuery: query, page: 1 }),

  setPage: (page) => set({ page }),

  setPageSize: (pageSize) => set({ pageSize, page: 1 }),

  setLoading: (loading) => set({ isLoading: loading }),

  setLoadingDetail: (loading) => set({ isLoadingDetail: loading }),

  setFetchingStats: (fetching) => set({ isFetchingStats: fetching }),

  setHasLoadedStats: (hasLoaded) => set({ hasLoadedStats: hasLoaded }),

  setError: (error) => set({ error }),

  setUnreadCount: (count) => set({ unreadCount: count }),

  setStarredCount: (count) => set({ starredCount: count }),

  setArchivedCount: (count) => set({ archivedCount: count }),

  setDeletedCount: (count) => set({ deletedCount: count, hasLoadedStats: true }),

  updateEmailStatus: (id, updates) => set((state) => ({
    emails: state.emails.map((email) =>
      email.id === id ? { ...email, ...updates } : email
    ),
    selectedEmail: state.selectedEmail?.id === id
      ? { ...state.selectedEmail, ...updates }
      : state.selectedEmail,
  })),

  removeEmail: (id) => set((state) => ({
    emails: state.emails.filter((email) => email.id !== id),
    selectedEmail: state.selectedEmail?.id === id ? null : state.selectedEmail,
    total: Math.max(0, state.total - 1),
  })),

  markAllAsRead: (accountUid) => set((state) => ({
    emails: state.emails.map((email) => {
      // 如果指定了账号，只更新该账号的邮件
      if (accountUid && email.account_uid !== accountUid) {
        return email;
      }
      // 否则更新所有邮件
      return { ...email, is_read: true };
    }),
    selectedEmail: state.selectedEmail && (!accountUid || state.selectedEmail.account_uid === accountUid)
      ? { ...state.selectedEmail, is_read: true }
      : state.selectedEmail,
  })),

  reset: () => set(initialState),
}));
