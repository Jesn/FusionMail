import { useState, useEffect } from 'react';
import { Plus, Trash2 } from 'lucide-react';
import { Button } from '../ui/button';
import { Input } from '../ui/input';
import { Label } from '../ui/label';
import { Textarea } from '../ui/textarea';
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '../ui/select';
import { Switch } from '../ui/switch';
import { Dialog, DialogContent, DialogHeader, DialogTitle } from '../ui/dialog';
import { Card, CardContent, CardHeader, CardTitle } from '../ui/card';
import { Badge } from '../ui/badge';
import { Webhook, CreateWebhookRequest, UpdateWebhookRequest } from '../../services/webhookService';
import { toast } from 'sonner';

interface WebhookFormProps {
  open: boolean;
  onClose: () => void;
  onSubmit: (data: CreateWebhookRequest | UpdateWebhookRequest) => void;
  webhook?: Webhook | null;
}

const EVENT_OPTIONS = [
  { value: 'email.received', label: '邮件接收' },
  { value: 'email.read', label: '邮件已读' },
  { value: 'email.starred', label: '邮件星标' },
  { value: 'email.archived', label: '邮件归档' },
  { value: 'email.deleted', label: '邮件删除' },
  { value: 'account.sync.started', label: '同步开始' },
  { value: 'account.sync.completed', label: '同步完成' },
  { value: 'account.sync.failed', label: '同步失败' },
];

const METHOD_OPTIONS = [
  { value: 'POST', label: 'POST' },
  { value: 'PUT', label: 'PUT' },
  { value: 'PATCH', label: 'PATCH' },
];

export const WebhookForm = ({ open, onClose, onSubmit, webhook }: WebhookFormProps) => {
  const isEditMode = !!webhook;
  
  const [formData, setFormData] = useState({
    name: '',
    description: '',
    url: '',
    method: 'POST',
    enabled: true,
    retry_enabled: true,
    max_retries: 3,
  });
  
  const [selectedEvents, setSelectedEvents] = useState<string[]>([]);
  const [headers, setHeaders] = useState<Array<{ key: string; value: string }>>([
    { key: '', value: '' }
  ]);
  const [retryIntervals, setRetryIntervals] = useState<number[]>([10, 30, 60]);

  useEffect(() => {
    if (webhook) {
      setFormData({
        name: webhook.name,
        description: webhook.description || '',
        url: webhook.url,
        method: webhook.method,
        enabled: webhook.enabled,
        retry_enabled: webhook.retry_enabled,
        max_retries: webhook.max_retries,
      });
      
      // 解析事件
      try {
        const events = JSON.parse(webhook.events || '[]');
        setSelectedEvents(events);
      } catch {
        setSelectedEvents([]);
      }
      
      // 解析请求头
      try {
        const headersObj = JSON.parse(webhook.headers || '{}');
        const headersArray = Object.entries(headersObj).map(([key, value]) => ({
          key,
          value: value as string
        }));
        setHeaders(headersArray.length > 0 ? headersArray : [{ key: '', value: '' }]);
      } catch {
        setHeaders([{ key: '', value: '' }]);
      }
      
      // 解析重试间隔
      try {
        const intervals = JSON.parse(webhook.retry_intervals || '[10,30,60]');
        setRetryIntervals(intervals);
      } catch {
        setRetryIntervals([10, 30, 60]);
      }
    } else {
      // 重置表单
      setFormData({
        name: '',
        description: '',
        url: '',
        method: 'POST',
        enabled: true,
        retry_enabled: true,
        max_retries: 3,
      });
      setSelectedEvents([]);
      setHeaders([{ key: '', value: '' }]);
      setRetryIntervals([10, 30, 60]);
    }
  }, [webhook, open]);

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault();
    
    // 验证表单
    if (!formData.name.trim()) {
      toast.error('请输入 Webhook 名称');
      return;
    }
    if (!formData.url.trim()) {
      toast.error('请输入 Webhook URL');
      return;
    }
    if (selectedEvents.length === 0) {
      toast.error('请至少选择一个事件类型');
      return;
    }

    // 构建请求头对象
    const headersObj: Record<string, string> = {};
    headers.forEach(({ key, value }) => {
      if (key.trim() && value.trim()) {
        headersObj[key.trim()] = value.trim();
      }
    });

    const submitData = {
      ...formData,
      events: selectedEvents,
      headers: Object.keys(headersObj).length > 0 ? JSON.stringify(headersObj) : '',
      retry_intervals: retryIntervals,
    };

    onSubmit(submitData);
  };

  const handleEventToggle = (eventValue: string) => {
    setSelectedEvents(prev => 
      prev.includes(eventValue)
        ? prev.filter(e => e !== eventValue)
        : [...prev, eventValue]
    );
  };

  const addHeader = () => {
    setHeaders(prev => [...prev, { key: '', value: '' }]);
  };

  const removeHeader = (index: number) => {
    if (headers.length > 1) {
      setHeaders(prev => prev.filter((_, i) => i !== index));
    }
  };

  const updateHeader = (index: number, field: 'key' | 'value', value: string) => {
    setHeaders(prev => prev.map((header, i) => 
      i === index ? { ...header, [field]: value } : header
    ));
  };

  const updateRetryInterval = (index: number, value: string) => {
    const numValue = parseInt(value) || 0;
    setRetryIntervals(prev => prev.map((interval, i) => 
      i === index ? numValue : interval
    ));
  };

  return (
    <Dialog open={open} onOpenChange={onClose}>
      <DialogContent className="max-w-4xl max-h-[90vh] overflow-y-auto">
        <DialogHeader>
          <DialogTitle>
            {isEditMode ? '编辑 Webhook' : '创建 Webhook'}
          </DialogTitle>
        </DialogHeader>

        <form onSubmit={handleSubmit} className="space-y-6">
          {/* 基本信息 */}
          <Card>
            <CardHeader>
              <CardTitle className="text-lg">基本信息</CardTitle>
            </CardHeader>
            <CardContent className="space-y-4">
              <div className="grid grid-cols-2 gap-4">
                <div>
                  <Label htmlFor="name">名称 *</Label>
                  <Input
                    id="name"
                    value={formData.name}
                    onChange={(e) => setFormData({ ...formData, name: e.target.value })}
                    placeholder="输入 Webhook 名称"
                  />
                </div>
                <div>
                  <Label htmlFor="method">请求方法</Label>
                  <Select
                    value={formData.method}
                    onValueChange={(value) => setFormData({ ...formData, method: value })}
                  >
                    <SelectTrigger>
                      <SelectValue />
                    </SelectTrigger>
                    <SelectContent>
                      {METHOD_OPTIONS.map((option) => (
                        <SelectItem key={option.value} value={option.value}>
                          {option.label}
                        </SelectItem>
                      ))}
                    </SelectContent>
                  </Select>
                </div>
              </div>

              <div>
                <Label htmlFor="url">URL *</Label>
                <Input
                  id="url"
                  type="url"
                  value={formData.url}
                  onChange={(e) => setFormData({ ...formData, url: e.target.value })}
                  placeholder="https://example.com/webhook"
                />
              </div>

              <div>
                <Label htmlFor="description">描述</Label>
                <Textarea
                  id="description"
                  value={formData.description}
                  onChange={(e) => setFormData({ ...formData, description: e.target.value })}
                  placeholder="输入 Webhook 描述（可选）"
                  rows={2}
                />
              </div>

              <div className="flex items-center space-x-4">
                <div className="flex items-center space-x-2">
                  <Switch
                    id="enabled"
                    checked={formData.enabled}
                    onCheckedChange={(checked) => setFormData({ ...formData, enabled: checked })}
                  />
                  <Label htmlFor="enabled">启用 Webhook</Label>
                </div>
              </div>
            </CardContent>
          </Card>

          {/* 事件配置 */}
          <Card>
            <CardHeader>
              <CardTitle className="text-lg">事件配置</CardTitle>
              <p className="text-sm text-gray-600 dark:text-gray-400">
                选择要监听的事件类型
              </p>
            </CardHeader>
            <CardContent>
              <div className="grid grid-cols-2 gap-3">
                {EVENT_OPTIONS.map((event) => (
                  <div
                    key={event.value}
                    className={`p-3 border rounded-lg cursor-pointer transition-colors ${
                      selectedEvents.includes(event.value)
                        ? 'border-blue-500 bg-blue-50 dark:bg-blue-900/20'
                        : 'border-gray-200 hover:border-gray-300'
                    }`}
                    onClick={() => handleEventToggle(event.value)}
                  >
                    <div className="flex items-center justify-between">
                      <span className="text-sm font-medium">{event.label}</span>
                      {selectedEvents.includes(event.value) && (
                        <Badge variant="default" className="text-xs">已选择</Badge>
                      )}
                    </div>
                    <div className="text-xs text-gray-500 mt-1">{event.value}</div>
                  </div>
                ))}
              </div>
            </CardContent>
          </Card>

          {/* 请求头配置 */}
          <Card>
            <CardHeader>
              <div className="flex items-center justify-between">
                <CardTitle className="text-lg">请求头</CardTitle>
                <Button type="button" variant="outline" size="sm" onClick={addHeader}>
                  <Plus className="h-4 w-4 mr-2" />
                  添加请求头
                </Button>
              </div>
            </CardHeader>
            <CardContent className="space-y-3">
              {headers.map((header, index) => (
                <div key={index} className="flex items-center gap-4">
                  <div className="flex-1">
                    <Input
                      placeholder="Header Name"
                      value={header.key}
                      onChange={(e) => updateHeader(index, 'key', e.target.value)}
                    />
                  </div>
                  <div className="flex-1">
                    <Input
                      placeholder="Header Value"
                      value={header.value}
                      onChange={(e) => updateHeader(index, 'value', e.target.value)}
                    />
                  </div>
                  {headers.length > 1 && (
                    <Button
                      type="button"
                      variant="ghost"
                      size="sm"
                      onClick={() => removeHeader(index)}
                      className="text-red-600 hover:text-red-700"
                    >
                      <Trash2 className="h-4 w-4" />
                    </Button>
                  )}
                </div>
              ))}
            </CardContent>
          </Card>

          {/* 重试配置 */}
          <Card>
            <CardHeader>
              <CardTitle className="text-lg">重试配置</CardTitle>
            </CardHeader>
            <CardContent className="space-y-4">
              <div className="flex items-center space-x-2">
                <Switch
                  id="retry_enabled"
                  checked={formData.retry_enabled}
                  onCheckedChange={(checked) => setFormData({ ...formData, retry_enabled: checked })}
                />
                <Label htmlFor="retry_enabled">启用重试</Label>
              </div>

              {formData.retry_enabled && (
                <>
                  <div>
                    <Label htmlFor="max_retries">最大重试次数</Label>
                    <Input
                      id="max_retries"
                      type="number"
                      min="1"
                      max="10"
                      value={formData.max_retries}
                      onChange={(e) => setFormData({ ...formData, max_retries: parseInt(e.target.value) || 3 })}
                    />
                  </div>

                  <div>
                    <Label>重试间隔（秒）</Label>
                    <div className="flex gap-2 mt-2">
                      {retryIntervals.map((interval, index) => (
                        <Input
                          key={index}
                          type="number"
                          min="1"
                          value={interval}
                          onChange={(e) => updateRetryInterval(index, e.target.value)}
                          className="w-20"
                        />
                      ))}
                    </div>
                    <p className="text-xs text-gray-500 mt-1">
                      第 1 次重试等待 {retryIntervals[0]} 秒，第 2 次等待 {retryIntervals[1]} 秒，以此类推
                    </p>
                  </div>
                </>
              )}
            </CardContent>
          </Card>

          {/* 提交按钮 */}
          <div className="flex justify-end gap-4">
            <Button type="button" variant="outline" onClick={onClose}>
              取消
            </Button>
            <Button type="submit">
              {isEditMode ? '更新 Webhook' : '创建 Webhook'}
            </Button>
          </div>
        </form>
      </DialogContent>
    </Dialog>
  );
};