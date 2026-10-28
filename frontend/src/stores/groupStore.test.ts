import { beforeEach, describe, expect, it, vi } from 'vitest';

vi.mock('../services/groupService', () => ({
  groupService: {
    getGroups: vi.fn(),
    createGroup: vi.fn(),
    updateGroup: vi.fn(),
    deleteGroup: vi.fn(),
    reorderGroups: vi.fn(),
  },
}));

import { useGroupStore } from './groupStore';
import { groupService } from '../services/groupService';
import type { AccountGroup, AccountGroupWithCount } from '../types';

const group = (id: number, name = `g${id}`): AccountGroupWithCount => ({
  id,
  name,
  description: '',
  display_order: id,
  created_at: '',
  updated_at: '',
  account_count: 0,
  unread_count: 0,
});

const bareGroup = (id: number, name = `g${id}`): AccountGroup => ({
  id,
  name,
  description: '',
  display_order: id,
  created_at: '',
  updated_at: '',
});

/** 一个明确「很旧」的时间戳，便于断言它既没被推进也没被置空 */
const OLD_TS = Date.now() - 60_000;

describe('groupStore 缓存时钟语义', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    useGroupStore.getState().reset();
  });

  it('setGroups（全量服务端数据）推进缓存时钟', () => {
    useGroupStore.setState({ cacheTimestamp: OLD_TS });
    useGroupStore.getState().setGroups([group(1)]);

    expect(useGroupStore.getState().cacheTimestamp).toBeGreaterThan(OLD_TS);
  });

  it('editGroup 不推进缓存时钟（回归：mutation 曾给 TTL 续命）', async () => {
    vi.mocked(groupService.updateGroup).mockResolvedValue(bareGroup(1, 'renamed'));
    useGroupStore.setState({ groups: [group(1)], cacheTimestamp: OLD_TS });

    await useGroupStore.getState().editGroup(1, 'renamed');

    // 既没被推进（否则等于续命 TTL），也没被置空（改名不改变聚合计数）
    expect(useGroupStore.getState().cacheTimestamp).toBe(OLD_TS);
    expect(useGroupStore.getState().groups[0].name).toBe('renamed');
  });

  it('createGroup 作废缓存（聚合计数由服务端算，本地推不出来）', async () => {
    vi.mocked(groupService.createGroup).mockResolvedValue(bareGroup(9, 'new'));
    useGroupStore.setState({ groups: [], cacheTimestamp: OLD_TS });

    await useGroupStore.getState().createGroup('new');

    expect(useGroupStore.getState().cacheTimestamp).toBeNull();
    expect(useGroupStore.getState().groups).toHaveLength(1);
  });

  it('deleteGroup 作废缓存', async () => {
    vi.mocked(groupService.deleteGroup).mockResolvedValue(undefined);
    useGroupStore.setState({ groups: [group(1), group(2)], cacheTimestamp: OLD_TS });

    await useGroupStore.getState().deleteGroup(1);

    expect(useGroupStore.getState().cacheTimestamp).toBeNull();
    expect(useGroupStore.getState().groups.map((g) => g.id)).toEqual([2]);
  });

  it('reorderGroups 不推进缓存时钟（只改顺序，不动聚合计数）', async () => {
    vi.mocked(groupService.reorderGroups).mockResolvedValue(undefined);
    useGroupStore.setState({ groups: [group(1), group(2)], cacheTimestamp: OLD_TS });

    await useGroupStore.getState().reorderGroups([2, 1]);

    expect(useGroupStore.getState().cacheTimestamp).toBe(OLD_TS);
  });

  it('reorderGroups 失败时回滚顺序且不动缓存时钟', async () => {
    vi.mocked(groupService.reorderGroups).mockRejectedValue(new Error('boom'));
    useGroupStore.setState({ groups: [group(1), group(2)], cacheTimestamp: OLD_TS });

    await expect(useGroupStore.getState().reorderGroups([2, 1])).rejects.toThrow('boom');

    expect(useGroupStore.getState().groups.map((g) => g.id)).toEqual([1, 2]);
    expect(useGroupStore.getState().cacheTimestamp).toBe(OLD_TS);
  });

  it('fetchGroups 在缓存未过期时跳过请求', async () => {
    useGroupStore.setState({ hasLoaded: true, cacheTimestamp: Date.now() });

    await useGroupStore.getState().fetchGroups();

    expect(groupService.getGroups).not.toHaveBeenCalled();
  });

  it('fetchGroups 在缓存被作废后重新请求', async () => {
    vi.mocked(groupService.getGroups).mockResolvedValue({
      groups: [group(1)],
      total_count: 1,
      ungrouped_count: 0,
      total_unread_count: 0,
      ungrouped_unread_count: 0,
    });
    useGroupStore.setState({ hasLoaded: true, cacheTimestamp: null });

    await useGroupStore.getState().fetchGroups();

    expect(groupService.getGroups).toHaveBeenCalledTimes(1);
    expect(useGroupStore.getState().cacheTimestamp).toBeGreaterThan(OLD_TS);
  });
});
