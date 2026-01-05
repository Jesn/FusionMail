/**
 * 主题 Hook
 * 提供主题状态和切换功能
 * 与 ThemeProvider 集成
 */
import { useTheme as useThemeContext } from '@/components/theme/ThemeProvider';

type ThemeMode = 'light' | 'dark' | 'system';

/**
 * 主题 Hook
 * 兼容旧版 API，同时支持新的颜色主题功能
 */
export const useTheme = () => {
  const context = useThemeContext();

  return {
    // 兼容旧版 API
    theme: context.mode,
    resolvedTheme: context.resolvedMode,
    setTheme: (mode: ThemeMode) => context.setMode(mode),
    
    // 新增：颜色主题相关
    colorTheme: context.theme,
    availableThemes: context.availableThemes,
    setColorTheme: context.setTheme,
  };
};

// 导出类型
export type { ThemeMode };
