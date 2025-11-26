import { useState, useEffect } from 'react';
import { Settings, Trash2, Save } from 'lucide-react';
import { Button } from '../components/ui/button';
import { Input } from '../components/ui/input';
import { Label } from '../components/ui/label';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '../components/ui/card';
import { toast } from 'sonner';
import { api } from '../services/api';

export const SystemSettingsPage = () => {
  const [trashAutoCleanupDays, setTrashAutoCleanupDays] = useState<number>(7);
  const [isLoading, setIsLoading] = useState(false);
  const [isSaving, setIsSaving] = useState(false);

  // 加载配置
  useEffect(() => {
    loadSettings();
  }, []);

  const loadSettings = async () => {
    try {
      setIsLoading(true);
      const response = await api.get<{ success: boolean; data: { value: string } }>(
        '/settings/system/trash_auto_cleanup_days'
      );
      if (response.data?.value) {
        setTrashAutoCleanupDays(parseInt(response.data.value));
      }
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
      await api.put('/settings/system/trash_auto_cleanup_days', {
        value: trashAutoCleanupDays.toString(),
      });
      toast.success('配置保存成功');
    } catch (err) {
      console.error('Failed to save settings:', err);
      toast.error('保存配置失败');
    } finally {
      setIsSaving(false);
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
                  min="-1"
                  value={trashAutoCleanupDays}
                  onChange={(e) => setTrashAutoCleanupDays(parseInt(e.target.value) || 0)}
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

            <div className="flex justify-end">
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
                    保存配置
                  </>
                )}
              </Button>
            </div>
          </CardContent>
        </Card>
      </div>
    </div>
  );
};
