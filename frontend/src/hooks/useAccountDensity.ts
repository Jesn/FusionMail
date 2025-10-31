import { useState, useEffect } from 'react';
import type { AccountDensity } from '../components/account/AccountToolbar';

interface UseAccountDensityOptions {
  accountCount: number;
  storageKey?: string;
}

export const useAccountDensity = ({ 
  accountCount, 
  storageKey = 'fusionmail_account_density' 
}: UseAccountDensityOptions) => {
  // 智能推荐密度
  const getRecommendedDensity = (count: number): AccountDensity => {
    if (count <= 20) return 'detailed';
    if (count <= 50) return 'compact';
    return 'minimal';
  };

  // 从本地存储读取用户偏好，如果没有则使用智能推荐
  const getInitialDensity = (): AccountDensity => {
    try {
      const stored = localStorage.getItem(storageKey);
      if (stored && ['detailed', 'compact', 'minimal'].includes(stored)) {
        return stored as AccountDensity;
      }
    } catch (error) {
      console.warn('Failed to read density preference from localStorage:', error);
    }
    return getRecommendedDensity(accountCount);
  };

  const [density, setDensityState] = useState<AccountDensity>(getInitialDensity);
  const [isAutoMode, setIsAutoMode] = useState(true);

  // 保存密度偏好到本地存储
  const setDensity = (newDensity: AccountDensity) => {
    setDensityState(newDensity);
    setIsAutoMode(false);
    try {
      localStorage.setItem(storageKey, newDensity);
      localStorage.setItem(`${storageKey}_auto`, 'false');
    } catch (error) {
      console.warn('Failed to save density preference to localStorage:', error);
    }
  };

  // 启用自动模式
  const enableAutoMode = () => {
    const recommendedDensity = getRecommendedDensity(accountCount);
    setDensityState(recommendedDensity);
    setIsAutoMode(true);
    try {
      localStorage.setItem(storageKey, recommendedDensity);
      localStorage.setItem(`${storageKey}_auto`, 'true');
    } catch (error) {
      console.warn('Failed to save auto mode preference to localStorage:', error);
    }
  };

  // 当账户数量变化时，如果是自动模式，则更新密度
  useEffect(() => {
    if (isAutoMode) {
      const recommendedDensity = getRecommendedDensity(accountCount);
      if (recommendedDensity !== density) {
        setDensityState(recommendedDensity);
        try {
          localStorage.setItem(storageKey, recommendedDensity);
        } catch (error) {
          console.warn('Failed to update density in localStorage:', error);
        }
      }
    }
  }, [accountCount, isAutoMode, density, storageKey]);

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
    density,
    setDensity,
    isAutoMode,
    enableAutoMode,
    recommendedDensity: getRecommendedDensity(accountCount),
  };
};