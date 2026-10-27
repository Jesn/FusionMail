import { useState, useEffect } from 'react';
import { Plus, RefreshCw, AlertCircle } from 'lucide-react';
import { Button } from '../components/ui/button';
import { Card, CardContent } from '../components/ui/card';
import { toast } from 'sonner';
import { oauth2ClientService } from '../services/oauth2ClientService';
import { useProviders } from '../hooks/useProviders';
import { OAuth2Client, OAuth2ClientCreateRequest, OAuth2ClientUpdateRequest } from '../types';
import { getOAuth2CallbackUri } from '../components/oauth2/oauth2ClientMeta';
import { OAuth2StatsCards } from '../components/oauth2/OAuth2StatsCards';
import { ClientCard } from '../components/oauth2/ClientCard';
import { CreateClientDialog } from '../components/oauth2/CreateClientDialog';
import { EditClientDialog } from '../components/oauth2/EditClientDialog';
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogHeader,
  AlertDialogTitle,
} from '../components/ui/alert-dialog';


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

  return (
    <div className="container mx-auto px-4 py-6 space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-3xl font-bold">OAuth2 客户端管理</h1>
          <p className="text-muted-foreground mt-2">
            管理多个 OAuth2 客户端配置，支持配额管理和智能切换
          </p>
        </div>
        <CreateClientDialog
          open={showCreateDialog}
          onOpenChange={setShowCreateDialog}
          providers={providers}
          form={createForm}
          onFormChange={setCreateForm}
          onSubmit={handleCreate}
          loading={loading}
        />
      </div>

      <OAuth2StatsCards clients={clients} />

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
            {clients.map((client) => (
              <ClientCard
                key={client.id}
                client={client}
                providerDisplayName={getProviderDisplayName(client.provider_id)}
                loading={loading}
                onEdit={handleEdit}
                onSetDefault={handleSetDefault}
                onDelete={(c) => {
                  setSelectedClient(c);
                  setShowDeleteDialog(true);
                }}
              />
            ))}
          </div>
        )}
      </div>

      <EditClientDialog
        open={showEditDialog}
        onOpenChange={setShowEditDialog}
        selectedClient={selectedClient}
        form={editForm}
        onFormChange={setEditForm}
        onSubmit={handleUpdate}
        loading={loading}
      />

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
