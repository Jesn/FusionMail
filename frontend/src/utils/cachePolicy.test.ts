import { describe, expect, it } from 'vitest';
import {
  ACCOUNT_CACHE_TTL,
  GROUP_CACHE_TTL,
  EMAIL_LIST_CACHE_TTL,
  EMAIL_DETAIL_CACHE_TTL,
  SEARCH_CACHE_TTL,
  SETTINGS_CACHE_TTL,
  PERSISTED_SETTING_KEYS,
  isPersistedSettingKey,
  isCacheExpired,
} from './cachePolicy';

describe('cachePolicy', () => {
  describe('isCacheExpired', () => {
    it('无时间戳视为过期', () => {
      expect(isCacheExpired(null, ACCOUNT_CACHE_TTL)).toBe(true);
      expect(isCacheExpired(undefined, ACCOUNT_CACHE_TTL)).toBe(true);
      expect(isCacheExpired(0, ACCOUNT_CACHE_TTL)).toBe(true);
    });

    it('TTL 内未过期', () => {
      expect(isCacheExpired(Date.now() - 1000, ACCOUNT_CACHE_TTL)).toBe(false);
    });

    it('超过 TTL 视为过期', () => {
      expect(isCacheExpired(Date.now() - ACCOUNT_CACHE_TTL - 1, ACCOUNT_CACHE_TTL)).toBe(true);
    });
  });

  describe('持久化白名单', () => {
    it('只放行显式登记的 key', () => {
      expect(isPersistedSettingKey('email_page_size')).toBe(true);
    });

    it('拒绝未登记的 key（含敏感项）', () => {
      expect(isPersistedSettingKey('smtp_password')).toBe(false);
      expect(isPersistedSettingKey('api_secret')).toBe(false);
      expect(isPersistedSettingKey('language')).toBe(false);
      expect(isPersistedSettingKey('')).toBe(false);
    });

    it('白名单非空且每一项都自洽', () => {
      expect(PERSISTED_SETTING_KEYS.length).toBeGreaterThan(0);
      for (const key of PERSISTED_SETTING_KEYS) {
        expect(isPersistedSettingKey(key)).toBe(true);
      }
    });
  });

  describe('TTL 取值', () => {
    it('均为正数毫秒', () => {
      for (const ttl of [
        ACCOUNT_CACHE_TTL,
        GROUP_CACHE_TTL,
        EMAIL_LIST_CACHE_TTL,
        EMAIL_DETAIL_CACHE_TTL,
        SEARCH_CACHE_TTL,
        SETTINGS_CACHE_TTL,
      ]) {
        expect(ttl).toBeGreaterThan(0);
      }
    });

    it('保持 P20 前的既有数值（不改变页面可见行为）', () => {
      expect(ACCOUNT_CACHE_TTL).toBe(10 * 60 * 1000);
      expect(GROUP_CACHE_TTL).toBe(5 * 60 * 1000);
      expect(EMAIL_LIST_CACHE_TTL).toBe(5 * 60 * 1000);
      expect(EMAIL_DETAIL_CACHE_TTL).toBe(10 * 60 * 1000);
      expect(SEARCH_CACHE_TTL).toBe(2 * 60 * 1000);
      expect(SETTINGS_CACHE_TTL).toBe(30 * 60 * 1000);
    });
  });
});
