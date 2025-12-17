import { Badge } from '../ui/badge';
import { Button } from '../ui/button';
import { Shield, RefreshCw, AlertTriangle, CheckCircle } from 'lucide-react';
import { Account } from '../../types';
import { oauth2Service, OAuth2Provider } from '../../services/oauth2Service';
import { toast } from 'sonner';
import { useState } from 'react';

interface OAuth2StatusBadgeProps {
  account: Account;
  onRefreshSuccess?: () => void;
}

/**
 * 根据 Account 的 provider_ref.supported_adapters 推导 OAuth2 提供商类型
 * gmail adapter -> google
 * graph adapter -> microsoft
 */
const getOAuth2ProviderFromAccount = (account: Account): OAuth2Provider | null => {
  // 优先使用 provider_ref.supported_adapters 推导
  const supportedAdapters = account.provider_ref?.supported_adapters;
  if (supportedAdapters && supportedAdapters.length > 0) {
    // 查找 OAuth2 类型的适配器
    const oauth2Adapter = supportedAdapters.find(a => a.auth_type === 'oauth2');
    if (oauth2Adapter) {
      if (oauth2Adapter.name === 'gmail') return 'google';
      if (oauth2Adapter.name === 'graph') return 'microsoft';
    }
  }
  
  // 回退：使用 adapter_ref 判断
  if (account.adapter_ref) {
    if (account.adapter_ref.name === 'gmail') return 'google';
    if (account.adapter_ref.name === 'graph') return 'microsoft';
  }
  
  // 最后回退：使用 provider 名称（兼容旧数据）
  if (account.provider === 'gmail') return 'google';
  if (account.provider === 'outlook') return 'microsoft';
  
  return null;
};

export const OAuth2StatusBadge = ({ account, onRefreshSuccess }: OAuth2StatusBadgeProps) => {
  const [isRefreshing, setIsRefreshing] = useState(false);

  // 判断是否是 OAuth2 账户
  const isOAuth2Account = account.auth_type === 'oauth2';
  
  // 使用新的推导函数获取提供商类型
  const provider = getOAuth2ProviderFromAccount(account);

  // 如果不是 OAuth2 账户，不显示
  if (!isOAuth2Account || !provider) {
    return null;
  }

  // 判断授权状态
  const getAuthStatus = () => {
    if (account.status === 'auth_required') {
      return {
        status: 'expired',
        label: '需要重新授权',
        color: 'destructive' as const,
        icon: AlertTriangle,
      };
    }
    
    if (account.status === 'active') {
      return {
        status: 'active',
        label: 'OAuth2 已授权',
        color: 'default' as const,
        icon: CheckCircle,
      };
    }

    return {
      status: 'unknown',
      label: '授权状态未知',
      color: 'secondary' as const,
      icon: Shield,
    };
  };

  const authStatus = getAuthStatus();

  const handleRefreshToken = async () => {
    if (!provider) return;

    setIsRefreshing(true);
    try {
      await oauth2Service.refreshToken(provider, account.uid);
      toast.success('访问令牌刷新成功');
      onRefreshSuccess?.();
    } catch (error) {
      console.error('Token refresh failed:', error);
      toast.error('访问令牌刷新失败，请重新授权');
    } finally {
      setIsRefreshing(false);
    }
  };

  const handleReauthorize = async () => {
    if (!provider) return;

    try {
      const authResponse = await oauth2Service.generateAuthUrl(provider, account.email);
      
      // 存储 state 到 sessionStorage
      sessionStorage.setItem(`oauth2_state_${provider}`, authResponse.state);
      sessionStorage.setItem(`oauth2_provider`, provider);
      sessionStorage.setItem(`oauth2_email`, account.email);

      // 打开授权页面
      const authWindow = window.open(
        authResponse.auth_url,
        'oauth2_reauth',
        'width=500,height=600,scrollbars=yes,resizable=yes'
      );

      if (!authWindow) {
        throw new Error('无法打开授权窗口，请检查浏览器弹窗设置');
      }

      toast.info('请在新窗口中完成重新授权');
    } catch (error) {
      console.error('Reauthorization failed:', error);
      toast.error('启动重新授权失败');
    }
  };

  const StatusIcon = authStatus.icon;

  return (
    <div className="flex items-center space-x-2">
      <Badge variant={authStatus.color} className="flex items-center space-x-1">
        <StatusIcon className="h-3 w-3" />
        <span>{authStatus.label}</span>
      </Badge>

      {authStatus.status === 'active' && (
        <Button
          variant="ghost"
          size="sm"
          onClick={handleRefreshToken}
          disabled={isRefreshing}
          className="h-6 px-2 text-xs"
        >
          <RefreshCw className={`h-3 w-3 ${isRefreshing ? 'animate-spin' : ''}`} />
          {isRefreshing ? '刷新中...' : '刷新令牌'}
        </Button>
      )}

      {authStatus.status === 'expired' && (
        <Button
          variant="outline"
          size="sm"
          onClick={handleReauthorize}
          className="h-6 px-2 text-xs"
        >
          <Shield className="h-3 w-3 mr-1" />
          重新授权
        </Button>
      )}
    </div>
  );
};