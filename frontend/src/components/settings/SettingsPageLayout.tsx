/**
 * 设置页面布局组件
 * 提供统一的设置页面布局和通用功能
 */

import { useMemo } from 'react';
import { Card, CardContent } from '../ui/card';
import { Button } from '../ui/button';
import { Alert, AlertDescription } from '../ui/alert';
import { Loader2, AlertTriangle } from 'lucide-react';
import { SettingsCategory } from './SettingsCategory';

import { transformSettings } from '../../utils/settingsUtils';
import type { SettingItem, CategoryName } from '../../types/settings';
import { getCategoryMeta } from '../../constants/settingsCategories';

interface SettingsPageLayoutProps {
  title: string;
  description: string;
  categories: CategoryName[];
  settingsQueries: Record<string, any>;
  onUpdate: (category: string, key: string, value: string) => Promise<void>;
  onReset: (category: string, key: string) => Promise<void>;
  isAdmin?: boolean;

  headerActions?: React.ReactNode;
  alerts?: React.ReactNode;
  footer?: React.ReactNode;
}

export function SettingsPageLayout({
  title,
  description,
  categories,
  settingsQueries,
  onUpdate,
  onReset,
  isAdmin = false,
  headerActions,
  alerts,
  footer,
}: SettingsPageLayoutProps) {



  // 转换设置
  const transformedSettings = useMemo(() => {
    const result: Record<string, SettingItem[]> = {};

    categories.forEach((category) => {
      const query = settingsQueries[category];
      if (!query?.settings) return;

      const items = transformSettings(query.settings, category);
      if (items.length > 0) {
        result[category] = items;
      }
    });

    return result;
  }, [categories, settingsQueries]);



  // 检查加载状态
  const isLoading = Object.values(settingsQueries).some((q) => q.isLoading);

  // 检查错误状态
  const hasError = Object.values(settingsQueries).some((q) => q.error);



  return (
    <div className="min-h-screen bg-gray-50">
      <div className="max-w-6xl mx-auto px-6 py-8">
        {/* 页面头部 */}
        <div className="mb-8">
          <div className="flex items-start justify-between">
            <div>
              <div className="flex items-center gap-3 mb-2">
                <div className="text-2xl">⚙️</div>
                <h1 className="text-2xl font-bold text-gray-900 dark:text-white">{title}</h1>
              </div>
              <p className="text-gray-600 dark:text-gray-400">{description}</p>
            </div>
            {/* 头部操作按钮 */}
            {headerActions && <div className="flex items-center gap-2">{headerActions}</div>}
          </div>
        </div>

        <div className="space-y-6">
        {/* 警告提示 */}
        {alerts}



        {/* 加载状态 */}
        {isLoading && (
          <Card>
            <CardContent className="flex items-center justify-center py-12">
              <Loader2 className="h-8 w-8 animate-spin text-muted-foreground" />
              <span className="ml-2 text-muted-foreground">加载中...</span>
            </CardContent>
          </Card>
        )}

        {/* 配置分类列表 */}
        {!isLoading && (
          <div className="space-y-6">
            {categories.map((category) => {
              const items = transformedSettings[category];
              if (!items || items.length === 0) return null;

              const meta = getCategoryMeta(category, isAdmin);

              return (
                <SettingsCategory
                  key={category}
                  name={category}
                  displayName={meta?.displayName || category}
                  description={meta?.description}
                  icon={meta?.icon}
                  items={items}
                  onUpdate={(key, value) => onUpdate(category, key, value)}
                  onReset={(key) => onReset(category, key)}
                  isLoading={false}
                  isEditable={true}
                />
              );
            })}
          </div>
        )}



        {/* 错误状态 */}
        {hasError && (
          <Alert variant="destructive">
            <AlertTriangle className="h-4 w-4" />
            <AlertDescription>
              加载设置失败，请刷新页面重试
              <Button
                variant="outline"
                size="sm"
                className="ml-4"
                onClick={() => {
                  Object.values(settingsQueries).forEach((q) => q.refetch?.());
                }}
              >
                重新加载
              </Button>
            </AlertDescription>
          </Alert>
        )}

        {/* 页脚 */}
        {footer}
        </div>
      </div>
    </div>
  );
}
