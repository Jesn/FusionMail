/**
 * 设置缓存工具
 * 用于在 localStorage 中缓存用户设置
 */

const SETTINGS_CACHE_KEY = 'user_settings_cache';
const SETTINGS_CACHE_TIMESTAMP_KEY = 'user_settings_cache_timestamp';
const CACHE_DURATION = 30 * 60 * 1000; // 30分钟

export interface SettingsCache {
  ui?: Record<string, string>;
  sync?: Record<string, string>;
  notification?: Record<string, string>;
}

/**
 * 保存设置到缓存
 */
export function saveSettingsCache(settings: SettingsCache): void {
  try {
    localStorage.setItem(SETTINGS_CACHE_KEY, JSON.stringify(settings));
    localStorage.setItem(SETTINGS_CACHE_TIMESTAMP_KEY, Date.now().toString());
    console.log('设置已缓存到 localStorage');
  } catch (error) {
    console.error('保存设置缓存失败:', error);
  }
}

/**
 * 从缓存读取设置
 */
export function loadSettingsCache(): SettingsCache | null {
  try {
    const cached = localStorage.getItem(SETTINGS_CACHE_KEY);
    const timestamp = localStorage.getItem(SETTINGS_CACHE_TIMESTAMP_KEY);

    if (!cached || !timestamp) {
      return null;
    }

    // 检查缓存是否过期
    const cacheAge = Date.now() - parseInt(timestamp, 10);
    if (cacheAge > CACHE_DURATION) {
      console.log('设置缓存已过期');
      clearSettingsCache();
      return null;
    }

    return JSON.parse(cached);
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
    console.log('设置缓存已清除');
  } catch (error) {
    console.error('清除设置缓存失败:', error);
  }
}

/**
 * 获取特定分类的设置
 */
export function getCachedSettings(category: 'ui' | 'sync' | 'notification'): Record<string, string> | null {
  const cache = loadSettingsCache();
  return cache?.[category] || null;
}

/**
 * 更新特定分类的设置缓存
 */
export function updateCachedSettings(
  category: 'ui' | 'sync' | 'notification',
  settings: Record<string, string>
): void {
  const cache = loadSettingsCache() || {};
  cache[category] = settings;
  saveSettingsCache(cache);
}
