import { beforeEach, describe, expect, it } from 'vitest';
import { useAccountStore } from './accountStore';
import type { Account, AccountStats } from '../types';

const account = (
  uid: string,
  email = `${uid}@example.com`,
  group_id: number | null = null
): Account => ({
  id: 1,
  uid,
  email,
  provider: 'test',
  protocol: 'imap',
  auth_type: 'password',
  sync_enabled: true,
  sync_interval: 5,
  status: 'active',
  created_at: '',
  updated_at: '',
  group_id,
  consecutive_auth_failures: 0,
});

const stats = (uid: string): AccountStats => ({
  account_uid: uid,
  total_count: 10,
  unread_count: 2,
  starred_count: 1,
});

/** 一个明确「很旧」的时间戳，便于断言它既没被推进也没被置空 */
const OLD_TS = Date.now() - 60_000;

describe('accountStore 缓存时钟语义', () => {
  beforeEach(() => {
    useAccountStore.getState().reset();
  });

  it('setAccounts（全量服务端数据）推进缓存时钟并按邮箱排序', () => {
    useAccountStore.setState({ cacheTimestamp: OLD_TS });
    useAccountStore.getState().setAccounts([account('b'), account('a')]);

    const state = useAccountStore.getState();
    expect(state.cacheTimestamp).toBeGreaterThan(OLD_TS);
    expect(state.hasLoaded).toBe(true);
    expect(state.accounts.map((a) => a.uid)).toEqual(['a', 'b']);
  });

  it('updateAccount 不推进缓存时钟（回归：mutation 曾给 TTL 续命）', () => {
    useAccountStore.setState({ accounts: [account('a')], cacheTimestamp: OLD_TS });

    useAccountStore.getState().updateAccount('a', { email: 'renamed@example.com' });

    expect(useAccountStore.getState().cacheTimestamp).toBe(OLD_TS);
    expect(useAccountStore.getState().accounts[0].email).toBe('renamed@example.com');
  });

  it('updateAccount 同步更新 selectedAccount', () => {
    const target = account('a');
    useAccountStore.setState({ accounts: [target], selectedAccount: target, cacheTimestamp: OLD_TS });

    useAccountStore.getState().updateAccount('a', { email: 'renamed@example.com' });

    expect(useAccountStore.getState().selectedAccount?.email).toBe('renamed@example.com');
  });

  it('addAccount 不推进缓存时钟', () => {
    useAccountStore.setState({ accounts: [], cacheTimestamp: OLD_TS });

    useAccountStore.getState().addAccount(account('a'));

    expect(useAccountStore.getState().cacheTimestamp).toBe(OLD_TS);
  });

  it('removeAccount 作废缓存并丢弃该账户的统计缓存', () => {
    useAccountStore.setState({
      accounts: [account('a'), account('b')],
      selectedAccount: account('a'),
      accountStats: { a: stats('a'), b: stats('b') },
      cacheTimestamp: OLD_TS,
    });

    useAccountStore.getState().removeAccount('a');

    const state = useAccountStore.getState();
    expect(state.cacheTimestamp).toBeNull();
    expect(state.accounts.map((a) => a.uid)).toEqual(['b']);
    expect(state.selectedAccount).toBeNull();
    expect(state.accountStats).toEqual({ b: stats('b') });
  });

  it('removeAccount 删除非选中账户时保留选中项', () => {
    useAccountStore.setState({
      accounts: [account('a'), account('b')],
      selectedAccount: account('b'),
      cacheTimestamp: OLD_TS,
    });

    useAccountStore.getState().removeAccount('a');

    expect(useAccountStore.getState().selectedAccount?.uid).toBe('b');
  });

  it('invalidate 置空缓存时间戳', () => {
    useAccountStore.setState({ cacheTimestamp: Date.now() });
    useAccountStore.getState().invalidate();
    expect(useAccountStore.getState().cacheTimestamp).toBeNull();
  });

  it('reset 回到初始状态', () => {
    useAccountStore.getState().setAccounts([account('a')]);
    useAccountStore.getState().reset();

    const state = useAccountStore.getState();
    expect(state.accounts).toEqual([]);
    expect(state.cacheTimestamp).toBeNull();
    expect(state.hasLoaded).toBe(false);
  });
});
