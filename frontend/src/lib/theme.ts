// Theme management for Tailwind v4 + shadcn/ui.
// 在 React 渲染前同步应用主题，避免深色模式刷新时闪白（FOUC）。
//
// 存储键与 ThemeProvider / lib/themes.ts 保持唯一来源：
//   - MODE_STORAGE_KEY  ('fusionmail_theme_mode')  明暗模式 light | dark | system
//   - THEME_STORAGE_KEY ('fusionmail_color_theme') 颜色主题名称

import {
  MODE_STORAGE_KEY,
  THEME_STORAGE_KEY,
  applyTheme,
  defaultTheme,
  getSystemTheme,
  getThemeByName,
} from './themes';

export function initTheme() {
  if (typeof window === 'undefined') {
    return;
  }

  // 读取 ThemeProvider 写入的明暗模式
  const savedMode = localStorage.getItem(MODE_STORAGE_KEY);

  let actualMode: 'light' | 'dark';
  if (savedMode === 'dark' || savedMode === 'light') {
    actualMode = savedMode;
  } else {
    // 'system' 或尚未设置时，跟随系统偏好
    actualMode = getSystemTheme();
  }

  // 同步应用颜色主题的 CSS 变量，避免首屏颜色跳变
  const themeName = localStorage.getItem(THEME_STORAGE_KEY) || defaultTheme.name;
  applyTheme(getThemeByName(themeName), actualMode);

  // 应用 dark 类，供 Tailwind 的 dark: 变体使用
  document.documentElement.classList.toggle('dark', actualMode === 'dark');
}
