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
  isRefreshing,
}: EmailToolbarProps) => {
  const hasSelection = selectedCount > 0;

  return (
    <div className="flex items-center justify-between border-b bg-background px-4 py-2">
      {/* 左侧：选择信息和操作按钮 */}
      <div className="flex items-center gap-2">
        {hasSelection ? (
          <>
            <Badge variant="secondary">{selectedCount} 已选择</Badge>
            <div className="flex items-center gap-1">
              <Button
                variant="ghost"
                size="icon"
                onClick={onMarkAsRead}
                title="标记为已读"
              >
                <MailOpen className="h-4 w-4" />
              </Button>
              <Button
                variant="ghost"
                size="icon"
                onClick={onMarkAsUnread}
                title="标记为未读"
              >
                <Mail className="h-4 w-4" />
              </Button>
              <Button
                variant="ghost"
                size="icon"
                onClick={onToggleStar}
                title="添加星标"
              >
                <Star className="h-4 w-4" />
              </Button>
              <Button
                variant="ghost"
                size="icon"
                onClick={onArchive}
                title="归档"
              >
                <Archive className="h-4 w-4" />
              </Button>
              <Button
                variant="ghost"
                size="icon"
                onClick={onDelete}
                title="删除"
              >
                <Trash2 className="h-4 w-4" />
              </Button>
            </div>
          </>
        ) : (
          <span className="text-sm text-muted-foreground">
            共 {totalCount} 封邮件
          </span>
        )}
      </div>

      {/* 右侧：刷新和更多操作 */}
      <div className="flex items-center gap-1">
        <Button
          variant="ghost"
          size="icon"
          onClick={onRefresh}
          disabled={isRefreshing}
          title="刷新"
        >
          <RefreshCw
            className={`h-4 w-4 ${isRefreshing ? 'animate-spin' : ''}`}
          />
        </Button>

        <DropdownMenu>
          <DropdownMenuTrigger asChild>
            <Button variant="ghost" size="icon" title="更多操作">
              <MoreVertical className="h-4 w-4" />
            </Button>
          </DropdownMenuTrigger>
          <DropdownMenuContent align="end">
            <DropdownMenuItem onClick={onMarkAsRead}>
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
