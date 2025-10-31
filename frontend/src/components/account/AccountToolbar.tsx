import { Search, Grid3X3, List, MoreHorizontal, Layers } from 'lucide-react';
import { Input } from '../ui/input';
import { Button } from '../ui/button';
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '../ui/select';
import { Badge } from '../ui/badge';
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from '../ui/dropdown-menu';

export type AccountDensity = 'detailed' | 'compact' | 'minimal';
export type AccountStatus = 'all' | 'active' | 'disabled' | 'error';
export type AccountProvider = 'all' | 'gmail' | 'outlook' | 'imap' | 'pop3';
export type ViewMode = 'list' | 'virtual' | 'groups';
export type GroupBy = 'provider' | 'status' | 'sync_status' | 'none';

interface AccountToolbarProps {
  // 搜索和筛选
  searchQuery: string;
  onSearchChange: (query: string) => void;
  statusFilter: AccountStatus;
  onStatusFilterChange: (status: AccountStatus) => void;
  providerFilter: AccountProvider;
  onProviderFilterChange: (provider: AccountProvider) => void;
  
  // 视图控制
  density: AccountDensity;
  onDensityChange: (density: AccountDensity) => void;
  viewMode: ViewMode;
  onViewModeChange: (mode: ViewMode) => void;
  groupBy: GroupBy;
  onGroupByChange: (groupBy: GroupBy) => void;
  
  // 批量操作
  selectedCount: number;
  totalCount: number;
  onSelectAll: () => void;
  onClearSelection: () => void;
  onBatchSync: () => void;
  onBatchToggleStatus: () => void;
  
  // 统计信息
  activeCount: number;
  errorCount: number;
}

export const AccountToolbar = ({
  searchQuery,
  onSearchChange,
  statusFilter,
  onStatusFilterChange,
  providerFilter,
  onProviderFilterChange,
  density,
  onDensityChange,
  viewMode,
  onViewModeChange,
  groupBy,
  onGroupByChange,
  selectedCount,
  totalCount,
  onSelectAll,
  onClearSelection,
  onBatchSync,
  onBatchToggleStatus,
  activeCount,
  errorCount,
}: AccountToolbarProps) => {
  const getDensityIcon = (densityType: AccountDensity) => {
    switch (densityType) {
      case 'detailed':
        return <Grid3X3 className="h-4 w-4" />;
      case 'compact':
        return <List className="h-4 w-4" />;
      case 'minimal':
        return <MoreHorizontal className="h-4 w-4" />;
    }
  };

  const getDensityLabel = (densityType: AccountDensity) => {
    switch (densityType) {
      case 'detailed':
        return '详细视图';
      case 'compact':
        return '紧凑视图';
      case 'minimal':
        return '极简视图';
    }
  };

  const getViewModeIcon = (mode: ViewMode) => {
    switch (mode) {
      case 'list':
        return <List className="h-4 w-4" />;
      case 'virtual':
        return <MoreHorizontal className="h-4 w-4" />;
      case 'groups':
        return <Layers className="h-4 w-4" />;
    }
  };

  const getViewModeLabel = (mode: ViewMode) => {
    switch (mode) {
      case 'list':
        return '列表视图';
      case 'virtual':
        return '虚拟滚动';
      case 'groups':
        return '分组视图';
    }
  };

  return (
    <div className="space-y-4">
      {/* 统计信息 */}
      <div className="flex items-center gap-4 text-sm text-muted-foreground">
        <span>共 {totalCount} 个账户</span>
        <Badge variant="default" className="bg-green-600">
          正常 {activeCount}
        </Badge>
        {errorCount > 0 && (
          <Badge variant="destructive">
            异常 {errorCount}
          </Badge>
        )}
        {selectedCount > 0 && (
          <Badge variant="secondary">
            已选择 {selectedCount}
          </Badge>
        )}
      </div>

      {/* 搜索和筛选栏 */}
      <div className="flex flex-col gap-4 sm:flex-row sm:items-center">
        {/* 搜索框 */}
        <div className="relative flex-1">
          <Search className="absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-muted-foreground" />
          <Input
            placeholder="搜索邮箱地址..."
            value={searchQuery}
            onChange={(e) => onSearchChange(e.target.value)}
            className="pl-10"
          />
        </div>

        {/* 筛选器 */}
        <div className="flex items-center gap-2">
          <Select value={statusFilter} onValueChange={onStatusFilterChange}>
            <SelectTrigger className="w-32">
              <SelectValue placeholder="状态" />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="all">全部状态</SelectItem>
              <SelectItem value="active">正常</SelectItem>
              <SelectItem value="disabled">已禁用</SelectItem>
              <SelectItem value="error">异常</SelectItem>
            </SelectContent>
          </Select>

          <Select value={providerFilter} onValueChange={onProviderFilterChange}>
            <SelectTrigger className="w-32">
              <SelectValue placeholder="服务商" />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="all">全部服务商</SelectItem>
              <SelectItem value="gmail">Gmail</SelectItem>
              <SelectItem value="outlook">Outlook</SelectItem>
              <SelectItem value="imap">IMAP</SelectItem>
              <SelectItem value="pop3">POP3</SelectItem>
            </SelectContent>
          </Select>

          {/* 视图模式切换 */}
          <DropdownMenu>
            <DropdownMenuTrigger asChild>
              <Button variant="outline" size="icon" title={getViewModeLabel(viewMode)}>
                {getViewModeIcon(viewMode)}
              </Button>
            </DropdownMenuTrigger>
            <DropdownMenuContent align="end">
              <DropdownMenuItem onClick={() => onViewModeChange('list')}>
                <List className="mr-2 h-4 w-4" />
                列表视图
              </DropdownMenuItem>
              <DropdownMenuItem onClick={() => onViewModeChange('virtual')}>
                <MoreHorizontal className="mr-2 h-4 w-4" />
                虚拟滚动
              </DropdownMenuItem>
              <DropdownMenuItem onClick={() => onViewModeChange('groups')}>
                <Layers className="mr-2 h-4 w-4" />
                分组视图
              </DropdownMenuItem>
            </DropdownMenuContent>
          </DropdownMenu>

          {/* 分组方式选择 */}
          {viewMode === 'groups' && (
            <Select value={groupBy} onValueChange={onGroupByChange}>
              <SelectTrigger className="w-32">
                <SelectValue placeholder="分组方式" />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="provider">按服务商</SelectItem>
                <SelectItem value="status">按状态</SelectItem>
                <SelectItem value="sync_status">按同步状态</SelectItem>
                <SelectItem value="none">不分组</SelectItem>
              </SelectContent>
            </Select>
          )}

          {/* 视图密度切换 */}
          <DropdownMenu>
            <DropdownMenuTrigger asChild>
              <Button variant="outline" size="icon" title={getDensityLabel(density)}>
                {getDensityIcon(density)}
              </Button>
            </DropdownMenuTrigger>
            <DropdownMenuContent align="end">
              <DropdownMenuItem onClick={() => onDensityChange('detailed')}>
                <Grid3X3 className="mr-2 h-4 w-4" />
                详细视图
              </DropdownMenuItem>
              <DropdownMenuItem onClick={() => onDensityChange('compact')}>
                <List className="mr-2 h-4 w-4" />
                紧凑视图
              </DropdownMenuItem>
              <DropdownMenuItem onClick={() => onDensityChange('minimal')}>
                <MoreHorizontal className="mr-2 h-4 w-4" />
                极简视图
              </DropdownMenuItem>
            </DropdownMenuContent>
          </DropdownMenu>
        </div>
      </div>

      {/* 批量操作栏 */}
      {selectedCount > 0 && (
        <div className="flex items-center justify-between rounded-lg bg-muted p-3">
          <div className="flex items-center gap-2">
            <span className="text-sm font-medium">
              已选择 {selectedCount} 个账户
            </span>
            <Button variant="ghost" size="sm" onClick={onClearSelection}>
              取消选择
            </Button>
          </div>
          <div className="flex items-center gap-2">
            <Button variant="outline" size="sm" onClick={onBatchSync}>
              批量同步
            </Button>
            <Button variant="outline" size="sm" onClick={onBatchToggleStatus}>
              批量启用/禁用
            </Button>
            <Button variant="outline" size="sm" onClick={onSelectAll}>
              全选
            </Button>
          </div>
        </div>
      )}
    </div>
  );
};