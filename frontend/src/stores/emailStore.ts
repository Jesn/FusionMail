import { create } from 'zustand';
import type { Email, EmailDetail } from '../types';

export interface EmailFilter {
  account_uid?: string;
  group_id?: number; // 分组 ID：-1 表示所有账号，0 表示未分组，>0 表示具体分组，undefined 表示不过滤
  is_read?: boolean;
  is_starred?: boolean;
  is_archived?: boolean;
  is_deleted?: boolean;
  is_spam?: boolean;
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
  selectedEmail: EmailDetail | null;
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
  spamCount: number;
  
  // Actions
  setEmails: (response: EmailListResponse) => void;
  setSelectedEmail: (email: EmailDetail | null) => void;
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
  setSpamCount: (count: number) => void;

  // 邮件操作
  updateEmailStatus: (id: number, updates: Partial<Email>) => void;
  removeEmail: (id: number) => void;
  removeEmailsByAccount: (accountUid: string) => void;
  markAllAsRead: (accountUid?: string) => void;
  
  // 重置
  reset: () => void;
}

import { getCachedSettings } from '../utils/settingsCache';

// 从设置缓存读取 pageSize
const getCachedPageSize = (): number => {
  try {
    const uiSettings = getCachedSettings('ui');
    if (uiSettings?.email_page_size) {
      const pageSize = parseInt(uiSettings.email_page_size, 10);
      if (!isNaN(pageSize) && pageSize > 0) {
        console.log('从设置缓存读取 pageSize:', pageSize);
        return pageSize;
      }
    }
  } catch (error) {
    console.error('读取 pageSize 缓存失败:', error);
  }

  // 返回默认值
  console.log('使用默认 pageSize: 20');
  return 20;
};

const initialState = {
  emails: [],
  selectedEmail: null,
  total: 0,
  page: 1,
  pageSize: getCachedPageSize(), // 从缓存读取或使用默认值 20
  totalPages: 0,
  filter: {
    is_archived: false,
    is_deleted: false,
    is_spam: false, // 默认不显示垃圾邮件
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
  spamCount: 0,
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

  setSpamCount: (count) => set({ spamCount: count }),

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

  removeEmailsByAccount: (accountUid) => set((state) => {
    const removedEmails = state.emails.filter((email) => email.account_uid === accountUid);
    const removedCount = removedEmails.length;
    
    return {
      emails: state.emails.filter((email) => email.account_uid !== accountUid),
      selectedEmail: state.selectedEmail?.account_uid === accountUid ? null : state.selectedEmail,
      total: Math.max(0, state.total - removedCount),
    };
  }),

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
