import { useState, useCallback } from 'react';
import { useSearchStore } from '../stores/searchStore';

/**
 * 搜索 Hook - 使用 Zustand store 持久化搜索状态
 * 这样从邮件详情页返回时，搜索结果不会丢失
 */
export const useSearch = () => {
  const {
    emails,
    total,
    isLoading,
    error,
    hasSearched,
    currentQuery,
    currentPage,
    search,
    loadMore,
    clearSearch,
  } = useSearchStore();

  return {
    emails,
    total,
    isLoading,
    error,
    hasSearched,
    currentQuery,
    currentPage,
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