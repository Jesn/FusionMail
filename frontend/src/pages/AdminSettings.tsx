/**
 * 管理员设置页面（重构版）
 * 使用共享的 SettingsPageLayout 组件
 */

import { useState } from 'react';
import {
  useSettingsByCategory,
  useUpdateSetting,
  useResetSetting,
  useGetStats,
  useWarmUp,
} from '../hooks/useSettings';
import { SettingsPageLayout } from '../components/settings/SettingsPageLayout';
import { Card, CardContent, CardHeader, CardTitle } from '../components/ui/card';
import { Button } from '../components/ui/button';
import { Alert, AlertDescription } from '../components/ui/alert';
import { Tabs, TabsContent, TabsList, TabsTrigger } from '../components/ui/tabs';
import { Shield, Loader2, RefreshCw, Activity } from 'lucide-react';
import { ADMIN_CATEGORIES, getCategoryMeta } from '../constants/settingsCategories';
import { transformSettings } from '../utils/settingsUtils';

export function AdminSettings() {
  const [activeTab, setActiveTab] = useState('ui');
  const [showSensitive, setShowSensitive] = useState(false);

  // 获取所有分类的设置
  const settingsQueries = {
    ui: useSettingsByCategory('ui'),
    sync: useSettingsByCategory('sync'),
    notification: useSettingsByCategory('notification'),
    security: useSettingsByCategory('security'),
    api: useSettingsByCategory('api'),
    oauth: useSettingsByCategory('oauth'),
    smtp: useSettingsByCategory('smtp'),
  };

  // 管理员操作
  const updateSetting = useUpdateSetting();
  const resetSetting = useResetSetting();
  const statsQuery = useGetStats();
  const warmUpMutation = useWarmUp();

  // 处理更新
  const handleUpdate = async (category: string, key: string, value: string) => {
    try {
      await updateSetting.mutateAsync({ category, key, value });
    } catch (error) {
      console.error('更新设置失败:', error);
      throw error;
    }
  };

  // 处理重置
  const handleReset = async (category: string, key: string) => {
    try {
      await resetSetting.mutateAsync({ category, key });
    } catch (error) {
      console.error('重置设置失败:', error);
      throw error;
    }
  };

  // 处理缓存预热
  const handleWarmUp = async () => {
    try {
      await warmUpMutation.mutateAsync(undefined);
    } catch (error) {
      console.error('缓存预热失败:', error);
    }
  };



  return (
    <div className="container mx-auto py-6 space-y-6">
      {/* 页面标题 */}
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-3xl font-bold tracking-tight">系统设置</h1>
          <p className="text-muted-foreground">管理系统级配置和安全设置</p>
        </div>

        <div className="flex items-center gap-2">
          {/* 缓存预热 */}
          <Button variant="outline" onClick={handleWarmUp} disabled={warmUpMutation.isPending}>
            {warmUpMutation.isPending ? (
              <Loader2 className="h-4 w-4 animate-spin mr-2" />
            ) : (
              <RefreshCw className="h-4 w-4 mr-2" />
            )}
            预热缓存
          </Button>

          {/* 显示/隐藏敏感配置 */}
          <Button
            variant={showSensitive ? 'destructive' : 'outline'}
            onClick={() => setShowSensitive(!showSensitive)}
          >
            <Shield className="h-4 w-4 mr-2" />
            {showSensitive ? '隐藏敏感配置' : '显示敏感配置'}
          </Button>
        </div>
      </div>

      {/* 权限警告 */}
      <Alert>
        <Shield className="h-4 w-4" />
        <AlertDescription>
          您正在访问管理员设置页面。请谨慎修改敏感配置，某些更改可能影响系统安全。
        </AlertDescription>
      </Alert>

      {/* 缓存统计 */}
      {statsQuery.data && (
        <Card>
          <CardHeader>
            <CardTitle className="text-sm flex items-center gap-2">
              <Activity className="h-4 w-4" />
              缓存统计
            </CardTitle>
          </CardHeader>
          <CardContent>
            <div className="grid grid-cols-2 md:grid-cols-4 gap-4 text-sm">
              <div>
                <p className="text-muted-foreground">本地缓存命中率</p>
                <p className="text-2xl font-bold">
                  {(statsQuery.data.localCache?.hitRate * 100 || 0).toFixed(1)}%
                </p>
              </div>
              <div>
                <p className="text-muted-foreground">Redis命中率</p>
                <p className="text-2xl font-bold">
                  {(statsQuery.data.redisCache?.hitRate * 100 || 0).toFixed(1)}%
                </p>
              </div>
              <div>
                <p className="text-muted-foreground">本地缓存大小</p>
                <p className="text-2xl font-bold">{statsQuery.data.localCache?.size || 0}</p>
              </div>
              <div>
                <p className="text-muted-foreground">总请求数</p>
                <p className="text-2xl font-bold">{statsQuery.data.totalRequests || 0}</p>
              </div>
            </div>
          </CardContent>
        </Card>
      )}

      {/* 配置分类标签页 */}
      <Tabs value={activeTab} onValueChange={setActiveTab}>
        <TabsList className="grid grid-cols-3 md:grid-cols-7 lg:w-full">
          {ADMIN_CATEGORIES.map((category) => {
            const meta = getCategoryMeta(category, true);
            const query = settingsQueries[category as keyof typeof settingsQueries];
            const settings = query.settings ? transformSettings(query.settings, category) : [];
            const sensitiveCount = settings.filter((s) => s.isSensitive).length;

            return (
              <TabsTrigger key={category} value={category} className="flex items-center gap-2">
                {meta?.icon}
                <span className="hidden md:inline">{meta?.displayName}</span>
                {sensitiveCount > 0 && (
                  <span className="ml-1 px-1.5 py-0.5 text-xs bg-red-100 text-red-700 rounded">
                    {sensitiveCount}
                  </span>
                )}
              </TabsTrigger>
            );
          })}
        </TabsList>

        {/* 标签页内容 */}
        {ADMIN_CATEGORIES.map((category) => (
          <TabsContent key={category} value={category as string} className="space-y-4 mt-6">
            <SettingsPageLayout
              title=""
              description=""
              categories={[category as any]}
              settingsQueries={{ [category]: settingsQueries[category as keyof typeof settingsQueries] }}
              onUpdate={handleUpdate}
              onReset={handleReset}
              isAdmin={true}
            />
          </TabsContent>
        ))}
      </Tabs>

      {/* 提示信息 */}
      <Card className="bg-muted/50">
        <CardHeader>
          <CardTitle className="text-sm">⚠️ 重要提示</CardTitle>
        </CardHeader>
        <CardContent>
          <ul className="text-sm text-muted-foreground space-y-1">
            <li>• 敏感配置已加密存储，修改后自动重新加密</li>
            <li>• 系统级配置影响所有用户，请谨慎修改</li>
            <li>• 建议在修改敏感配置前先导出当前设置作为备份</li>
            <li>• 缓存预热可以提升配置读取性能</li>
          </ul>
        </CardContent>
      </Card>
    </div>
  );
}

export default AdminSettings;
