/**
 * Sidebar 组件 - 侧边栏容器
 * 根据当前路由自动切换显示 MailMenu 或 AdminMenu
 */
import { useLocation } from 'react-router-dom';
import { useUIStore } from '../../stores/uiStore';
import { getViewMode } from '../../utils/routeUtils';
import { MailMenu } from './MailMenu';
import { AdminMenu } from './AdminMenu';

export const Sidebar = () => {
  const location = useLocation();
  const { sidebarCollapsed } = useUIStore();
  
  // 根据当前路由判断视图模式
  const viewMode = getViewMode(location.pathname);

  // 侧边栏折叠时不显示
  if (sidebarCollapsed) {
    return null;
  }

  // 根据视图模式渲染对应菜单
  if (viewMode === 'settings') {
    return <AdminMenu />;
  }

  return <MailMenu />;
};
