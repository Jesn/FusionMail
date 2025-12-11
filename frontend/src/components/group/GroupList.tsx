import { useEffect } from 'react';
import { FolderOpen, Folder, Plus, MoreHorizontal, Pencil, Trash2, GripVertical } from 'lucide-react';
import { cn } from '../../lib/utils';
import { Button } from '../ui/button';
import { Badge } from '../ui/badge';
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from '../ui/dropdown-menu';
import { useGroupStore, ALL_ACCOUNTS_GROUP_ID, UNGROUPED_GROUP_ID } from '../../stores/groupStore';
import type { AccountGroupWithCount } from '../../types';

interface GroupListProps {
  /** 点击创建分组按钮的回调 */
  onCreateGroup?: () => void;
  /** 点击编辑分组的回调 */
  onEditGroup?: (group: AccountGroupWithCount) => void;
  /** 点击删除分组的回调 */
  onDeleteGroup?: (group: AccountGroupWithCount) => void;
  /** 未分组账号数量 */
  ungroupedCount?: number;
  /** 所有账号数量 */
  totalCount?: number;
  /** 是否显示操作按钮 */
  showActions?: boolean;
  /** 自定义类名 */
  className?: string;
}

/**
 * 分组列表组件
 * 显示在侧边栏，用于筛选账号
 */
export const GroupList = ({
  onCreateGroup,
  onEditGroup,
  onDeleteGroup,
  ungroupedCount = 0,
  totalCount = 0,
  showActions = true,
  className,
}: GroupListProps) => {
  const {
    groups,
    selectedGroupId,
    isLoading,
    setSelectedGroupId,
    fetchGroups,
  } = useGroupStore();

  // 组件挂载时获取分组列表
  useEffect(() => {
    fetchGroups();
  }, [fetchGroups]);

  // 渲染分组项
  const renderGroupItem = (
    id: number,
    name: string,
    count: number,
    isSpecial: boolean = false,
    group?: AccountGroupWithCount
  ) => {
    const isSelected = selectedGroupId === id;
    const Icon = isSelected ? FolderOpen : Folder;

    return (
      <div
        key={id}
        className={cn(
          'group flex items-center justify-between px-3 py-2 rounded-md cursor-pointer transition-colors',
          isSelected
            ? 'bg-primary/10 text-primary'
            : 'hover:bg-muted text-muted-foreground hover:text-foreground'
        )}
        onClick={() => setSelectedGroupId(id)}
      >
        <div className="flex items-center gap-2 min-w-0 flex-1">
          {!isSpecial && showActions && (
            <GripVertical className="h-4 w-4 opacity-0 group-hover:opacity-50 cursor-grab" />
          )}
          <Icon className="h-4 w-4 flex-shrink-0" />
          <span className="truncate text-sm">{name}</span>
        </div>
        <div className="flex items-center gap-1">
          <Badge variant="secondary" className="text-xs">
            {count}
          </Badge>
          {!isSpecial && showActions && group && (
            <DropdownMenu>
              <DropdownMenuTrigger asChild>
                <Button
                  variant="ghost"
                  size="icon"
                  className="h-6 w-6 opacity-0 group-hover:opacity-100"
                  onClick={(e) => e.stopPropagation()}
                >
                  <MoreHorizontal className="h-4 w-4" />
                </Button>
              </DropdownMenuTrigger>
              <DropdownMenuContent align="end">
                <DropdownMenuItem onClick={() => onEditGroup?.(group)}>
                  <Pencil className="h-4 w-4 mr-2" />
                  编辑
                </DropdownMenuItem>
                <DropdownMenuItem
                  className="text-destructive"
                  onClick={() => onDeleteGroup?.(group)}
                >
                  <Trash2 className="h-4 w-4 mr-2" />
                  删除
                </DropdownMenuItem>
              </DropdownMenuContent>
            </DropdownMenu>
          )}
        </div>
      </div>
    );
  };

  return (
    <div className={cn('space-y-1', className)}>
      {/* 标题和创建按钮 */}
      <div className="flex items-center justify-between px-3 py-2">
        <span className="text-sm font-medium text-muted-foreground">分组</span>
        {showActions && (
          <Button
            variant="ghost"
            size="icon"
            className="h-6 w-6"
            onClick={onCreateGroup}
            title="创建分组"
          >
            <Plus className="h-4 w-4" />
          </Button>
        )}
      </div>

      {/* 特殊分组：所有账号 */}
      {renderGroupItem(ALL_ACCOUNTS_GROUP_ID, '所有账号', totalCount, true)}

      {/* 特殊分组：未分组 */}
      {renderGroupItem(UNGROUPED_GROUP_ID, '未分组', ungroupedCount, true)}

      {/* 分隔线 */}
      {groups.length > 0 && <div className="border-t my-2" />}

      {/* 自定义分组列表 */}
      {isLoading ? (
        <div className="px-3 py-2 text-sm text-muted-foreground">加载中...</div>
      ) : (
        groups.map((group) =>
          renderGroupItem(group.id, group.name, group.account_count, false, group)
        )
      )}

      {/* 空状态 */}
      {!isLoading && groups.length === 0 && (
        <div className="px-3 py-4 text-center text-sm text-muted-foreground">
          暂无分组
          {showActions && (
            <Button
              variant="link"
              size="sm"
              className="px-1"
              onClick={onCreateGroup}
            >
              创建一个
            </Button>
          )}
        </div>
      )}
    </div>
  );
};
