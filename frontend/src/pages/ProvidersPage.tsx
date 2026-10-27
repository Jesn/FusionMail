import { useState, useEffect } from 'react';
import { Button } from '../components/ui/button';
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogHeader,
  AlertDialogTitle,
} from '../components/ui/alert-dialog';
import { toast } from 'sonner';
import { providerService } from '../services/providerService';
import adapterService from '../services/adapterService';
import { Provider, ProviderCreateRequest, ProviderUpdateRequest, AdapterResponse } from '../types';
import { useProviders } from '../hooks/useProviders';
import { ProviderCreateDialog } from '../components/provider/ProviderCreateDialog';
import { ProviderEditDialog } from '../components/provider/ProviderEditDialog';
import { ProviderStatsCards } from '../components/provider/ProviderStatsCards';
import { ProviderListSection } from '../components/provider/ProviderListSection';

export const ProvidersPage = () => {
  const [providers, setProviders] = useState<Provider[]>([]);
  const [adapters, setAdapters] = useState<AdapterResponse[]>([]);
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
    default_adapter_id: undefined,
    email_domains: [],
    adapter_ids: [],
    supported_protocols: ['imap'],
    recommended_protocol: 'imap',
    requires_oauth: false,
    imap_host: '',
    imap_port: 993,
    pop3_host: '',
    pop3_port: 995,
    smtp_host: '',
    smtp_port: 587,
    imap_encryption: 'ssl',
    pop3_encryption: 'ssl',
    smtp_encryption: 'ssl',
    enabled: true,
    sort_order: 0,
    description: '',
  });
  
  // 邮箱域名输入临时状态
  const [emailDomainsInput, setEmailDomainsInput] = useState('');

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

  // 加载适配器列表
  const loadAdapters = async () => {
    try {
      const data = await adapterService.listEnabled();
      setAdapters(data);
    } catch (error) {
      console.error('加载适配器列表失败:', error);
    }
  };

  useEffect(() => {
    loadProviders();
    loadAdapters();
  }, []);

  // 更新创建表单（子组件传回部分字段补丁）
  const updateCreateForm = (patch: Partial<ProviderCreateRequest>) => {
    setCreateForm((prev) => ({ ...prev, ...patch }));
  };

  // 更新编辑表单（子组件传回部分字段补丁）
  const updateEditForm = (patch: Partial<ProviderUpdateRequest>) => {
    setEditForm((prev) => ({ ...prev, ...patch }));
  };

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
      // 处理邮箱域名
      const formToSubmit = {
        ...createForm,
        email_domains: emailDomainsInput
          .split(',')
          .map(d => d.trim().toLowerCase())
          .filter(Boolean),
      };
      await providerService.create(formToSubmit);
      toast.success('创建成功');
      setShowCreateDialog(false);
      setCreateForm({
        name: '',
        display_name: '',
        default_adapter_id: undefined,
        email_domains: [],
        adapter_ids: [],
        supported_protocols: ['imap'],
        recommended_protocol: 'imap',
        requires_oauth: false,
        imap_host: '',
        imap_port: 993,
        pop3_host: '',
        pop3_port: 995,
        smtp_host: '',
        smtp_port: 587,
        imap_encryption: 'ssl',
        pop3_encryption: 'ssl',
        smtp_encryption: 'ssl',
        enabled: true,
        sort_order: 0,
        description: '',
      });
      setEmailDomainsInput('');
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

  // 判断是否为 WebAPI 供应商（系统内置，不可由用户新增）
  const isWebAPIProvider = (provider: Provider): boolean => {
    return provider.recommended_protocol === 'webapi' || 
           provider.supported_protocols.includes('webapi');
  };

  // 开始编辑
  const handleEdit = (provider: Provider) => {
    setSelectedProvider(provider);
    // WebAPI 供应商只需要编辑基础信息，但需要保留协议字段以通过后端验证
    if (isWebAPIProvider(provider)) {
      setEditForm({
        name: provider.name,
        display_name: provider.display_name,
        description: provider.description,
        enabled: provider.enabled,
        sort_order: provider.sort_order,
        // 保留协议字段以通过后端验证
        supported_protocols: provider.supported_protocols,
        recommended_protocol: provider.recommended_protocol,
        // 回填当前关联的适配器 ID，避免更新时被后端清空
        adapter_ids: provider.supported_adapters?.map(pa => pa.adapter_id) ?? [],
      });
    } else {
      // 传统供应商需要编辑完整配置
      setEditForm({
        name: provider.name,
        display_name: provider.display_name,
        supported_protocols: provider.supported_protocols,
        recommended_protocol: provider.recommended_protocol,
        requires_oauth: provider.requires_oauth,
        // 回填当前关联的适配器 ID，避免更新时被后端清空
        adapter_ids: provider.supported_adapters?.map(pa => pa.adapter_id) ?? [],
        imap_host: provider.imap_host,
        imap_port: provider.imap_port,
        pop3_host: provider.pop3_host,
        pop3_port: provider.pop3_port,
        smtp_host: provider.smtp_host,
        smtp_port: provider.smtp_port,
        imap_encryption: provider.imap_encryption || 'ssl',
        pop3_encryption: provider.pop3_encryption || 'ssl',
        smtp_encryption: provider.smtp_encryption || 'ssl',
        enabled: provider.enabled,
        sort_order: provider.sort_order,
        description: provider.description,
      });
    }
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

  // 打开删除确认对话框
  const openDeleteDialog = (provider: Provider) => {
    setSelectedProvider(provider);
    setShowDeleteDialog(true);
  };

  return (
    <div className="container mx-auto px-4 py-6 space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-3xl font-bold">邮箱提供商管理</h1>
          <p className="text-muted-foreground mt-2">
            管理支持的邮箱提供商配置，包括服务器地址、协议和认证方式
          </p>
        </div>
        <ProviderCreateDialog
          open={showCreateDialog}
          onOpenChange={setShowCreateDialog}
          form={createForm}
          onFormChange={updateCreateForm}
          adapters={adapters}
          emailDomainsInput={emailDomainsInput}
          onEmailDomainsInputChange={setEmailDomainsInput}
          onSubmit={handleCreate}
          loading={loading}
        />
      </div>

      {/* 统计卡片 */}
      <ProviderStatsCards providers={providers} />

      {/* 提供商列表 */}
      <ProviderListSection
        providers={providers}
        isLoading={loading}
        isWebAPIProvider={isWebAPIProvider}
        onRefresh={loadProviders}
        onCreateFirst={() => setShowCreateDialog(true)}
        onEditProvider={handleEdit}
        onDeleteProvider={openDeleteDialog}
      />

      {/* 编辑对话框 */}
      <ProviderEditDialog
        open={showEditDialog}
        onOpenChange={setShowEditDialog}
        provider={selectedProvider}
        isWebAPI={selectedProvider ? isWebAPIProvider(selectedProvider) : false}
        form={editForm}
        onFormChange={updateEditForm}
        onSubmit={handleUpdate}
        loading={loading}
      />

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
