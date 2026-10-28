import { beforeEach, describe, expect, it, vi } from 'vitest';
import { saveSettingsCache } from '../utils/settingsCache';
import type { Account } from '../types';

/**
 * 模拟「刷新页面」：vi.resetModules() + 重新 import，等价于浏览器重新加载
 * 整个 JS bundle，模块级 initialState 会重新求值。
 *
 * 这覆盖 P20 持久化决策里可自动化的部分：
 *   - 业务数据（账户/分组）不跨刷新
 *   - 设置白名单（email_page_size）跨刷新，且敏感项不进入运行时
 */

const CACHE_KEY = 'user_settings_cache';
const TIMESTAMP_KEY = 'user_settings_cache_timestamp';

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

const account = (uid: string): Account => ({
  id: 1,
  uid,
  email: `${uid}@example.com`,
  provider: 'test',
  protocol: 'imap',
  auth_type: 'password',
  sync_enabled: true,
  sync_interval: 5,
  status: 'active',
  created_at: '',
  updated_at: '',
  group_id: null,
  consecutive_auth_failures: 0,
});

describe('刷新（模块重载）后的缓存一致性', () => {
  beforeEach(() => {
    installLocalStorage();
    vi.resetModules();
  });

  it('账户数据不跨刷新：新模块实例为空且缓存时间戳为 null', async () => {
    const first = await import('./accountStore');
    first.useAccountStore.getState().setAccounts([account('a')]);
    expect(first.useAccountStore.getState().accounts).toHaveLength(1);

    vi.resetModules();
    const refreshed = await import('./accountStore');

    expect(refreshed.useAccountStore.getState().accounts).toEqual([]);
    expect(refreshed.useAccountStore.getState().cacheTimestamp).toBeNull();
    expect(refreshed.useAccountStore.getState().hasLoaded).toBe(false);
  });

  it('分组数据不跨刷新', async () => {
    const first = await import('./groupStore');
    first.useGroupStore.getState().setGroups([
      {
        id: 1,
        name: 'g1',
        description: '',
        display_order: 1,
        created_at: '',
        updated_at: '',
        account_count: 3,
        unread_count: 1,
      },
    ]);
    expect(first.useGroupStore.getState().groups).toHaveLength(1);

    vi.resetModules();
    const refreshed = await import('./groupStore');

    expect(refreshed.useGroupStore.getState().groups).toEqual([]);
    expect(refreshed.useGroupStore.getState().cacheTimestamp).toBeNull();
  });

  it('email_page_size 跨刷新：刷新后 pageSize 初值取自白名单设置', async () => {
    saveSettingsCache({ email_page_size: '50' });

    vi.resetModules();
    const { useEmailStore } = await import('./emailStore');

    expect(useEmailStore.getState().pageSize).toBe(50);
  });

  it('无设置缓存时 pageSize 回落到默认 20', async () => {
    vi.resetModules();
    const { useEmailStore } = await import('./emailStore');

    expect(useEmailStore.getState().pageSize).toBe(20);
  });

  it('localStorage 中的非白名单（敏感）设置不会进入运行时状态', async () => {
    // 模拟旧版本或旁路代码留下的脏数据
    localStorage.setItem(
      CACHE_KEY,
      JSON.stringify({ email_page_size: '50', smtp_password: 'leaked-secret' })
    );
    localStorage.setItem(TIMESTAMP_KEY, Date.now().toString());

    vi.resetModules();
    const { useEmailStore } = await import('./emailStore');

    expect(useEmailStore.getState().pageSize).toBe(50);
    expect(JSON.stringify(useEmailStore.getState())).not.toContain('leaked-secret');
  });
});
