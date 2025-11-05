import { ReactNode, useEffect } from 'react';
import { Header } from './Header';
import { Sidebar } from './Sidebar';
import { useUIStore } from '../../stores/uiStore';
import { useAccounts } from '../../hooks/useAccounts';
import { cn } from '../../lib/utils';

interface MainLayoutProps {
  children: ReactNode;
}

export const MainLayout = ({ children }: MainLayoutProps) => {
  const { sidebarCollapsed } = useUIStore();
  const { loadAccounts } = useAccounts();

  // 全局加载账户列表（只加载一次）
  useEffect(() => {
    loadAccounts();
  }, []);

  return (
    <div className="flex h-screen overflow-hidden bg-background">
      {/* 侧边栏 */}
      <Sidebar />

      {/* 主内容区域 */}
      <div className="flex flex-1 flex-col overflow-hidden">
        {/* 头部 */}
        <Header />

        {/* 内容区域 */}
        <main
          className={cn(
            'flex-1 overflow-auto transition-all duration-300',
            sidebarCollapsed ? 'ml-0' : 'ml-0'
          )}
        >
          {children}
        </main>
      </div>
    </div>
  );
};
