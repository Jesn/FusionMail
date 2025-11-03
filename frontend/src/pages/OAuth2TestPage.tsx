import { useState } from 'react';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '../components/ui/card';
import { Button } from '../components/ui/button';
import { Input } from '../components/ui/input';
import { Label } from '../components/ui/label';
import { OAuth2AuthButton } from '../components/auth/OAuth2AuthButton';
import { oauth2Service } from '../services/oauth2Service';
import { toast } from 'sonner';

export const OAuth2TestPage = () => {
  const [email, setEmail] = useState('');
  const [testResults, setTestResults] = useState<string[]>([]);

  const addTestResult = (result: string) => {
    setTestResults(prev => [...prev, `${new Date().toLocaleTimeString()}: ${result}`]);
  };

  const testGoogleAuthUrl = async () => {
    try {
      const response = await oauth2Service.generateGoogleAuthUrl(email);
      addTestResult(`Google 授权 URL 生成成功: ${response.auth_url.substring(0, 100)}...`);
      toast.success('Google 授权 URL 生成成功');
    } catch (error) {
      const errorMessage = error instanceof Error ? error.message : '未知错误';
      addTestResult(`Google 授权 URL 生成失败: ${errorMessage}`);
      toast.error('Google 授权 URL 生成失败');
    }
  };

  const testMicrosoftAuthUrl = async () => {
    try {
      const response = await oauth2Service.generateMicrosoftAuthUrl(email);
      addTestResult(`Microsoft 授权 URL 生成成功: ${response.auth_url.substring(0, 100)}...`);
      toast.success('Microsoft 授权 URL 生成成功');
    } catch (error) {
      const errorMessage = error instanceof Error ? error.message : '未知错误';
      addTestResult(`Microsoft 授权 URL 生成失败: ${errorMessage}`);
      toast.error('Microsoft 授权 URL 生成失败');
    }
  };

  const clearResults = () => {
    setTestResults([]);
  };

  return (
    <div className="container mx-auto p-6 space-y-6">
      <div className="text-center">
        <h1 className="text-3xl font-bold">OAuth2 集成测试</h1>
        <p className="text-muted-foreground mt-2">
          测试 Gmail 和 Microsoft OAuth2 认证功能
        </p>
      </div>

      <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
        {/* 测试控制面板 */}
        <Card>
          <CardHeader>
            <CardTitle>测试控制</CardTitle>
            <CardDescription>
              测试 OAuth2 服务的各项功能
            </CardDescription>
          </CardHeader>
          <CardContent className="space-y-4">
            <div className="space-y-2">
              <Label htmlFor="email">测试邮箱地址</Label>
              <Input
                id="email"
                type="email"
                placeholder="test@example.com"
                value={email}
                onChange={(e) => setEmail(e.target.value)}
              />
            </div>

            <div className="space-y-2">
              <h4 className="font-medium">API 测试</h4>
              <div className="flex space-x-2">
                <Button onClick={testGoogleAuthUrl} variant="outline" size="sm">
                  测试 Google API
                </Button>
                <Button onClick={testMicrosoftAuthUrl} variant="outline" size="sm">
                  测试 Microsoft API
                </Button>
              </div>
            </div>

            <div className="space-y-2">
              <h4 className="font-medium">OAuth2 组件测试</h4>
              <div className="space-y-3">
                <OAuth2AuthButton
                  provider="google"
                  email={email}
                  onSuccess={(accountUid, email) => {
                    addTestResult(`Google OAuth2 成功: ${email} (${accountUid})`);
                    toast.success('Google OAuth2 认证成功');
                  }}
                  onError={(error) => {
                    addTestResult(`Google OAuth2 失败: ${error}`);
                    toast.error('Google OAuth2 认证失败');
                  }}
                />
                
                <OAuth2AuthButton
                  provider="microsoft"
                  email={email}
                  onSuccess={(accountUid, email) => {
                    addTestResult(`Microsoft OAuth2 成功: ${email} (${accountUid})`);
                    toast.success('Microsoft OAuth2 认证成功');
                  }}
                  onError={(error) => {
                    addTestResult(`Microsoft OAuth2 失败: ${error}`);
                    toast.error('Microsoft OAuth2 认证失败');
                  }}
                />
              </div>
            </div>
          </CardContent>
        </Card>

        {/* 测试结果 */}
        <Card>
          <CardHeader>
            <CardTitle className="flex items-center justify-between">
              测试结果
              <Button onClick={clearResults} variant="outline" size="sm">
                清空
              </Button>
            </CardTitle>
            <CardDescription>
              OAuth2 功能测试的实时结果
            </CardDescription>
          </CardHeader>
          <CardContent>
            <div className="space-y-2 max-h-96 overflow-y-auto">
              {testResults.length === 0 ? (
                <p className="text-muted-foreground text-sm">
                  暂无测试结果
                </p>
              ) : (
                testResults.map((result, index) => (
                  <div
                    key={index}
                    className="text-sm p-2 bg-gray-50 dark:bg-gray-800 rounded border-l-4 border-blue-500"
                  >
                    {result}
                  </div>
                ))
              )}
            </div>
          </CardContent>
        </Card>
      </div>

      {/* 使用说明 */}
      <Card>
        <CardHeader>
          <CardTitle>使用说明</CardTitle>
        </CardHeader>
        <CardContent className="space-y-4">
          <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
            <div>
              <h4 className="font-medium mb-2">测试步骤</h4>
              <ol className="text-sm space-y-1 list-decimal list-inside text-muted-foreground">
                <li>输入测试邮箱地址</li>
                <li>点击"测试 API"按钮验证后端连接</li>
                <li>点击 OAuth2 按钮测试完整认证流程</li>
                <li>查看测试结果和控制台日志</li>
              </ol>
            </div>
            <div>
              <h4 className="font-medium mb-2">注意事项</h4>
              <ul className="text-sm space-y-1 list-disc list-inside text-muted-foreground">
                <li>需要配置有效的 OAuth2 客户端 ID</li>
                <li>确保后端服务正在运行</li>
                <li>检查浏览器弹窗设置</li>
                <li>测试环境需要添加测试用户</li>
              </ul>
            </div>
          </div>
        </CardContent>
      </Card>
    </div>
  );
};