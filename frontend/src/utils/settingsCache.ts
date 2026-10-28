/**
 * 设置缓存（P20）—— localStorage 中唯一允许持久化的设置值。
 *
 * 决策见 utils/cachePolicy.ts：业务数据不跨刷新，只有 PERSISTED_SETTING_KEYS
 * 白名单内的 key 落盘（目前仅 email_page_size，因为 stores/emailStore.ts 在
 * 模块初始化时同步读取它来定 pageSize 初值）。
 *
 * 历史问题：旧实现按整个分类（ui / sync / notification）落盘，而设置项带
 * isSensitive 标记（components/settings/SettingsCategory.tsx 自己都在统计
 * sensitiveCount），存在敏感值被写入 localStorage 的风险。现在按 key 过滤，
 * 非白名单 key 一律丢弃。
 */

import {
  SETTINGS_CACHE_TTL,
  PERSISTED_SETTING_KEYS,
  isPersistedSettingKey,
} from './cachePolicy';

const SETTINGS_CACHE_KEY = 'user_settings_cache';
const SETTINGS_CACHE_TIMESTAMP_KEY = 'user_settings_cache_timestamp';

/** 只保留白名单内的 key */
const pickPersisted = (settings: Record<string, string>): Record<string, string> => {
  const result: Record<string, string> = {};
  for (const key of PERSISTED_SETTING_KEYS) {
    const value = settings[key];
    if (typeof value === 'string') {
      result[key] = value;
    }
  }
  return result;
};

/**
 * 兼容旧格式：旧版按分类存 `{ui:{...}, sync:{...}, notification:{...}}`，
 * 新版是扁平 `{key: value}`。读取时把旧结构摊平，再按白名单过滤。
 */
const normalize = (raw: unknown): Record<string, string> => {
  if (!raw || typeof raw !== 'object') return {};
  const flat: Record<string, string> = {};
  for (const value of Object.values(raw as Record<string, unknown>)) {
    if (value && typeof value === 'object') {
      for (const [innerKey, innerValue] of Object.entries(value as Record<string, unknown>)) {
        if (typeof innerValue === 'string') flat[innerKey] = innerValue;
      }
    }
  }
  // 新版扁平结构：直接摊平顶层字符串值
  for (const [key, value] of Object.entries(raw as Record<string, unknown>)) {
    if (typeof value === 'string') flat[key] = value;
  }
  return pickPersisted(flat);
};

/**
 * 保存设置到缓存（仅白名单 key 落盘）
 */
export function saveSettingsCache(settings: Record<string, string>): void {
  try {
    localStorage.setItem(SETTINGS_CACHE_KEY, JSON.stringify(pickPersisted(settings)));
    localStorage.setItem(SETTINGS_CACHE_TIMESTAMP_KEY, Date.now().toString());
  } catch (error) {
    console.error('保存设置缓存失败:', error);
  }
}

/**
 * 从缓存读取设置（已按白名单过滤；无有效内容时返回 null）
 */
export function loadSettingsCache(): Record<string, string> | null {
  try {
    const cached = localStorage.getItem(SETTINGS_CACHE_KEY);
    const timestamp = localStorage.getItem(SETTINGS_CACHE_TIMESTAMP_KEY);

    if (!cached || !timestamp) {
      return null;
    }

    const cacheAge = Date.now() - parseInt(timestamp, 10);
    if (Number.isNaN(cacheAge) || cacheAge > SETTINGS_CACHE_TTL) {
      clearSettingsCache();
      return null;
    }

    const normalized = normalize(JSON.parse(cached));
    return Object.keys(normalized).length > 0 ? normalized : null;
  } catch (error) {
    console.error('读取设置缓存失败:', error);
    return null;
  }
}

/**
 * 清除设置缓存
 */
export function clearSettingsCache(): void {
  try {
    localStorage.removeItem(SETTINGS_CACHE_KEY);
    localStorage.removeItem(SETTINGS_CACHE_TIMESTAMP_KEY);
  } catch (error) {
    console.error('清除设置缓存失败:', error);
  }
}

/**
 * 获取单个设置项（非白名单 key 直接返回 null，不读盘）
 */
export function getCachedSetting(key: string): string | null {
  if (!isPersistedSettingKey(key)) {
    return null;
  }
  const cache = loadSettingsCache();
  return cache?.[key] ?? null;
}

/**
 * 更新设置缓存（非白名单 key 被丢弃）
 */
export function updateCachedSettings(settings: Record<string, string>): void {
  const cache = loadSettingsCache() || {};
  saveSettingsCache({ ...cache, ...settings });
}
