import { FolderOpen, Folder, Users, MoreHorizontal, Pencil, Trash2 } from 'lucide-react';
import { Button } from '../ui/button';
import { Badge } from '../ui/badge';
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from '../ui/dropdown-menu';
import {
  Tooltip,
  TooltipContent,
  TooltipProvider,
  TooltipTrigger,
} from '../ui/tooltip';
import { cn } from '../../lib/utils';
import type { AccountGroupWithCount } from '../../types';
import { ALL_ACCOUNTS_GROUP_ID, UNGROUPED_GROUP_ID } from '../../stores/groupStore';

interface GroupSidebarProps {
  /** 自定义分组列表（不含"所有账号/未分组"两个特殊项） */
  groups: AccountGroupWithCount[];
  /** 当前选中的分组 ID */
  selectedGroupId: number;
  /** 选中某个分组（页面负责联动清空表格选中项） */
  onSelectGroup: (groupId: number) => void;
  /** 拖拽账号时悬停中的分组 ID */
  dragOverGroupId: number | null;
  onDragOver: (e: React.DragEvent, groupId: number) => void;
  onDragLeave: () => void;
  onDrop: (e: React.DragEvent, targetGroupId: number) => void;
  /** 每个分组的异常账号数量 */
  groupErrorCounts: Record<number, number>;
  /** 获取分组账号总数（后端统计） */
  getGroupAccountCount: (groupId: number) => number;
  /** 获取分组数字 Badge 的警告背景色 */
  getGroupBadgeWarningColor: (groupId: number) => string;
  onEditGroup: (group: AccountGroupWithCount) => void;
  onDeleteGroup: (group: AccountGroupWithCount) => void;
  /** 正在被拖拽的账号 UID（用于显示拖拽提示） */
  draggedAccountUid: string | null;
}

/**
 * 账户页左侧分组树（纯展示组件）：
 * 不自己请求数据、不直接读写 store，全部数据与回调来自页面。
 */
export const GroupSidebar = ({
  groups,
  selectedGroupId,
  onSelectGroup,
  dragOverGroupId,
  onDragOver,
  onDragLeave,
  onDrop,
  groupErrorCounts,
  getGroupAccountCount,
  getGroupBadgeWarningColor,
  onEditGroup,
  onDeleteGroup,
  draggedAccountUid,
}: GroupSidebarProps) => {
  // 渲染分组树项
  const renderGroupItem = (
    id: number,
    name: string,
    isSpecial: boolean = false,
    group?: AccountGroupWithCount
  ) => {
    const isSelected = selectedGroupId === id;
    const Icon = isSelected ? FolderOpen : Folder;
    const isDragOver = dragOverGroupId === id;
    const canDrop = id !== ALL_ACCOUNTS_GROUP_ID;
    const count = getGroupAccountCount(id);
    const errorCount = groupErrorCounts[id] || 0;
    const badgeWarningColor = getGroupBadgeWarningColor(id);

    return (
      <div
        key={id}
        className={cn(
          'group flex items-center justify-between px-3 py-2 rounded-md cursor-pointer transition-all duration-200',
          isSelected ? 'bg-primary text-primary-foreground' : 'hover:bg-muted',
          isDragOver && canDrop && 'ring-2 ring-primary ring-offset-2 bg-primary/10',
          isDragOver && !canDrop && 'ring-2 ring-destructive ring-offset-2 bg-destructive/10'
        )}
        onClick={() => {
          onSelectGroup(id);
        }}
        onDragOver={(e) => canDrop && onDragOver(e, id)}
        onDragLeave={onDragLeave}
        onDrop={(e) => canDrop && onDrop(e, id)}
      >
        <div className="flex items-center gap-2 min-w-0 flex-1">
          {isSpecial ? (
            <Users className="h-4 w-4 flex-shrink-0" />
          ) : (
            <Icon className="h-4 w-4 flex-shrink-0" />
          )}
          <span className="truncate text-sm font-medium">{name}</span>
        </div>
        <div className="flex items-center gap-1 ml-auto shrink-0">
          {/* Badge 带警告背景色（选中和未选中状态都显示） */}
          {/* 格式：异常数/总数 */}
          {errorCount > 0 ? (
            <TooltipProvider>
              <Tooltip>
                <TooltipTrigger asChild>
                  <Badge
                    variant="outline"
                    className={cn(
                      'text-xs min-w-[24px] justify-center cursor-help',
                      badgeWarningColor
                    )}
                  >
                    {errorCount}/{count}
                  </Badge>
                </TooltipTrigger>
                <TooltipContent side="right">
                  <p className="text-xs">{errorCount} 个异常 / 共 {count} 个账号</p>
                </TooltipContent>
              </Tooltip>
            </TooltipProvider>
          ) : (
            <Badge
              variant={isSelected ? 'secondary' : 'outline'}
              className="text-xs min-w-[24px] justify-center"
            >
              {count}
            </Badge>
          )}
          {!isSpecial && group ? (
            <DropdownMenu>
              <DropdownMenuTrigger asChild>
                <Button
                  variant="ghost"
                  size="icon"
                  className={cn(
                    'h-6 w-6 opacity-0 group-hover:opacity-100 flex-shrink-0',
                    isSelected && 'text-primary-foreground hover:bg-primary/80'
                  )}
                  onClick={(e) => e.stopPropagation()}
                >
                  <MoreHorizontal className="h-4 w-4" />
                </Button>
              </DropdownMenuTrigger>
              <DropdownMenuContent align="end">
                <DropdownMenuItem onClick={() => onEditGroup(group)}>
                  <Pencil className="h-4 w-4 mr-2" />
                  编辑
                </DropdownMenuItem>
                <DropdownMenuItem
                  className="text-destructive"
                  onClick={() => onDeleteGroup(group)}
                >
                  <Trash2 className="h-4 w-4 mr-2" />
                  删除
                </DropdownMenuItem>
              </DropdownMenuContent>
            </DropdownMenu>
          ) : (
            <div className="w-6 h-6 flex-shrink-0" />
          )}
        </div>
      </div>
    );
  };

  return (
    <div className="w-64 border-r flex flex-col bg-muted/30">
      <div className="p-3 border-b">
        <span className="text-sm font-medium text-muted-foreground">分组列表</span>
        {draggedAccountUid && (
          <p className="text-xs text-primary mt-1">拖拽到分组上放置</p>
        )}
      </div>
      <div className="flex-1 overflow-auto p-2 space-y-1">
        {renderGroupItem(ALL_ACCOUNTS_GROUP_ID, '所有账号', true)}
        {renderGroupItem(UNGROUPED_GROUP_ID, '未分组', true)}
        {groups.length > 0 && <div className="border-t my-2" />}
        {groups.map((group) =>
          renderGroupItem(group.id, group.name, false, group)
        )}
        {groups.length === 0 && (
          <div className="px-3 py-4 text-center text-sm text-muted-foreground">
            暂无自定义分组
          </div>
        )}
      </div>
    </div>
  );
};
