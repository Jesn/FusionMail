import { useState, useEffect, useCallback } from 'react';
import { systemService, type Provider } from '../services/systemService';
import { toast } from 'sonner';

export const useProviders = () => {
  const [providers, setProviders] = useState<Provider[]>([]);
  const [isLoading, setIsLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  // 获取提供商列表
  const fetchProviders = useCallback(async () => {
    setIsLoading(true);
    setError(null);
    
    try {
      const data = await systemService.getProviders();
      setProviders(data);
    } catch (err) {
      const errorMessage = err instanceof Error ? err.message : '获取邮箱提供商列表失败';
      setError(errorMessage);
      toast.error(errorMessage);
    } finally {
      setIsLoading(false);
    }
  }, []);

  // 根据邮箱地址获取推荐的提供商
  const getProviderByEmail = useCallback((email: string): Provider | null => {
    if (!email || !email.includes('@')) {
      return null;
    }

    const domain = email.split('@')[1].toLowerCase();
    
    // 域名映射
    const domainMappings: Record<string, string> = {
      'qq.com': 'qq',
      '163.com': '163',
      '126.com': '163', // 126邮箱使用163的配置
      'gmail.com': 'gmail',
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

    // 如果没有匹配的预设提供商，返回通用邮箱
    return providers.find(p => p.name === 'generic') || null;
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

  // 组件挂载时获取提供商列表
  useEffect(() => {
    fetchProviders();
  }, [fetchProviders]);

  return {
    providers,
    isLoading,
    error,
    fetchProviders,
    getProviderByEmail,
    getProviderByName,
    getPresetProviders,
    getGenericProvider,
  };
};