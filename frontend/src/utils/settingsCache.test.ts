import { beforeEach, describe, expect, it, vi } from 'vitest';
import {
  saveSettingsCache,
  loadSettingsCache,
  getCachedSetting,
  updateCachedSettings,
  clearSettingsCache,
} from './settingsCache';
import { SETTINGS_CACHE_TTL } from './cachePolicy';

const CACHE_KEY = 'user_settings_cache';
const TIMESTAMP_KEY = 'user_settings_cache_timestamp';

/** 可用的内存版 localStorage（全局 mock 恒返回 null，无法用于读写断言） */
const installLocalStorage = () => {
  const store = new Map<string, string>();
  Object.defineProperty(window, 'localStorage', {
    configurable: true,
    value: {
      getItem: (key: string) => store.get(key) ?? null,
      setItem: (key: string, value: string) => void store.set(key, value),
      removeItem: (key: string) => void store.delete(key),
      clear: () => store.clear(),
      key: (index: number) => Array.from(store.keys())[index] ?? null,
      get length() {
        return store.size;
      },
    },
  });
  return store;
};

describe('settingsCache', () => {
  let store: Map<string, string>;

  beforeEach(() => {
    store = installLocalStorage();
    vi.restoreAllMocks();
  });

  it('只持久化白名单内的 key，敏感项被丢弃', () => {
    saveSettingsCache({
      email_page_size: '50',
      smtp_password: 'super-secret',
      api_secret: 'another-secret',
      language: 'zh-CN',
    });

    const raw = store.get(CACHE_KEY);
    expect(raw).toBeDefined();
    expect(JSON.parse(raw!)).toEqual({ email_page_size: '50' });
    expect(raw).not.toContain('super-secret');
    expect(raw).not.toContain('another-secret');
  });

  it('读回白名单值', () => {
    saveSettingsCache({ email_page_size: '50' });
    expect(loadSettingsCache()).toEqual({ email_page_size: '50' });
  });

  it('只写了非白名单 key 时，视为无缓存', () => {
    saveSettingsCache({ smtp_password: 'secret' });
    expect(loadSettingsCache()).toBeNull();
  });

  it('getCachedSetting 对非白名单 key 直接返回 null 且不读盘', () => {
    saveSettingsCache({ email_page_size: '50' });
    expect(getCachedSetting('email_page_size')).toBe('50');
    expect(getCachedSetting('smtp_password')).toBeNull();
    expect(getCachedSetting('language')).toBeNull();
  });

  it('兼容旧的分分类格式并按白名单过滤', () => {
    store.set(
      CACHE_KEY,
      JSON.stringify({
        ui: { email_page_size: '30', language: 'zh-CN' },
        sync: { sync_interval: '5' },
        notification: { smtp_password: 'legacy-secret' },
      })
    );
    store.set(TIMESTAMP_KEY, Date.now().toString());

    expect(loadSettingsCache()).toEqual({ email_page_size: '30' });
  });

  it('超过 TTL 视为过期并清除', () => {
    saveSettingsCache({ email_page_size: '50' });
    store.set(TIMESTAMP_KEY, (Date.now() - SETTINGS_CACHE_TTL - 1).toString());

    expect(loadSettingsCache()).toBeNull();
    expect(store.has(CACHE_KEY)).toBe(false);
    expect(store.has(TIMESTAMP_KEY)).toBe(false);
  });

  it('时间戳非法时视为过期', () => {
    saveSettingsCache({ email_page_size: '50' });
    store.set(TIMESTAMP_KEY, 'not-a-number');
    expect(loadSettingsCache()).toBeNull();
  });

  it('updateCachedSettings 合并写入且过滤非白名单 key', () => {
    updateCachedSettings({ email_page_size: '50' });
    updateCachedSettings({ smtp_password: 'secret', language: 'en' });

    expect(loadSettingsCache()).toEqual({ email_page_size: '50' });
    expect(store.get(CACHE_KEY)).not.toContain('secret');
  });

  it('clearSettingsCache 清除两个键', () => {
    saveSettingsCache({ email_page_size: '50' });
    clearSettingsCache();
    expect(store.has(CACHE_KEY)).toBe(false);
    expect(store.has(TIMESTAMP_KEY)).toBe(false);
  });

  it('损坏的 JSON 不抛异常', () => {
    const consoleError = vi.spyOn(console, 'error').mockImplementation(() => {});
    store.set(CACHE_KEY, '{not json');
    store.set(TIMESTAMP_KEY, Date.now().toString());
    expect(loadSettingsCache()).toBeNull();
    expect(consoleError).toHaveBeenCalled();
  });
});
