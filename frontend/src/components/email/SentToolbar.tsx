import { Search, RefreshCw, Trash2, MoreVertical } from 'lucide-react';
import { Button } from '../ui/button';
import { Badge } from '../ui/badge';
import { Input } from '../ui/input';
import { Checkbox } from '../ui/checkbox';
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '../ui/select';
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from '../ui/dropdown-menu';
import { cn } from '../../lib/utils';
import type { SentEmailFilter, Account } from '../../types';
import type { StatusFilter } from '../../pages/SentPage';

interface SentToolbarProps {
  searchKeyword: string;
  onSearchKeywordChange: (v: string) => void;
  onSearch: () => void;
  filter: SentEmailFilter;
  onAccountFilter: (accountUid: string) => void;
  statusFilter: StatusFilter;
  onStatusFilter: (status: StatusFilter) => void;
  activeAccounts: Account[];
  selectedIds: number[];
  onSelectAll: () => void;
  onClearSelection: () => void;
  onDeleteSelected: () => void;
  onRefresh: () => void;
  isLoading: boolean;
  total: number;
  isAllSelected: boolean;
}

/** 已发送页工具栏（自 SentPage 同位搬移）。
 * 纯展示：不请求、不碰 store；StatusFilter 类型仍从页面导入。 */
export function SentToolbar({
  searchKeyword,
  onSearchKeywordChange,
  onSearch,
  filter,
  onAccountFilter,
  statusFilter,
  onStatusFilter,
  activeAccounts,
  selectedIds,
  onSelectAll,
  onClearSelection,
  onDeleteSelected,
  onRefresh,
  isLoading,
  total,
  isAllSelected,
}: SentToolbarProps) {
  return (
      <div className="flex items-center justify-between border-b bg-background px-4 py-1.5">
        {/* 左侧 */}
        <div className="flex items-center gap-2">
          <Checkbox
            checked={isAllSelected}
            onCheckedChange={onSelectAll}
            title={isAllSelected ? '取消全选' : '全选当前页'}
          />

          {/* 状态筛选 */}
          <div className="flex items-center gap-1">
            <Button
              variant={statusFilter === 'all' ? 'secondary' : 'ghost'}
              size="sm"
              onClick={() => onStatusFilter('all')}
              className="h-7 px-2 text-xs"
            >
              全部
            </Button>
            <Button
              variant={statusFilter === 'sent' ? 'secondary' : 'ghost'}
              size="sm"
              onClick={() => onStatusFilter('sent')}
              className="h-7 px-2 text-xs"
            >
              已发送
            </Button>
            <Button
              variant={statusFilter === 'failed' ? 'secondary' : 'ghost'}
              size="sm"
              onClick={() => onStatusFilter('failed')}
              className="h-7 px-2 text-xs"
            >
              失败
            </Button>
          </div>

          <div className="h-4 w-px bg-border" />

          {selectedIds.length > 0 ? (
            <>
              <Badge variant="secondary" className="h-6 text-xs px-2">
                {selectedIds.length} 已选择
              </Badge>
              <Button
                variant="ghost"
                size="sm"
                onClick={() => onDeleteSelected()}
                title="删除记录"
                className="h-7 w-7 p-0 text-destructive hover:text-destructive"
              >
                <Trash2 className="h-3.5 w-3.5" />
              </Button>
            </>
          ) : (
            <span className="text-xs text-muted-foreground">
              共 {total} 封
            </span>
          )}
        </div>

        {/* 右侧 */}
        <div className="flex items-center gap-2">
          <div className="flex items-center gap-1">
            <Input
              placeholder="搜索..."
              value={searchKeyword}
              onChange={(e) => onSearchKeywordChange(e.target.value)}
              onKeyDown={(e) => e.key === 'Enter' && onSearch()}
              className="h-7 w-32 text-xs"
            />
            <Button
              variant="ghost"
              size="sm"
              onClick={onSearch}
              className="h-7 w-7 p-0"
            >
              <Search className="h-3.5 w-3.5" />
            </Button>
          </div>

          <Select
            value={filter.account_uid || 'all'}
            onValueChange={onAccountFilter}
          >
            <SelectTrigger className="h-7 w-32 text-xs">
              <SelectValue placeholder="所有账户" />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="all">所有账户</SelectItem>
              {activeAccounts.map((account: Account) => (
                <SelectItem key={account.uid} value={account.uid}>
                  {account.email}
                </SelectItem>
              ))}
            </SelectContent>
          </Select>

          <Button
            variant="ghost"
            size="sm"
            onClick={onRefresh}
            disabled={isLoading}
            className="h-7 w-7 p-0"
          >
            <RefreshCw className={cn('h-3.5 w-3.5', isLoading && 'animate-spin')} />
          </Button>

          <DropdownMenu>
            <DropdownMenuTrigger asChild>
              <Button variant="ghost" size="sm" className="h-7 w-7 p-0">
                <MoreVertical className="h-3.5 w-3.5" />
              </Button>
            </DropdownMenuTrigger>
            <DropdownMenuContent align="end">
              <DropdownMenuItem onClick={onSelectAll}>
                选择全部
              </DropdownMenuItem>
              <DropdownMenuItem onClick={onClearSelection}>
                取消选择
              </DropdownMenuItem>
              <DropdownMenuSeparator />
              <DropdownMenuItem
                onClick={() => onDeleteSelected()}
                disabled={selectedIds.length === 0}
                className="text-destructive"
              >
                删除选中
              </DropdownMenuItem>
            </DropdownMenuContent>
          </DropdownMenu>
        </div>
      </div>

  );
}
