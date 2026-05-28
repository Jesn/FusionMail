/**
 * 配置管理React Hook
 * 提供配置数据的获取、更新、缓存等功能
 */

import { useQuery, useMutation, useQueryClient, useQueries } from '@tanstack/react-query';
import { useCallback, useMemo } from 'react';
import { settingsService } from '../services/settings';
import type {
  UseSettingsReturn,
} from '../types/settings';

// Query Keys
export const SETTINGS_QUERY_KEYS = {
  all: ['settings'] as const,
  category: (category: string, userId?: number) =>
    [...SETTINGS_QUERY_KEYS.all, 'category', category, userId] as const,
  setting: (category: string, key: string, userId?: number) =>
    [...SETTINGS_QUERY_KEYS.all, 'setting', category, key, userId] as const,
  public: () => [...SETTINGS_QUERY_KEYS.all, 'public'] as const,
  categories: () => [...SETTINGS_QUERY_KEYS.all, 'categories'] as const,
  search: (query: string, category?: string) =>
    [...SETTINGS_QUERY_KEYS.all, 'search', query, category] as const,
} as const;

// Query Keys 定义

/**
 * 按分类获取配置
 */
export function useSettingsByCategory(
  category: string,
  options?: {
    userId?: number;
    includeSensitive?: boolean;
    enabled?: boolean;
  }
): UseSettingsReturn {
  const queryKey = SETTINGS_QUERY_KEYS.category(category, options?.userId);

  const query = useQuery({
    queryKey,
    queryFn: async () => {
      const response = await settingsService.getSettingsByCategory(category, {
        userId: options?.userId,
        includeSensitive: options?.includeSensitive,
      });
      return response.settings;
    },
    staleTime: 5 * 60 * 1000, // 5分钟
    gcTime: 10 * 60 * 1000, // 10分钟
    enabled: options?.enabled !== false,
  });

  return {
    settings: query.data,
    isLoading: query.isLoading,
    error: query.error as Error | null,
    refetch: async () => { await query.refetch(); },
  };
}

/**
 * 获取单个配置项
 */
export function useSetting(
  category: string,
  key: string,
  options?: {
    userId?: number;
    includeSensitive?: boolean;
    enabled?: boolean;
  }
): UseSettingsReturn {
  const queryKey = SETTINGS_QUERY_KEYS.setting(category, key, options?.userId);

  const query = useQuery({
    queryKey,
    queryFn: async () => {
      const value = await settingsService.getSetting(category, key, {
        userId: options?.userId,
        includeSensitive: options?.includeSensitive,
      });
      return { [key]: value };
    },
    staleTime: 5 * 60 * 1000,
    enabled: options?.enabled !== false && !!key,
  });

  return {
    settings: query.data,
    isLoading: query.isLoading,
    error: query.error as Error | null,
    refetch: async () => { await query.refetch(); },
  };
}

/**
 * 获取公开配置
 */
export function usePublicSettings(options?: {
  enabled?: boolean;
}) {
  const queryKey = SETTINGS_QUERY_KEYS.public();

  const query = useQuery({
    queryKey,
    queryFn: async () => {
      return await settingsService.getPublicSettings();
    },
    staleTime: 10 * 60 * 1000, // 公开配置缓存更久
    enabled: options?.enabled !== false,
  });

  return {
    settings: query.data,
    isLoading: query.isLoading,
    error: query.error as Error | null,
    refetch: async () => { await query.refetch(); },
  };
}

/**
 * 获取配置分类列表
 */
export function useSettingCategories() {
  const queryKey = SETTINGS_QUERY_KEYS.categories();

  const query = useQuery({
    queryKey,
    queryFn: async () => {
      return await settingsService.getSettingCategories();
    },
    staleTime: 30 * 60 * 1000, // 分类列表缓存30分钟
  });

  return {
    categories: query.data,
    isLoading: query.isLoading,
    error: query.error as Error | null,
    refetch: async () => { await query.refetch(); },
  };
}

/**
 * 搜索配置
 */
export function useSearchSettings(
  query: string,
  options?: {
    category?: string;
    onlyPublic?: boolean;
    enabled?: boolean;
  }
) {
  const queryKey = SETTINGS_QUERY_KEYS.search(query, options?.category);

  const query_obj = useQuery({
    queryKey,
    queryFn: async () => {
      return await settingsService.searchSettings(query, {
        category: options?.category,
        onlyPublic: options?.onlyPublic,
      });
    },
    staleTime: 2 * 60 * 1000,
    enabled:
      options?.enabled !== false &&
      !!query &&
      query.trim().length >= 2,
  });

  return {
    results: query_obj.data,
    isLoading: query_obj.isLoading,
    error: query_obj.error as Error | null,
    refetch: async () => { await query_obj.refetch(); },
  };
}

/**
 * 更新单个配置
 */
export function useUpdateSetting(options?: {
  onSuccess?: () => void;
  onError?: (error: Error) => void;
}) {
  const queryClient = useQueryClient();

  const mutation = useMutation({
    mutationFn: async (variables: {
      category: string;
      key: string;
      value: string;
      userId?: number;
      isSensitive?: boolean;
    }) => {
      await settingsService.setSetting(
        variables.category,
        variables.key,
        variables.value,
        {
          userId: variables.userId,
          isSensitive: variables.isSensitive,
        }
      );
    },
    onSuccess: (_data, variables) => {
      // 更新相关缓存
      queryClient.invalidateQueries({
        queryKey: SETTINGS_QUERY_KEYS.category(variables.category, variables.userId),
      });
      queryClient.invalidateQueries({
        queryKey: SETTINGS_QUERY_KEYS.setting(
          variables.category,
          variables.key,
          variables.userId
        ),
      });

      options?.onSuccess?.();
    },
    onError: (error) => {
      console.error('Failed to update setting:', error);
      options?.onError?.(error as Error);
    },
  });

  return {
    mutate: mutation.mutate,
    mutateAsync: mutation.mutateAsync,
    isPending: mutation.isPending,
    error: mutation.error as Error | null,
  };
}

/**
 * 批量更新配置
 */
export function useBatchUpdateSettings(options?: {
  onSuccess?: () => void;
  onError?: (error: Error) => void;
}) {
  const queryClient = useQueryClient();

  const mutation = useMutation({
    mutationFn: async (variables: {
      category: string;
      settings: Record<string, string>;
      userId?: number;
    }) => {
      await settingsService.batchSetSettings(
        variables.category,
        variables.settings,
        {
          userId: variables.userId,
        }
      );
    },
    onSuccess: (_data, variables) => {
      // 批量更新缓存
      queryClient.invalidateQueries({
        queryKey: SETTINGS_QUERY_KEYS.category(variables.category, variables.userId),
      });

      options?.onSuccess?.();
    },
    onError: (error) => {
      console.error('Failed to batch update settings:', error);
      options?.onError?.(error as Error);
    },
  });

  return {
    mutate: mutation.mutate,
    mutateAsync: mutation.mutateAsync,
    isPending: mutation.isPending,
    error: mutation.error as Error | null,
  };
}

/**
 * 删除配置
 */
export function useDeleteSetting(options?: {
  onSuccess?: () => void;
  onError?: (error: Error) => void;
}) {
  const queryClient = useQueryClient();

  const mutation = useMutation({
    mutationFn: async (variables: {
      category: string;
      key: string;
      userId?: number;
    }) => {
      await settingsService.deleteSetting(variables.category, variables.key);
    },
    onSuccess: (_data, variables) => {
      // 删除相关缓存
      queryClient.invalidateQueries({
        queryKey: SETTINGS_QUERY_KEYS.category(variables.category, variables.userId),
      });
      queryClient.removeQueries({
        queryKey: SETTINGS_QUERY_KEYS.setting(
          variables.category,
          variables.key,
          variables.userId
        ),
      });

      options?.onSuccess?.();
    },
    onError: (error) => {
      console.error('Failed to delete setting:', error);
      options?.onError?.(error as Error);
    },
  });

  return {
    mutate: mutation.mutate,
    mutateAsync: mutation.mutateAsync,
    isPending: mutation.isPending,
    error: mutation.error as Error | null,
  };
}

/**
 * 重置配置
 */
export function useResetSetting(options?: {
  onSuccess?: () => void;
  onError?: (error: Error) => void;
}) {
  const queryClient = useQueryClient();

  const mutation = useMutation({
    mutationFn: async (variables: {
      category: string;
      key: string;
      userId?: number;
    }) => {
      await settingsService.resetSetting(variables.category, variables.key, {
        userId: variables.userId,
      });
    },
    onSuccess: (_data, variables) => {
      // 重置后刷新缓存
      queryClient.invalidateQueries({
        queryKey: SETTINGS_QUERY_KEYS.category(variables.category, variables.userId),
      });

      options?.onSuccess?.();
    },
    onError: (error) => {
      console.error('Failed to reset setting:', error);
      options?.onError?.(error as Error);
    },
  });

  return {
    mutate: mutation.mutate,
    mutateAsync: mutation.mutateAsync,
    isPending: mutation.isPending,
    error: mutation.error as Error | null,
  };
}

/**
 * 预热缓存
 */
export function useWarmUpSettings() {
  const queryClient = useQueryClient();

  const warmUp = useCallback(async () => {
    // 预热常用配置分类
    const categories = ['ui', 'sync', 'notification'];

    await Promise.all(
      categories.map((category) =>
        queryClient.prefetchQuery({
          queryKey: SETTINGS_QUERY_KEYS.category(category),
          queryFn: async () => {
            const response = await settingsService.getSettingsByCategory(category);
            return response.settings;
          },
          staleTime: 5 * 60 * 1000,
        })
      )
    );
  }, [queryClient]);

  return { warmUp };
}

/**
 * 组合Hook：获取分类下的所有配置
 */
export function useSettings(category: string, options?: {
  userId?: number;
  includeSensitive?: boolean;
  watch?: boolean; // 是否实时监听变化
}) {
  const queryClient = useQueryClient();
  const { settings, isLoading, error, refetch } = useSettingsByCategory(
    category,
    options
  );

  // 监听配置变化（如果启用）
  useMemo(() => {
    if (!options?.watch) return;

    queryClient.setDefaultOptions({
      queries: {
        refetchOnWindowFocus: true,
        refetchOnReconnect: true,
      },
    });

    return () => {
      queryClient.setDefaultOptions({
        queries: {
          refetchOnWindowFocus: false,
          refetchOnReconnect: false,
        },
      });
    };
  }, [category, options?.watch, options?.userId, queryClient]);

  // 更新配置的便捷方法
  const updateSetting = useUpdateSetting();
  const deleteSetting = useDeleteSetting();
  const resetSetting = useResetSetting();

  return {
    settings,
    isLoading,
    error,
    refetch,
    updateSetting,
    deleteSetting,
    resetSetting,
  };
}

/**
 * 获取多个分类的配置
 */
export function useMultipleSettings(
  categories: string[],
  options?: {
    userId?: number;
  }
) {
  const queries = useQueries({
    queries: categories.map((category) => ({
      queryKey: SETTINGS_QUERY_KEYS.category(category, options?.userId),
      queryFn: async () => {
        const response = await settingsService.getSettingsByCategory(category, {
          userId: options?.userId,
        });
        return response.settings;
      },
      staleTime: 5 * 60 * 1000,
      gcTime: 10 * 60 * 1000,
    })),
  });

  return {
    settings: queries.map((q) => q.data),
    isLoading: queries.some((q) => q.isLoading),
    errors: queries.map((q) => q.error as Error | null),
    refetches: queries.map((q) => async () => {
      await q.refetch();
    }),
  };
}

/**
 * 配置订阅（实时更新）
 */
export function useSettingsSubscription(
  category: string,
  options?: {
    userId?: number;
    onUpdate?: (key: string, value: string) => void;
  }
) {
  const queryClient = useQueryClient();

  // 这里可以集成WebSocket或Server-Sent Events
  // 实现实时配置更新监听
  useMemo(() => {
    const queryKey = SETTINGS_QUERY_KEYS.category(category, options?.userId);
    // 示例：WebSocket订阅
    // const ws = new WebSocket(`ws://localhost:3333/ws/settings/${category}`);
    // ws.onmessage = (event) => {
    //   const { key, value } = JSON.parse(event.data);
    //   queryClient.setQueryData(queryKey, (old) => ({
    //     ...old,
    //     [key]: value,
    //   }));
    //   options?.onUpdate?.(key, value);
    // };
    // return () => ws.close();
    
    // 避免未使用变量警告
    void queryKey;
  }, [category, options?.userId, queryClient, options?.onUpdate]);
}

/**
 * 获取缓存统计信息（管理员功能）
 */
export function useGetStats() {
  const query = useQuery({
    queryKey: [...SETTINGS_QUERY_KEYS.all, 'stats'],
    queryFn: async () => {
      // TODO: 实现统计信息API调用
      return {
        localCache: { hitRate: 0.85, size: 12 },
        redisCache: { hitRate: 0.92 },
        totalRequests: 15420,
      };
    },
    staleTime: 30 * 1000, // 30秒
  });

  return {
    data: query.data,
    isLoading: query.isLoading,
    error: query.error as Error | null,
    refetch: async () => { await query.refetch(); },
  };
}

/**
 * 缓存预热（管理员功能）
 */
export function useWarmUp() {
  const queryClient = useQueryClient();

  const mutation = useMutation({
    mutationFn: async (_categories?: string[]) => {
      // TODO: 实现缓存预热API调用
      await new Promise((resolve) => setTimeout(resolve, 1000));
      return { success: true };
    },
    onSuccess: () => {
      // 预热后刷新相关缓存
      queryClient.invalidateQueries({ queryKey: SETTINGS_QUERY_KEYS.all });
    },
  });

  return {
    mutate: mutation.mutate,
    mutateAsync: mutation.mutateAsync,
    isPending: mutation.isPending,
    error: mutation.error as Error | null,
  };
}

