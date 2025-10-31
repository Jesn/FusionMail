import { Search, Settings, User, History } from 'lucide-react';
import { Button } from '../ui/button';
import { Input } from '../ui/input';
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuLabel,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from '../ui/dropdown-menu';
import { useAuthStore } from '../../stores/authStore';
import { useEmailStore } from '../../stores/emailStore';
import { SyncStatusIndicator } from '../sync/SyncStatusIndicator';
import { SyncLogsDialog } from '../sync/SyncLogsDialog';
import { useState } from 'react';
import { useLocation } from 'react-router-dom';

export const Header = () => {
  const { user, logout } = useAuthStore();
  const { searchQuery, setSearchQuery } = useEmailStore();
  const [localSearch, setLocalSearch] = useState(searchQuery);
  const [showSyncLogs, setShowSyncLogs] = useState(false);
  const location = useLocation();
  
  // 在 accounts 页面禁用自动刷新，避免页面刷新
  const isAccountsPage = location.pathname === '/accounts' || location.pathname.startsWith('/accounts/');

  const handleSearch = (e: React.FormEvent) => {
    e.preventDefault();
    setSearchQuery(localSearch);
  };

  const handleLogout = () => {
    logout();
    window.location.href = '/login';
  };

  return (
    <header className="flex h-16 items-center justify-between border-b bg-background px-6">
      {/* 左侧：搜索框 */}
      <div className="flex flex-1 items-center gap-4">
        <form onSubmit={handleSearch} className="flex w-full max-w-md items-center gap-2">
          <div className="relative flex-1">
            <Search className="absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-muted-foreground" />
            <Input
              type="search"
              placeholder="搜索邮件..."
              className="pl-9"
              value={localSearch}
              onChange={(e) => setLocalSearch(e.target.value)}
            />
          </div>
        </form>
      </div>

      {/* 右侧：操作按钮 */}
      <div className="flex items-center gap-4">
        {/* 同步状态指示器 */}
        <SyncStatusIndicator compact={false} showControls={true} autoRefresh={!isAccountsPage} />
        
        {/* 同步历史按钮 */}
        <Button
          variant="ghost"
          size="icon"
          onClick={() => setShowSyncLogs(true)}
          title="查看同步历史"
        >
          <History className="h-5 w-5" />
        </Button>

        {/* 设置按钮 */}
        <Button variant="ghost" size="icon" title="设置">
          <Settings className="h-5 w-5" />
        </Button>

        {/* 用户菜单 */}
        <DropdownMenu>
          <DropdownMenuTrigger asChild>
            <Button variant="ghost" size="icon">
              <User className="h-5 w-5" />
            </Button>
          </DropdownMenuTrigger>
          <DropdownMenuContent align="end" className="w-56">
            <DropdownMenuLabel>
              <div className="flex flex-col space-y-1">
                <p className="text-sm font-medium">{user?.name || '用户'}</p>
                <p className="text-xs text-muted-foreground">{user?.email}</p>
              </div>
            </DropdownMenuLabel>
            <DropdownMenuSeparator />
            <DropdownMenuItem onClick={() => window.location.href = '/accounts'}>
              账户管理
            </DropdownMenuItem>
            <DropdownMenuItem>个人设置</DropdownMenuItem>
            <DropdownMenuSeparator />
            <DropdownMenuItem onClick={handleLogout} className="text-red-600">
              退出登录
            </DropdownMenuItem>
          </DropdownMenuContent>
        </DropdownMenu>
      </div>

      {/* 同步日志对话框 */}
      <SyncLogsDialog
        open={showSyncLogs}
        onClose={() => setShowSyncLogs(false)}
      />
    </header>
  );
};
