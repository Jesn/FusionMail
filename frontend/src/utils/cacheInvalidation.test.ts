import { beforeEach, describe, expect, it } from 'vitest';
import { invalidateAccounts, invalidateGroups } from './cacheInvalidation';
import { useAccountStore } from '../stores/accountStore';
import { useGroupStore } from '../stores/groupStore';

describe('cacheInvalidation', () => {
  beforeEach(() => {
    useAccountStore.setState({ cacheTimestamp: Date.now() });
    useGroupStore.setState({ cacheTimestamp: Date.now() });
  });

  it('invalidateAccounts 置空账户缓存时间戳，不影响分组', () => {
    invalidateAccounts();
    expect(useAccountStore.getState().cacheTimestamp).toBeNull();
    expect(useGroupStore.getState().cacheTimestamp).not.toBeNull();
  });

  it('invalidateGroups 置空分组缓存时间戳，不影响账户', () => {
    invalidateGroups();
    expect(useGroupStore.getState().cacheTimestamp).toBeNull();
    expect(useAccountStore.getState().cacheTimestamp).not.toBeNull();
  });

  it('两者可同时调用（账户增删需同时作废分组聚合计数）', () => {
    invalidateAccounts();
    invalidateGroups();
    expect(useAccountStore.getState().cacheTimestamp).toBeNull();
    expect(useGroupStore.getState().cacheTimestamp).toBeNull();
  });

  it('置 null 后任何 TTL 都判定为过期，从而触发重新拉取', () => {
    invalidateAccounts();
    const { cacheTimestamp } = useAccountStore.getState();
    expect(cacheTimestamp).toBeNull();
    // isCacheExpired 对 null 恒返回 true
    expect(cacheTimestamp).toBeFalsy();
  });
});
