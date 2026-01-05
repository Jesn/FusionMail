/**
 * 路由工具函数
 * 用于判断当前路由类型和视图模式
 */

// 视图模式类型
export type ViewMode = 'mail' | 'settings';

// 设置相关路由列表
export const SETTINGS_ROUTES = [
  '/settings',
  '/settings/system',
  '/accounts',
  '/trash',
  '/rules',
  '/webhooks',
  '/api-keys',
  '/providers',
  '/oauth2-clients',
  '/email-list',
  '/logs',
  '/api-docs',
];

// 邮件相关路由列表（用于参考，实际判断使用排除法）
export const MAIL_ROUTES = [
  '/inbox',
  '/sent',
  '/spam',
  '/search',
  '/email',
];

/**
 * 判断当前路由是否为设置视图路由
 * @param pathname 当前路由路径
 * @returns 是否为设置路由
 */
export function isSettingsRoute(pathname: string): boolean {
  return SETTINGS_ROUTES.some(route => 
    pathname === route || pathname.startsWith(route + '/')
  );
}

/**
 * 获取当前视图模式
 * @param pathname 当前路由路径
 * @returns 视图模式：'mail' 或 'settings'
 */
export function getViewMode(pathname: string): ViewMode {
  return isSettingsRoute(pathname) ? 'settings' : 'mail';
}
