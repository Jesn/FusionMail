/**
 * 配置分类组件
 * 用于分组显示和管理配置项
 */

import { useState } from 'react';
import { Settings as SettingsIcon } from 'lucide-react';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '../ui/card';
import { Badge } from '../ui/badge';
import { Button } from '../ui/button';
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '../ui/select';
import type { SettingsCategoryProps, SettingsToolbarProps, SettingItem } from '../../types/settings';
import { updateCachedSettings } from '../../utils/settingsCache';
import { getFieldConfig } from './settingOptions';

export type { SettingItem };

export function SettingsCategory({
  name,
  displayName,
  description,
  icon,
  items,
  onUpdate,
  onReset,
  isLoading = false,
  isEditable = true,
}: SettingsCategoryProps) {
  const [localItems, setLocalItems] = useState<Record<string, string>>(
    items.reduce((acc, item) => {
      acc[item.key] = item.value;
      return acc;
    }, {} as Record<string, string>)
  );

  // 处理值变化
  const handleValueChange = (key: string, value: string) => {
    setLocalItems((prev) => {
      const newItems = {
        ...prev,
        [key]: value,
      };
      
      // 更新缓存（仅支持 ui/sync/notification 分类）
      if (name === 'ui' || name === 'sync' || name === 'notification') {
        updateCachedSettings(name, newItems);
        console.log(`已更新 ${name} 设置缓存`);
      }
      
      return newItems;
    });
    
    onUpdate(key, value);
  };

  // 处理重置
  const handleReset = (key: string) => {
    const item = items.find((i) => i.key === key);
    if (item) {
      // 特殊字段的默认值
      let resetValue = item.value;
      if (key === 'language') {
        resetValue = 'zh-CN';
      } else if (key === 'email_page_size') {
        resetValue = '20';
      }
      
      setLocalItems((prev) => {
        const newItems = {
          ...prev,
          [key]: resetValue,
        };
        
        // 更新缓存
        if (name === 'ui' || name === 'sync' || name === 'notification') {
          updateCachedSettings(name, newItems);
          console.log(`已重置 ${name}.${key} 并更新缓存`);
        }
        
        return newItems;
      });
      
      onReset?.(key);
    }
  };

  // 统计信息
  const sensitiveCount = items.filter((item) => item.isSensitive).length;
  const totalCount = items.length;

  return (
    <Card className="w-full">
      <CardHeader>
        <div className="flex items-center justify-between">
          <div className="flex items-center gap-3">
            <div className="flex items-center gap-2">
              {icon || <SettingsIcon className="h-5 w-5" />}
              <CardTitle className="text-lg">{displayName}</CardTitle>
            </div>

            <div className="flex items-center gap-2">
              {sensitiveCount > 0 && (
                <Badge variant="outline" className="text-xs">
                  {sensitiveCount} 敏感
                </Badge>
              )}
              <Badge variant="secondary" className="text-xs">
                {totalCount} 项
              </Badge>
            </div>
          </div>
        </div>

        {description && (
          <CardDescription className="mt-2">{description}</CardDescription>
        )}
      </CardHeader>

      <CardContent>
            {items.length === 0 ? (
              <div className="text-center py-8 text-muted-foreground">
                <SettingsIcon className="h-12 w-12 mx-auto mb-2 opacity-50" />
                <p>该分类下暂无配置</p>
              </div>
            ) : (
              <div className="space-y-6">
                {items.map((item) => (
                  <div key={item.key} className="border-b pb-4 last:border-b-0 last:pb-0">
                    <div className="space-y-2">
                      <div className="flex items-center justify-between gap-2">
                        <div className="flex items-center gap-2">
                          <label className="text-sm font-medium">
                            {item.key}
                          </label>

                          {item.isSensitive && (
                            <Badge variant="outline" className="text-xs text-amber-600">
                              敏感
                            </Badge>
                          )}

                          <Badge variant="secondary" className="text-xs">
                            {item.valueType}
                          </Badge>
                        </div>

                        {isEditable && onReset && (
                          <Button
                            variant="ghost"
                            size="sm"
                            onClick={() => handleReset(item.key)}
                            disabled={isLoading}
                          >
                            重置
                          </Button>
                        )}
                      </div>

                      {/* 描述 */}
                      {item.description && (
                        <p className="text-xs text-muted-foreground">
                          {item.description}
                        </p>
                      )}

                      {/* 值输入区域 */}
                      <div className="mt-2">
                        {(() => {
                          const fieldConfig = getFieldConfig(item.key);
                          
                          // 通用下拉框渲染（支持 theme、language、default_view 等）
                          if (fieldConfig?.type === 'select' && fieldConfig.options) {
                            return (
                              <Select
                                value={localItems[item.key] || fieldConfig.options[0]?.value || ''}
                                onValueChange={(value) => handleValueChange(item.key, value)}
                                disabled={!isEditable || isLoading}
                              >
                                <SelectTrigger className="w-full">
                                  <SelectValue placeholder={fieldConfig.placeholder || '请选择'} />
                                </SelectTrigger>
                                <SelectContent>
                                  {fieldConfig.options.map((option) => (
                                    <SelectItem key={option.value} value={option.value}>
                                      <div className="flex flex-col">
                                        <span>{option.label}</span>
                                        {option.description && (
                                          <span className="text-xs text-muted-foreground">
                                            {option.description}
                                          </span>
                                        )}
                                      </div>
                                    </SelectItem>
                                  ))}
                                </SelectContent>
                              </Select>
                            );
                          }
                          
                          // 布尔值开关
                          if (item.valueType === 'boolean') {
                            return (
                              <div className="flex items-center gap-2">
                                <input
                                  type="checkbox"
                                  id={`${name}-${item.key}`}
                                  checked={localItems[item.key] === 'true'}
                                  onChange={(e) =>
                                    handleValueChange(item.key, e.target.checked ? 'true' : 'false')
                                  }
                                  disabled={!isEditable || isLoading}
                                  className="h-4 w-4 rounded border-gray-300"
                                />
                                <label
                                  htmlFor={`${name}-${item.key}`}
                                  className="text-sm text-muted-foreground"
                                >
                                  {localItems[item.key] === 'true' ? '已启用' : '已禁用'}
                                </label>
                              </div>
                            );
                          }
                          
                          // 数字输入
                          if (item.valueType === 'number') {
                            return (
                              <input
                                type="number"
                                value={localItems[item.key]}
                                onChange={(e) => handleValueChange(item.key, e.target.value)}
                                disabled={!isEditable || isLoading}
                                className="w-full px-3 py-2 border rounded-md"
                              />
                            );
                          }
                          
                          // JSON 文本域
                          if (item.valueType === 'json') {
                            return (
                              <textarea
                                value={localItems[item.key]}
                                onChange={(e) => handleValueChange(item.key, e.target.value)}
                                disabled={!isEditable || isLoading}
                                rows={4}
                                className="w-full px-3 py-2 border rounded-md font-mono text-sm"
                                placeholder="输入有效的JSON格式"
                              />
                            );
                          }
                          
                          // 密码输入
                          if (fieldConfig?.type === 'password') {
                            return (
                              <input
                                type="password"
                                value={localItems[item.key]}
                                onChange={(e) => handleValueChange(item.key, e.target.value)}
                                disabled={!isEditable || isLoading}
                                className="w-full px-3 py-2 border rounded-md"
                                placeholder={fieldConfig.placeholder || item.description || `输入${item.key}的值`}
                              />
                            );
                          }
                          
                          // 默认文本输入
                          return (
                            <input
                              type="text"
                              value={localItems[item.key]}
                              onChange={(e) => handleValueChange(item.key, e.target.value)}
                              disabled={!isEditable || isLoading}
                              className="w-full px-3 py-2 border rounded-md"
                              placeholder={fieldConfig?.placeholder || item.description || `输入${item.key}的值`}
                            />
                          );
                        })()}
                      </div>

                      {/* 当前值显示（只读模式） */}
                      {!isEditable && (
                        <div className="mt-1 text-xs text-muted-foreground">
                          当前值：{item.value}
                        </div>
                      )}
                    </div>
                  </div>
                ))}
              </div>
            )}
          </CardContent>
    </Card>
  );
}

/**
 * 批量操作工具栏
 */
export function SettingsToolbar({
  category,
  itemCount,
  onRefresh,
  onExport,
  onImport,
  isLoading = false,
}: SettingsToolbarProps) {
  return (
    <div className="flex items-center justify-between mb-4">
      <div>
        <h2 className="text-lg font-semibold">{category}</h2>
        <p className="text-sm text-muted-foreground">{itemCount} 个配置项</p>
      </div>

      <div className="flex items-center gap-2">
        {onRefresh && (
          <Button variant="outline" size="sm" onClick={onRefresh} disabled={isLoading}>
            刷新
          </Button>
        )}
        {onExport && (
          <Button variant="outline" size="sm" onClick={onExport}>
            导出
          </Button>
        )}
        {onImport && (
          <Button variant="outline" size="sm" onClick={onImport}>
            导入
          </Button>
        )}
      </div>
    </div>
  );
}
