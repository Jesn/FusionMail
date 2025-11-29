import { useState, useEffect, useCallback, useRef } from 'react';
import { systemService, type Provider } from '../services/systemService';
import { toast } from 'sonner';

// 全局缓存，避免重复请求
let cachedProviders: Provider[] | null = null;
let fetchPromise: Promise<Provider[]> | null = null;

export const useProviders = () => {
  const [providers, setProviders] = useState<Provider[]>(cachedProviders || []);
  const [isLoading, setIsLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const isMounted = useRef(true);

  // 获取提供商列表
  const fetchProviders = useCallback(async () => {
    // 如果已有缓存，直接使用
    if (cachedProviders) {
      setProviders(cachedProviders);
      return;
    }

    // 如果正在请求中，等待现有请求完成
    if (fetchPromise) {
      try {
        const data = await fetchPromise;
        if (isMounted.current) {
          setProviders(data);
        }
      } catch (err) {
        // 错误已在原始请求中处理
      }
      return;
    }

    setIsLoading(true);
    setError(null);
    
    // 创建新的请求 Promise
    fetchPromise = systemService.getProviders();
    
    try {
      const data = await fetchPromise;
      cachedProviders = data; // 缓存结果
      if (isMounted.current) {
        setProviders(data);
      }
    } catch (err) {
      const errorMessage = err instanceof Error ? err.message : '获取邮箱提供商列表失败';
      if (isMounted.current) {
        setError(errorMessage);
        toast.error(errorMessage);
      }
    } finally {
      fetchPromise = null; // 清除请求 Promise
      if (isMounted.current) {
        setIsLoading(false);
      }
    }
  }, []);

  // 根据邮箱地址获取推荐的提供商
  const getProviderByEmail = useCallback((email: string): Provider | null => {
    if (!email || !email.includes('@')) {
      return null;
    }

    const domain = email.split('@')[1]?.toLowerCase();
    
    // 如果域名为空或无效，返回 null（保持当前选择）
    if (!domain) {
      return null;
    }
    
    // 域名映射
    const domainMappings: Record<string, string> = {
      'qq.com': 'qq',
      '163.com': '163',
      '126.com': '163', // 126邮箱使用163的配置
      'gmail.com': 'gmail', // Match backend case
      'outlook.com': 'outlook',
      'hotmail.com': 'outlook',
      'live.com': 'outlook',
      'icloud.com': 'icloud',
      'me.com': 'icloud',
    };

    const providerName = domainMappings[domain];
    if (providerName) {
      return providers.find(p => p.name === providerName) || null;
    }

    // 对于未知域名，返回 null（保持当前选择，不自动切换到通用邮箱）
    return null;
  }, [providers]);

  // 根据提供商名称获取提供商信息
  const getProviderByName = useCallback((name: string): Provider | null => {
    return providers.find(p => p.name === name) || null;
  }, [providers]);

  // 获取预设提供商列表（排除通用邮箱）
  const getPresetProviders = useCallback((): Provider[] => {
    return providers.filter(p => p.name !== 'generic');
  }, [providers]);

  // 获取通用邮箱提供商
  const getGenericProvider = useCallback((): Provider | null => {
    return providers.find(p => p.name === 'generic') || null;
  }, [providers]);

  // 强制刷新提供商列表（清除缓存）
  const refreshProviders = useCallback(async () => {
    cachedProviders = null; // 清除缓存
    fetchPromise = null;
    await fetchProviders();
  }, [fetchProviders]);

  // 组件挂载时获取提供商列表
  useEffect(() => {
    isMounted.current = true;
    fetchProviders();
    
    return () => {
      isMounted.current = false;
    };
  }, [fetchProviders]);

  return {
    providers,
    isLoading,
    error,
    fetchProviders,
    refreshProviders, // 新增：强制刷新方法
    getProviderByEmail,
    getProviderByName,
    getPresetProviders,
    getGenericProvider,
  };
};