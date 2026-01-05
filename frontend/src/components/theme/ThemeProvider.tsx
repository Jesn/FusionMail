/**
 * 主题提供者组件
 * 管理颜色主题和明暗模式
 */
import { createContext, useContext, useEffect, useState, useCallback, type ReactNode } from 'react';
import {
  type Theme,
  getThemeByName,
  applyTheme,
  getSystemTheme,
  defaultTheme,
  themes,
  THEME_STORAGE_KEY,
  MODE_STORAGE_KEY,
} from '@/lib/themes';

type ThemeMode = 'light' | 'dark' | 'system';

interface ThemeContextValue {
  // 当前颜色主题
  theme: Theme;
  // 当前模式（light/dark/system）
  mode: ThemeMode;
  // 实际应用的模式（light/dark）
  resolvedMode: 'light' | 'dark';
  // 所有可用主题
  availableThemes: Theme[];
  // 设置颜色主题
  setTheme: (themeName: string) => void;
  // 设置明暗模式
  setMode: (mode: ThemeMode) => void;
}

const ThemeContext = createContext<ThemeContextValue | undefined>(undefined);

interface ThemeProviderProps {
  children: ReactNode;
  defaultThemeName?: string;
  defaultMode?: ThemeMode;
}

export function ThemeProvider({
  children,
  defaultThemeName = 'default',
  defaultMode = 'system',
}: ThemeProviderProps) {
  // 从 localStorage 读取初始值
  const [themeName, setThemeName] = useState<string>(() => {
    if (typeof window !== 'undefined') {
      return localStorage.getItem(THEME_STORAGE_KEY) || defaultThemeName;
    }
    return defaultThemeName;
  });

  const [mode, setModeState] = useState<ThemeMode>(() => {
    if (typeof window !== 'undefined') {
      return (localStorage.getItem(MODE_STORAGE_KEY) as ThemeMode) || defaultMode;
    }
    return defaultMode;
  });

  const [resolvedMode, setResolvedMode] = useState<'light' | 'dark'>(() => {
    if (mode === 'system') {
      return getSystemTheme();
    }
    return mode;
  });

  const theme = getThemeByName(themeName);

  // 应用主题
  const applyCurrentTheme = useCallback(() => {
    const actualMode = mode === 'system' ? getSystemTheme() : mode;
    setResolvedMode(actualMode);

    // 应用 CSS 变量
    applyTheme(theme, actualMode);

    // 设置 dark 类
    if (actualMode === 'dark') {
      document.documentElement.classList.add('dark');
    } else {
      document.documentElement.classList.remove('dark');
    }
  }, [theme, mode]);

  // 监听系统主题变化
  useEffect(() => {
    const mediaQuery = window.matchMedia('(prefers-color-scheme: dark)');

    const handleChange = () => {
      if (mode === 'system') {
        applyCurrentTheme();
      }
    };

    mediaQuery.addEventListener('change', handleChange);
    return () => mediaQuery.removeEventListener('change', handleChange);
  }, [mode, applyCurrentTheme]);

  // 主题或模式变化时应用
  useEffect(() => {
    applyCurrentTheme();
  }, [applyCurrentTheme]);

  // 设置颜色主题
  const setTheme = useCallback((name: string) => {
    setThemeName(name);
    localStorage.setItem(THEME_STORAGE_KEY, name);
  }, []);

  // 设置明暗模式
  const setMode = useCallback((newMode: ThemeMode) => {
    setModeState(newMode);
    localStorage.setItem(MODE_STORAGE_KEY, newMode);
  }, []);

  const value: ThemeContextValue = {
    theme,
    mode,
    resolvedMode,
    availableThemes: themes,
    setTheme,
    setMode,
  };

  return <ThemeContext.Provider value={value}>{children}</ThemeContext.Provider>;
}

// Hook 用于访问主题上下文
export function useTheme() {
  const context = useContext(ThemeContext);
  if (context === undefined) {
    throw new Error('useTheme must be used within a ThemeProvider');
  }
  return context;
}

export { defaultTheme, themes };
