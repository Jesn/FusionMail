import { create } from 'zustand';
import type { Email } from '../types';

export interface EmailFilter {
  account_uid?: string;
  is_read?: boolean;
  is_starred?: boolean;
  is_archived?: boolean;
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
  setError: (error: string | null) => void;
  setUnreadCount: (count: number) => void;
  setStarredCount: (count: number) => void;
  setArchivedCount: (count: number) => void;
  setDeletedCount: (count: number) => void;
  
  // 邮件操作
  updateEmailStatus: (id: number, updates: Partial<Email>) => void;
  removeEmail: (id: number) => void;
  
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
  filter: {},
  searchQuery: '',
  isLoading: false,
  isLoadingDetail: false,
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

  setError: (error) => set({ error }),

  setUnreadCount: (count) => set({ unreadCount: count }),

  setStarredCount: (count) => set({ starredCount: count }),

  setArchivedCount: (count) => set({ archivedCount: count }),

  setDeletedCount: (count) => set({ deletedCount: count }),

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
  })),

  reset: () => set(initialState),
}));
