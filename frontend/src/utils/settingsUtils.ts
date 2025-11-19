/**
 * 配置管理工具函数
 * 提供配置数据转换、验证等通用功能
 */

import type { SettingItem, SettingValueType } from '../types/settings';
import { getFieldConfig, isSwitchKey, isNumberKey } from '../components/settings/settingOptions';

/**
 * 将 API 返回的配置对象转换为 SettingItem 数组
 */
export function transformSettings(
  settingsObj: Record<string, string>,
  category: string
): SettingItem[] {
  return Object.entries(settingsObj).map(([key, value]) => {
    // 根据配置项名称推断类型
    let valueType: SettingValueType = 'string';

    if (isSwitchKey(key)) {
      valueType = 'boolean';
    } else if (isNumberKey(key)) {
      valueType = 'number';
    } else if (key.includes('config') || key.includes('json')) {
      valueType = 'json';
    }

    // 检查是否为敏感配置
    const fieldConfig = getFieldConfig(key);
    const isSensitive =
      fieldConfig?.type === 'password' ||
      key.includes('password') ||
      key.includes('secret') ||
      key.includes('token');

    return {
      category,
      key,
      value: value || '',
      valueType,
      isSensitive,
      isPublic: !isSensitive,
      description: fieldConfig?.placeholder || '',
    };
  });
}

/**
 * 验证配置值
 */
export function validateSettingValue(
  key: string,
  value: string,
  valueType: SettingValueType
): { valid: boolean; error?: string } {
  const fieldConfig = getFieldConfig(key);

  // 类型验证
  switch (valueType) {
    case 'number':
      if (isNaN(Number(value))) {
        return { valid: false, error: '请输入有效的数字' };
      }
      if (fieldConfig?.numberRange) {
        const num = Number(value);
        if (num < fieldConfig.numberRange.min || num > fieldConfig.numberRange.max) {
          return {
            valid: false,
            error: `值必须在 ${fieldConfig.numberRange.min} 到 ${fieldConfig.numberRange.max} 之间`,
          };
        }
      }
      break;

    case 'boolean':
      if (value !== 'true' && value !== 'false') {
        return { valid: false, error: '请输入有效的布尔值（true/false）' };
      }
      break;

    case 'json':
      try {
        JSON.parse(value);
      } catch {
        return { valid: false, error: '请输入有效的 JSON 格式' };
      }
      break;
  }

  // 自定义验证规则
  if (fieldConfig?.validation) {
    const { pattern, minLength, maxLength } = fieldConfig.validation;

    if (pattern && !pattern.test(value)) {
      return { valid: false, error: '格式不正确' };
    }

    if (minLength && value.length < minLength) {
      return { valid: false, error: `长度不能少于 ${minLength} 个字符` };
    }

    if (maxLength && value.length > maxLength) {
      return { valid: false, error: `长度不能超过 ${maxLength} 个字符` };
    }
  }

  return { valid: true };
}

/**
 * 格式化配置值用于显示
 */
export function formatSettingValue(value: string, valueType: SettingValueType): string {
  switch (valueType) {
    case 'boolean':
      return value === 'true' ? '已启用' : '已禁用';

    case 'json':
      try {
        return JSON.stringify(JSON.parse(value), null, 2);
      } catch {
        return value;
      }

    default:
      return value;
  }
}

/**
 * 检查配置是否已修改
 */
export function isSettingModified(
  originalValue: string,
  currentValue: string
): boolean {
  return originalValue !== currentValue;
}

/**
 * 批量验证配置
 */
export function validateSettings(
  settings: Record<string, { value: string; valueType: SettingValueType }>
): Record<string, string> {
  const errors: Record<string, string> = {};

  Object.entries(settings).forEach(([key, { value, valueType }]) => {
    const result = validateSettingValue(key, value, valueType);
    if (!result.valid && result.error) {
      errors[key] = result.error;
    }
  });

  return errors;
}

/**
 * 导出配置为 JSON 字符串
 */
export function exportSettingsToJSON(settings: SettingItem[]): string {
  const data = settings.reduce((acc, item) => {
    if (!acc[item.category]) {
      acc[item.category] = {};
    }
    acc[item.category][item.key] = item.value;
    return acc;
  }, {} as Record<string, Record<string, string>>);

  return JSON.stringify(data, null, 2);
}

/**
 * 导出配置为 ENV 格式
 */
export function exportSettingsToENV(settings: SettingItem[]): string {
  return settings
    .map((item) => {
      const key = `${item.category.toUpperCase()}_${item.key.toUpperCase()}`;
      const value = item.value.includes(' ') ? `"${item.value}"` : item.value;
      return `${key}=${value}`;
    })
    .join('\n');
}

/**
 * 从 JSON 字符串导入配置
 */
export function importSettingsFromJSON(
  jsonString: string
): Record<string, Record<string, string>> {
  try {
    return JSON.parse(jsonString);
  } catch (error) {
    throw new Error('无效的 JSON 格式');
  }
}

/**
 * 从 ENV 格式导入配置
 */
export function importSettingsFromENV(
  envString: string
): Record<string, Record<string, string>> {
  const result: Record<string, Record<string, string>> = {};

  envString.split('\n').forEach((line) => {
    line = line.trim();
    if (!line || line.startsWith('#')) return;

    const [key, ...valueParts] = line.split('=');
    if (!key || valueParts.length === 0) return;

    let value = valueParts.join('=').trim();
    // 移除引号
    if ((value.startsWith('"') && value.endsWith('"')) ||
        (value.startsWith("'") && value.endsWith("'"))) {
      value = value.slice(1, -1);
    }

    // 解析分类和键名
    const parts = key.split('_');
    if (parts.length < 2) return;

    const category = parts[0].toLowerCase();
    const settingKey = parts.slice(1).join('_').toLowerCase();

    if (!result[category]) {
      result[category] = {};
    }
    result[category][settingKey] = value;
  });

  return result;
}

/**
 * 计算配置统计信息
 */
export function calculateSettingsStats(settings: SettingItem[]): {
  total: number;
  sensitive: number;
  public: number;
  byCategory: Record<string, number>;
  byType: Record<SettingValueType, number>;
} {
  const stats = {
    total: settings.length,
    sensitive: 0,
    public: 0,
    byCategory: {} as Record<string, number>,
    byType: {} as Record<SettingValueType, number>,
  };

  settings.forEach((item) => {
    if (item.isSensitive) stats.sensitive++;
    if (item.isPublic) stats.public++;

    stats.byCategory[item.category] = (stats.byCategory[item.category] || 0) + 1;
    stats.byType[item.valueType] = (stats.byType[item.valueType] || 0) + 1;
  });

  return stats;
}

/**
 * 搜索配置项
 */
export function searchSettings(
  settings: SettingItem[],
  query: string
): SettingItem[] {
  const lowerQuery = query.toLowerCase();

  return settings.filter((item) => {
    return (
      item.key.toLowerCase().includes(lowerQuery) ||
      item.value.toLowerCase().includes(lowerQuery) ||
      item.description?.toLowerCase().includes(lowerQuery) ||
      item.category.toLowerCase().includes(lowerQuery)
    );
  });
}

/**
 * 按分类分组配置
 */
export function groupSettingsByCategory(
  settings: SettingItem[]
): Record<string, SettingItem[]> {
  return settings.reduce((acc, item) => {
    if (!acc[item.category]) {
      acc[item.category] = [];
    }
    acc[item.category].push(item);
    return acc;
  }, {} as Record<string, SettingItem[]>);
}

/**
 * 合并配置（用于导入时）
 */
export function mergeSettings(
  existing: SettingItem[],
  imported: SettingItem[],
  overwrite: boolean = false
): SettingItem[] {
  const existingMap = new Map(
    existing.map((item) => [`${item.category}:${item.key}`, item])
  );

  imported.forEach((item) => {
    const key = `${item.category}:${item.key}`;
    if (overwrite || !existingMap.has(key)) {
      existingMap.set(key, item);
    }
  });

  return Array.from(existingMap.values());
}
