/**
 * 配置搜索和筛选工具函数
 */

import type { SettingItem } from './SettingsCategory';
import type { FilterCriteria } from './SearchAndFilter';

// 筛选配置项
export function filterSettings(
  settings: SettingItem[],
  criteria: FilterCriteria
): SettingItem[] {
  let filtered = [...settings];

  // 关键字搜索
  if (criteria.query.trim()) {
    const query = criteria.query.toLowerCase().trim();
    filtered = filtered.filter(
      (item) =>
        item.key.toLowerCase().includes(query) ||
        item.value.toLowerCase().includes(query) ||
        (item.description && item.description.toLowerCase().includes(query))
    );
  }

  // 分类筛选
  if (criteria.category !== 'all') {
    // 注意：这里需要从父组件传递分类信息
    // 或者在SettingItem中添加category字段
  }

  // 类型筛选
  if (criteria.valueType !== 'all') {
    filtered = filtered.filter((item) => item.valueType === criteria.valueType);
  }

  // 敏感度筛选
  if (criteria.sensitivity === 'sensitive') {
    filtered = filtered.filter((item) => item.isSensitive);
  } else if (criteria.sensitivity === 'public') {
    filtered = filtered.filter((item) => !item.isSensitive);
  }

  // 描述筛选
  if (criteria.hasDescription !== null) {
    filtered = filtered.filter((item) => {
      const hasDescription = item.description && item.description.trim().length > 0;
      return criteria.hasDescription ? hasDescription : !hasDescription;
    });
  }

  return filtered;
}

// 排序配置项
export function sortSettings(
  settings: SettingItem[],
  sortBy: 'key' | 'value' | 'type' | 'sensitivity' = 'key',
  sortOrder: 'asc' | 'desc' = 'asc'
): SettingItem[] {
  return [...settings].sort((a, b) => {
    let comparison = 0;

    switch (sortBy) {
      case 'key':
        comparison = a.key.localeCompare(b.key);
        break;
      case 'value':
        comparison = a.value.localeCompare(b.value);
        break;
      case 'type':
        comparison = a.valueType.localeCompare(b.valueType);
        break;
      case 'sensitivity':
        const aSensitive = a.isSensitive ? 1 : 0;
        const bSensitive = b.isSensitive ? 1 : 0;
        comparison = aSensitive - bSensitive;
        break;
    }

    return sortOrder === 'asc' ? comparison : -comparison;
  });
}

// 高亮搜索关键词
export function highlightSearchTerm(text: string, searchTerm: string): string {
  if (!searchTerm.trim()) return text;

  const regex = new RegExp(`(${escapeRegExp(searchTerm)})`, 'gi');
  return text.replace(regex, '<mark>$1</mark>');
}

// 转义正则表达式特殊字符
function escapeRegExp(string: string): string {
  return string.replace(/[.*+?^${}()|[\]\\]/g, '\\$&');
}

// 获取配置项统计信息
export function getSettingStats(settings: SettingItem[]) {
  const total = settings.length;
  const sensitive = settings.filter((s) => s.isSensitive).length;
  const publicCount = total - sensitive;
  const byType = settings.reduce((acc, setting) => {
    acc[setting.valueType] = (acc[setting.valueType] || 0) + 1;
    return acc;
  }, {} as Record<string, number>);
  const withDescription = settings.filter((s) => s.description).length;

  return {
    total,
    sensitive,
    public: publicCount,
    byType,
    withDescription,
    withoutDescription: total - withDescription,
  };
}

// 检查是否匹配筛选条件
export function matchesCriteria(
  setting: SettingItem,
  criteria: FilterCriteria,
  category?: string
): boolean {
  // 分类匹配
  if (criteria.category !== 'all' && category && category !== criteria.category) {
    return false;
  }

  // 关键字搜索
  if (criteria.query.trim()) {
    const query = criteria.query.toLowerCase().trim();
    const matchesKey = setting.key.toLowerCase().includes(query);
    const matchesValue = setting.value.toLowerCase().includes(query);
    const matchesDescription =
      setting.description && setting.description.toLowerCase().includes(query);

    if (!matchesKey && !matchesValue && !matchesDescription) {
      return false;
    }
  }

  // 类型筛选
  if (criteria.valueType !== 'all' && setting.valueType !== criteria.valueType) {
    return false;
  }

  // 敏感度筛选
  if (
    criteria.sensitivity === 'sensitive' &&
    !setting.isSensitive
  ) {
    return false;
  }

  if (
    criteria.sensitivity === 'public' &&
    setting.isSensitive
  ) {
    return false;
  }

  // 描述筛选
  if (criteria.hasDescription !== null) {
    const hasDescription = setting.description && setting.description.trim().length > 0;
    if (criteria.hasDescription !== hasDescription) {
      return false;
    }
  }

  return true;
}
