// 子邮箱列表区（WebAPI 账户编辑对话框 · 纯展示）
import { Badge } from '../ui/badge';
import {
  Collapsible,
  CollapsibleContent,
  CollapsibleTrigger,
} from '../ui/collapsible';
import { ChevronRight, Loader2, Mail, Users } from 'lucide-react';

// 子邮箱账户信息
export interface ChildAccountInfo {
  uid: string;
  email: string;
  status: string;
  disable_reason?: string;
  total_emails: number;
  unread_count: number;
  last_sync_at: string | null;
  created_at: string;
  orphaned?: boolean;
}

interface ChildAccountsSectionProps {
  /** 子邮箱列表（分页首页预览数据） */
  childAccounts: ChildAccountInfo[];
  /** 子邮箱列表是否加载中 */
  isLoadingChildren: boolean;
  /** 折叠面板是否展开 */
  expanded: boolean;
  /** 切换折叠面板展开状态 */
  onExpandedChange: (open: boolean) => void;
}

/**
 * 子邮箱列表区（纯展示组件）：展示关联邮箱账户预览，不自己请求数据。
 */
export const ChildAccountsSection = ({
  childAccounts,
  isLoadingChildren,
  expanded,
  onExpandedChange,
}: ChildAccountsSectionProps) => {
  return (
    <Collapsible open={expanded} onOpenChange={onExpandedChange}>
      <CollapsibleTrigger className="flex items-center gap-2 text-sm font-medium w-full py-2">
        <Users className="h-4 w-4" />
        关联邮箱账户
        <Badge variant="secondary" className="ml-1 text-xs">
          {childAccounts.length}
        </Badge>
        <ChevronRight className={`h-4 w-4 ml-auto transition-transform ${expanded ? 'rotate-90' : ''}`} />
      </CollapsibleTrigger>
      <CollapsibleContent className="pt-2">
        {isLoadingChildren ? (
          <div className="flex items-center justify-center py-4">
            <Loader2 className="h-4 w-4 animate-spin text-muted-foreground" />
            <span className="ml-2 text-sm text-muted-foreground">加载中...</span>
          </div>
        ) : (
          <div className="space-y-2 max-h-[200px] overflow-y-auto">
            {childAccounts.map((child) => (
              <div
                key={child.uid}
                className="p-3 rounded-lg bg-muted/30 border text-sm"
              >
                <div className="flex items-center gap-2 mb-1">
                  <Mail className="h-4 w-4 text-muted-foreground" />
                  <span className="font-medium truncate flex-1">{child.email}</span>
                  <Badge
                    variant={child.orphaned ? 'outline' : child.status === 'active' ? 'default' : 'secondary'}
                    className="text-xs"
                  >
                    {child.orphaned ? '远端已失效' : child.status === 'active' ? '正常' : child.status}
                  </Badge>
                </div>
                <div className="flex items-center gap-4 text-xs text-muted-foreground ml-6">
                  <span>邮件: {child.total_emails}</span>
                  {child.unread_count > 0 && (
                    <span className="text-primary">未读: {child.unread_count}</span>
                  )}
                  {child.last_sync_at && (
                    <span>
                      同步: {new Date(child.last_sync_at).toLocaleString('zh-CN', {
                        month: 'numeric',
                        day: 'numeric',
                        hour: '2-digit',
                        minute: '2-digit',
                      })}
                    </span>
                  )}
                </div>
              </div>
            ))}
          </div>
        )}
        <p className="text-xs text-muted-foreground mt-2">
          这些邮箱账户由此 WebAPI 账户统一管理和同步
        </p>
      </CollapsibleContent>
    </Collapsible>
  );
};
