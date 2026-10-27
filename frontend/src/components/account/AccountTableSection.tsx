import {
  FolderInput, ArrowRightLeft, RefreshCw, Power, Search, X, Loader2, Mail,
  GripVertical, MoreHorizontal, Pencil, Square, Copy, Inbox, AlertCircle,
  Trash2, ChevronLeft, ChevronRight, Plus,
} from 'lucide-react';
import { Button } from '../ui/button';
import { Badge } from '../ui/badge';
import { Input } from '../ui/input';
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '../ui/table';
import { Checkbox } from '../ui/checkbox';
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from '../ui/dropdown-menu';
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '../ui/select';
import {
  Tooltip,
  TooltipContent,
  TooltipProvider,
  TooltipTrigger,
} from '../ui/tooltip';
import { cn } from '../../lib/utils';
import type { Account, AccountGroupWithCount, SyncProgress } from '../../types';
import { toast } from 'sonner';

interface AccountTableSectionProps {
  // 头部：当前分组与批量操作
  currentGroupName: string;
  totalAccounts: number;
  selectedGroupId: number;
  selectedAccountUids: string[];
  isSubmitting: boolean;
  syncingAccounts: string[];
  // 筛选栏
  searchEmail: string;
  filterProvider: string;
  filterStatus: string;
  providerList: string[];
  hasFilters: boolean;
  onSearchEmailChange: (email: string) => void;
  onFilterProviderChange: (provider: string) => void;
  onFilterStatusChange: (status: string) => void;
  onClearFilters: () => void;
  // 表格数据
  accounts: Account[];
  isLoadingAccounts: boolean;
  groups: AccountGroupWithCount[];
  syncProgressMap: Record<string, SyncProgress>;
  // 头部批量操作回调
  onAddToCurrentGroup: () => void;
  onOpenBatchAssign: () => void;
  onBatchSync: () => void;
  onBatchEnable: () => void;
  onBatchDisable: () => void;
  // 表格行回调
  onAddAccount: () => void;
  onSelectAll: (checked: boolean) => void;
  onSelectAccount: (uid: string, checked: boolean) => void;
  onSyncAccount: (uid: string) => void;
  onCancelSync: (uid: string) => void;
  onEditAccount: (account: Account) => void;
  onViewChildAccounts: (account: Account) => void;
  onToggleAccountStatus: (account: Account) => void;
  onClearSyncError: (account: Account) => void;
  onReauthorize: (uid: string, provider: string) => void;
  onDeleteAccount: (uid: string, email: string) => void;
  // 行拖拽（把账号拖入分组）
  draggedAccountUid: string | null;
  onDragStart: (e: React.DragEvent, accountUid: string) => void;
  onDragEnd: (e: React.DragEvent) => void;
  // 分页
  currentPage: number;
  totalPages: number;
  pageSize: number;
  onPageSizeChange: (size: number) => void;
  onPageChange: (page: number) => void;
}

// 获取账号状态显示
const getStatusBadge = (status: string) => {
  switch (status) {
    case 'active':
      return <Badge variant="default" className="bg-green-500">正常</Badge>;
    case 'disabled':
      return <Badge variant="destructive" className="bg-red-500 hover:bg-red-600">禁用</Badge>;
    case 'error':
      return <Badge variant="destructive">错误</Badge>;
    default:
      return <Badge variant="outline">{status}</Badge>;
  }
};

/**
 * 账户页右侧账号表格区块（纯展示组件）：
 * 包含批量操作头部、筛选栏、账号表格与分页控件。
 * 不自己请求数据、不直接读写 store，全部数据与回调来自页面。
 */
export const AccountTableSection = ({
  currentGroupName,
  totalAccounts,
  selectedGroupId,
  selectedAccountUids,
  isSubmitting,
  syncingAccounts,
  searchEmail,
  filterProvider,
  filterStatus,
  providerList,
  hasFilters,
  onSearchEmailChange,
  onFilterProviderChange,
  onFilterStatusChange,
  onClearFilters,
  accounts,
  isLoadingAccounts,
  groups,
  syncProgressMap,
  onAddToCurrentGroup,
  onOpenBatchAssign,
  onBatchSync,
  onBatchEnable,
  onBatchDisable,
  onAddAccount,
  onSelectAll,
  onSelectAccount,
  onSyncAccount,
  onCancelSync,
  onEditAccount,
  onViewChildAccounts,
  onToggleAccountStatus,
  onClearSyncError,
  onReauthorize,
  onDeleteAccount,
  draggedAccountUid,
  onDragStart,
  onDragEnd,
  currentPage,
  totalPages,
  pageSize,
  onPageSizeChange,
  onPageChange,
}: AccountTableSectionProps) => {
  return (
    <div className="flex-1 flex flex-col overflow-hidden">
      {/* 表格头部 - 固定高度避免批量操作按钮出现时页面抖动 */}
      <div className="px-6 py-3 border-b flex items-center justify-between bg-background min-h-[56px]">
        <div className="flex items-center gap-2">
          <h2 className="text-lg font-semibold">{currentGroupName}</h2>
          <Badge variant="outline">{totalAccounts} 个账号</Badge>
        </div>
        <div className="flex items-center gap-2 h-8">
          {selectedAccountUids.length > 0 && (
            <>
              {selectedGroupId > 0 && (
                <Button
                  variant="default"
                  size="sm"
                  onClick={onAddToCurrentGroup}
                  disabled={isSubmitting}
                >
                  <FolderInput className="mr-2 h-4 w-4" />
                  加入当前分组 ({selectedAccountUids.length})
                </Button>
              )}
              <Button
                variant="outline"
                size="sm"
                onClick={onOpenBatchAssign}
              >
                <ArrowRightLeft className="mr-2 h-4 w-4" />
                移动到分组 ({selectedAccountUids.length})
              </Button>
              <Button
                variant="outline"
                size="sm"
                onClick={onBatchSync}
                disabled={syncingAccounts.length > 0}
              >
                <RefreshCw className="mr-2 h-4 w-4" />
                批量同步
              </Button>
              <Button
                variant="outline"
                size="sm"
                onClick={onBatchEnable}
                disabled={isSubmitting}
              >
                <Power className="mr-2 h-4 w-4" />
                批量启用 ({selectedAccountUids.length})
              </Button>
              <Button
                variant="outline"
                size="sm"
                onClick={onBatchDisable}
                disabled={isSubmitting}
              >
                <Power className="mr-2 h-4 w-4" />
                批量禁用 ({selectedAccountUids.length})
              </Button>
            </>
          )}
        </div>
      </div>


      {/* 筛选栏 */}
      <div className="px-6 py-2 border-b flex items-center gap-3 bg-muted/30">
        <div className="relative flex-1 max-w-xs">
          <Search className="absolute left-2.5 top-1/2 -translate-y-1/2 h-4 w-4 text-muted-foreground" />
          <Input
            placeholder="搜索邮箱..."
            value={searchEmail}
            onChange={(e) => onSearchEmailChange(e.target.value)}
            className="pl-8 h-8"
          />
        </div>

        <Select value={filterProvider} onValueChange={onFilterProviderChange}>
          <SelectTrigger className="w-[120px] h-8">
            <SelectValue placeholder="提供商" />
          </SelectTrigger>
          <SelectContent>
            <SelectItem value="all">全部提供商</SelectItem>
            {providerList.map((provider) => (
              <SelectItem key={provider} value={provider}>
                {provider}
              </SelectItem>
            ))}
          </SelectContent>
        </Select>

        <Select value={filterStatus} onValueChange={onFilterStatusChange}>
          <SelectTrigger className="w-[100px] h-8">
            <SelectValue placeholder="状态" />
          </SelectTrigger>
          <SelectContent>
            <SelectItem value="all">全部状态</SelectItem>
            <SelectItem value="active">正常</SelectItem>
            <SelectItem value="disabled">禁用</SelectItem>
            <SelectItem value="error">错误</SelectItem>
          </SelectContent>
        </Select>

        {hasFilters && (
          <Button variant="ghost" size="sm" onClick={onClearFilters} className="h-8">
            <X className="h-4 w-4 mr-1" />
            清除
          </Button>
        )}
      </div>

      {/* 表格内容 */}
      <div className="flex-1 overflow-auto">
        {isLoadingAccounts ? (
          <div className="flex flex-col items-center justify-center h-full text-muted-foreground">
            <Loader2 className="h-8 w-8 mb-4 animate-spin" />
            <p>加载中...</p>
          </div>
        ) : accounts.length === 0 ? (
          <div className="flex flex-col items-center justify-center h-full text-muted-foreground">
            <Mail className="h-12 w-12 mb-4 opacity-50" />
            <p>{hasFilters ? '没有符合条件的账号' : '该分组暂无账号'}</p>
            {!hasFilters && (
              <Button variant="outline" className="mt-4" onClick={onAddAccount}>
                <Plus className="mr-2 h-4 w-4" />
                添加第一个账户
              </Button>
            )}
          </div>
        ) : (
          <Table className="table-fixed w-full">
            <TableHeader>
              <TableRow>
                <TableHead className="w-16">
                  <div className="flex items-center gap-2">
                    <GripVertical className="h-4 w-4 text-muted-foreground opacity-0" />
                    <Checkbox
                      checked={
                        accounts.length > 0 &&
                        accounts.every((acc) => selectedAccountUids.includes(acc.uid))
                      }
                      onCheckedChange={onSelectAll}
                    />
                  </div>
                </TableHead>
                <TableHead style={{ width: '180px' }}>邮箱地址</TableHead>
                <TableHead style={{ width: '120px' }}>提供商</TableHead>
                <TableHead style={{ width: '120px' }}>所属分组</TableHead>
                <TableHead style={{ width: '70px' }}>状态</TableHead>
                <TableHead style={{ width: '80px' }}>同步模式</TableHead>
                <TableHead style={{ width: '100px' }}>最后同步</TableHead>
                <TableHead style={{ width: '60px' }} className="text-right">操作</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {accounts.map((account) => {
                const accountGroup = account.group_id
                  ? groups.find((g) => g.id === account.group_id)
                  : null;
                const isDragging = draggedAccountUid === account.uid;
                const isSyncing = syncingAccounts.includes(account.uid);
                const syncProgress = syncProgressMap[account.uid];


                return (
                  <TableRow
                    key={account.uid}
                    draggable
                    onDragStart={(e) => onDragStart(e, account.uid)}
                    onDragEnd={onDragEnd}
                    className={cn(
                      'cursor-grab active:cursor-grabbing',
                      isDragging && 'opacity-50'
                    )}
                  >
                    <TableCell>
                      <div className="flex items-center gap-2">
                        <GripVertical className="h-4 w-4 text-muted-foreground cursor-grab" />
                        <Checkbox
                          checked={selectedAccountUids.includes(account.uid)}
                          onCheckedChange={(checked) =>
                            onSelectAccount(account.uid, checked as boolean)
                          }
                          onClick={(e) => e.stopPropagation()}
                        />
                      </div>
                    </TableCell>
                    <TableCell className="font-medium">
                      <div className="flex items-center gap-2 max-w-[200px]">
                        <span className="truncate" title={account.email}>{account.email}</span>
                        <TooltipProvider>
                          <Tooltip>
                            <TooltipTrigger asChild>
                              <Button
                                variant="ghost"
                                size="icon"
                                className="h-6 w-6 flex-shrink-0 cursor-pointer"
                                onClick={(e) => {
                                  e.stopPropagation();
                                  navigator.clipboard.writeText(account.email);
                                  toast.success('邮箱地址已复制');
                                }}
                              >
                                <Copy className="h-3.5 w-3.5 text-muted-foreground hover:text-foreground cursor-pointer" />
                              </Button>
                            </TooltipTrigger>
                            <TooltipContent side="top">
                              <p className="text-sm">复制邮箱地址</p>
                            </TooltipContent>
                          </Tooltip>
                        </TooltipProvider>
                        {(account.last_sync_error || account.consecutive_auth_failures > 0) && (
                          <TooltipProvider>
                            <Tooltip>
                              <TooltipTrigger asChild>
                                <AlertCircle className="h-4 w-4 text-destructive flex-shrink-0 cursor-help" />
                              </TooltipTrigger>
                              <TooltipContent side="top" className="max-w-xs">
                                <div className="space-y-1">
                                  {account.consecutive_auth_failures > 0 && (
                                    <p className="text-sm font-medium text-orange-600">
                                      连续失败 {account.consecutive_auth_failures} 次
                                    </p>
                                  )}
                                  {account.last_sync_error && (
                                    <p className="text-sm">同步错误：{account.last_sync_error}</p>
                                  )}
                                </div>
                              </TooltipContent>
                            </Tooltip>
                          </TooltipProvider>
                        )}
                      </div>
                    </TableCell>
                    <TableCell className="max-w-[120px]">
                      <Badge variant="outline" className="truncate max-w-full" title={account.provider}>{account.provider}</Badge>
                    </TableCell>
                    <TableCell className="max-w-[120px]">
                      {accountGroup ? (
                        <Badge variant="secondary" className="truncate max-w-full" title={accountGroup.name}>{accountGroup.name}</Badge>
                      ) : (
                        <span className="text-muted-foreground">未分组</span>
                      )}
                    </TableCell>
                    <TableCell>{getStatusBadge(account.status)}</TableCell>
                    <TableCell>
                      {account.sync_mode === 'webhook' ? (
                        <Badge variant="outline" className="bg-green-50 text-green-700 border-green-200">
                          Webhook
                        </Badge>
                      ) : (
                        <Badge variant="outline" className="text-muted-foreground">
                          轮询
                        </Badge>
                      )}
                    </TableCell>
                    <TableCell className="text-muted-foreground text-sm">
                      {account.last_sync_at
                        ? new Date(account.last_sync_at).toLocaleString('zh-CN', {
                            month: '2-digit',
                            day: '2-digit',
                            hour: '2-digit',
                            minute: '2-digit'
                          })
                        : '从未同步'}
                    </TableCell>
                    <TableCell className="text-right">
                      <DropdownMenu>
                        <DropdownMenuTrigger asChild>
                          <Button variant="ghost" size="icon" className="h-8 w-8">
                            <MoreHorizontal className="h-4 w-4" />
                          </Button>
                        </DropdownMenuTrigger>
                        <DropdownMenuContent align="end">
                          {isSyncing || syncProgress ? (
                            <DropdownMenuItem onClick={() => onCancelSync(account.uid)}>
                              <Square className="h-4 w-4 mr-2" />
                              取消同步
                            </DropdownMenuItem>
                          ) : (
                            // 仅在账号未禁用时显示"立即同步"
                            account.status !== 'disabled' && (
                              <DropdownMenuItem onClick={() => onSyncAccount(account.uid)}>
                                <RefreshCw className="h-4 w-4 mr-2" />
                                立即同步
                              </DropdownMenuItem>
                            )
                          )}
                          <DropdownMenuItem onClick={() => onEditAccount(account)}>
                            <Pencil className="h-4 w-4 mr-2" />
                            编辑
                          </DropdownMenuItem>
                          {/* 查看子邮箱（仅 WebAPI 账户且未禁用时显示） */}
                          {account.status !== 'disabled' &&
                            (account.protocol === 'webapi' ||
                              (account.email && /^(cloudflare|cloud_mail|webapi)[-_]/.test(account.email)) ||
                              account.provider === 'Cloud Mail' ||
                              account.provider === 'Cloudflare Temp Email') && (
                            <DropdownMenuItem onClick={() => onViewChildAccounts(account)}>
                              <Inbox className="h-4 w-4 mr-2" />
                              查看子邮箱
                            </DropdownMenuItem>
                          )}
                          <DropdownMenuItem
                            onClick={() => onToggleAccountStatus(account)}
                          >
                            <Power className="h-4 w-4 mr-2" />
                            {account.status === 'active' ? '禁用' : '启用'}
                          </DropdownMenuItem>
                          {account.last_sync_error && (
                            <DropdownMenuItem onClick={() => onClearSyncError(account)}>
                              <AlertCircle className="h-4 w-4 mr-2" />
                              清除错误
                            </DropdownMenuItem>
                          )}
                          {/* 重新授权（仅 OAuth2 账户且有错误时显示） */}
                          {account.last_sync_error &&
                            (account.auth_type === 'oauth2' || account.provider === 'gmail' || account.provider === 'outlook') &&
                            (account.last_sync_error.toLowerCase().includes('invalid_grant') ||
                              account.last_sync_error.toLowerCase().includes('token') ||
                              account.last_sync_error.toLowerCase().includes('oauth2') ||
                              account.last_sync_error.toLowerCase().includes('unauthorized') ||
                              account.last_sync_error.toLowerCase().includes('401')) && (
                            <DropdownMenuItem onClick={() => onReauthorize(account.uid, account.provider)}>
                              <RefreshCw className="h-4 w-4 mr-2" />
                              重新授权
                            </DropdownMenuItem>
                          )}
                          <DropdownMenuSeparator />
                          <DropdownMenuItem
                            className="text-destructive"
                            onClick={() => onDeleteAccount(account.uid, account.email)}
                          >
                            <Trash2 className="h-4 w-4 mr-2" />
                            删除
                          </DropdownMenuItem>
                        </DropdownMenuContent>
                      </DropdownMenu>
                    </TableCell>
                  </TableRow>
                );
              })}
            </TableBody>
          </Table>
        )}
      </div>


      {/* 分页控件 */}
      {totalAccounts > 0 && (
        <div className="px-6 py-3 border-t flex items-center justify-between bg-background">
          <div className="flex items-center gap-2 text-sm text-muted-foreground">
            <span>每页显示</span>
            <Select
              value={String(pageSize)}
              onValueChange={(value) => onPageSizeChange(Number(value))}
            >
              <SelectTrigger className="w-[70px] h-8">
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="10">10</SelectItem>
                <SelectItem value="20">20</SelectItem>
                <SelectItem value="50">50</SelectItem>
                <SelectItem value="100">100</SelectItem>
              </SelectContent>
            </Select>
            <span>条，共 {totalAccounts} 条</span>
          </div>
          <div className="flex items-center gap-2">
            <Button
              variant="outline"
              size="sm"
              onClick={() => onPageChange(Math.max(1, currentPage - 1))}
              disabled={currentPage === 1}
            >
              <ChevronLeft className="h-4 w-4" />
            </Button>
            <span className="text-sm">
              {currentPage} / {totalPages || 1}
            </span>
            <Button
              variant="outline"
              size="sm"
              onClick={() => onPageChange(Math.min(totalPages, currentPage + 1))}
              disabled={currentPage >= totalPages}
            >
              <ChevronRight className="h-4 w-4" />
            </Button>
          </div>
        </div>
      )}
    </div>
  );
};
