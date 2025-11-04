import { useEffect, useState } from 'react';
import { useSearchParams, useNavigate } from 'react-router-dom';
import { Loader2, CheckCircle, XCircle, ArrowLeft } from 'lucide-react';
import { Button } from '../components/ui/button';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '../components/ui/card';


export const OAuth2CallbackPage = () => {
  const [searchParams] = useSearchParams();
  const navigate = useNavigate();
  const [status, setStatus] = useState<'processing' | 'success' | 'error'>('processing');
  const [message, setMessage] = useState('正在处理授权...');
  const [accountInfo, setAccountInfo] = useState<{ uid: string; email: string } | null>(null);

  useEffect(() => {
    const handleCallback = async () => {
      try {
        // 获取 URL 参数
        const success = searchParams.get('success');
        const accountUid = searchParams.get('account_uid');
        const email = searchParams.get('email');
        const error = searchParams.get('error');

        // 检查是否有错误
        if (error) {
          throw new Error(getErrorMessage(error));
        }

        // 检查是否是成功回调
        if (success === 'true' && accountUid && email) {
          // 后端已经处理完成，直接显示成功
          setStatus('success');
          setMessage('账户添加成功！');
          setAccountInfo({ uid: accountUid, email: decodeURIComponent(email) });

          // 清理 sessionStorage
          const provider = sessionStorage.getItem('oauth2_provider');
          if (provider) {
            sessionStorage.removeItem(`oauth2_state_${provider}`);
            sessionStorage.removeItem('oauth2_provider');
            sessionStorage.removeItem('oauth2_email');
          }

          // 存储结果供父窗口使用
          sessionStorage.setItem('oauth2_auth_result', JSON.stringify({
            success: true,
            account_uid: accountUid,
            email: decodeURIComponent(email),
          }));

          // 如果是弹窗，关闭窗口
          if (window.opener) {
            setTimeout(() => {
              window.close();
            }, 2000);
          } else {
            // 如果不是弹窗，3秒后跳转到账户页面
            setTimeout(() => {
              navigate('/accounts');
            }, 3000);
          }
          return;
        }

        // 如果没有成功参数，说明可能是旧的流程，抛出错误
        throw new Error('授权处理失败，请重试');

      } catch (error) {
        console.error('OAuth2 callback error:', error);
        
        const errorMessage = error instanceof Error ? error.message : '授权处理失败';
        setStatus('error');
        setMessage(errorMessage);

        // 存储错误结果
        sessionStorage.setItem('oauth2_auth_result', JSON.stringify({
          success: false,
          error: errorMessage,
        }));

        // 清理 sessionStorage
        const provider = sessionStorage.getItem('oauth2_provider');
        if (provider) {
          sessionStorage.removeItem(`oauth2_state_${provider}`);
          sessionStorage.removeItem('oauth2_provider');
          sessionStorage.removeItem('oauth2_email');
        }

        // 如果是弹窗，延迟关闭
        if (window.opener) {
          setTimeout(() => {
            window.close();
          }, 3000);
        }
      }
    };

    handleCallback();
  }, [searchParams, navigate]);

  const getErrorMessage = (error: string): string => {
    switch (error) {
      case 'access_denied':
        return '用户拒绝了授权请求';
      case 'invalid_request':
        return '授权请求无效';
      case 'unauthorized_client':
        return '客户端未授权';
      case 'unsupported_response_type':
        return '不支持的响应类型';
      case 'invalid_scope':
        return '无效的授权范围';
      case 'server_error':
        return '服务器错误，请稍后重试';
      case 'temporarily_unavailable':
        return '服务暂时不可用，请稍后重试';
      case 'missing_parameters':
        return '缺少必要的授权参数';
      case 'callback_failed':
        return '授权回调处理失败，请重试';
      default:
        return `授权失败：${error}`;
    }
  };

  const handleRetry = () => {
    if (window.opener) {
      window.close();
    } else {
      navigate('/accounts');
    }
  };

  const getStatusIcon = () => {
    switch (status) {
      case 'processing':
        return <Loader2 className="h-8 w-8 animate-spin text-blue-500" />;
      case 'success':
        return <CheckCircle className="h-8 w-8 text-green-500" />;
      case 'error':
        return <XCircle className="h-8 w-8 text-red-500" />;
    }
  };

  const getStatusColor = () => {
    switch (status) {
      case 'processing':
        return 'border-blue-200 bg-blue-50 dark:border-blue-800 dark:bg-blue-950';
      case 'success':
        return 'border-green-200 bg-green-50 dark:border-green-800 dark:bg-green-950';
      case 'error':
        return 'border-red-200 bg-red-50 dark:border-red-800 dark:bg-red-950';
    }
  };

  return (
    <div className="min-h-screen flex items-center justify-center bg-gray-50 dark:bg-gray-900 p-4">
      <Card className={`w-full max-w-md ${getStatusColor()}`}>
        <CardHeader className="text-center">
          <div className="flex justify-center mb-4">
            {getStatusIcon()}
          </div>
          <CardTitle className="text-xl">
            {status === 'processing' && 'OAuth2 授权处理中'}
            {status === 'success' && '授权成功'}
            {status === 'error' && '授权失败'}
          </CardTitle>
          <CardDescription>
            {message}
          </CardDescription>
        </CardHeader>

        <CardContent className="space-y-4">
          {status === 'success' && accountInfo && (
            <div className="bg-white dark:bg-gray-800 rounded-lg p-4 border">
              <h4 className="font-medium text-sm text-gray-900 dark:text-white mb-2">
                账户信息
              </h4>
              <div className="space-y-1 text-sm text-gray-600 dark:text-gray-400">
                <div>邮箱：{accountInfo.email}</div>
                <div>账户 ID：{accountInfo.uid}</div>
              </div>
            </div>
          )}

          {status === 'success' && (
            <div className="text-center text-sm text-gray-600 dark:text-gray-400">
              {window.opener ? (
                '窗口将在 2 秒后自动关闭...'
              ) : (
                '3 秒后将跳转到账户管理页面...'
              )}
            </div>
          )}

          {status === 'error' && (
            <div className="space-y-3">
              <div className="bg-white dark:bg-gray-800 rounded-lg p-4 border">
                <h4 className="font-medium text-sm text-red-600 dark:text-red-400 mb-2">
                  错误详情
                </h4>
                <p className="text-sm text-gray-600 dark:text-gray-400">
                  {message}
                </p>
              </div>

              <div className="flex space-x-2">
                <Button
                  onClick={handleRetry}
                  variant="outline"
                  className="flex-1"
                >
                  <ArrowLeft className="mr-2 h-4 w-4" />
                  返回
                </Button>
                {!window.opener && (
                  <Button
                    onClick={() => window.location.reload()}
                    variant="default"
                    className="flex-1"
                  >
                    重试
                  </Button>
                )}
              </div>
            </div>
          )}

          {status === 'processing' && (
            <div className="text-center">
              <div className="animate-pulse space-y-2">
                <div className="h-2 bg-blue-200 dark:bg-blue-800 rounded w-3/4 mx-auto"></div>
                <div className="h-2 bg-blue-200 dark:bg-blue-800 rounded w-1/2 mx-auto"></div>
              </div>
            </div>
          )}
        </CardContent>
      </Card>
    </div>
  );
};