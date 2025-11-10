import { create } from 'zustand';
import { persist } from 'zustand/middleware';

interface CacheEntry<T> {
  data: T;
  timestamp: number;
  expiresIn: number; // 过期时间（毫秒）
}

/**
 * 邮件缓存存储接口
 */
interface EmailCacheStore {
  // 缓存数据
  emailCache: Map<string, CacheEntry<any>>;
  emailDetailCache: Map<string, CacheEntry<any>>;
  searchCache: Map<string, CacheEntry<any>>;

  // Actions - 设置缓存
  setEmailCache: <T>(key: string, data: T, expiresIn?: number) => void;
  setEmailDetailCache: <T>(key: string, data: T, expiresIn?: number) => void;
  setSearchCache: <T>(key: string, data: T, expiresIn?: number) => void;

  // Actions - 获取缓存
  getEmailCache: <T>(key: string) => T | null;
  getEmailDetailCache: <T>(key: string) => T | null;
  getSearchCache: <T>(key: string) => T | null;

  // Actions - 清除缓存
  clearEmailCache: (pattern?: string) => void;
  clearEmailDetailCache: (pattern?: string) => void;
  clearSearchCache: (pattern?: string) => void;
  clearAllCache: () => void;

  // Actions - 清理过期缓存
  cleanup: () => void;

  // 获取缓存统计
  getCacheStats: () => {
    emailCacheCount: number;
    emailDetailCacheCount: number;
    searchCacheCount: number;
    totalCount: number;
  };
}

// 默认缓存过期时间（毫秒）
const DEFAULT_EMAIL_CACHE_EXPIRY = 5 * 60 * 1000; // 5分钟
const DEFAULT_EMAIL_DETAIL_CACHE_EXPIRY = 10 * 60 * 1000; // 10分钟
const DEFAULT_SEARCH_CACHE_EXPIRY = 2 * 60 * 1000; // 2分钟

// 缓存条目上限（LRU 淘汰）
const MAX_EMAIL_CACHE_ENTRIES = 20; // 邮件列表页缓存最多 20 页
const MAX_EMAIL_DETAIL_CACHE_ENTRIES = 100; // 邮件详情最多 100 封
const MAX_SEARCH_CACHE_ENTRIES = 30; // 搜索结果最多 30 条（不同 query）

// 简单按时间戳的 LRU 淘汰：超过上限时删除最早写入的条目
function pruneMap<T>(map: Map<string, CacheEntry<T>>, maxEntries: number) {
  if (map.size <= maxEntries) return;
  const entries = Array.from(map.entries());
  entries.sort((a, b) => a[1].timestamp - b[1].timestamp); // 最早的在前
  const toDelete = entries.length - maxEntries;
  for (let i = 0; i < toDelete; i++) {
    map.delete(entries[i][0]);
  }
}

/**
 * 创建邮件缓存存储
 * 使用 persist 中间件实现本地存储
 */
export const useEmailCacheStore = create<EmailCacheStore>()(
  persist(
    (set, get) => ({
      // 初始状态
      emailCache: new Map(),
      emailDetailCache: new Map(),
      searchCache: new Map(),

      // 设置缓存
      setEmailCache: <T>(key: string, data: T, expiresIn: number = DEFAULT_EMAIL_CACHE_EXPIRY) => {
        set((state) => {
          const newCache = new Map(state.emailCache);
          newCache.set(key, {
            data,
            timestamp: Date.now(),
            expiresIn,
          });
          pruneMap(newCache, MAX_EMAIL_CACHE_ENTRIES);
          return { emailCache: newCache };
        });
      },

      setEmailDetailCache: <T>(key: string, data: T, expiresIn: number = DEFAULT_EMAIL_DETAIL_CACHE_EXPIRY) => {
        set((state) => {
          const newCache = new Map(state.emailDetailCache);
          newCache.set(key, {
            data,
            timestamp: Date.now(),
            expiresIn,
          });
          pruneMap(newCache, MAX_EMAIL_DETAIL_CACHE_ENTRIES);
          return { emailDetailCache: newCache };
        });
      },

      setSearchCache: <T>(key: string, data: T, expiresIn: number = DEFAULT_SEARCH_CACHE_EXPIRY) => {
        set((state) => {
          const newCache = new Map(state.searchCache);
          newCache.set(key, {
            data,
            timestamp: Date.now(),
            expiresIn,
          });
          pruneMap(newCache, MAX_SEARCH_CACHE_ENTRIES);
          return { searchCache: newCache };
        });
      },

      // 获取缓存
      getEmailCache: <T>(key: string): T | null => {
        const entry = get().emailCache.get(key);
        if (!entry) return null;

        // 检查是否过期
        if (Date.now() - entry.timestamp > entry.expiresIn) {
          get().emailCache.delete(key);
          return null;
        }

        return entry.data as T;
      },

      getEmailDetailCache: <T>(key: string): T | null => {
        const entry = get().emailDetailCache.get(key);
        if (!entry) return null;

        if (Date.now() - entry.timestamp > entry.expiresIn) {
          get().emailDetailCache.delete(key);
          return null;
        }

        return entry.data as T;
      },

      getSearchCache: <T>(key: string): T | null => {
        const entry = get().searchCache.get(key);
        if (!entry) return null;

        if (Date.now() - entry.timestamp > entry.expiresIn) {
          get().searchCache.delete(key);
          return null;
        }

        return entry.data as T;
      },

      // 清除缓存
      clearEmailCache: (pattern?: string) => {
        set((state) => {
          if (!pattern) {
            return { emailCache: new Map() };
          }

          const newCache = new Map(state.emailCache);
          for (const key of newCache.keys()) {
            if (key.match(pattern)) {
              newCache.delete(key);
            }
          }
          return { emailCache: newCache };
        });
      },

      clearEmailDetailCache: (pattern?: string) => {
        set((state) => {
          if (!pattern) {
            return { emailDetailCache: new Map() };
          }

          const newCache = new Map(state.emailDetailCache);
          for (const key of newCache.keys()) {
            if (key.match(pattern)) {
              newCache.delete(key);
            }
          }
          return { emailDetailCache: newCache };
        });
      },

      clearSearchCache: (pattern?: string) => {
        set((state) => {
          if (!pattern) {
            return { searchCache: new Map() };
          }

          const newCache = new Map(state.searchCache);
          for (const key of newCache.keys()) {
            if (key.match(pattern)) {
              newCache.delete(key);
            }
          }
          return { searchCache: newCache };
        });
      },

      clearAllCache: () => {
        set({
          emailCache: new Map(),
          emailDetailCache: new Map(),
          searchCache: new Map(),
        });
      },

      // 清理过期缓存
      cleanup: () => {
        const now = Date.now();
        set((state) => {
          const newEmailCache = new Map(state.emailCache);
          const newEmailDetailCache = new Map(state.emailDetailCache);
          const newSearchCache = new Map(state.searchCache);

          // 清理邮件列表缓存
          for (const [key, entry] of newEmailCache.entries()) {
            if (now - entry.timestamp > entry.expiresIn) {
              newEmailCache.delete(key);
            }
          }

          // 清理邮件详情缓存
          for (const [key, entry] of newEmailDetailCache.entries()) {
            if (now - entry.timestamp > entry.expiresIn) {
              newEmailDetailCache.delete(key);
            }
          }

          // 清理搜索缓存
          for (const [key, entry] of newSearchCache.entries()) {
            if (now - entry.timestamp > entry.expiresIn) {
              newSearchCache.delete(key);
            }
          }

          return {
            emailCache: newEmailCache,
            emailDetailCache: newEmailDetailCache,
            searchCache: newSearchCache,
          };
        });
      },

      // 获取缓存统计
      getCacheStats: () => {
        const state = get();
        return {
          emailCacheCount: state.emailCache.size,
          emailDetailCacheCount: state.emailDetailCache.size,
          searchCacheCount: state.searchCache.size,
          totalCount: state.emailCache.size + state.emailDetailCache.size + state.searchCache.size,
        };
      },
    }),
    {
      // 配置持久化
      name: 'fusionmail-email-cache', // localStorage 键名
      version: 1, // 版本号
      // 只持久化 Map 的数据部分
      partialize: (state) => ({
        emailCache: Array.from(state.emailCache.entries()),
        emailDetailCache: Array.from(state.emailDetailCache.entries()),
        searchCache: Array.from(state.searchCache.entries()),
      }),
      // 恢复时转换回 Map
      onRehydrateStorage: () => (state) => {
        if (state) {
          state.emailCache = new Map(state.emailCache);
          state.emailDetailCache = new Map(state.emailDetailCache);
          state.searchCache = new Map(state.searchCache);
        }
      },
    }
  )
);

/**
 * 缓存工具函数
 * 提供便捷的缓存操作方法
 */

/**
 * 获取或设置缓存
 * 如果缓存存在且未过期，返回缓存数据；否则调用 fetcher 获取新数据
 */
export const getOrSetEmailCache = async <T>(
  cacheFn: (key: string) => T | null,
  setCacheFn: (key: string, data: T, expiresIn?: number) => void,
  key: string,
  fetcher: () => Promise<T>,
  expiresIn?: number
): Promise<T> => {
  // 尝试从缓存获取
  const cached = cacheFn(key);
  if (cached !== null) {
    return cached;
  }

  // 缓存未命中，调用 fetcher 获取新数据
  const data = await fetcher();

  // 设置缓存
  setCacheFn(key, data, expiresIn);

  return data;
};

/**
 * 预加载邮件列表缓存
 */
export const preloadEmailCache = (
  accountUid: string,
  page: number,
  pageSize: number,
  fetcher: () => Promise<any>
): Promise<any> => {
  const cacheKey = `emails:${accountUid}:${page}:${pageSize}`;
  const { getEmailCache, setEmailCache } = useEmailCacheStore.getState();

  return getOrSetEmailCache(
    getEmailCache,
    setEmailCache,
    cacheKey,
    fetcher,
    DEFAULT_EMAIL_CACHE_EXPIRY
  );
};

/**
 * 预加载邮件详情缓存
 */
export const preloadEmailDetailCache = (
  accountUid: string,
  emailId: string,
  fetcher: () => Promise<any>
): Promise<any> => {
  const cacheKey = `email:${accountUid}:${emailId}`;
  const { getEmailDetailCache, setEmailDetailCache } = useEmailCacheStore.getState();

  return getOrSetEmailCache(
    getEmailDetailCache,
    setEmailDetailCache,
    cacheKey,
    fetcher,
    DEFAULT_EMAIL_DETAIL_CACHE_EXPIRY
  );
};

/**
 * 预加载搜索结果缓存
 */
export const preloadSearchCache = (
  query: string,
  accountUid: string,
  fetcher: () => Promise<any>
): Promise<any> => {
  const cacheKey = `search:${accountUid}:${query}`;
  const { getSearchCache, setSearchCache } = useEmailCacheStore.getState();

  return getOrSetEmailCache(
    getSearchCache,
    setSearchCache,
    cacheKey,
    fetcher,
    DEFAULT_SEARCH_CACHE_EXPIRY
  );
};

// 定期清理过期缓存（每分钟）
if (typeof window !== 'undefined') {
  setInterval(() => {
    useEmailCacheStore.getState().cleanup();
  }, 60 * 1000);
}
