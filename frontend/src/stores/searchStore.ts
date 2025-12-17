import { create } from 'zustand';
import { emailService } from '../services/emailService';
import { Email, PaginationParams } from '../types';

interface SearchState {
  // 搜索结果
  emails: Email[];
  total: number;
  isLoading: boolean;
  error: string | null;
  hasSearched: boolean;
  
  // 当前搜索参数
  currentQuery: string;
  currentAccountUid: string | undefined;
  currentPage: number;
  
  // Actions
  search: (params: SearchParams) => Promise<void>;
  loadMore: (page: number) => Promise<void>;
  clearSearch: () => void;
  
  // 恢复搜索状态（从详情页返回时使用）
  restoreSearch: () => void;
}

interface SearchParams {
  query: string;
  accountUid?: string;
  pagination?: PaginationParams;
}

export const useSearchStore = create<SearchState>((set, get) => ({
  // 初始状态
  emails: [],
  total: 0,
  isLoading: false,
  error: null,
  hasSearched: false,
  currentQuery: '',
  currentAccountUid: undefined,
  currentPage: 1,

  // 执行搜索
  search: async ({ query, accountUid, pagination }: SearchParams) => {
    if (!query.trim()) {
      set({
        emails: [],
        total: 0,
        hasSearched: false,
        error: null,
        currentQuery: '',
        currentAccountUid: undefined,
        currentPage: 1,
      });
      return;
    }

    set({ isLoading: true, error: null });

    try {
      const result = await emailService.search(query, accountUid, pagination);
      const page = pagination?.page || 1;
      
      console.log('[searchStore] 搜索结果:', {
        query,
        emailsLength: result.emails?.length || 0,
        total: result.total,
        page,
      });

      set(state => ({
        emails: page === 1 ? result.emails : [...state.emails, ...result.emails],
        total: result.total,
        isLoading: false,
        hasSearched: true,
        currentQuery: query,
        currentAccountUid: accountUid,
        currentPage: page,
      }));
    } catch (error) {
      console.error('搜索邮件失败:', error);
      set({
        isLoading: false,
        error: '搜索失败，请重试',
        hasSearched: true,
      });
    }
  },

  // 加载更多
  loadMore: async (page: number) => {
    const { currentQuery, currentAccountUid, isLoading } = get();
    if (!currentQuery || isLoading) return;

    await get().search({
      query: currentQuery,
      accountUid: currentAccountUid,
      pagination: { page, page_size: 20 },
    });
  },

  // 清除搜索
  clearSearch: () => {
    set({
      emails: [],
      total: 0,
      isLoading: false,
      error: null,
      hasSearched: false,
      currentQuery: '',
      currentAccountUid: undefined,
      currentPage: 1,
    });
  },

  // 恢复搜索状态（从详情页返回时，状态已经在 store 中保持）
  restoreSearch: () => {
    // 状态已经在 store 中，不需要额外操作
    // 这个方法主要用于触发组件重新渲染或执行其他恢复逻辑
    console.log('[searchStore] 恢复搜索状态:', {
      query: get().currentQuery,
      total: get().total,
      emailsCount: get().emails.length,
    });
  },
}));
