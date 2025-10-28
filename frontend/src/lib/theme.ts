// Theme management for Tailwind v4 + shadcn/ui.
// Fixed to light mode - ensures dark class is never applied.

export function initTheme() {
  // 固定为浅色模式,确保移除 dark class
  document.documentElement.classList.remove('dark')
}