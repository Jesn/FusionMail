import { useState, useEffect } from 'react';
import { Plus, Trash2, AlertCircle, RefreshCw, ShieldCheck } from 'lucide-react';
import { Button } from '../components/ui/button';
import { Card, CardContent, CardHeader, CardTitle } from '../components/ui/card';
import { Input } from '../components/ui/input';
import { Label } from '../components/ui/label';
import { Textarea } from '../components/ui/textarea';
import { Switch } from '../components/ui/switch';
import { Badge } from '../components/ui/badge';
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '../components/ui/select';
import { toast } from 'sonner';
import { providerService } from '../services/providerService';
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
import { Provider, ProviderCreateRequest, ProviderUpdateRequest } from '../types';
import { ProviderType, getProviderTypeDisplayName } from '../types/providerType';
import { useProviders } from '../hooks/useProviders';

export const ProvidersPage = () => {
  const [providers, setProviders] = useState<Provider[]>([]);
  const [loading, setLoading] = useState(false);
  const [showCreateDialog, setShowCreateDialog] = useState(false);
  const [showEditDialog, setShowEditDialog] = useState(false);
  const [showDeleteDialog, setShowDeleteDialog] = useState(false);
  const [selectedProvider, setSelectedProvider] = useState<Provider | null>(null);
  
  // 使用 useProviders Hook 来刷新全局缓存
  const { refreshProviders } = useProviders();

  // 创建表单状态
  const [createForm, setCreateForm] = useState<ProviderCreateRequest>({
    name: '',
    display_name: '',
    provider_type: ProviderType.Generic,
    supported_protocols: ['imap'],
    recommended_protocol: 'imap',
    requires_oauth: false,
    imap_host: '',
    imap_port: 993,
    pop3_host: '',
    pop3_port: 995,
    imap_encryption: 'ssl',
    pop3_encryption: 'ssl',
    smtp_encryption: 'ssl',
    enabled: true,
    sort_order: 0,
    description: '',
  });

  // 编辑表单状态
  const [editForm, setEditForm] = useState<ProviderUpdateRequest>({});

  // 加载 Provider 列表
  const loadProviders = async () => {
    try {
      setLoading(true);
      const response = await providerService.getList(1, 100);
      // 确保 data 是数组
      setProviders(Array.isArray(response.data) ? response.data : response.data || []);
    } catch (error) {
      toast.error('加载 Provider 列表失败');
      console.error(error);
      // 出错时设置为空数组
      setProviders([]);
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    loadProviders();
  }, []);

  // 创建 Provider
  const handleCreate = async () => {
    if (!createForm.name.trim()) {
      toast.error('请输入 Provider 标识');
      return;
    }
    if (!createForm.display_name.trim()) {
      toast.error('请输入显示名称');
      return;
    }

    try {
      setLoading(true);
      await providerService.create(createForm);
      toast.success('创建成功');
      setShowCreateDialog(false);
      setCreateForm({
        name: '',
        display_name: '',
        provider_type: ProviderType.Generic,
        supported_protocols: ['imap'],
        recommended_protocol: 'imap',
        requires_oauth: false,
        imap_host: '',
        imap_port: 993,
        pop3_host: '',
        pop3_port: 995,
        imap_encryption: 'ssl',
        pop3_encryption: 'ssl',
        smtp_encryption: 'ssl',
        enabled: true,
        sort_order: 0,
        description: '',
      });
      loadProviders();
      // 刷新全局提供商缓存，确保其他页面能看到新增的提供商
      await refreshProviders();
    } catch (error) {
      toast.error('创建失败');
      console.error(error);
    } finally {
      setLoading(false);
    }
  };

  // 开始编辑
  const handleEdit = (provider: Provider) => {
    setSelectedProvider(provider);
    setEditForm({
      name: provider.name,
      display_name: provider.display_name,
      provider_type: provider.provider_type,
      supported_protocols: provider.supported_protocols,
      recommended_protocol: provider.recommended_protocol,
      requires_oauth: provider.requires_oauth,
      imap_host: provider.imap_host,
      imap_port: provider.imap_port,
      pop3_host: provider.pop3_host,
      pop3_port: provider.pop3_port,
      imap_encryption: provider.imap_encryption || 'ssl',
      pop3_encryption: provider.pop3_encryption || 'ssl',
      smtp_encryption: provider.smtp_encryption || 'ssl',
      enabled: provider.enabled,
      sort_order: provider.sort_order,
      description: provider.description,
    });
    setShowEditDialog(true);
  };

  // 更新 Provider
  const handleUpdate = async () => {
    if (!selectedProvider) return;

    try {
      setLoading(true);
      await providerService.update(selectedProvider.id, editForm);
      toast.success('更新成功');
      setShowEditDialog(false);
      loadProviders();
      // 刷新全局提供商缓存
      await refreshProviders();
    } catch (error) {
      toast.error('更新失败');
      console.error(error);
    } finally {
      setLoading(false);
    }
  };

  // 删除 Provider
  const handleDelete = async () => {
    if (!selectedProvider) return;

    try {
      setLoading(true);
      await providerService.delete(selectedProvider.id);
      toast.success('删除成功');
      setShowDeleteDialog(false);
      loadProviders();
      // 刷新全局提供商缓存
      await refreshProviders();
    } catch (error) {
      toast.error('删除失败');
      console.error(error);
    } finally {
      setLoading(false);
    }
  };

  // 渲染 Provider 卡片
  const renderProviderCard = (provider: Provider) => (
    <Card key={provider.id} className="transition-all hover:shadow-md">
      <CardHeader className="pb-3">
        <div className="flex items-start justify-between">
          <div className="space-y-1">
            <div className="flex items-center gap-2">
              <CardTitle className="text-lg">{provider.display_name}</CardTitle>
              {!provider.enabled && (
                <Badge variant="outline">已禁用</Badge>
              )}
              {provider.requires_oauth && (
                <Badge variant="secondary" className="gap-1">
                  <ShieldCheck className="h-3 w-3" />
                  OAuth2
                </Badge>
              )}
            </div>
            <div className="flex items-center gap-2 text-sm text-muted-foreground">
              <span>{provider.name}</span>
              <span>•</span>
              <span>排序: {provider.sort_order}</span>
            </div>
          </div>
        </div>
      </CardHeader>
      <CardContent className="space-y-4">
        <div className="grid grid-cols-2 gap-4 text-sm">
          <div>
            <Label className="text-muted-foreground">IMAP 服务器</Label>
            <p className="font-mono text-xs mt-1 truncate">{provider.imap_host}:{provider.imap_port}</p>
          </div>
          <div>
            <Label className="text-muted-foreground">推荐协议</Label>
            <p className="mt-1">{provider.recommended_protocol.toUpperCase()}</p>
          </div>
        </div>

        {provider.description && (
          <div className="text-sm">
            <Label className="text-muted-foreground">描述</Label>
            <p className="mt-1 text-muted-foreground">{provider.description}</p>
          </div>
        )}

        <div className="flex items-center gap-2 pt-2">
          <Button
            size="sm"
            variant="outline"
            onClick={() => handleEdit(provider)}
            disabled={loading}
          >
            编辑
          </Button>
          <Button
            size="sm"
            variant="destructive"
            onClick={() => {
              setSelectedProvider(provider);
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
          <h1 className="text-3xl font-bold">邮箱提供商管理</h1>
          <p className="text-muted-foreground mt-2">
            管理支持的邮箱提供商配置，包括服务器地址、协议和认证方式
          </p>
        </div>
        <Dialog open={showCreateDialog} onOpenChange={setShowCreateDialog}>
          <DialogTrigger asChild>
            <Button onClick={() => setShowCreateDialog(true)} disabled={loading}>
              <Plus className="h-4 w-4 mr-2" />
              新增提供商
            </Button>
          </DialogTrigger>
          <DialogContent className="sm:max-w-[600px] max-h-[90vh] overflow-y-auto">
            <DialogHeader>
              <DialogTitle>创建邮箱提供商配置</DialogTitle>
              <DialogDescription>
                为新邮箱提供商创建服务器和协议配置
              </DialogDescription>
            </DialogHeader>
            <div className="space-y-4 py-4">
              <div className="grid grid-cols-2 gap-4">
                <div className="space-y-2">
                  <Label>提供商标识 *</Label>
                  <Input
                    placeholder="例如：gmail, outlook"
                    value={createForm.name}
                    onChange={(e) =>
                      setCreateForm({ ...createForm, name: e.target.value })
                    }
                  />
                </div>
                <div className="space-y-2">
                  <Label>显示名称 *</Label>
                  <Input
                    placeholder="例如：Google Gmail"
                    value={createForm.display_name}
                    onChange={(e) =>
                      setCreateForm({ ...createForm, display_name: e.target.value })
                    }
                  />
                </div>
              </div>

              <div className="space-y-2">
                <Label>邮箱提供商类型 *</Label>
                <Select
                  value={(createForm.provider_type ?? ProviderType.Generic).toString()}
                  onValueChange={(value) => {
                    const providerType = parseInt(value) as ProviderType;
                    const requiresOAuth = providerType === ProviderType.Gmail || providerType === ProviderType.Outlook;
                    setCreateForm({
                      ...createForm,
                      provider_type: providerType,
                      requires_oauth: requiresOAuth,
                      // 如果选择了Gmail或Outlook，默认推荐使用OAuth2
                      recommended_protocol: requiresOAuth ? 'oauth2' : 'imap',
                      supported_protocols: requiresOAuth ? ['oauth2', 'imap'] : ['imap'],
                    });
                  }}
                >
                  <SelectTrigger>
                    <SelectValue placeholder="选择邮箱提供商类型" />
                  </SelectTrigger>
                  <SelectContent>
                    <SelectItem value={ProviderType.Gmail.toString()}>
                      {getProviderTypeDisplayName(ProviderType.Gmail)}
                    </SelectItem>
                    <SelectItem value={ProviderType.Outlook.toString()}>
                      {getProviderTypeDisplayName(ProviderType.Outlook)}
                    </SelectItem>
                    <SelectItem value={ProviderType.Icloud.toString()}>
                      {getProviderTypeDisplayName(ProviderType.Icloud)}
                    </SelectItem>
                    <SelectItem value={ProviderType.QQ.toString()}>
                      {getProviderTypeDisplayName(ProviderType.QQ)}
                    </SelectItem>
                    <SelectItem value={ProviderType.Email163.toString()}>
                      {getProviderTypeDisplayName(ProviderType.Email163)}
                    </SelectItem>
                    <SelectItem value={ProviderType.Generic.toString()}>
                      {getProviderTypeDisplayName(ProviderType.Generic)}
                    </SelectItem>
                  </SelectContent>
                </Select>
              </div>

              <div className="space-y-2">
                <Label>描述信息</Label>
                <Textarea
                  placeholder="请输入邮箱提供商的描述信息（可选）"
                  value={createForm.description}
                  onChange={(e) =>
                    setCreateForm({ ...createForm, description: e.target.value })
                  }
                  rows={3}
                />
                <p className="text-xs text-muted-foreground">
                  描述信息将显示在提供商列表中，帮助用户了解该提供商的特点
                </p>
              </div>

              <div className="grid grid-cols-3 gap-4">
                <div className="space-y-2">
                  <Label>IMAP 服务器</Label>
                  <Input
                    placeholder="imap.example.com"
                    value={createForm.imap_host}
                    onChange={(e) =>
                      setCreateForm({ ...createForm, imap_host: e.target.value })
                    }
                  />
                </div>
                <div className="space-y-2">
                  <Label>IMAP 端口</Label>
                  <Input
                    type="number"
                    placeholder="993"
                    value={createForm.imap_port}
                    onChange={(e) =>
                      setCreateForm({
                        ...createForm,
                        imap_port: parseInt(e.target.value) || 993,
                      })
                    }
                  />
                </div>
                <div className="space-y-2">
                  <Label>IMAP 加密</Label>
                  <Select
                    value={createForm.imap_encryption || 'ssl'}
                    onValueChange={(value) =>
                      setCreateForm({ ...createForm, imap_encryption: value })
                    }
                  >
                    <SelectTrigger>
                      <SelectValue placeholder="选择加密方式" />
                    </SelectTrigger>
                    <SelectContent>
                      <SelectItem value="ssl">SSL/TLS</SelectItem>
                      <SelectItem value="starttls">STARTTLS</SelectItem>
                      <SelectItem value="none">无加密</SelectItem>
                    </SelectContent>
                  </Select>
                </div>
              </div>

              <div className="space-y-2">
                <Label>推荐协议 *</Label>
                <Select
                  value={createForm.recommended_protocol}
                  onValueChange={(value) =>
                    setCreateForm({ ...createForm, recommended_protocol: value })
                  }
                >
                  <SelectTrigger>
                    <SelectValue placeholder="选择推荐协议" />
                  </SelectTrigger>
                  <SelectContent>
                    <SelectItem value="oauth2">OAuth2</SelectItem>
                    <SelectItem value="imap">IMAP</SelectItem>
                    <SelectItem value="pop3">POP3</SelectItem>
                  </SelectContent>
                </Select>
                <p className="text-xs text-muted-foreground">
                  选择推荐的认证和协议类型
                </p>
              </div>

              <div className="flex items-center space-x-2">
                <Switch
                  id="enabled"
                  checked={createForm.enabled}
                  onCheckedChange={(checked) =>
                    setCreateForm({ ...createForm, enabled: checked })
                  }
                />
                <Label htmlFor="enabled">启用此提供商</Label>
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
              总提供商数
            </CardTitle>
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold">{Array.isArray(providers) ? providers.length : 0}</div>
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
              {Array.isArray(providers) ? providers.filter(p => p.enabled).length : 0}
            </div>
          </CardContent>
        </Card>
        <Card>
          <CardHeader className="pb-2">
            <CardTitle className="text-sm font-medium text-muted-foreground">
              支持 OAuth2
            </CardTitle>
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold">
              {Array.isArray(providers) ? providers.filter(p => p.requires_oauth).length : 0}
            </div>
          </CardContent>
        </Card>
        <Card>
          <CardHeader className="pb-2">
            <CardTitle className="text-sm font-medium text-muted-foreground">
              IMAP 支持
            </CardTitle>
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold">
              {Array.isArray(providers) ? providers.filter(p => p.supported_protocols.includes('imap')).length : 0}
            </div>
          </CardContent>
        </Card>
      </div>

      {/* 提供商列表 */}
      <div className="space-y-4">
        <div className="flex items-center justify-between">
          <h2 className="text-xl font-semibold">配置列表</h2>
          <Button variant="outline" size="sm" onClick={loadProviders} disabled={loading}>
            <RefreshCw className={`h-4 w-4 mr-2 ${loading ? 'animate-spin' : ''}`} />
            刷新
          </Button>
        </div>

        {loading && (!Array.isArray(providers) || providers.length === 0) ? (
          <Card>
            <CardContent className="flex items-center justify-center py-8">
              <RefreshCw className="h-6 w-6 animate-spin mr-2" />
              加载中...
            </CardContent>
          </Card>
        ) : !Array.isArray(providers) || providers.length === 0 ? (
          <Card>
            <CardContent className="flex flex-col items-center justify-center py-8">
              <AlertCircle className="h-12 w-12 text-muted-foreground mb-4" />
              <p className="text-muted-foreground">暂无邮箱提供商配置</p>
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
            {providers.map(renderProviderCard)}
          </div>
        )}
      </div>

      {/* 编辑对话框 */}
      <Dialog open={showEditDialog} onOpenChange={setShowEditDialog}>
        <DialogContent className="sm:max-w-[600px] max-h-[90vh] overflow-y-auto">
          <DialogHeader>
            <DialogTitle>编辑邮箱提供商配置</DialogTitle>
            <DialogDescription>修改 "{selectedProvider?.display_name}" 的配置信息</DialogDescription>
          </DialogHeader>
          <div className="space-y-4 py-4">
            <div className="grid grid-cols-2 gap-4">
              <div className="space-y-2">
                <Label>提供商标识</Label>
                <Input
                  placeholder="例如：gmail, outlook"
                  value={editForm.name || ''}
                  onChange={(e) => setEditForm({ ...editForm, name: e.target.value })}
                />
              </div>
              <div className="space-y-2">
                <Label>显示名称</Label>
                <Input
                  placeholder="例如：Google Gmail"
                  value={editForm.display_name || ''}
                  onChange={(e) => setEditForm({ ...editForm, display_name: e.target.value })}
                />
              </div>
            </div>

            <div className="space-y-2">
              <Label>描述信息</Label>
              <Textarea
                placeholder="请输入邮箱提供商的描述信息（可选）"
                value={editForm.description || ''}
                onChange={(e) => setEditForm({ ...editForm, description: e.target.value })}
                rows={3}
              />
              <p className="text-xs text-muted-foreground">
                描述信息将显示在提供商列表中，帮助用户了解该提供商的特点
              </p>
            </div>

            <div className="grid grid-cols-3 gap-4">
              <div className="space-y-2">
                <Label>IMAP 服务器</Label>
                <Input
                  placeholder="imap.example.com"
                  value={editForm.imap_host || ''}
                  onChange={(e) => setEditForm({ ...editForm, imap_host: e.target.value })}
                />
              </div>
              <div className="space-y-2">
                <Label>IMAP 端口</Label>
                <Input
                  type="number"
                  placeholder="993"
                  value={editForm.imap_port || ''}
                  onChange={(e) =>
                    setEditForm({
                      ...editForm,
                      imap_port: parseInt(e.target.value) || 993,
                    })
                  }
                />
              </div>
              <div className="space-y-2">
                <Label>IMAP 加密</Label>
                <Select
                  value={editForm.imap_encryption || 'ssl'}
                  onValueChange={(value) =>
                    setEditForm({ ...editForm, imap_encryption: value })
                  }
                >
                  <SelectTrigger>
                    <SelectValue placeholder="选择加密方式" />
                  </SelectTrigger>
                  <SelectContent>
                    <SelectItem value="ssl">SSL/TLS</SelectItem>
                    <SelectItem value="starttls">STARTTLS</SelectItem>
                    <SelectItem value="none">无加密</SelectItem>
                  </SelectContent>
                </Select>
              </div>
            </div>

            <div className="space-y-2">
              <Label>推荐协议 *</Label>
              <Select
                value={editForm.recommended_protocol || 'imap'}
                onValueChange={(value) =>
                  setEditForm({ ...editForm, recommended_protocol: value })
                }
              >
                <SelectTrigger>
                  <SelectValue placeholder="选择推荐协议" />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="oauth2">OAuth2</SelectItem>
                  <SelectItem value="imap">IMAP</SelectItem>
                  <SelectItem value="pop3">POP3</SelectItem>
                </SelectContent>
              </Select>
              <p className="text-xs text-muted-foreground">
                选择推荐的认证和协议类型
              </p>
            </div>

            <div className="flex items-center space-x-2">
              <Switch
                id="enabled-edit"
                checked={editForm.enabled ?? true}
                onCheckedChange={(checked) =>
                  setEditForm({ ...editForm, enabled: checked })
                }
              />
              <Label htmlFor="enabled-edit">启用此提供商</Label>
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
              您确定要删除提供商 "{selectedProvider?.display_name}" 吗？此操作无法撤销。
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
