import { useState, useEffect } from 'react';
import { Settings, Trash2, Save, FileText } from 'lucide-react';
import { Button } from '../components/ui/button';
import { Input } from '../components/ui/input';
import { Label } from '../components/ui/label';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '../components/ui/card';
import { toast } from 'sonner';
import { api } from '../services/api';

// 日志清理配置类型
interface LogCleanupSettings {
  syncLogsRetentionDays: number;
  webhookLogsRetentionDays: number;
  spamDetectionLogsRetentionDays: number;
}

export const SystemSettingsPage = () => {
  const [trashAutoCleanupDays, setTrashAutoCleanupDays] = useState<number>(-1);
  const [logSettings, setLogSettings] = useState<LogCleanupSettings>({
    syncLogsRetentionDays: 7,
    webhookLogsRetentionDays: 14,
    spamDetectionLogsRetentionDays: 7,
  });
  const [isLoading, setIsLoading] = useState(false);
  const [isSaving, setIsSaving] = useState(false);

  // 加载配置
  useEffect(() => {
    loadSettings();
  }, []);

  const loadSettings = async () => {
    try {
      setIsLoading(true);
      
      // 加载回收站清理配置
      const trashResponse = await api.get<{ success: boolean; data: { value: string } }>(
        '/settings/system/system/trash_auto_cleanup_days'
      );
      if (trashResponse.data?.value) {
        setTrashAutoCleanupDays(parseInt(trashResponse.data.value));
      }

      // 加载日志清理配置
      const [syncLogsRes, webhookLogsRes, spamLogsRes] = await Promise.all([
        api.get<{ success: boolean; data: { value: string } }>('/settings/system/system/sync_logs_retention_days'),
        api.get<{ success: boolean; data: { value: string } }>('/settings/system/system/webhook_logs_retention_days'),
        api.get<{ success: boolean; data: { value: string } }>('/settings/system/system/spam_detection_logs_retention_days'),
      ]);

      setLogSettings({
        syncLogsRetentionDays: syncLogsRes.data?.value ? parseInt(syncLogsRes.data.value) : 7,
        webhookLogsRetentionDays: webhookLogsRes.data?.value ? parseInt(webhookLogsRes.data.value) : 14,
        spamDetectionLogsRetentionDays: spamLogsRes.data?.value ? parseInt(spamLogsRes.data.value) : 7,
      });
    } catch (err) {
      console.error('Failed to load settings:', err);
      toast.error('加载配置失败');
    } finally {
      setIsLoading(false);
    }
  };

  const handleSave = async () => {
    try {
      setIsSaving(true);
      
      // 保存所有配置
      await Promise.all([
        api.post('/settings/system/system/trash_auto_cleanup_days', {
          value: trashAutoCleanupDays.toString(),
        }),
        api.post('/settings/system/system/sync_logs_retention_days', {
          value: logSettings.syncLogsRetentionDays.toString(),
        }),
        api.post('/settings/system/system/webhook_logs_retention_days', {
          value: logSettings.webhookLogsRetentionDays.toString(),
        }),
        api.post('/settings/system/system/spam_detection_logs_retention_days', {
          value: logSettings.spamDetectionLogsRetentionDays.toString(),
        }),
      ]);
      
      toast.success('配置保存成功');
    } catch (err) {
      console.error('Failed to save settings:', err);
      toast.error('保存配置失败');
    } finally {
      setIsSaving(false);
    }
  };

  // 处理日志配置变更
  const handleLogSettingChange = (key: keyof LogCleanupSettings, value: string) => {
    if (value === '' || value === '-') {
      setLogSettings(prev => ({ ...prev, [key]: 0 }));
    } else {
      const num = parseInt(value, 10);
      if (!isNaN(num) && num >= -1) {
        setLogSettings(prev => ({ ...prev, [key]: num }));
      }
    }
  };

  return (
    <div className="h-full overflow-auto">
      <div className="mx-auto max-w-4xl p-6">
        {/* 头部 */}
        <div className="mb-6">
          <div className="flex items-center gap-2 mb-2">
            <Settings className="h-8 w-8 text-muted-foreground" />
            <h1 className="text-3xl font-bold">系统设置</h1>
          </div>
          <p className="text-muted-foreground">
            管理系统级配置和功能
          </p>
        </div>

        {/* 回收站设置 */}
        <Card>
          <CardHeader>
            <div className="flex items-center gap-2">
              <Trash2 className="h-5 w-5" />
              <CardTitle>回收站自动清理</CardTitle>
            </div>
            <CardDescription>
              配置回收站中已删除账号的自动清理策略
            </CardDescription>
          </CardHeader>
          <CardContent className="space-y-4">
            <div className="space-y-2">
              <Label htmlFor="cleanup-days">
                自动清理天数
              </Label>
              <div className="flex items-center gap-4">
                <Input
                  id="cleanup-days"
                  type="number"
                  value={trashAutoCleanupDays}
                  onChange={(e) => {
                    const value = e.target.value;
                    if (value === '' || value === '-') {
                      setTrashAutoCleanupDays(0);
                    } else {
                      const num = parseInt(value, 10);
                      if (!isNaN(num) && num >= -1) {
                        setTrashAutoCleanupDays(num);
                      }
                    }
                  }}
                  className="w-32"
                  disabled={isLoading}
                />
                <span className="text-sm text-muted-foreground">
                  天
                </span>
              </div>
              <p className="text-sm text-muted-foreground">
                设置为 -1 表示永不自动清理，设置为 0 或正数表示删除后保留的天数
              </p>
              <div className="mt-4 rounded-lg border border-blue-200 bg-blue-50 p-4 dark:border-blue-900 dark:bg-blue-950/20">
                <p className="text-sm text-blue-700 dark:text-blue-300">
                  <strong>说明：</strong>
                  <br />
                  • -1：永不自动清理，需要手动永久删除
                  <br />
                  • 0：立即清理（不推荐）
                  <br />
                  • 7（默认）：删除后保留 7 天
                  <br />
                  • 30：删除后保留 30 天
                  <br />
                  <br />
                  自动清理会在系统后台定期执行，永久删除超过指定天数的已删除账号及其所有数据。
                </p>
              </div>
            </div>

          </CardContent>
        </Card>

        {/* 日志清理设置 */}
        <Card className="mt-6">
          <CardHeader>
            <div className="flex items-center gap-2">
              <FileText className="h-5 w-5" />
              <CardTitle>日志自动清理</CardTitle>
            </div>
            <CardDescription>
              配置各类系统日志的自动清理策略，定期清理可节省存储空间
            </CardDescription>
          </CardHeader>
          <CardContent className="space-y-6">
            {/* 同步日志 */}
            <div className="space-y-2">
              <Label htmlFor="sync-logs-days">同步日志保留天数</Label>
              <div className="flex items-center gap-4">
                <Input
                  id="sync-logs-days"
                  type="number"
                  value={logSettings.syncLogsRetentionDays}
                  onChange={(e) => handleLogSettingChange('syncLogsRetentionDays', e.target.value)}
                  className="w-32"
                  disabled={isLoading}
                />
                <span className="text-sm text-muted-foreground">天</span>
              </div>
              <p className="text-xs text-muted-foreground">
                记录邮件同步的执行情况，包括成功/失败状态和耗时统计
              </p>
            </div>

            {/* Webhook 日志 */}
            <div className="space-y-2">
              <Label htmlFor="webhook-logs-days">Webhook 日志保留天数</Label>
              <div className="flex items-center gap-4">
                <Input
                  id="webhook-logs-days"
                  type="number"
                  value={logSettings.webhookLogsRetentionDays}
                  onChange={(e) => handleLogSettingChange('webhookLogsRetentionDays', e.target.value)}
                  className="w-32"
                  disabled={isLoading}
                />
                <span className="text-sm text-muted-foreground">天</span>
              </div>
              <p className="text-xs text-muted-foreground">
                记录 Webhook 调用的请求和响应详情，用于调试集成问题
              </p>
            </div>

            {/* 垃圾邮件检测日志 */}
            <div className="space-y-2">
              <Label htmlFor="spam-logs-days">垃圾邮件检测日志保留天数</Label>
              <div className="flex items-center gap-4">
                <Input
                  id="spam-logs-days"
                  type="number"
                  value={logSettings.spamDetectionLogsRetentionDays}
                  onChange={(e) => handleLogSettingChange('spamDetectionLogsRetentionDays', e.target.value)}
                  className="w-32"
                  disabled={isLoading}
                />
                <span className="text-sm text-muted-foreground">天</span>
              </div>
              <p className="text-xs text-muted-foreground">
                记录垃圾邮件检测的评分和详情，数据量较大建议短期保留
              </p>
            </div>

            <div className="rounded-lg border border-amber-200 bg-amber-50 p-4 dark:border-amber-900 dark:bg-amber-950/20">
              <p className="text-sm text-amber-700 dark:text-amber-300">
                <strong>提示：</strong>
                <br />
                • 设置为 -1 表示永不自动清理
                <br />
                • 日志清理任务每天凌晨 4:00 自动执行
                <br />
                • 建议根据存储空间和排查需求合理设置保留天数
              </p>
            </div>
          </CardContent>
        </Card>

        {/* 保存按钮 */}
        <div className="mt-6 flex justify-end">
          <Button
            onClick={handleSave}
            disabled={isSaving || isLoading}
          >
            {isSaving ? (
              <>
                <span className="mr-2 h-4 w-4 animate-spin rounded-full border-2 border-current border-t-transparent" />
                保存中...
              </>
            ) : (
              <>
                <Save className="mr-2 h-4 w-4" />
                保存所有配置
              </>
            )}
          </Button>
        </div>
      </div>
    </div>
  );
};
