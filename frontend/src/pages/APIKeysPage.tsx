import { useState, useEffect } from 'react';
import { Plus, Copy, Trash2, Eye, EyeOff, Check } from 'lucide-react';
import { Button } from '../components/ui/button';
import { Card, CardContent, CardHeader, CardTitle } from '../components/ui/card';
import { Input } from '../components/ui/input';
import { Label } from '../components/ui/label';
import { Separator } from '../components/ui/separator';
import { toast } from 'sonner';
import { apiKeyService, APIKey, CreateAPIKeyRequest } from '../services/apiKeyService';
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
  DialogTrigger,
} from '../components/ui/dialog';
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogHeader,
  AlertDialogTitle,
} from '../components/ui/alert-dialog';

export const APIKeysPage = () => {
  const [apiKeys, setApiKeys] = useState<APIKey[]>([]);
  const [loading, setLoading] = useState(false);
  const [showCreateDialog, setShowCreateDialog] = useState(false);
  const [showDeleteDialog, setShowDeleteDialog] = useState(false);
  const [selectedKeyId, setSelectedKeyId] = useState<number | null>(null);
  const [plaintextKey, setPlaintextKey] = useState('');
  const [showKeyDialog, setShowKeyDialog] = useState(false);

  // 创建表单状态
  const [formData, setFormData] = useState<CreateAPIKeyRequest>({
    name: '',
    description: '',
    rate_limit: 100,
  });

  // 加载 API Key 列表
  const loadApiKeys = async () => {
    try {
      setLoading(true);
      const keys = await apiKeyService.list();
      setApiKeys(keys);
    } catch (error) {
      toast.error('加载 API Key 列表失败');
      console.error(error);
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    loadApiKeys();
  }, []);

  // 创建 API Key
  const handleCreate = async () => {
    if (!formData.name.trim()) {
      toast.error('请输入 API Key 名称');
      return;
    }

    try {
      setLoading(true);
      const response = await apiKeyService.create(formData);
      
      // 显示明文 Key
      setPlaintextKey(response.api_key);
      setShowKeyDialog(true);
      
      // 重置表单
      setFormData({
        name: '',
        description: '',
        rate_limit: 100,
      });
      setShowCreateDialog(false);
      
      // 重新加载列表
      await loadApiKeys();
      toast.success('API Key 创建成功');
    } catch (error) {
      toast.error('创建 API Key 失败');
      console.error(error);
    } finally {
      setLoading(false);
    }
  };

  // 删除 API Key
  const handleDelete = async () => {
    if (selectedKeyId === null) return;

    try {
      setLoading(true);
      await apiKeyService.delete(selectedKeyId);
      await loadApiKeys();
      setShowDeleteDialog(false);
      setSelectedKeyId(null);
      toast.success('API Key 删除成功');
    } catch (error) {
      toast.error('删除 API Key 失败');
      console.error(error);
    } finally {
      setLoading(false);
    }
  };

  // 启用/禁用 API Key
  const handleToggleStatus = async (id: number, enabled: boolean) => {
    try {
      setLoading(true);
      if (enabled) {
        await apiKeyService.disable(id);
      } else {
        await apiKeyService.enable(id);
      }
      await loadApiKeys();
      toast.success(enabled ? 'API Key 已禁用' : 'API Key 已启用');
    } catch (error) {
      toast.error('操作失败');
      console.error(error);
    } finally {
      setLoading(false);
    }
  };

  // 复制到剪贴板
  const handleCopyToClipboard = (text: string) => {
    navigator.clipboard.writeText(text);
    toast.success('已复制到剪贴板');
  };

  return (
    <div className="container mx-auto px-4 py-6">
      <div className="max-w-6xl mx-auto">
        {/* 页面头部 */}
        <div className="flex items-center justify-between mb-6">
          <div>
            <h1 className="text-2xl font-bold text-gray-900 dark:text-white">
              API Key 管理
            </h1>
            <p className="text-gray-600 dark:text-gray-400 mt-1">
              创建和管理用于程序化访问的 API Key
            </p>
          </div>
          <Dialog open={showCreateDialog} onOpenChange={setShowCreateDialog}>
            <DialogTrigger asChild>
              <Button>
                <Plus className="h-4 w-4 mr-2" />
                创建 API Key
              </Button>
            </DialogTrigger>
            <DialogContent>
              <DialogHeader>
                <DialogTitle>创建新的 API Key</DialogTitle>
                <DialogDescription>
                  创建一个新的 API Key 用于程序化访问
                </DialogDescription>
              </DialogHeader>
              <div className="space-y-4">
                <div className="space-y-2">
                  <Label htmlFor="name">名称 *</Label>
                  <Input
                    id="name"
                    placeholder="例如：My App"
                    value={formData.name}
                    onChange={(e) =>
                      setFormData({ ...formData, name: e.target.value })
                    }
                  />
                </div>
                <div className="space-y-2">
                  <Label htmlFor="description">描述</Label>
                  <Input
                    id="description"
                    placeholder="例如：用于生产环境的 API Key"
                    value={formData.description}
                    onChange={(e) =>
                      setFormData({ ...formData, description: e.target.value })
                    }
                  />
                </div>
                <div className="space-y-2">
                  <Label htmlFor="rateLimit">速率限制 (请求/分钟)</Label>
                  <Input
                    id="rateLimit"
                    type="number"
                    min="1"
                    max="10000"
                    value={formData.rate_limit}
                    onChange={(e) =>
                      setFormData({
                        ...formData,
                        rate_limit: parseInt(e.target.value) || 100,
                      })
                    }
                  />
                </div>
                <Button
                  onClick={handleCreate}
                  disabled={loading}
                  className="w-full"
                >
                  {loading ? '创建中...' : '创建'}
                </Button>
              </div>
            </DialogContent>
          </Dialog>
        </div>

        {/* API Key 列表 */}
        <Card>
          <CardHeader>
            <CardTitle>API Key 列表</CardTitle>
          </CardHeader>
          <CardContent>
            {loading && apiKeys.length === 0 ? (
              <div className="text-center py-8 text-gray-500">
                加载中...
              </div>
            ) : apiKeys.length === 0 ? (
              <div className="text-center py-8 text-gray-500">
                还没有创建任何 API Key
              </div>
            ) : (
              <div className="space-y-4">
                {apiKeys.map((key) => (
                  <div
                    key={key.id}
                    className="flex items-center justify-between p-4 border rounded-lg hover:bg-gray-50 dark:hover:bg-gray-800"
                  >
                    <div className="flex-1">
                      <div className="flex items-center gap-2">
                        <h3 className="font-semibold text-gray-900 dark:text-white">
                          {key.name}
                        </h3>
                        <span
                          className={`px-2 py-1 text-xs rounded-full ${
                            key.enabled
                              ? 'bg-green-100 text-green-800 dark:bg-green-900 dark:text-green-200'
                              : 'bg-red-100 text-red-800 dark:bg-red-900 dark:text-red-200'
                          }`}
                        >
                          {key.enabled ? '启用' : '禁用'}
                        </span>
                      </div>
                      <p className="text-sm text-gray-600 dark:text-gray-400 mt-1">
                        {key.description}
                      </p>
                      <div className="flex gap-4 mt-2 text-xs text-gray-500 dark:text-gray-400">
                        <span>速率限制: {key.rate_limit} 请求/分钟</span>
                        <span>总请求数: {key.total_requests}</span>
                        {key.last_used_at && (
                          <span>
                            最后使用: {new Date(key.last_used_at).toLocaleString()}
                          </span>
                        )}
                      </div>
                    </div>
                    <div className="flex items-center gap-2">
                      <Button
                        variant="outline"
                        size="sm"
                        onClick={() =>
                          handleToggleStatus(key.id, key.enabled)
                        }
                        disabled={loading}
                      >
                        {key.enabled ? (
                          <>
                            <EyeOff className="h-4 w-4 mr-1" />
                            禁用
                          </>
                        ) : (
                          <>
                            <Eye className="h-4 w-4 mr-1" />
                            启用
                          </>
                        )}
                      </Button>
                      <Button
                        variant="outline"
                        size="sm"
                        onClick={() => {
                          setSelectedKeyId(key.id);
                          setShowDeleteDialog(true);
                        }}
                        disabled={loading}
                      >
                        <Trash2 className="h-4 w-4" />
                      </Button>
                    </div>
                  </div>
                ))}
              </div>
            )}
          </CardContent>
        </Card>

        {/* 使用说明 */}
        <Card className="mt-6">
          <CardHeader>
            <CardTitle>使用说明</CardTitle>
          </CardHeader>
          <CardContent className="space-y-4">
            <div>
              <h4 className="font-semibold mb-2">鉴权方式</h4>
              <div className="space-y-2 text-sm text-gray-600 dark:text-gray-400">
                <p>
                  使用 <code className="bg-gray-100 dark:bg-gray-800 px-2 py-1 rounded">Authorization: Bearer &lt;API_KEY&gt;</code> 格式进行鉴权
                </p>
                <p className="text-xs text-gray-500 dark:text-gray-500 mt-2">
                  示例：
                </p>
                <p>
                  <code className="bg-gray-100 dark:bg-gray-800 px-2 py-1 rounded text-xs">
                    curl -H "Authorization: Bearer your_api_key_here" https://api.example.com/api/v1/emails
                  </code>
                </p>
              </div>
            </div>
            <Separator />
            <div>
              <h4 className="font-semibold mb-2">安全建议</h4>
              <ul className="list-disc list-inside space-y-1 text-sm text-gray-600 dark:text-gray-400">
                <li>妥善保管 API Key，不要在代码中硬编码</li>
                <li>定期轮换 API Key</li>
                <li>为不同的应用创建不同的 API Key</li>
                <li>禁用不再使用的 API Key</li>
              </ul>
            </div>
          </CardContent>
        </Card>
      </div>

      {/* 显示明文 Key 的对话框 */}
      <Dialog open={showKeyDialog} onOpenChange={setShowKeyDialog}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>API Key 已创建</DialogTitle>
            <DialogDescription>
              请妥善保管此密钥，此密钥仅显示一次
            </DialogDescription>
          </DialogHeader>
          <div className="space-y-4">
            <div className="bg-yellow-50 dark:bg-yellow-900/20 border border-yellow-200 dark:border-yellow-800 rounded-lg p-4">
              <p className="text-sm text-yellow-800 dark:text-yellow-200 mb-2">
                ⚠️ 重要提示
              </p>
              <p className="text-sm text-yellow-700 dark:text-yellow-300">
                这是你唯一一次看到这个密钥的机会。请立即复制并保存到安全的地方。
              </p>
            </div>
            <div className="bg-gray-100 dark:bg-gray-800 rounded-lg p-4 break-all font-mono text-sm">
              {plaintextKey}
            </div>
            <Button
              onClick={() => handleCopyToClipboard(plaintextKey)}
              className="w-full"
            >
              <Copy className="h-4 w-4 mr-2" />
              复制到剪贴板
            </Button>
            <Button
              onClick={() => setShowKeyDialog(false)}
              variant="outline"
              className="w-full"
            >
              我已保存
            </Button>
          </div>
        </DialogContent>
      </Dialog>

      {/* 删除确认对话框 */}
      <AlertDialog open={showDeleteDialog} onOpenChange={setShowDeleteDialog}>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>删除 API Key</AlertDialogTitle>
            <AlertDialogDescription>
              确定要删除此 API Key 吗？此操作无法撤销。
            </AlertDialogDescription>
          </AlertDialogHeader>
          <div className="flex gap-4">
            <AlertDialogCancel>取消</AlertDialogCancel>
            <AlertDialogAction
              onClick={handleDelete}
              disabled={loading}
              className="bg-red-600 hover:bg-red-700"
            >
              {loading ? '删除中...' : '删除'}
            </AlertDialogAction>
          </div>
        </AlertDialogContent>
      </AlertDialog>
    </div>
  );
};

