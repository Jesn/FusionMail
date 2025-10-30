import { Plus } from 'lucide-react';
import { useState, useEffect } from 'react';
import { Button } from '../components/ui/button';
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
} from '../components/ui/alert-dialog';
import { WebhookList } from '../components/webhook/WebhookList';
import { WebhookForm } from '../components/webhook/WebhookForm';
import { WebhookLogsDialog } from '../components/webhook/WebhookLogsDialog';
import { useWebhooks } from '../hooks/useWebhooks';
import { Webhook } from '../services/webhookService';

export const WebhooksPage = () => {
  const {
    webhooks,
    webhookLogs,
    isLoading,
    isLoadingLogs,
    fetchWebhooks,
    createWebhook,
    updateWebhook,
    deleteWebhook,
    toggleWebhook,
    testWebhook,
    fetchWebhookLogs,
    setSelectedWebhook,
  } = useWebhooks();

  const [isFormOpen, setIsFormOpen] = useState(false);
  const [editingWebhook, setEditingWebhook] = useState<Webhook | null>(null);
  const [deletingWebhook, setDeletingWebhook] = useState<{ id: number; name: string } | null>(null);
  const [logsDialogWebhook, setLogsDialogWebhook] = useState<Webhook | null>(null);

  useEffect(() => {
    fetchWebhooks();
  }, [fetchWebhooks]);

  const handleCreate = () => {
    setEditingWebhook(null);
    setIsFormOpen(true);
  };

  const handleEdit = (webhook: Webhook) => {
    setEditingWebhook(webhook);
    setIsFormOpen(true);
  };

  const handleFormClose = () => {
    setIsFormOpen(false);
    setEditingWebhook(null);
  };

  const handleFormSubmit = async (data: any) => {
    try {
      if (editingWebhook) {
        await updateWebhook(editingWebhook.id, data);
      } else {
        await createWebhook(data);
      }
      handleFormClose();
    } catch (error) {
      // 错误已在 hook 中处理
    }
  };

  const handleDeleteClick = (id: number, name: string) => {
    setDeletingWebhook({ id, name });
  };

  const handleDeleteConfirm = async () => {
    if (deletingWebhook) {
      try {
        await deleteWebhook(deletingWebhook.id);
        setDeletingWebhook(null);
      } catch (error) {
        // 错误已在 hook 中处理
      }
    }
  };

  const handleDeleteCancel = () => {
    setDeletingWebhook(null);
  };

  const handleToggle = async (id: number) => {
    try {
      await toggleWebhook(id);
    } catch (error) {
      // 错误已在 hook 中处理
    }
  };

  const handleTest = async (id: number) => {
    try {
      await testWebhook(id);
    } catch (error) {
      // 错误已在 hook 中处理
    }
  };

  const handleViewLogs = (webhook: Webhook) => {
    setLogsDialogWebhook(webhook);
    setSelectedWebhook(webhook);
    fetchWebhookLogs(webhook.id);
  };

  const handleLogsDialogClose = () => {
    setLogsDialogWebhook(null);
    setSelectedWebhook(null);
  };

  const handleRefreshLogs = () => {
    if (logsDialogWebhook) {
      fetchWebhookLogs(logsDialogWebhook.id);
    }
  };

  return (
    <div className="container mx-auto px-4 py-6">
      <div className="flex items-center justify-between mb-6">
        <div>
          <h1 className="text-2xl font-bold text-gray-900 dark:text-white">
            Webhook 管理
          </h1>
          <p className="text-gray-600 dark:text-gray-400 mt-1">
            管理邮件事件的 Webhook 通知
          </p>
        </div>
        <Button onClick={handleCreate} className="flex items-center gap-2">
          <Plus className="h-4 w-4" />
          创建 Webhook
        </Button>
      </div>

      <WebhookList
        webhooks={webhooks}
        isLoading={isLoading}
        onEdit={handleEdit}
        onDelete={handleDeleteClick}
        onToggle={handleToggle}
        onTest={handleTest}
        onViewLogs={handleViewLogs}
      />

      {/* Webhook 表单对话框 */}
      <WebhookForm
        open={isFormOpen}
        onClose={handleFormClose}
        onSubmit={handleFormSubmit}
        webhook={editingWebhook}
      />

      {/* 删除确认对话框 */}
      <AlertDialog open={!!deletingWebhook} onOpenChange={() => setDeletingWebhook(null)}>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>确认删除</AlertDialogTitle>
            <AlertDialogDescription>
              确定要删除 Webhook "{deletingWebhook?.name}" 吗？此操作无法撤销。
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel onClick={handleDeleteCancel}>
              取消
            </AlertDialogCancel>
            <AlertDialogAction
              onClick={handleDeleteConfirm}
              className="bg-red-600 hover:bg-red-700"
            >
              删除
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>

      {/* Webhook 日志对话框 */}
      <WebhookLogsDialog
        open={!!logsDialogWebhook}
        onClose={handleLogsDialogClose}
        webhook={logsDialogWebhook}
        logs={webhookLogs}
        isLoading={isLoadingLogs}
        onRefresh={handleRefreshLogs}
      />
    </div>
  );
};