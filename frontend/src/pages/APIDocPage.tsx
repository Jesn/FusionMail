import { useState } from 'react';
import { Copy, Check, ChevronDown, ChevronUp } from 'lucide-react';
import { Button } from '@/components/ui/button';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import { toast } from 'sonner';

interface APIEndpoint {
  method: 'GET' | 'POST' | 'PUT' | 'DELETE';
  title: string;
  path: string;
  description: string;
  params?: Array<{
    name: string;
    type: string;
    required: boolean;
    description: string;
  }>;
  curlExample: string;
  responseExample: string;
}

export const APIDocPage = () => {
  const [copiedId, setCopiedId] = useState<string | null>(null);
  const [expandedEndpoint, setExpandedEndpoint] = useState<string | null>(null);

  const copyToClipboard = (text: string, id: string) => {
    navigator.clipboard.writeText(text);
    setCopiedId(id);
    toast.success('已复制到剪贴板');
    setTimeout(() => setCopiedId(null), 2000);
  };

  const apiKey = '9JSMwAyfXgKM9N9ApPAOsIjuoZDDp0SyAhXxjPiRPyg=';
  const baseUrl = 'http://localhost:3333/api/v1/public';

  const endpoints: APIEndpoint[] = [
    {
      method: 'GET',
      title: '实时拉取邮件',
      path: '/mail/receive',
      description: '通过 API Key 实时拉取指定邮箱的邮件列表',
      params: [
        { name: 'email', type: 'string', required: true, description: '邮箱地址' },
        { name: 'limit', type: 'int', required: false, description: '返回邮件数量（默认 20，最大 100）' },
        { name: 'offset', type: 'int', required: false, description: '偏移量（默认 0）' },
        { name: 'is_read', type: 'bool', required: false, description: '是否已读' },
        { name: 'is_starred', type: 'bool', required: false, description: '是否星标' },
        { name: 'is_archived', type: 'bool', required: false, description: '是否归档' },
      ],
      curlExample: `curl -X GET "${baseUrl}/mail/receive?email=user@example.com&limit=5" \\
  -H "Authorization: Bearer ${apiKey}"`,
      responseExample: `{
  "success": true,
  "data": {
    "emails": [
      {
        "id": 1,
        "subject": "Hello",
        "from_address": "sender@example.com",
        "from_name": "Sender",
        "to_address": "user@example.com",
        "snippet": "Email content...",
        "is_read": false,
        "is_starred": false,
        "is_archived": false,
        "sent_at": "2024-01-15T10:30:00Z"
      }
    ],
    "total": 100,
    "limit": 5,
    "offset": 0
  }
}`,
    },
    {
      method: 'GET',
      title: '搜索邮件',
      path: '/mail/search',
      description: '通过 API Key 搜索指定邮箱的邮件',
      params: [
        { name: 'email', type: 'string', required: true, description: '邮箱地址' },
        { name: 'q', type: 'string', required: true, description: '搜索关键词' },
        { name: 'limit', type: 'int', required: false, description: '返回邮件数量（默认 20，最大 100）' },
        { name: 'offset', type: 'int', required: false, description: '偏移量（默认 0）' },
      ],
      curlExample: `curl -X GET "${baseUrl}/mail/search?email=user@example.com&q=important" \\
  -H "Authorization: Bearer ${apiKey}"`,
      responseExample: `{
  "success": true,
  "data": {
    "emails": [
      {
        "id": 1,
        "subject": "Important Email",
        "from_address": "sender@example.com",
        "to_address": "user@example.com",
        "snippet": "This is important...",
        "sent_at": "2024-01-15T10:30:00Z"
      }
    ],
    "total": 5,
    "limit": 10,
    "offset": 0,
    "query": "important"
  }
}`,
    },
  ];

  const getMethodColor = (method: string) => {
    switch (method) {
      case 'GET':
        return 'bg-blue-100 text-blue-800';
      case 'POST':
        return 'bg-green-100 text-green-800';
      case 'PUT':
        return 'bg-yellow-100 text-yellow-800';
      case 'DELETE':
        return 'bg-red-100 text-red-800';
      default:
        return 'bg-gray-100 text-gray-800';
    }
  };

  return (
    <div className="min-h-screen bg-background">
      {/* 页面头部 */}
      <div className="bg-card border-b">
        <div className="max-w-6xl mx-auto px-6 py-8">
          <div className="flex items-center gap-3 mb-4">
            <div className="text-3xl">📚</div>
            <h1 className="text-3xl font-bold">FusionMail API 文档</h1>
          </div>
          <p className="text-gray-600">详细的 API 端点说明和使用示例</p>
        </div>
      </div>

      <div className="max-w-6xl mx-auto px-6 py-8">
        {/* 快速开始 */}
        <Card className="mb-8">
          <CardHeader>
            <CardTitle className="text-xl">🚀 快速开始</CardTitle>
          </CardHeader>
          <CardContent className="space-y-4">
            <ol className="space-y-2 text-sm">
              <li className="flex gap-3">
                <span className="font-semibold text-blue-600 min-w-6">1.</span>
                <span>前往 <a href="/api-keys" className="text-blue-600 hover:underline">API 密钥管理页面</a> 创建 API 密钥</span>
              </li>
              <li className="flex gap-3">
                <span className="font-semibold text-blue-600 min-w-6">2.</span>
                <span>复制生成的 API 密钥</span>
              </li>
              <li className="flex gap-3">
                <span className="font-semibold text-blue-600 min-w-6">3.</span>
                <span>选择下方任意端点，将示例中的 <code className="bg-gray-100 px-2 py-1 rounded text-xs">YOUR_API_KEY</code> 替换为你的实际密钥</span>
              </li>
              <li className="flex gap-3">
                <span className="font-semibold text-blue-600 min-w-6">4.</span>
                <span>在终端中执行 curl 命令开始使用</span>
              </li>
            </ol>
          </CardContent>
        </Card>

        {/* API 使用文档 */}
        <div className="mb-8">
          <div className="mb-6">
            <h2 className="text-2xl font-bold mb-2">API 使用文档</h2>
            <p className="text-gray-600 text-sm">
              以下是外部 API 端点的详细说明。请将示例中的 <code className="bg-gray-100 px-2 py-1 rounded text-xs">YOUR_API_KEY</code> 替换为你的实际 API 密钥。
            </p>
          </div>

          {/* API 端点列表 */}
          <div className="space-y-3">
            {endpoints.map((endpoint, index) => {
              const endpointId = `${endpoint.method}-${index}`;
              const isExpanded = expandedEndpoint === endpointId;

              return (
                <div key={endpointId} className="border rounded-lg bg-card overflow-hidden">
                  {/* 端点头部 - 可点击 */}
                  <button
                    onClick={() => setExpandedEndpoint(isExpanded ? null : endpointId)}
                    className="w-full px-6 py-4 hover:bg-accent transition-colors flex items-center justify-between"
                  >
                    <div className="flex items-center gap-4 flex-1 text-left">
                      <span className={`px-3 py-1 rounded font-semibold text-sm ${getMethodColor(endpoint.method)}`}>
                        {endpoint.method}
                      </span>
                      <div className="flex-1">
                        <div className="font-semibold text-gray-900">{endpoint.title}</div>
                        <div className="text-sm text-gray-600 font-mono">{endpoint.path}</div>
                      </div>
                    </div>
                    {isExpanded ? (
                      <ChevronUp className="h-5 w-5 text-gray-400" />
                    ) : (
                      <ChevronDown className="h-5 w-5 text-gray-400" />
                    )}
                  </button>

                  {/* 端点详情 - 展开时显示 */}
                  {isExpanded && (
                    <div className="border-t px-6 py-4 bg-muted/50 space-y-6">
                      {/* 描述 */}
                      <div>
                        <p className="text-gray-700">{endpoint.description}</p>
                      </div>

                      {/* 请求参数 */}
                      {endpoint.params && endpoint.params.length > 0 && (
                        <div>
                          <h4 className="font-semibold mb-3 text-gray-900">请求参数</h4>
                          <div className="overflow-x-auto">
                            <table className="w-full text-sm">
                              <thead>
                                <tr className="border-b bg-muted/30">
                                  <th className="text-left py-2 px-3 font-semibold">参数</th>
                                  <th className="text-left py-2 px-3 font-semibold">类型</th>
                                  <th className="text-left py-2 px-3 font-semibold text-gray-700">必需</th>
                                  <th className="text-left py-2 px-3 font-semibold text-gray-700">说明</th>
                                </tr>
                              </thead>
                              <tbody>
                                {endpoint.params.map((param, idx) => (
                                  <tr key={idx} className="border-b">
                                    <td className="py-2 px-3 font-mono text-xs text-blue-600">{param.name}</td>
                                    <td className="py-2 px-3 text-gray-600">{param.type}</td>
                                    <td className="py-2 px-3">{param.required ? '✅' : '❌'}</td>
                                    <td className="py-2 px-3 text-gray-600">{param.description}</td>
                                  </tr>
                                ))}
                              </tbody>
                            </table>
                          </div>
                        </div>
                      )}

                      {/* cURL 示例 */}
                      <div>
                        <div className="flex items-center justify-between mb-2">
                          <h4 className="font-semibold text-gray-900">cURL 示例</h4>
                          <Button
                            size="sm"
                            variant="outline"
                            onClick={() => copyToClipboard(endpoint.curlExample, `curl-${endpointId}`)}
                            className="gap-2"
                          >
                            {copiedId === `curl-${endpointId}` ? (
                              <>
                                <Check className="h-4 w-4" />
                                已复制
                              </>
                            ) : (
                              <>
                                <Copy className="h-4 w-4" />
                                复制
                              </>
                            )}
                          </Button>
                        </div>
                        <pre className="bg-gray-900 text-gray-100 p-4 rounded overflow-x-auto text-xs font-mono">
                          {endpoint.curlExample}
                        </pre>
                      </div>

                      {/* 响应示例 */}
                      <div>
                        <div className="flex items-center justify-between mb-2">
                          <h4 className="font-semibold text-gray-900">成功响应</h4>
                          <Button
                            size="sm"
                            variant="outline"
                            onClick={() => copyToClipboard(endpoint.responseExample, `response-${endpointId}`)}
                            className="gap-2"
                          >
                            {copiedId === `response-${endpointId}` ? (
                              <>
                                <Check className="h-4 w-4" />
                                已复制
                              </>
                            ) : (
                              <>
                                <Copy className="h-4 w-4" />
                                复制
                              </>
                            )}
                          </Button>
                        </div>
                        <pre className="bg-gray-900 text-gray-100 p-4 rounded overflow-x-auto text-xs font-mono">
                          {endpoint.responseExample}
                        </pre>
                      </div>
                    </div>
                  )}
                </div>
              );
            })}
          </div>
        </div>

        {/* 重要提示 */}
        <Card className="border-yellow-200 bg-yellow-50">
          <CardHeader>
            <CardTitle className="text-lg">⚠️ 重要提示</CardTitle>
          </CardHeader>
          <CardContent>
            <ul className="space-y-2 text-sm text-gray-700">
              <li className="flex gap-2">
                <span>•</span>
                <span>所有 API 请求都需要在请求头中包含 <code className="bg-white px-2 py-1 rounded text-xs">Authorization: Bearer &lt;API_KEY&gt;</code></span>
              </li>
              <li className="flex gap-2">
                <span>•</span>
                <span>API Key 只能访问 <code className="bg-white px-2 py-1 rounded text-xs">/api/v1/public/</code> 下的公共接口</span>
              </li>
              <li className="flex gap-2">
                <span>•</span>
                <span>请确保在安全环境下使用 API Key，不要在公开的代码中暴露密钥</span>
              </li>
              <li className="flex gap-2">
                <span>•</span>
                <span>删除 API 密钥后，使用该密钥的请求将返回 401 错误</span>
              </li>
            </ul>
          </CardContent>
        </Card>
      </div>
    </div>
  );
};

