import { useState, useCallback } from 'react';
import { toast } from 'sonner';
import { 
  webhookService, 
  Webhook, 
  WebhookLog, 
  CreateWebhookRequest, 
  UpdateWebhookRequest 
} from '../services/webhookService';

export const useWebhooks = () => {
  const [webhooks, setWebhooks] = useState<Webhook[]>([]);
  const [selectedWebhook, setSelectedWebhook] = useState<Webhook | null>(null);
  const [webhookLogs, setWebhookLogs] = useState<WebhookLog[]>([]);
  const [isLoading, setIsLoading] = useState(false);
  const [isLoadingLogs, setIsLoadingLogs] = useState(false);
  const [total, setTotal] = useState(0);
  const [logsTotal, setLogsTotal] = useState(0);

  // 获取 Webhook 列表
  const fetchWebhooks = useCallback(async (page = 1, pageSize = 20) => {
    setIsLoading(true);
    try {
      const result = await webhookService.getList(page, pageSize);
      setWebhooks(result.data);
      setTotal(result.total);
    } catch (error) {
      console.error('获取 Webhook 列表失败:', error);
      toast.error('获取 Webhook 列表失败');
    } finally {
      setIsLoading(false);
    }
  }, []);

  // 获取 Webhook 详情
  const fetchWebhookDetail = useCallback(async (id: number) => {
    try {
      const webhook = await webhookService.getById(id);
      setSelectedWebhook(webhook);
      return webhook;
    } catch (error) {
      console.error('获取 Webhook 详情失败:', error);
      toast.error('获取 Webhook 详情失败');
      return null;
    }
  }, []);

  // 创建 Webhook
  const createWebhook = useCallback(async (data: CreateWebhookRequest) => {
    try {
      const newWebhook = await webhookService.create(data);
      setWebhooks(prev => [newWebhook, ...prev]);
      toast.success('Webhook 创建成功');
      return newWebhook;
    } catch (error) {
      console.error('创建 Webhook 失败:', error);
      toast.error('创建 Webhook 失败');
      throw error;
    }
  }, []); 
 // 更新 Webhook
  const updateWebhook = useCallback(async (id: number, data: UpdateWebhookRequest) => {
    try {
      const updatedWebhook = await webhookService.update(id, data);
      setWebhooks(prev => prev.map(w => w.id === id ? updatedWebhook : w));
      if (selectedWebhook?.id === id) {
        setSelectedWebhook(updatedWebhook);
      }
      toast.success('Webhook 更新成功');
      return updatedWebhook;
    } catch (error) {
      console.error('更新 Webhook 失败:', error);
      toast.error('更新 Webhook 失败');
      throw error;
    }
  }, [selectedWebhook]);

  // 删除 Webhook
  const deleteWebhook = useCallback(async (id: number) => {
    try {
      await webhookService.delete(id);
      setWebhooks(prev => prev.filter(w => w.id !== id));
      if (selectedWebhook?.id === id) {
        setSelectedWebhook(null);
      }
      toast.success('Webhook 删除成功');
    } catch (error) {
      console.error('删除 Webhook 失败:', error);
      toast.error('删除 Webhook 失败');
      throw error;
    }
  }, [selectedWebhook]);

  // 切换 Webhook 启用状态
  const toggleWebhook = useCallback(async (id: number) => {
    try {
      const result = await webhookService.toggle(id);
      setWebhooks(prev => prev.map(w => 
        w.id === id ? { ...w, enabled: result.enabled } : w
      ));
      if (selectedWebhook?.id === id) {
        setSelectedWebhook(prev => prev ? { ...prev, enabled: result.enabled } : null);
      }
      toast.success(`Webhook 已${result.enabled ? '启用' : '禁用'}`);
    } catch (error) {
      console.error('切换 Webhook 状态失败:', error);
      toast.error('切换 Webhook 状态失败');
      throw error;
    }
  }, [selectedWebhook]);

  // 测试 Webhook
  const testWebhook = useCallback(async (id: number, testData?: Record<string, any>) => {
    try {
      await webhookService.test(id, testData);
      toast.success('Webhook 测试请求已发送');
    } catch (error) {
      console.error('测试 Webhook 失败:', error);
      toast.error('测试 Webhook 失败');
      throw error;
    }
  }, []);

  // 获取 Webhook 调用日志
  const fetchWebhookLogs = useCallback(async (id: number, page = 1, pageSize = 20) => {
    setIsLoadingLogs(true);
    try {
      const result = await webhookService.getLogs(id, page, pageSize);
      setWebhookLogs(result.data);
      setLogsTotal(result.total);
    } catch (error) {
      console.error('获取 Webhook 日志失败:', error);
      toast.error('获取 Webhook 日志失败');
    } finally {
      setIsLoadingLogs(false);
    }
  }, []);

  return {
    // 状态
    webhooks,
    selectedWebhook,
    webhookLogs,
    isLoading,
    isLoadingLogs,
    total,
    logsTotal,
    
    // 操作
    fetchWebhooks,
    fetchWebhookDetail,
    createWebhook,
    updateWebhook,
    deleteWebhook,
    toggleWebhook,
    testWebhook,
    fetchWebhookLogs,
    setSelectedWebhook,
  };
};