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
const MAX_EMAIL_CACHE_ENTRIES = 5; // 邮件列表页缓存最多 5 页（减少以避免配额超限）
const MAX_EMAIL_DETAIL_CACHE_ENTRIES = 20; // 邮件详情最多 20 封（大幅减少）
const MAX_SEARCH_CACHE_ENTRIES = 10; // 搜索结果最多 10 条（大幅减少）

// localStorage 配额检查（估算）
const MAX_STORAGE_SIZE = 4 * 1024 * 1024; // 4MB 阈值（低于 5-10MB 限制）

/**
 * 检查 localStorage 使用情况
 */
function getStorageSize(): number {
  try {
    const data = localStorage.getItem('fusionmail-email-cache');
    return data ? new Blob([data]).size : 0;
  } catch (e) {
    return 0;
  }
}

/**
 * 清理部分缓存以释放空间
 */
function cleanupStorage(): void {
  try {
    // 清理一半的缓存
    const state = useEmailCacheStore.getState();
    const newEmailCache = new Map(Array.from(state.emailCache.entries()).slice(0, Math.floor(MAX_EMAIL_CACHE_ENTRIES / 2)));
    const newEmailDetailCache = new Map(Array.from(state.emailDetailCache.entries()).slice(0, Math.floor(MAX_EMAIL_DETAIL_CACHE_ENTRIES / 2)));
    const newSearchCache = new Map(Array.from(state.searchCache.entries()).slice(0, Math.floor(MAX_SEARCH_CACHE_ENTRIES / 2)));

    useEmailCacheStore.setState({
      emailCache: newEmailCache,
      emailDetailCache: newEmailDetailCache,
      searchCache: newSearchCache,
    });
  } catch (e) {
    // 如果清理失败，清空所有缓存
    console.warn('Failed to cleanup storage, clearing all cache:', e);
    try {
      useEmailCacheStore.getState().clearAllCache();
    } catch (e2) {
      console.error('Failed to clear cache:', e2);
    }
  }
}

/**
 * 简单按时间戳的 LRU 淘汰：超过上限时删除最早写入的条目
 * 改进版：带存储空间检查
 */
function pruneMap<T>(map: Map<string, CacheEntry<T>>, maxEntries: number): boolean {
  let pruned = false;
  if (map.size <= maxEntries) return pruned;

  const entries = Array.from(map.entries());
  entries.sort((a, b) => a[1].timestamp - b[1].timestamp); // 最早的在前
  const toDelete = entries.length - maxEntries;
  for (let i = 0; i < toDelete; i++) {
    map.delete(entries[i][0]);
  }
  pruned = true;

  // 检查存储空间
  const size = getStorageSize();
  if (size > MAX_STORAGE_SIZE) {
    console.warn('Storage size exceeded, cleaning up:', size);
    cleanupStorage();
  }

  return pruned;
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

        // 更新访问时间以实现真正的 LRU
        try {
          set((state) => {
            const newCache = new Map(state.emailCache);
            const cacheEntry = newCache.get(key);
            if (cacheEntry) {
              newCache.set(key, {
                ...cacheEntry,
                timestamp: Date.now(),
              });
            }
            return { emailCache: newCache };
          });
        } catch (e) {
          // 如果更新失败，忽略错误
          console.warn('Failed to update cache access time:', e);
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

        // 更新访问时间以实现真正的 LRU
        try {
          set((state) => {
            const newCache = new Map(state.emailDetailCache);
            const cacheEntry = newCache.get(key);
            if (cacheEntry) {
              newCache.set(key, {
                ...cacheEntry,
                timestamp: Date.now(),
              });
            }
            return { emailDetailCache: newCache };
          });
        } catch (e) {
          console.warn('Failed to update cache access time:', e);
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

        // 更新访问时间以实现真正的 LRU
        try {
          set((state) => {
            const newCache = new Map(state.searchCache);
            const cacheEntry = newCache.get(key);
            if (cacheEntry) {
              newCache.set(key, {
                ...cacheEntry,
                timestamp: Date.now(),
              });
            }
            return { searchCache: newCache };
          });
        } catch (e) {
          console.warn('Failed to update cache access time:', e);
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
      partialize: (state) => {
        try {
          return {
            emailCache: Array.from(state.emailCache.entries()),
            emailDetailCache: Array.from(state.emailDetailCache.entries()),
            searchCache: Array.from(state.searchCache.entries()),
          };
        } catch (e) {
          console.error('Failed to serialize cache data:', e);
          // 返回空缓存
          return {
            emailCache: [],
            emailDetailCache: [],
            searchCache: [],
          };
        }
      },
      // 恢复时转换回 Map
      onRehydrateStorage: () => (state) => {
        if (state) {
          try {
            state.emailCache = new Map(state.emailCache);
            state.emailDetailCache = new Map(state.emailDetailCache);
            state.searchCache = new Map(state.searchCache);
          } catch (e) {
            console.error('Failed to restore cache data:', e);
            // 初始化为空 Map
            state.emailCache = new Map();
            state.emailDetailCache = new Map();
            state.searchCache = new Map();
          }
        }
      },
    }
  )
);

/**
 * 缓存工具函数
 * 提供便捷的缓存操作方法
 */

// 正在进行的请求映射，防止重复请求
const inflightRequests = new Map<string, Promise<any>>();

/**
 * 获取或设置缓存
 * 如果缓存存在且未过期，返回缓存数据；否则调用 fetcher 获取新数据
 * 添加并发控制，防止同一时间对同一 key 的重复请求
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

  // 检查是否有正在进行的请求
  if (inflightRequests.has(key)) {
    return inflightRequests.get(key);
  }

  // 缓存未命中，调用 fetcher 获取新数据
  const requestPromise = (async () => {
    try {
      const data = await fetcher();
      setCacheFn(key, data, expiresIn);
      return data;
    } finally {
      // 请求完成后清除 in-flight 记录
      inflightRequests.delete(key);
    }
  })();

  // 存储请求Promise
  inflightRequests.set(key, requestPromise);

  return requestPromise;
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
    try {
      useEmailCacheStore.getState().cleanup();
      // 检查存储空间
      const size = getStorageSize();
      if (size > MAX_STORAGE_SIZE) {
        console.warn('Storage size exceeded during periodic cleanup:', size);
        cleanupStorage();
      }
    } catch (e) {
      console.error('Failed to perform periodic cleanup:', e);
    }
  }, 60 * 1000);

  // 初始化时检查并清理过量缓存
  setTimeout(() => {
    try {
      const state = useEmailCacheStore.getState();

      // 强制执行 LRU 淘汰
      pruneMap(state.emailCache, MAX_EMAIL_CACHE_ENTRIES);
      pruneMap(state.emailDetailCache, MAX_EMAIL_DETAIL_CACHE_ENTRIES);
      pruneMap(state.searchCache, MAX_SEARCH_CACHE_ENTRIES);

      // 检查存储空间
      const size = getStorageSize();
      if (size > MAX_STORAGE_SIZE) {
        console.warn('Initial storage size check: exceeded, cleaning up. Size:', size);
        cleanupStorage();
      } else {
        console.log('Cache initialized successfully. Storage size:', Math.round(size / 1024), 'KB');
      }
    } catch (e) {
      console.error('Failed to initialize cache cleanup:', e);
      // 尝试清理所有缓存
      try {
        useEmailCacheStore.getState().clearAllCache();
      } catch (e2) {
        console.error('Failed to clear cache on initialization:', e2);
      }
    }
  }, 100);
}
