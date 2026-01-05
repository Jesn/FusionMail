import { Search, Settings, User, BookOpen, Mail, Key } from 'lucide-react';
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
import { useState } from 'react';
import { useNavigate, useLocation } from 'react-router-dom';
import { ChangePasswordDialog } from '@/components/settings/ChangePasswordDialog';
import { getViewMode } from '../../utils/routeUtils';
import { ThemeSwitcher } from '@/components/theme';

export const Header = () => {
  const { user, logout } = useAuthStore();
  const { searchQuery, setSearchQuery, unreadCount, filter, setFilter } = useEmailStore();
  const [localSearch, setLocalSearch] = useState(searchQuery);
  const hasUnread = unreadCount > 0;
  const navigate = useNavigate();
  const location = useLocation();
  
  // 获取当前视图模式
  const viewMode = getViewMode(location.pathname);
  const isMailView = viewMode === 'mail';

  const handleSearch = (e: React.FormEvent) => {
    e.preventDefault();
    setSearchQuery(localSearch);
  };

  const handleLogout = () => {
    logout();
    window.location.href = '/login';
  };

  const handleUnreadClick = () => {
    const newFilter: any = {};

    if (filter.account_uid) {
      newFilter.account_uid = filter.account_uid;
    }

    // 未读列表视图：只看未读且不展示已删除/归档
    newFilter.is_read = false;
    newFilter.is_archived = false;
    newFilter.is_deleted = false;

    setFilter(newFilter);
    navigate('/inbox');
  };

  return (
    <header className="flex h-16 items-center justify-between border-b bg-background px-6">
      <div className="flex flex-1 items-center gap-4">
        {/* 搜索框：仅在邮件视图显示 */}
        {isMailView && (
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
        )}
      </div>

      <div className="flex items-center gap-1">
        {/* API 文档按钮：仅在设置视图显示 */}
        {!isMailView && (
          <Button
            variant="ghost"
            size="icon"
            title="API 文档"
            onClick={() => navigate('/api-docs')}
          >
            <BookOpen className="h-5 w-5" />
          </Button>
        )}

        {/* 未读邮件徽章：仅在邮件视图且有未读邮件时显示 */}
        {isMailView && hasUnread && (
          <Button
            variant="ghost"
            size="icon"
            title={`未读邮件 (${unreadCount})`}
            onClick={handleUnreadClick}
            className="relative"
          >
            <Mail className="h-4 w-4" />
            <span className="absolute -top-1 -right-1 flex h-4 min-w-[16px] items-center justify-center rounded-full bg-red-500 px-1 text-[10px] font-medium text-white animate-pulse">
              {unreadCount > 99 ? '99+' : unreadCount}
            </span>
          </Button>
        )}

        {/* 主题切换按钮 */}
        <ThemeSwitcher />

        <Button
          variant="ghost"
          size="icon"
          title="设置"
          onClick={() => navigate('/accounts')}
        >
          <Settings className="h-5 w-5" />
        </Button>

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
            <DropdownMenuItem onClick={() => navigate('/settings')}>个人设置</DropdownMenuItem>
            <DropdownMenuSeparator />
            <ChangePasswordDialog
              trigger={
                <DropdownMenuItem onSelect={(e) => e.preventDefault()}>
                  <Key className="mr-2 h-4 w-4" />
                  修改密码
                </DropdownMenuItem>
              }
            />
            <DropdownMenuSeparator />
            <DropdownMenuItem onClick={handleLogout} className="text-red-600">
              退出登录
            </DropdownMenuItem>
          </DropdownMenuContent>
        </DropdownMenu>
      </div>
    </header>
  );
};
