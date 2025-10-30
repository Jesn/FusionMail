import { useState, useCallback } from 'react';
import { emailService } from '../services/emailService';
import { Email, PaginationParams } from '../types';
// import { toast } from 'react-hot-toast';

interface SearchState {
  emails: Email[];
  total: number;
  isLoading: boolean;
  error: string | null;
  hasSearched: boolean;
}

interface SearchParams {
  query: string;
  accountUid?: string;
  pagination?: PaginationParams;
}

export const useSearch = () => {
  const [state, setState] = useState<SearchState>({
    emails: [],
    total: 0,
    isLoading: false,
    error: null,
    hasSearched: false,
  });

  const [currentQuery, setCurrentQuery] = useState('');
  const [currentAccountUid, setCurrentAccountUid] = useState<string | undefined>();

  const search = useCallback(async ({ query, accountUid, pagination }: SearchParams) => {
    if (!query.trim()) {
      setState(prev => ({
        ...prev,
        emails: [],
        total: 0,
        hasSearched: false,
        error: null,
      }));
      return;
    }

    setState(prev => ({ ...prev, isLoading: true, error: null }));
    setCurrentQuery(query);
    setCurrentAccountUid(accountUid);

    try {
      const result = await emailService.search(query, accountUid, pagination);
      setState(prev => ({
        ...prev,
        emails: pagination?.page === 1 ? result.emails : [...prev.emails, ...result.emails],
        total: result.total,
        isLoading: false,
        hasSearched: true,
      }));
    } catch (error) {
      console.error('搜索邮件失败:', error);
      setState(prev => ({
        ...prev,
        isLoading: false,
        error: '搜索失败，请重试',
        hasSearched: true,
      }));
      // toast.error('搜索失败，请重试');
    }
  }, []);

  const loadMore = useCallback(async (page: number) => {
    if (!currentQuery || state.isLoading) return;

    await search({
      query: currentQuery,
      accountUid: currentAccountUid,
      pagination: { page, page_size: 20 },
    });
  }, [currentQuery, currentAccountUid, state.isLoading, search]);

  const clearSearch = useCallback(() => {
    setState({
      emails: [],
      total: 0,
      isLoading: false,
      error: null,
      hasSearched: false,
    });
    setCurrentQuery('');
    setCurrentAccountUid(undefined);
  }, []);

  return {
    ...state,
    currentQuery,
    search,
    loadMore,
    clearSearch,
  };
};

// 搜索历史管理
const SEARCH_HISTORY_KEY = 'fusionmail_search_history';
const MAX_HISTORY_ITEMS = 10;

export const useSearchHistory = () => {
  const [history, setHistory] = useState<string[]>(() => {
    try {
      const saved = localStorage.getItem(SEARCH_HISTORY_KEY);
      return saved ? JSON.parse(saved) : [];
    } catch {
      return [];
    }
  });

  const addToHistory = useCallback((query: string) => {
    if (!query.trim()) return;

    setHistory(prev => {
      const filtered = prev.filter(item => item !== query);
      const newHistory = [query, ...filtered].slice(0, MAX_HISTORY_ITEMS);
      localStorage.setItem(SEARCH_HISTORY_KEY, JSON.stringify(newHistory));
      return newHistory;
    });
  }, []);

  const removeFromHistory = useCallback((query: string) => {
    setHistory(prev => {
      const newHistory = prev.filter(item => item !== query);
      localStorage.setItem(SEARCH_HISTORY_KEY, JSON.stringify(newHistory));
      return newHistory;
    });
  }, []);

  const clearHistory = useCallback(() => {
    setHistory([]);
    localStorage.removeItem(SEARCH_HISTORY_KEY);
  }, []);

  return {
    history,
    addToHistory,
    removeFromHistory,
    clearHistory,
  };
};