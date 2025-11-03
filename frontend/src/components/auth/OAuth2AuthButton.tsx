import { useState } from 'react';
import { Button } from '../ui/button';
import { Badge } from '../ui/badge';
import { Loader2, Shield, ExternalLink, AlertCircle } from 'lucide-react';
import { oauth2Service, OAuth2Provider } from '../../services/oauth2Service';
import { toast } from 'sonner';

interface OAuth2AuthButtonProps {
  provider: OAuth2Provider;
  email?: string;
  onSuccess?: (accountUid: string, email: string) => void;
  onError?: (error: string) => void;
  disabled?: boolean;
  className?: string;
}

const providerConfig = {
  google: {
    name: 'Google',
    displayName: 'Gmail',
    color: 'bg-red-500 hover:bg-red-600',
    icon: '📧',
    description: '使用 Google 账号安全登录',
  },
  microsoft: {
    name: 'Microsoft',
    displayName: 'Outlook',
    color: 'bg-blue-500 hover:bg-blue-600',
    icon: '📮',
    description: '使用 Microsoft 账号安全登录',
  },
};

export const OAuth2AuthButton = ({
  provider,
  email,
  onSuccess,
  onError,
  disabled = false,
  className = '',
}: OAuth2AuthButtonProps) => {
  const [isLoading, setIsLoading] = useState(false);
  const config = providerConfig[provider];

  const handleAuth = async () => {
    if (disabled || isLoading) return;

    setIsLoading(true);
    try {
      // 生成授权 URL
      const authResponse = await oauth2Service.generateAuthUrl(provider, email);
      
      // 存储 state 到 sessionStorage 用于回调验证
      sessionStorage.setItem(`oauth2_state_${provider}`, authResponse.state);
      sessionStorage.setItem(`oauth2_provider`, provider);
      if (email) {
        sessionStorage.setItem(`oauth2_email`, email);
      }

      // 打开授权页面
      const authWindow = window.open(
        authResponse.auth_url,
        'oauth2_auth',
        'width=500,height=600,scrollbars=yes,resizable=yes'
      );

      if (!authWindow) {
        throw new Error('无法打开授权窗口，请检查浏览器弹窗设置');
      }

      // 监听来自弹窗的消息
      const handleMessage = (event: MessageEvent) => {
        console.log('收到消息:', event.data);
        
        if (event.data && event.data.type === 'oauth2_result') {
          console.log('收到 OAuth2 结果消息:', event.data.data);
          
          clearInterval(checkClosed);
          setIsLoading(false);
          
          const result = event.data.data;
          
          if (result.success) {
            console.log('授权成功，调用 onSuccess 回调');
            toast.success(`${config.displayName} 账户添加成功！`);
            onSuccess?.(result.account_uid, result.email);
          } else {
            const errorMessage = result.error || '授权失败';
            console.error('授权失败:', errorMessage);
            toast.error(errorMessage);
            onError?.(errorMessage);
          }
          
          // 移除事件监听器
          window.removeEventListener('message', handleMessage);
        }
      };
      
      // 添加消息监听器
      window.addEventListener('message', handleMessage);

      // 监听授权窗口关闭
      const checkClosed = setInterval(() => {
        if (authWindow.closed) {
          clearInterval(checkClosed);
          setIsLoading(false);
          
          console.log('OAuth2 窗口已关闭，检查授权结果...');
          
          // 移除事件监听器
          window.removeEventListener('message', handleMessage);
          
          // 检查是否有授权结果（备用方案）
          const authResult = sessionStorage.getItem('oauth2_auth_result');
          console.log('获取到的授权结果:', authResult);
          
          if (authResult) {
            try {
              const result = JSON.parse(authResult);
              sessionStorage.removeItem('oauth2_auth_result');
              
              console.log('解析的授权结果:', result);
              
              if (result.success) {
                console.log('授权成功，调用 onSuccess 回调');
                toast.success(`${config.displayName} 账户添加成功！`);
                onSuccess?.(result.account_uid, result.email);
              } else {
                throw new Error(result.error || '授权失败');
              }
            } catch (error) {
              const errorMessage = error instanceof Error ? error.message : '授权处理失败';
              console.error('处理授权结果时出错:', error);
              toast.error(errorMessage);
              onError?.(errorMessage);
            }
          } else {
            console.log('没有找到授权结果，可能用户取消了授权');
          }
        }
      }, 1000);

      // 设置超时
      setTimeout(() => {
        if (!authWindow.closed) {
          authWindow.close();
          clearInterval(checkClosed);
          setIsLoading(false);
          toast.error('授权超时，请重试');
        }
      }, 300000); // 5分钟超时

    } catch (error) {
      setIsLoading(false);
      const errorMessage = error instanceof Error ? error.message : '启动授权失败';
      toast.error(errorMessage);
      onError?.(errorMessage);
    }
  };

  return (
    <div className={`space-y-2 ${className}`}>
      <Button
        onClick={handleAuth}
        disabled={disabled || isLoading}
        className={`w-full ${config.color} text-white relative`}
        size="lg"
      >
        {isLoading ? (
          <>
            <Loader2 className="mr-2 h-4 w-4 animate-spin" />
            正在授权...
          </>
        ) : (
          <>
            <span className="mr-2 text-lg">{config.icon}</span>
            使用 {config.displayName} 账号登录
            <ExternalLink className="ml-2 h-4 w-4" />
          </>
        )}
      </Button>

      <div className="flex items-center justify-between text-xs text-muted-foreground">
        <div className="flex items-center space-x-1">
          <Shield className="h-3 w-3" />
          <span>安全的 OAuth2 认证</span>
        </div>
        <Badge variant="secondary" className="text-xs">
          推荐
        </Badge>
      </div>

      <p className="text-xs text-muted-foreground">
        {config.description}，无需提供密码
      </p>

      {email && (
        <div className="flex items-center space-x-1 text-xs text-blue-600 dark:text-blue-400">
          <AlertCircle className="h-3 w-3" />
          <span>将为 {email} 添加 {config.displayName} 账户</span>
        </div>
      )}
    </div>
  );
};