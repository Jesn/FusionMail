/**
 * 用户设置页面（Tabs 版本）
 * 使用 shadcn Tabs 组件展示不同设置分类
 */

import { useState, useMemo, useEffect } from 'react';
import { useSettingsByCategory, useUpdateSetting, useResetSetting } from '../hooks/useSettings';
import { SettingsCategory } from '../components/settings/SettingsCategory';
import { TwoFactorSettings } from '../components/settings/TwoFactorSettings';
import { ColorThemeSettings } from '../components/settings/ColorThemeSettings';
import { Card, CardContent, CardHeader, CardTitle } from '../components/ui/card';
import { Tabs, TabsContent, TabsList, TabsTrigger } from '../components/ui/tabs';
import { Alert, AlertDescription } from '../components/ui/alert';
import { Loader2, AlertTriangle, Shield, Palette } from 'lucide-react';
import { USER_CATEGORIES, getCategoryMeta } from '../constants/settingsCategories';
import { transformSettings } from '../utils/settingsUtils';
import { useTheme } from '../hooks/useTheme';
import type { SettingItem } from '../types/settings';

export function UserSettings() {
  const [activeTab, setActiveTab] = useState('ui');
  const { theme, setTheme } = useTheme();

  // 获取各类别设置
  const settingsQueries = {
    ui: useSettingsByCategory('ui'),
    sync: useSettingsByCategory('sync'),
    notification: useSettingsByCategory('notification'),
  };

  // 获取更新和重置设置的 mutations
  const updateSetting = useUpdateSetting();
  const resetSetting = useResetSetting();

  // 监听 UI 设置中的 theme 变化，同步到主题系统
  useEffect(() => {
    const uiSettings = settingsQueries.ui.settings;
    if (uiSettings?.theme) {
      const savedTheme = uiSettings.theme as 'light' | 'dark' | 'system';
      if (savedTheme !== theme) {
        setTheme(savedTheme);
      }
    }
  }, [settingsQueries.ui.settings?.theme]);

  // 转换设置数据
  const transformedSettings = useMemo(() => {
    const result: Record<string, SettingItem[]> = {};

    USER_CATEGORIES.forEach((category) => {
      const query = settingsQueries[category as keyof typeof settingsQueries];
      if (!query?.settings) return;

      const items = transformSettings(query.settings, category);
      if (items.length > 0) {
        result[category] = items;
      }
    });

    return result;
  }, [settingsQueries]);

  // 检查加载状态
  const isLoading = Object.values(settingsQueries).some((q) => q.isLoading);

  // 检查错误状态
  const hasError = Object.values(settingsQueries).some((q) => q.error);

  // 处理更新
  const handleUpdate = async (category: string, key: string, value: string) => {
    try {
      await updateSetting.mutateAsync({ category, key, value });
      
      // 如果更新的是主题设置，立即应用主题
      if (category === 'ui' && key === 'theme') {
        setTheme(value as 'light' | 'dark' | 'system');
      }
    } catch (error) {
      console.error('更新设置失败:', error);
      throw error;
    }
  };

  // 处理重置
  const handleReset = async (category: string, key: string) => {
    try {
      await resetSetting.mutateAsync({ category, key });
      
      // 如果重置的是主题设置，恢复默认主题
      if (category === 'ui' && key === 'theme') {
        setTheme('system');
      }
    } catch (error) {
      console.error('重置设置失败:', error);
      throw error;
    }
  };

  return (
    <div className="min-h-screen bg-background">
      <div className="max-w-6xl mx-auto px-6 py-8">
        {/* 页面头部 */}
        <div className="mb-8">
          <div className="flex items-center gap-3 mb-2">
            <div className="text-2xl">⚙️</div>
            <h1 className="text-2xl font-bold text-gray-900 dark:text-white">设置</h1>
          </div>
          <p className="text-gray-600 dark:text-gray-400">管理您的个人偏好和配置选项</p>
        </div>

        <div className="space-y-6">
          {/* 加载状态 */}
          {isLoading && (
            <Card>
              <CardContent className="flex items-center justify-center py-12">
                <Loader2 className="h-8 w-8 animate-spin text-muted-foreground" />
                <span className="ml-2 text-muted-foreground">加载中...</span>
              </CardContent>
            </Card>
          )}

          {/* 错误状态 */}
          {hasError && (
            <Alert variant="destructive">
              <AlertTriangle className="h-4 w-4" />
              <AlertDescription>
                加载设置失败，请刷新页面重试
              </AlertDescription>
            </Alert>
          )}

          {/* Tabs 组件 */}
          {!isLoading && !hasError && (
            <Tabs value={activeTab} onValueChange={setActiveTab} className="w-full">
              <TabsList className="grid w-full grid-cols-5">
                {USER_CATEGORIES.map((category) => {
                  const meta = getCategoryMeta(category, false);
                  return (
                    <TabsTrigger key={category} value={category} className="flex items-center gap-2">
                      {meta?.icon}
                      <span>{meta?.displayName}</span>
                    </TabsTrigger>
                  );
                })}
                {/* 主题设置标签 */}
                <TabsTrigger value="theme" className="flex items-center gap-2">
                  <Palette className="h-4 w-4" />
                  <span>主题</span>
                </TabsTrigger>
                {/* 安全设置标签 */}
                <TabsTrigger value="security" className="flex items-center gap-2">
                  <Shield className="h-4 w-4" />
                  <span>安全</span>
                </TabsTrigger>
              </TabsList>

              {/* 标签页内容 */}
              {USER_CATEGORIES.map((category) => {
                const items = transformedSettings[category];
                if (!items || items.length === 0) return null;

                const meta = getCategoryMeta(category, false);

                return (
                  <TabsContent key={category} value={category} className="mt-6">
                    <SettingsCategory
                      name={category}
                      displayName={meta?.displayName || category}
                      description={meta?.description}
                      icon={meta?.icon}
                      items={items}
                      onUpdate={(key, value) => handleUpdate(category, key, value)}
                      onReset={(key) => handleReset(category, key)}
                      isLoading={false}
                      isEditable={true}
                    />
                  </TabsContent>
                );
              })}

              {/* 主题设置标签页 */}
              <TabsContent value="theme" className="mt-6">
                <ColorThemeSettings />
              </TabsContent>

              {/* 安全设置标签页 */}
              <TabsContent value="security" className="mt-6">
                <TwoFactorSettings />
              </TabsContent>
            </Tabs>
          )}

          {/* 提示信息 */}
          <Card className="bg-muted/50">
            <CardHeader>
              <CardTitle className="text-sm">💡 提示</CardTitle>
            </CardHeader>
            <CardContent>
              <ul className="text-sm text-muted-foreground space-y-1">
                <li>• 您的设置会自动保存</li>
                <li>• 重置功能可以恢复到默认值</li>
                <li>• 某些设置需要刷新页面后生效</li>
              </ul>
            </CardContent>
          </Card>
        </div>
      </div>
    </div>
  );
}

export default UserSettings;
