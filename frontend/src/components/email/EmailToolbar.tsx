import {
  Archive,
  Trash2,
  Mail,
  MailOpen,
  Star,
  MoreVertical,
  RefreshCw,
} from 'lucide-react';
import { Button } from '../ui/button';
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from '../ui/dropdown-menu';
import { Badge } from '../ui/badge';

interface EmailToolbarProps {
  selectedCount: number;
  totalCount: number;
  onMarkAsRead: () => void;
  onMarkAsUnread: () => void;
  onToggleStar: () => void;
  onArchive: () => void;
  onDelete: () => void;
  onRefresh: () => void;
  onMarkAllAsRead: () => void;
  isRefreshing?: boolean;
}

export const EmailToolbar = ({
  selectedCount,
  totalCount,
  onMarkAsRead,
  onMarkAsUnread,
  onToggleStar,
  onArchive,
  onDelete,
  onRefresh,
  onMarkAllAsRead,
  isRefreshing,
}: EmailToolbarProps) => {
  const hasSelection = selectedCount > 0;

  return (
    <div className="flex items-center justify-between border-b bg-background px-4 py-1.5">
      {/* 左侧：选择信息和操作按钮 */}
      <div className="flex items-center gap-2">
        {hasSelection ? (
          <>
            <Badge variant="secondary" className="h-6 text-xs px-2">{selectedCount} 已选择</Badge>
            <div className="flex items-center gap-1">
              <Button
                variant="ghost"
                size="sm"
                onClick={onMarkAsRead}
                title="标记为已读"
                className="h-7 w-7 p-0"
              >
                <MailOpen className="h-3.5 w-3.5" />
              </Button>
              <Button
                variant="ghost"
                size="sm"
                onClick={onMarkAsUnread}
                title="标记为未读"
                className="h-7 w-7 p-0"
              >
                <Mail className="h-3.5 w-3.5" />
              </Button>
              <Button
                variant="ghost"
                size="sm"
                onClick={onToggleStar}
                title="添加星标"
                className="h-7 w-7 p-0"
              >
                <Star className="h-3.5 w-3.5" />
              </Button>
              <Button
                variant="ghost"
                size="sm"
                onClick={onArchive}
                title="归档"
                className="h-7 w-7 p-0"
              >
                <Archive className="h-3.5 w-3.5" />
              </Button>
              <Button
                variant="ghost"
                size="sm"
                onClick={onDelete}
                title="删除"
                className="h-7 w-7 p-0"
              >
                <Trash2 className="h-3.5 w-3.5" />
              </Button>
            </div>
          </>
        ) : (
          <span className="text-xs text-muted-foreground">
            共 {totalCount} 封邮件
          </span>
        )}
      </div>

      {/* 右侧：刷新和更多操作 */}
      <div className="flex items-center gap-1">
        <Button
          variant="ghost"
          size="sm"
          onClick={onRefresh}
          disabled={isRefreshing}
          title="刷新"
          className="h-7 w-7 p-0"
        >
          <RefreshCw
            className={`h-3.5 w-3.5 ${isRefreshing ? 'animate-spin' : ''}`}
          />
        </Button>

        <DropdownMenu>
          <DropdownMenuTrigger asChild>
            <Button variant="ghost" size="sm" title="更多操作" className="h-7 w-7 p-0">
              <MoreVertical className="h-3.5 w-3.5" />
            </Button>
          </DropdownMenuTrigger>
          <DropdownMenuContent align="end">
            <DropdownMenuItem onClick={onMarkAllAsRead}>
              全部标记为已读
            </DropdownMenuItem>
            <DropdownMenuItem>选择全部</DropdownMenuItem>
            <DropdownMenuItem>取消选择</DropdownMenuItem>
          </DropdownMenuContent>
        </DropdownMenu>
      </div>
    </div>
  );
};
