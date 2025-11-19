/**
 * 配置项组件
 * 根据配置类型渲染不同的输入组件
 */

import React, { useState, useCallback } from 'react';
import { Input } from '../ui/input';
import { Switch } from '../ui/switch';
import { Textarea } from '../ui/textarea';
import { Label } from '../ui/label';
import { Button } from '../ui/button';
import { Badge } from '../ui/badge';
import { Eye, EyeOff, RotateCcw, AlertTriangle, CheckCircle2 } from 'lucide-react';
import { Tooltip, TooltipContent, TooltipProvider, TooltipTrigger } from '../ui/tooltip';
import { Popover, PopoverContent, PopoverTrigger } from '../ui/popover';
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '../ui/select';
import { getFieldConfig } from './settingOptions';
import type { SettingItemProps } from '../../types/settings';

export function SettingItem({
  item,
  onChange,
  onReset,
  isLoading = false,
  isDirty = false,
  error,
  disabled = false,
}: SettingItemProps) {
  const [showValue, setShowValue] = useState(false);
  const [localValue, setLocalValue] = useState(item.value);

  // 处理值变化
  const handleChange = useCallback(
    (newValue: string) => {
      setLocalValue(newValue);
      onChange(item.key, newValue);
    },
    [item.key, onChange]
  );

  // 重置值
  const handleReset = useCallback(() => {
    setLocalValue(item.value);
    onReset?.(item.key);
  }, [item.key, item.value, onReset]);

  // 获取配置项的UI配置
  const fieldConfig = getFieldConfig(item.key);
  const configType = fieldConfig?.type || 'input';

  // 渲染输入组件
  const renderInput = () => {
    const commonProps = {
      id: `setting-${item.key}`,
      value: localValue,
      onChange: (e: React.ChangeEvent<HTMLInputElement | HTMLTextAreaElement>) =>
        handleChange(e.target.value),
      disabled: disabled || isLoading,
      className: error ? 'border-red-500' : '',
    };

    // 优先使用配置项定义的类型
    switch (configType) {
      case 'select':
        if (!fieldConfig?.options) return <Input {...commonProps} />;
        return (
          <Select
            value={localValue}
            onValueChange={handleChange}
            disabled={disabled || isLoading}
          >
            <SelectTrigger>
              <SelectValue placeholder={fieldConfig.placeholder || '请选择...'} />
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

      case 'number':
        return (
          <div className="space-y-2">
            <div className="flex items-center gap-2">
              <Input
                {...commonProps}
                type="number"
                min={fieldConfig?.numberRange?.min || 0}
                max={fieldConfig?.numberRange?.max}
                step={fieldConfig?.numberRange?.step || 1}
              />
              {fieldConfig?.numberRange?.suffix && (
                <span className="text-sm text-muted-foreground whitespace-nowrap">
                  {fieldConfig.numberRange.suffix}
                </span>
              )}
            </div>
            {fieldConfig?.numberRange && (
              <input
                type="range"
                min={fieldConfig.numberRange.min}
                max={fieldConfig.numberRange.max}
                step={fieldConfig.numberRange.step}
                value={Number(localValue) || fieldConfig.numberRange.min}
                onChange={(e) => handleChange(e.target.value)}
                className="w-full h-2 bg-gray-200 rounded-lg appearance-none cursor-pointer accent-blue-600"
                disabled={disabled || isLoading}
              />
            )}
          </div>
        );

      case 'switch':
        return (
          <div className="flex items-center justify-between p-3 bg-muted rounded-lg">
            <div className="flex-1">
              <p className="text-sm font-medium">{fieldConfig?.placeholder || '启用此设置'}</p>
              {localValue === 'true' && (
                <p className="text-xs text-green-600 flex items-center gap-1 mt-1">
                  <CheckCircle2 className="h-3 w-3" />
                  已启用
                </p>
              )}
            </div>
            <Switch
              checked={localValue === 'true'}
              onCheckedChange={(checked) => handleChange(checked.toString())}
              disabled={disabled || isLoading}
            />
          </div>
        );

      case 'password':
        return (
          <div className="relative flex items-center">
            <Input
              {...commonProps}
              type={showValue ? 'text' : 'password'}
              className="pr-10"
              placeholder={fieldConfig?.placeholder || '输入密码'}
            />
            <TooltipProvider>
              <Tooltip>
                <TooltipTrigger asChild>
                  <Button
                    type="button"
                    variant="ghost"
                    size="icon"
                    className="absolute right-2 h-6 w-6"
                    onClick={() => setShowValue(!showValue)}
                    disabled={disabled || isLoading}
                  >
                    {showValue ? (
                      <EyeOff className="h-4 w-4" />
                    ) : (
                      <Eye className="h-4 w-4" />
                    )}
                  </Button>
                </TooltipTrigger>
                <TooltipContent>
                  {showValue ? '隐藏' : '显示'}
                </TooltipContent>
              </Tooltip>
            </TooltipProvider>
          </div>
        );

      case 'textarea':
        return (
          <Textarea
            {...commonProps}
            rows={4}
            placeholder={fieldConfig?.placeholder || '输入内容'}
          />
        );

      case 'input':
      default:
        if (localValue.length > 50 || localValue.includes('\n')) {
          return (
            <Textarea
              {...commonProps}
              rows={3}
              placeholder={fieldConfig?.placeholder || '输入值'}
            />
          );
        }
        return (
          <Input
            {...commonProps}
            placeholder={fieldConfig?.placeholder || '输入值'}
          />
        );
    }
  };

  return (
    <div className="space-y-2">
      <div className="flex items-start justify-between gap-2">
        <div className="space-y-1 flex-1">
          <div className="flex items-center gap-2">
            <Label
              htmlFor={`setting-${item.key}`}
              className="text-sm font-medium"
            >
              {item.key}
            </Label>

            {/* 敏感配置标记 */}
            {item.isSensitive && (
              <TooltipProvider>
                <Tooltip>
                  <TooltipTrigger>
                    <Badge variant="outline" className="text-xs">
                      敏感
                    </Badge>
                  </TooltipTrigger>
                  <TooltipContent>
                    <p>此配置包含敏感信息，将被加密存储</p>
                  </TooltipContent>
                </Tooltip>
              </TooltipProvider>
            )}

            {/* 配置类型标记 */}
            {configType && configType !== 'input' && (
              <Badge variant="secondary" className="text-xs">
                {configType === 'number' ? '数字' :
                 configType === 'select' ? '选择' :
                 configType === 'switch' ? '开关' :
                 configType === 'password' ? '密码' :
                 configType === 'textarea' ? '多行文本' :
                 item.valueType}
              </Badge>
            )}

            {/* 数据类型标记 */}
            {configType === 'input' && (
              <Badge variant="secondary" className="text-xs">
                {item.valueType}
              </Badge>
            )}

            {/* 脏数据标记 */}
            {isDirty && (
              <Badge variant="secondary" className="text-xs bg-yellow-100 text-yellow-700">
                未保存
              </Badge>
            )}
          </div>

          {/* 配置描述 */}
          {item.description && (
            <p className="text-xs text-muted-foreground">
              {item.description}
            </p>
          )}

          {/* 错误信息 */}
          {error && (
            <div className="flex items-center gap-1 text-xs text-red-600">
              <AlertTriangle className="h-3 w-3" />
              {error}
            </div>
          )}
        </div>

        {/* 操作按钮 */}
        <div className="flex items-center gap-2">
          {/* 重置按钮 */}
          {onReset && (
            <TooltipProvider>
              <Tooltip>
                <TooltipTrigger asChild>
                  <Button
                    type="button"
                    variant="ghost"
                    size="icon"
                    onClick={handleReset}
                    disabled={disabled || isLoading}
                  >
                    <RotateCcw className="h-4 w-4" />
                  </Button>
                </TooltipTrigger>
                <TooltipContent>重置为默认值</TooltipContent>
              </Tooltip>
            </TooltipProvider>
          )}

          {/* JSON预览按钮 */}
          {item.valueType === 'json' && (
            <Popover>
              <PopoverTrigger asChild>
                <Button
                  type="button"
                  variant="outline"
                  size="sm"
                  disabled={disabled || isLoading}
                >
                  预览
                </Button>
              </PopoverTrigger>
              <PopoverContent className="w-[400px]" align="end">
                <div className="space-y-2">
                  <h4 className="font-medium">JSON 预览</h4>
                  <pre className="p-4 bg-muted rounded-md text-xs overflow-auto max-h-[300px]">
                    {localValue ? JSON.stringify(JSON.parse(localValue), null, 2) : '无数据'}
                  </pre>
                </div>
              </PopoverContent>
            </Popover>
          )}
        </div>
      </div>

      {/* 输入组件 */}
      <div className="mt-2">
        {renderInput()}
      </div>

      {/* 加载指示器 */}
      {isLoading && (
        <div className="text-xs text-muted-foreground">
          保存中...
        </div>
      )}
    </div>
  );
}

/**
 * 配置项组容器
 */
interface SettingItemGroupProps {
  title: string;
  description?: string;
  children: React.ReactNode;
  className?: string;
}

export function SettingItemGroup({
  title,
  description,
  children,
  className,
}: SettingItemGroupProps) {
  return (
    <div className={`space-y-4 ${className || ''}`}>
      <div>
        <h3 className="text-lg font-semibold">{title}</h3>
        {description && (
          <p className="text-sm text-muted-foreground">{description}</p>
        )}
      </div>
      <div className="space-y-4">{children}</div>
    </div>
  );
}
