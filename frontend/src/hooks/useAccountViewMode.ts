import { useState, useEffect } from 'react';
import type { ViewMode, GroupBy } from '../components/account/AccountToolbar';

interface UseAccountViewModeOptions {
  accountCount: number;
  storageKey?: string;
}

export const useAccountViewMode = ({ 
  accountCount, 
  storageKey = 'fusionmail_account_view_mode' 
}: UseAccountViewModeOptions) => {
  // 智能推荐视图模式
  const getRecommendedViewMode = (count: number): ViewMode => {
    if (count <= 15) return 'list';      // 少量账户用列表
    if (count <= 30) return 'virtual';   // 中等数量用虚拟滚动
    if (count <= 50) return 'groups';    // 较多账户用分组
    return 'table';                      // 大量账户用表格分页
  };

  // 从本地存储读取用户偏好
  const getInitialViewMode = (): ViewMode => {
    try {
      const stored = localStorage.getItem(storageKey);
      if (stored && ['list', 'virtual', 'groups', 'table'].includes(stored)) {
        return stored as ViewMode;
      }
    } catch (error) {
      console.warn('Failed to read view mode preference from localStorage:', error);
    }
    return getRecommendedViewMode(accountCount);
  };

  const getInitialGroupBy = (): GroupBy => {
    try {
      const stored = localStorage.getItem(`${storageKey}_group_by`);
      if (stored && ['provider', 'status', 'sync_status', 'none'].includes(stored)) {
        return stored as GroupBy;
      }
    } catch (error) {
      console.warn('Failed to read group by preference from localStorage:', error);
    }
    return 'provider';
  };

  const [viewMode, setViewModeState] = useState<ViewMode>(getInitialViewMode);
  const [groupBy, setGroupByState] = useState<GroupBy>(getInitialGroupBy);
  const [isAutoMode, setIsAutoMode] = useState(true);

  // 保存视图模式偏好
  const setViewMode = (newViewMode: ViewMode) => {
    setViewModeState(newViewMode);
    setIsAutoMode(false);
    try {
      localStorage.setItem(storageKey, newViewMode);
      localStorage.setItem(`${storageKey}_auto`, 'false');
    } catch (error) {
      console.warn('Failed to save view mode preference to localStorage:', error);
    }
  };

  // 保存分组方式偏好
  const setGroupBy = (newGroupBy: GroupBy) => {
    setGroupByState(newGroupBy);
    try {
      localStorage.setItem(`${storageKey}_group_by`, newGroupBy);
    } catch (error) {
      console.warn('Failed to save group by preference to localStorage:', error);
    }
  };

  // 启用自动模式
  const enableAutoMode = () => {
    const recommendedViewMode = getRecommendedViewMode(accountCount);
    setViewModeState(recommendedViewMode);
    setIsAutoMode(true);
    try {
      localStorage.setItem(storageKey, recommendedViewMode);
      localStorage.setItem(`${storageKey}_auto`, 'true');
    } catch (error) {
      console.warn('Failed to save auto mode preference to localStorage:', error);
    }
  };

  // 当账户数量变化时，如果是自动模式，则更新视图模式
  useEffect(() => {
    if (isAutoMode) {
      const recommendedViewMode = getRecommendedViewMode(accountCount);
      if (recommendedViewMode !== viewMode) {
        setViewModeState(recommendedViewMode);
        try {
          localStorage.setItem(storageKey, recommendedViewMode);
        } catch (error) {
          console.warn('Failed to update view mode in localStorage:', error);
        }
      }
    }
  }, [accountCount, isAutoMode, viewMode, storageKey]);

  // 初始化时检查是否是自动模式
  useEffect(() => {
    try {
      const autoMode = localStorage.getItem(`${storageKey}_auto`);
      if (autoMode === 'false') {
        setIsAutoMode(false);
      }
    } catch (error) {
      console.warn('Failed to read auto mode preference from localStorage:', error);
    }
  }, [storageKey]);

  return {
    viewMode,
    setViewMode,
    groupBy,
    setGroupBy,
    isAutoMode,
    enableAutoMode,
    recommendedViewMode: getRecommendedViewMode(accountCount),
  };
};