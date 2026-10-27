export const getOAuth2CallbackUri = (provider: 'google' | 'microsoft'): string => {
  // 优先使用环境变量中的 API 基础路径
  const apiBase = import.meta.env.VITE_API_BASE_URL;
  
  if (apiBase) {
    // 如果是相对路径（如 /api/v1），则拼接当前域名
    if (apiBase.startsWith('/')) {
      return `${window.location.origin}${apiBase}/auth/${provider}/callback`;
    }
    // 如果是完整 URL，直接使用
    return `${apiBase}/auth/${provider}/callback`;
  }
  
  // 默认使用当前域名 + 标准 API 路径
  return `${window.location.origin}/api/v1/auth/${provider}/callback`;
};

/**
 * 根据 Provider 的 supported_adapters 推导 OAuth2 提供商类型
 * gmail adapter -> google
 * graph adapter -> microsoft
 */
export const getOAuth2ProviderType = (provider: { supported_adapters?: { name: string; auth_type: string }[]; name?: string } | undefined): 'google' | 'microsoft' => {
  // 优先使用 supported_adapters 推导
  if (provider?.supported_adapters && provider.supported_adapters.length > 0) {
    const oauth2Adapter = provider.supported_adapters.find(a => a.auth_type === 'oauth2');
    if (oauth2Adapter) {
      if (oauth2Adapter.name === 'gmail') return 'google';
      if (oauth2Adapter.name === 'graph') return 'microsoft';
    }
  }
  
  // 回退：使用 provider 名称（兼容旧数据）
  if (provider?.name === 'gmail') return 'google';
  if (provider?.name === 'outlook') return 'microsoft';
  
  // 默认返回 google
  return 'google';
};
