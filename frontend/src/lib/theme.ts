// Theme management for Tailwind v4 + shadcn/ui.
// Initialize theme from localStorage or system preference

export function initTheme() {
  // 从本地存储获取保存的主题设置
  const savedTheme = localStorage.getItem('fusionmail_theme') as 'light' | 'dark' | 'system' | null;

  let actualTheme: 'light' | 'dark' = 'light';

  if (savedTheme === 'dark') {
    actualTheme = 'dark';
  } else if (savedTheme === 'system' || !savedTheme) {
    // 如果是系统主题或没有保存的主题，检查系统偏好
    actualTheme = window.matchMedia('(prefers-color-scheme: dark)').matches ? 'dark' : 'light';
  }

  // 应用主题到 DOM
  if (actualTheme === 'dark') {
    document.documentElement.classList.add('dark');
  } else {
    document.documentElement.classList.remove('dark');
  }
}