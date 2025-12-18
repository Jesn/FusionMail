import { create } from 'zustand';
import { persist } from 'zustand/middleware';
import type { AccountGroupWithCount } from '../types';
import { groupService } from '../services/groupService';

// 缓存过期时间：5分钟
export const GROUP_CACHE_TTL = 5 * 60 * 1000;

// 辅助函数：检查缓存是否过期
export const isGroupCacheExpired = (cacheTimestamp: number | null): boolean => {
  if (!cacheTimestamp) return true;
  return Date.now() - cacheTimestamp > GROUP_CACHE_TTL;
};

// 特殊分组 ID 常量
export const ALL_ACCOUNTS_GROUP_ID = -1; // 所有账号
export const UNGROUPED_GROUP_ID = 0; // 未分组

interface GroupState {
  // 分组列表
  groups: AccountGroupWithCount[];

  // 统计信息（来自后端）
  totalCount: number;      // 所有账号总数
  ungroupedCount: number;  // 未分组账号数

  // 当前选中的分组 ID
  // -1: 所有账号, 0: 未分组, >0: 具体分组 ID
  selectedGroupId: number;

  // 加载状态
  isLoading: boolean;
  isFetching: boolean;
  hasLoaded: boolean;
  error: string | null;

  // 缓存时间戳
  cacheTimestamp: number | null;

  // Actions
  setGroups: (groups: AccountGroupWithCount[]) => void;
  addGroup: (group: AccountGroupWithCount) => void;
  updateGroup: (id: number, updates: Partial<AccountGroupWithCount>) => void;
  removeGroup: (id: number) => void;
  setSelectedGroupId: (id: number) => void;
  setLoading: (loading: boolean) => void;
  setFetching: (fetching: boolean) => void;
  setHasLoaded: (hasLoaded: boolean) => void;
  setError: (error: string | null) => void;
  setCacheTimestamp: (timestamp: number) => void;

  // 异步 Actions
  fetchGroups: () => Promise<void>;
  createGroup: (name: string, description?: string) => Promise<AccountGroupWithCount>;
  editGroup: (id: number, name: string, description?: string) => Promise<void>;
  deleteGroup: (id: number) => Promise<void>;
  reorderGroups: (groupIds: number[]) => Promise<void>;

  // 重置
  reset: () => void;
}

const initialState = {
  groups: [],
  totalCount: 0,
  ungroupedCount: 0,
  selectedGroupId: ALL_ACCOUNTS_GROUP_ID, // 默认选中"所有账号"
  isLoading: false,
  isFetching: false,
  hasLoaded: false,
  error: null,
  cacheTimestamp: null,
};

export const useGroupStore = create<GroupState>()(
  persist(
    (set, get) => ({
      ...initialState,

      setGroups: (groups) => set({
        groups: groups.sort((a, b) => a.display_order - b.display_order),
        hasLoaded: true,
        cacheTimestamp: Date.now(),
      }),

      addGroup: (group) => set((state) => ({
        groups: [...state.groups, group].sort((a, b) => a.display_order - b.display_order),
        cacheTimestamp: Date.now(),
      })),

      updateGroup: (id, updates) => set((state) => ({
        groups: state.groups.map((group) =>
          group.id === id ? { ...group, ...updates } : group
        ),
        cacheTimestamp: Date.now(),
      })),

      removeGroup: (id) => set((state) => ({
        groups: state.groups.filter((group) => group.id !== id),
        // 如果删除的是当前选中的分组，切换到"所有账号"
        selectedGroupId: state.selectedGroupId === id ? ALL_ACCOUNTS_GROUP_ID : state.selectedGroupId,
        cacheTimestamp: Date.now(),
      })),

      setSelectedGroupId: (id) => set({ selectedGroupId: id }),

      setLoading: (loading) => set({ isLoading: loading }),

      setFetching: (fetching) => set({ isFetching: fetching }),

      setHasLoaded: (hasLoaded) => set({ hasLoaded }),

      setError: (error) => set({ error }),

      setCacheTimestamp: (timestamp) => set({ cacheTimestamp: timestamp }),

      // 获取分组列表（带统计信息）
      fetchGroups: async () => {
        const state = get();

        // 如果正在请求中，跳过
        if (state.isFetching) return;

        // 如果缓存未过期，跳过
        if (state.hasLoaded && !isGroupCacheExpired(state.cacheTimestamp)) {
          return;
        }

        set({ isFetching: true, error: null });

        try {
          const response = await groupService.getGroups();
          // 兼容处理：确保 groups 存在且为数组
          const groups = response.groups || [];
          set({
            groups: groups.sort((a, b) => a.display_order - b.display_order),
            totalCount: response.total_count || 0,
            ungroupedCount: response.ungrouped_count || 0,
            hasLoaded: true,
            cacheTimestamp: Date.now(),
            isFetching: false,
          });
        } catch (error) {
          set({
            error: error instanceof Error ? error.message : '获取分组列表失败',
            isFetching: false,
          });
          throw error;
        }
      },

      // 创建分组
      createGroup: async (name, description) => {
        set({ isLoading: true, error: null });

        try {
          const newGroup = await groupService.createGroup({ name, description });
          const groupWithCount: AccountGroupWithCount = {
            ...newGroup,
            account_count: 0,
          };

          set((state) => ({
            groups: [...state.groups, groupWithCount].sort((a, b) => a.display_order - b.display_order),
            isLoading: false,
            cacheTimestamp: Date.now(),
          }));

          return groupWithCount;
        } catch (error) {
          set({
            error: error instanceof Error ? error.message : '创建分组失败',
            isLoading: false,
          });
          throw error;
        }
      },

      // 编辑分组
      editGroup: async (id, name, description) => {
        set({ isLoading: true, error: null });

        try {
          const updatedGroup = await groupService.updateGroup(id, { name, description });

          set((state) => ({
            groups: state.groups.map((group) =>
              group.id === id ? { ...group, ...updatedGroup } : group
            ),
            isLoading: false,
            cacheTimestamp: Date.now(),
          }));
        } catch (error) {
          set({
            error: error instanceof Error ? error.message : '更新分组失败',
            isLoading: false,
          });
          throw error;
        }
      },

      // 删除分组
      deleteGroup: async (id) => {
        set({ isLoading: true, error: null });

        try {
          await groupService.deleteGroup(id);

          set((state) => ({
            groups: state.groups.filter((group) => group.id !== id),
            selectedGroupId: state.selectedGroupId === id ? ALL_ACCOUNTS_GROUP_ID : state.selectedGroupId,
            isLoading: false,
            cacheTimestamp: Date.now(),
          }));
        } catch (error) {
          set({
            error: error instanceof Error ? error.message : '删除分组失败',
            isLoading: false,
          });
          throw error;
        }
      },

      // 重排序分组
      reorderGroups: async (groupIds) => {
        const state = get();
        const originalGroups = [...state.groups];

        // 乐观更新：先更新本地状态
        const reorderedGroups = groupIds.map((id, index) => {
          const group = state.groups.find((g) => g.id === id);
          return group ? { ...group, display_order: index } : null;
        }).filter((g): g is AccountGroupWithCount => g !== null);

        set({ groups: reorderedGroups });

        try {
          await groupService.reorderGroups(groupIds);
          set({ cacheTimestamp: Date.now() });
        } catch (error) {
          // 回滚
          set({ groups: originalGroups });
          set({
            error: error instanceof Error ? error.message : '重排序分组失败',
          });
          throw error;
        }
      },

      reset: () => set(initialState),
    }),
    {
      name: 'fusionmail-groups',
      version: 1,
      partialize: (state) => ({
        groups: state.groups,
        selectedGroupId: state.selectedGroupId,
        hasLoaded: state.hasLoaded,
        cacheTimestamp: state.cacheTimestamp,
      }),
    }
  )
);
