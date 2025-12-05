import { useState, useEffect } from 'react';
import { Plus, Trash2, Star, AlertCircle, RefreshCw } from 'lucide-react';
import { Button } from '../components/ui/button';
import { Card, CardContent, CardHeader, CardTitle } from '../components/ui/card';
import { Input } from '../components/ui/input';
import { Label } from '../components/ui/label';
import { Textarea } from '../components/ui/textarea';
import { Switch } from '../components/ui/switch';
import { Badge } from '../components/ui/badge';
import { toast } from 'sonner';
import { oauth2ClientService } from '../services/oauth2ClientService';
import { useProviders } from '../hooks/useProviders';
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
import { OAuth2Client, OAuth2ClientCreateRequest, OAuth2ClientUpdateRequest } from '../types';
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '../components/ui/select';

// 动态生成 OAuth2 回调 URI，基于当前域名
const getOAuth2CallbackUri = (provider: 'google' | 'microsoft'): string => {
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

export const OAuth2ClientsPage = () => {
  const [clients, setClients] = useState<OAuth2Client[]>([]);
  const [loading, setLoading] = useState(false);
  const [showCreateDialog, setShowCreateDialog] = useState(false);
  const [showEditDialog, setShowEditDialog] = useState(false);
  const [showDeleteDialog, setShowDeleteDialog] = useState(false);
  const [selectedClient, setSelectedClient] = useState<OAuth2Client | null>(null);
  const { providers } = useProviders();

  // 创建表单状态
  const [createForm, setCreateForm] = useState<OAuth2ClientCreateRequest>({
    provider_id: 1, // Default to Gmail provider ID
    name: '',
    client_id: '',
    client_secret: '',
    redirect_uri: getOAuth2CallbackUri('google'),
    quota_daily: 100,
    quota_monthly: 2000,
    metadata: '',
  });

  // 编辑表单状态
  const [editForm, setEditForm] = useState<OAuth2ClientUpdateRequest>({});

  // 加载 OAuth2 客户端列表
  const loadClients = async () => {
    try {
      setLoading(true);
      const response = await oauth2ClientService.getList(1, 100);
      // 确保 data 是数组
      setClients(Array.isArray(response.data) ? response.data : []);
    } catch (error) {
      toast.error('加载 OAuth2 客户端列表失败');
      console.error(error);
      // 出错时设置为空数组
      setClients([]);
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    loadClients();
  }, []);

  // 创建 OAuth2 客户端
  const handleCreate = async () => {
    if (!createForm.name.trim()) {
      toast.error('请输入配置名称');
      return;
    }
    if (!createForm.client_id.trim()) {
      toast.error('请输入 Client ID');
      return;
    }
    if (!createForm.client_secret.trim()) {
      toast.error('请输入 Client Secret');
      return;
    }

    try {
      setLoading(true);
      await oauth2ClientService.create(createForm);
      toast.success('创建成功');
      setShowCreateDialog(false);
      setCreateForm({
        provider_id: 1, // Default to Gmail provider ID
        name: '',
        client_id: '',
        client_secret: '',
        redirect_uri: getOAuth2CallbackUri('google'),
        quota_daily: 100,
        quota_monthly: 2000,
        metadata: '',
      });
      loadClients();
    } catch (error) {
      toast.error('创建失败');
      console.error(error);
    } finally {
      setLoading(false);
    }
  };

  // 开始编辑
  const handleEdit = (client: OAuth2Client) => {
    setSelectedClient(client);

    // 解析metadata字段，如果是JSON格式则提取raw字段，否则使用原始值
    let description = client.metadata;
    if (client.metadata) {
      try {
        const parsed = JSON.parse(client.metadata);
        description = parsed.raw || client.metadata;
      } catch {
        description = client.metadata;
      }
    }

    setEditForm({
      name: client.name,
      client_id: client.client_id,
      enabled: client.enabled,
      quota_daily: client.quota_daily,
      quota_monthly: client.quota_monthly,
      metadata: description,
    });
    setShowEditDialog(true);
  };

  // 更新 OAuth2 客户端
  const handleUpdate = async () => {
    if (!selectedClient) return;

    try {
      setLoading(true);
      await oauth2ClientService.update(selectedClient.id, editForm);
      toast.success('更新成功');
      setShowEditDialog(false);
      loadClients();
    } catch (error) {
      toast.error('更新失败');
      console.error(error);
    } finally {
      setLoading(false);
    }
  };

  // 删除 OAuth2 客户端
  const handleDelete = async () => {
    if (!selectedClient) return;

    try {
      setLoading(true);
      await oauth2ClientService.delete(selectedClient.id);
      toast.success('删除成功');
      setShowDeleteDialog(false);
      loadClients();
    } catch (error) {
      toast.error('删除失败');
      console.error(error);
    } finally {
      setLoading(false);
    }
  };

  // 设置默认客户端
  const handleSetDefault = async (client: OAuth2Client) => {
    try {
      setLoading(true);
      // 根据客户端的 provider_id 设置默认
      await oauth2ClientService.setDefault(client.id, client.provider_id);
      toast.success(`已设置 "${client.name}" 为默认配置`);
      loadClients();
    } catch (error) {
      toast.error('设置默认失败');
      console.error(error);
    } finally {
      setLoading(false);
    }
  };

  // 获取提供商显示名称
  const getProviderDisplayName = (providerId: number) => {
    const provider = providers.find(p => p.id === providerId);
    return provider?.display_name || `提供商 ${providerId}`;
  };

  // 渲染客户端卡片
  const renderClientCard = (client: OAuth2Client) => (
    <Card key={client.id} className="transition-all hover:shadow-md">
      <CardHeader className="pb-3">
        <div className="flex items-start justify-between">
          <div className="space-y-1">
            <div className="flex items-center gap-2">
              <CardTitle className="text-lg">{client.name}</CardTitle>
              {client.is_default && (
                <Badge variant="secondary" className="gap-1">
                  <Star className="h-3 w-3" />
                  默认
                </Badge>
              )}
              {!client.enabled && (
                <Badge variant="outline">已禁用</Badge>
              )}
            </div>
            <div className="flex items-center gap-2 text-sm text-muted-foreground">
              <span>{getProviderDisplayName(client.provider_id)}</span>
              <span>•</span>
              <span>使用 {client.usage_count} 次</span>
            </div>
          </div>
        </div>
      </CardHeader>
      <CardContent className="space-y-4">
        <div className="grid grid-cols-2 gap-4 text-sm">
          <div>
            <Label className="text-muted-foreground">Client ID</Label>
            <p className="font-mono text-xs mt-1 truncate">{client.client_id}</p>
          </div>
          <div>
            <Label className="text-muted-foreground">配额</Label>
            <p className="mt-1">
              日: {client.quota_daily >= 0 ? client.quota_daily : '无限制'} /
              月: {client.quota_monthly >= 0 ? client.quota_monthly : '无限制'}
            </p>
          </div>
        </div>

        {client.metadata && (
          <div className="text-sm">
            <Label className="text-muted-foreground">描述</Label>
            <p className="mt-1 text-muted-foreground">{
              (() => {
                try {
                  const parsed = JSON.parse(client.metadata);
                  return parsed.raw || client.metadata;
                } catch {
                  return client.metadata;
                }
              })()
            }</p>
          </div>
        )}

        <div className="flex items-center gap-2 pt-2">
          <Button
            size="sm"
            variant="outline"
            onClick={() => handleEdit(client)}
            disabled={loading}
          >
            编辑
          </Button>
          {!client.is_default && (
            <Button
              size="sm"
              variant="outline"
              onClick={() => handleSetDefault(client)}
              disabled={loading}
            >
              <Star className="h-4 w-4 mr-1" />
              设为默认
            </Button>
          )}
          <Button
            size="sm"
            variant="destructive"
            onClick={() => {
              setSelectedClient(client);
              setShowDeleteDialog(true);
            }}
            disabled={loading}
          >
            <Trash2 className="h-4 w-4 mr-1" />
            删除
          </Button>
        </div>
      </CardContent>
    </Card>
  );

  return (
    <div className="container mx-auto px-4 py-6 space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-3xl font-bold">OAuth2 客户端管理</h1>
          <p className="text-muted-foreground mt-2">
            管理多个 OAuth2 客户端配置，支持配额管理和智能切换
          </p>
        </div>
        <Dialog open={showCreateDialog} onOpenChange={setShowCreateDialog}>
          <DialogTrigger asChild>
            <Button onClick={() => setShowCreateDialog(true)} disabled={loading}>
              <Plus className="h-4 w-4 mr-2" />
              新增配置
            </Button>
          </DialogTrigger>
          <DialogContent className="sm:max-w-[500px]">
            <DialogHeader>
              <DialogTitle>创建 OAuth2 客户端配置</DialogTitle>
              <DialogDescription>
                为指定邮箱提供商创建新的 OAuth2 客户端配置
              </DialogDescription>
            </DialogHeader>
            <div className="space-y-4 py-4">
              <div className="space-y-2">
                <Label>邮箱提供商 *</Label>
                <Select
                  value={createForm.provider_id?.toString() || ''}
                  onValueChange={(value: string) => {
                    const providerId = parseInt(value, 10);
                    const provider = providers.find(p => p.id === providerId);
                    setCreateForm({
                      ...createForm,
                      provider_id: providerId,
                      redirect_uri: provider?.name === 'gmail'
                        ? getOAuth2CallbackUri('google')
                        : getOAuth2CallbackUri('microsoft'),
                    });
                  }}
                >
                  <SelectTrigger>
                    <SelectValue placeholder="选择提供商" />
                  </SelectTrigger>
                  <SelectContent>
                    {providers.filter(p => p.enabled).map((provider) => (
                      <SelectItem key={provider.id} value={provider.id.toString()}>
                        {provider.display_name}
                      </SelectItem>
                    ))}
                  </SelectContent>
                </Select>
              </div>

              <div className="space-y-2">
                <Label>配置名称 *</Label>
                <Input
                  placeholder="例如：生产环境、测试环境等"
                  value={createForm.name}
                  onChange={(e) =>
                    setCreateForm({ ...createForm, name: e.target.value })
                  }
                />
              </div>

              <div className="space-y-2">
                <Label>Client ID *</Label>
                <Input
                  placeholder="输入 Client ID"
                  value={createForm.client_id}
                  onChange={(e) =>
                    setCreateForm({ ...createForm, client_id: e.target.value })
                  }
                />
              </div>

              <div className="space-y-2">
                <Label>Client Secret *</Label>
                <Input
                  type="password"
                  placeholder="输入 Client Secret"
                  value={createForm.client_secret}
                  onChange={(e) =>
                    setCreateForm({ ...createForm, client_secret: e.target.value })
                  }
                />
              </div>

              <div className="space-y-2">
                <Label>重定向 URI</Label>
                <Input
                  placeholder="输入重定向 URI"
                  value={createForm.redirect_uri}
                  onChange={(e) =>
                    setCreateForm({ ...createForm, redirect_uri: e.target.value })
                  }
                />
              </div>

              <div className="space-y-2">
                <Label>描述信息</Label>
                <Textarea
                  placeholder="请输入OAuth2客户端配置的描述信息（可选）"
                  value={createForm.metadata}
                  onChange={(e) =>
                    setCreateForm({ ...createForm, metadata: e.target.value })
                  }
                  rows={3}
                />
                <p className="text-xs text-muted-foreground">
                  描述信息将显示在客户端配置列表中，帮助识别配置的用途
                </p>
              </div>

              <div className="grid grid-cols-2 gap-4">
                <div className="space-y-2">
                  <Label>日配额（-1为无限制）</Label>
                  <Input
                    type="number"
                    value={createForm.quota_daily}
                    onChange={(e) =>
                      setCreateForm({
                        ...createForm,
                        quota_daily: parseInt(e.target.value) || -1,
                      })
                    }
                  />
                </div>
                <div className="space-y-2">
                  <Label>月配额（-1为无限制）</Label>
                  <Input
                    type="number"
                  value={createForm.quota_monthly}
                    onChange={(e) =>
                      setCreateForm({
                        ...createForm,
                        quota_monthly: parseInt(e.target.value) || -1,
                      })
                    }
                  />
                </div>
              </div>
            </div>
            <div className="flex justify-end gap-2">
              <Button
                variant="outline"
                onClick={() => setShowCreateDialog(false)}
                disabled={loading}
              >
                取消
              </Button>
              <Button onClick={handleCreate} disabled={loading}>
                {loading ? '创建中...' : '创建'}
              </Button>
            </div>
          </DialogContent>
        </Dialog>
      </div>

      {/* 统计卡片 */}
      <div className="grid grid-cols-1 md:grid-cols-4 gap-4">
        <Card>
          <CardHeader className="pb-2">
            <CardTitle className="text-sm font-medium text-muted-foreground">
              总配置数
            </CardTitle>
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold">{Array.isArray(clients) ? clients.length : 0}</div>
          </CardContent>
        </Card>
        <Card>
          <CardHeader className="pb-2">
            <CardTitle className="text-sm font-medium text-muted-foreground">
              启用中
            </CardTitle>
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold">
              {Array.isArray(clients) ? clients.filter(c => c.enabled).length : 0}
            </div>
          </CardContent>
        </Card>
        <Card>
          <CardHeader className="pb-2">
            <CardTitle className="text-sm font-medium text-muted-foreground">
              默认配置
            </CardTitle>
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold">
              {Array.isArray(clients) ? clients.filter(c => c.is_default).length : 0}
            </div>
          </CardContent>
        </Card>
        <Card>
          <CardHeader className="pb-2">
            <CardTitle className="text-sm font-medium text-muted-foreground">
              总使用次数
            </CardTitle>
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold">
              {Array.isArray(clients) ? clients.reduce((sum, c) => sum + c.usage_count, 0) : 0}
            </div>
          </CardContent>
        </Card>
      </div>

      {/* 客户端列表 */}
      <div className="space-y-4">
        <div className="flex items-center justify-between">
          <h2 className="text-xl font-semibold">配置列表</h2>
          <Button variant="outline" size="sm" onClick={loadClients} disabled={loading}>
            <RefreshCw className={`h-4 w-4 mr-2 ${loading ? 'animate-spin' : ''}`} />
            刷新
          </Button>
        </div>

        {loading && (!Array.isArray(clients) || clients.length === 0) ? (
          <Card>
            <CardContent className="flex items-center justify-center py-8">
              <RefreshCw className="h-6 w-6 animate-spin mr-2" />
              加载中...
            </CardContent>
          </Card>
        ) : !Array.isArray(clients) || clients.length === 0 ? (
          <Card>
            <CardContent className="flex flex-col items-center justify-center py-8">
              <AlertCircle className="h-12 w-12 text-muted-foreground mb-4" />
              <p className="text-muted-foreground">暂无 OAuth2 客户端配置</p>
              <Button
                className="mt-4"
                onClick={() => setShowCreateDialog(true)}
              >
                <Plus className="h-4 w-4 mr-2" />
                创建第一个配置
              </Button>
            </CardContent>
          </Card>
        ) : (
          <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
            {clients.map(renderClientCard)}
          </div>
        )}
      </div>

      {/* 编辑对话框 */}
      <Dialog open={showEditDialog} onOpenChange={setShowEditDialog}>
        <DialogContent className="sm:max-w-[500px]">
          <DialogHeader>
            <DialogTitle>编辑 OAuth2 客户端配置</DialogTitle>
            <DialogDescription>修改 "{selectedClient?.name}" 的配置信息</DialogDescription>
          </DialogHeader>
          <div className="space-y-4 py-4">
            <div className="space-y-2">
              <Label>配置名称</Label>
              <Input
                placeholder="输入配置名称"
                value={editForm.name || ''}
                onChange={(e) => setEditForm({ ...editForm, name: e.target.value })}
              />
            </div>

            <div className="space-y-2">
              <Label>Client ID</Label>
              <Input
                placeholder="输入 Client ID"
                value={editForm.client_id || ''}
                onChange={(e) =>
                  setEditForm({ ...editForm, client_id: e.target.value })
                }
              />
            </div>

            <div className="space-y-2">
              <Label>Client Secret（留空则不修改）</Label>
              <Input
                type="password"
                placeholder="输入新 Client Secret"
                value={editForm.client_secret || ''}
                onChange={(e) =>
                  setEditForm({ ...editForm, client_secret: e.target.value })
                }
              />
            </div>

            <div className="space-y-2">
              <Label>描述信息</Label>
              <Textarea
                placeholder="请输入OAuth2客户端配置的描述信息（可选）"
                value={editForm.metadata || ''}
                onChange={(e) =>
                  setEditForm({ ...editForm, metadata: e.target.value })
                }
                rows={3}
              />
              <p className="text-xs text-muted-foreground">
                描述信息将显示在客户端配置列表中，帮助识别配置的用途
              </p>
            </div>

            <div className="grid grid-cols-2 gap-4">
              <div className="space-y-2">
                <Label>日配额</Label>
                <Input
                  type="number"
                  value={editForm.quota_daily ?? 100}
                  onChange={(e) =>
                    setEditForm({
                      ...editForm,
                      quota_daily: parseInt(e.target.value) || -1,
                    })
                  }
                />
              </div>
              <div className="space-y-2">
                <Label>月配额</Label>
                <Input
                  type="number"
                  value={editForm.quota_monthly ?? 2000}
                  onChange={(e) =>
                    setEditForm({
                      ...editForm,
                      quota_monthly: parseInt(e.target.value) || -1,
                    })
                  }
                />
              </div>
            </div>

            <div className="flex items-center space-x-2">
              <Switch
                id="enabled"
                checked={editForm.enabled ?? true}
                onCheckedChange={(checked) =>
                  setEditForm({ ...editForm, enabled: checked })
                }
              />
              <Label htmlFor="enabled">启用此配置</Label>
            </div>
          </div>
          <div className="flex justify-end gap-2">
            <Button variant="outline" onClick={() => setShowEditDialog(false)} disabled={loading}>
              取消
            </Button>
            <Button onClick={handleUpdate} disabled={loading}>
              {loading ? '保存中...' : '保存'}
            </Button>
          </div>
        </DialogContent>
      </Dialog>

      {/* 删除确认对话框 */}
      <AlertDialog open={showDeleteDialog} onOpenChange={setShowDeleteDialog}>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>确认删除</AlertDialogTitle>
            <AlertDialogDescription>
              您确定要删除配置 "{selectedClient?.name}" 吗？此操作无法撤销。
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogAction asChild>
            <Button variant="destructive" onClick={handleDelete} disabled={loading}>
              {loading ? '删除中...' : '删除'}
            </Button>
          </AlertDialogAction>
          <AlertDialogCancel asChild>
            <Button variant="outline" onClick={() => setShowDeleteDialog(false)} disabled={loading}>
              取消
            </Button>
          </AlertDialogCancel>
        </AlertDialogContent>
      </AlertDialog>
    </div>
  );
};
