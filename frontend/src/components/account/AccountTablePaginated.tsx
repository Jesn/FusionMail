import { useState, useMemo } from 'react';
import { 
  ChevronDown, 
  ChevronUp, 
  MoreHorizontal, 
  Edit, 
  RefreshCw, 
  Trash2, 
  Power, 
  PowerOff,
  AlertCircle,
  CheckCircle,
  Clock,
  Mail
} from 'lucide-react';
import { Button } from '../ui/button';
import { Badge } from '../ui/badge';
import { Checkbox } from '../ui/checkbox';
import { 
  Table, 
  TableBody, 
  TableCell, 
  TableHead, 
  TableHeader, 
  TableRow 
} from '../ui/table';
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
import { Account } from '../../types';
import { formatDistanceToNow } from 'date-fns';
import { zhCN } from 'date-fns/locale';

interface AccountTablePaginatedProps {
  accounts: Account[];
  searchQuery: string;
  statusFilter: string;
  providerFilter: string;
  selectedAccounts: string[];
  onAccountSelect: (uid: string, selected: boolean) => void;
  showSelection: boolean;
  onSync: (uid: string) => void;
  onDelete: (uid: string, email: string) => void;
  onEdit: (account: Account) => void;
  onToggleStatus: (uid: string, status: string) => void;
  onClearError: (uid: string) => void;
  syncingAccounts?: string[];
}

type SortField = 'email' | 'provider' | 'status' | 'unread_count' | 'last_sync_at';
type SortDirection = 'asc' | 'desc';

export const AccountTablePaginated = ({
  accounts,
  searchQuery,
  statusFilter,
  providerFilter,
  selectedAccounts,
  onAccountSelect,
  showSelection,
  onSync,
  onDelete,
  onEdit,
  onToggleStatus,
  onClearError,
  syncingAccounts = [],
}: AccountTablePaginatedProps) => {
  const [currentPage, setCurrentPage] = useState(1);
  const [pageSize, setPageSize] = useState(20);
  const [sortField, setSortField] = useState<SortField>('email');
  const [sortDirection, setSortDirection] = useState<SortDirection>('asc');

  // 筛选和搜索
  const filteredAccounts = useMemo(() => {
    return accounts.filter((account) => {
      // 搜索筛选
      if (searchQuery) {
        const query = searchQuery.toLowerCase();
        const emailMatch = account.email.toLowerCase().includes(query);
        const domainMatch = account.email.split('@')[1]?.toLowerCase().includes(query);
        if (!emailMatch && !domainMatch) {
          return false;
        }
      }

      // 状态筛选
      if (statusFilter !== 'all' && account.status !== statusFilter) {
        return false;
      }

      // 服务商筛选
      if (providerFilter !== 'all' && account.provider !== providerFilter) {
        return false;
      }

      return true;
    });
  }, [accounts, searchQuery, statusFilter, providerFilter]);

  // 排序
  const sortedAccounts = useMemo(() => {
    return [...filteredAccounts].sort((a, b) => {
      let aValue: any;
      let bValue: any;

      switch (sortField) {
        case 'email':
          aValue = a.email.toLowerCase();
          bValue = b.email.toLowerCase();
          break;
        case 'provider':
          aValue = a.provider;
          bValue = b.provider;
          break;
        case 'status':
          aValue = a.status;
          bValue = b.status;
          break;
        case 'unread_count':
          aValue = a.unread_count || 0;
          bValue = b.unread_count || 0;
          break;
        case 'last_sync_at':
          aValue = new Date(a.last_sync_at || 0).getTime();
          bValue = new Date(b.last_sync_at || 0).getTime();
          break;
        default:
          return 0;
      }

      if (aValue < bValue) return sortDirection === 'asc' ? -1 : 1;
      if (aValue > bValue) return sortDirection === 'asc' ? 1 : -1;
      return 0;
    });
  }, [filteredAccounts, sortField, sortDirection]);

  // 分页
  const totalPages = Math.ceil(sortedAccounts.length / pageSize);
  const startIndex = (currentPage - 1) * pageSize;
  const endIndex = startIndex + pageSize;
  const currentAccounts = sortedAccounts.slice(startIndex, endIndex);

  // 排序处理
  const handleSort = (field: SortField) => {
    if (sortField === field) {
      setSortDirection(sortDirection === 'asc' ? 'desc' : 'asc');
    } else {
      setSortField(field);
      setSortDirection('asc');
    }
  };

  // 分页处理
  const handlePageChange = (page: number) => {
    setCurrentPage(Math.max(1, Math.min(page, totalPages)));
  };

  const handlePageSizeChange = (newPageSize: string) => {
    setPageSize(Number(newPageSize));
    setCurrentPage(1); // 重置到第一页
  };

  // 渲染排序图标
  const renderSortIcon = (field: SortField) => {
    if (sortField !== field) return null;
    return sortDirection === 'asc' ? 
      <ChevronUp className="ml-1 h-4 w-4" /> : 
      <ChevronDown className="ml-1 h-4 w-4" />;
  };

  // 渲染状态徽章
  const renderStatusBadge = (status: string) => {
    switch (status) {
      case 'active':
        return (
          <Badge variant="default" className="bg-green-600">
            <CheckCircle className="mr-1 h-3 w-3" />
            正常
          </Badge>
        );
      case 'disabled':
        return (
          <Badge variant="secondary">
            <PowerOff className="mr-1 h-3 w-3" />
            已禁用
          </Badge>
        );
      case 'error':
        return (
          <Badge variant="destructive">
            <AlertCircle className="mr-1 h-3 w-3" />
            异常
          </Badge>
        );
      default:
        return <Badge variant="outline">{status}</Badge>;
    }
  };

  // 渲染服务商
  const renderProvider = (provider: string) => {
    const getProviderLabel = (p: string) => {
      switch (p) {
        case 'gmail': return 'Gmail';
        case 'outlook': return 'Outlook';
        case 'imap': return 'IMAP';
        case 'pop3': return 'POP3';
        default: return p.toUpperCase();
      }
    };

    return (
      <div className="flex items-center gap-2">
        <Mail className="h-4 w-4 text-muted-foreground" />
        <span>{getProviderLabel(provider)}</span>
      </div>
    );
  };

  // 格式化时间
  const formatTime = (dateString?: string) => {
    if (!dateString) return '从未同步';
    try {
      return formatDistanceToNow(new Date(dateString), {
        addSuffix: true,
        locale: zhCN,
      });
    } catch {
      return '时间错误';
    }
  };

  if (filteredAccounts.length === 0) {
    return (
      <div className="flex flex-col items-center justify-center rounded-lg border border-dashed py-12">
        <p className="mb-4 text-lg text-muted-foreground">
          {searchQuery || statusFilter !== 'all' || providerFilter !== 'all'
            ? '没有找到匹配的账户'
            : '还没有添加邮箱账户'}
        </p>
        {!searchQuery && statusFilter === 'all' && providerFilter === 'all' && (
          <p className="text-sm text-muted-foreground">
            点击右上角的"添加账户"按钮开始添加
          </p>
        )}
      </div>
    );
  }

  return (
    <div className="space-y-4">
      {/* 表格 */}
      <div className="rounded-lg border">
        <Table>
          <TableHeader>
            <TableRow>
              {showSelection && (
                <TableHead className="w-12">
                  <Checkbox
                    checked={selectedAccounts.length === currentAccounts.length && currentAccounts.length > 0}
                    onCheckedChange={(checked) => {
                      if (checked) {
                        currentAccounts.forEach(account => {
                          if (!selectedAccounts.includes(account.uid)) {
                            onAccountSelect(account.uid, true);
                          }
                        });
                      } else {
                        currentAccounts.forEach(account => {
                          if (selectedAccounts.includes(account.uid)) {
                            onAccountSelect(account.uid, false);
                          }
                        });
                      }
                    }}
                  />
                </TableHead>
              )}
              <TableHead 
                className="cursor-pointer hover:bg-muted/50"
                onClick={() => handleSort('email')}
              >
                <div className="flex items-center">
                  邮箱地址
                  {renderSortIcon('email')}
                </div>
              </TableHead>
              <TableHead 
                className="cursor-pointer hover:bg-muted/50"
                onClick={() => handleSort('provider')}
              >
                <div className="flex items-center">
                  服务商
                  {renderSortIcon('provider')}
                </div>
              </TableHead>
              <TableHead 
                className="cursor-pointer hover:bg-muted/50"
                onClick={() => handleSort('status')}
              >
                <div className="flex items-center">
                  状态
                  {renderSortIcon('status')}
                </div>
              </TableHead>
              <TableHead 
                className="cursor-pointer hover:bg-muted/50 text-right"
                onClick={() => handleSort('unread_count')}
              >
                <div className="flex items-center justify-end">
                  未读数
                  {renderSortIcon('unread_count')}
                </div>
              </TableHead>
              <TableHead 
                className="cursor-pointer hover:bg-muted/50"
                onClick={() => handleSort('last_sync_at')}
              >
                <div className="flex items-center">
                  最后同步
                  {renderSortIcon('last_sync_at')}
                </div>
              </TableHead>
              <TableHead className="w-20">操作</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {currentAccounts.map((account) => (
              <TableRow key={account.uid} className="hover:bg-muted/50">
                {showSelection && (
                  <TableCell>
                    <Checkbox
                      checked={selectedAccounts.includes(account.uid)}
                      onCheckedChange={(checked) => onAccountSelect(account.uid, !!checked)}
                    />
                  </TableCell>
                )}
                <TableCell className="font-medium">
                  <div className="max-w-xs truncate" title={account.email}>
                    {account.email}
                  </div>
                </TableCell>
                <TableCell>
                  {renderProvider(account.provider)}
                </TableCell>
                <TableCell>
                  {renderStatusBadge(account.status)}
                </TableCell>
                <TableCell className="text-right">
                  {account.unread_count > 0 ? (
                    <Badge variant="secondary">
                      {account.unread_count}
                    </Badge>
                  ) : (
                    <span className="text-muted-foreground">0</span>
                  )}
                </TableCell>
                <TableCell>
                  <div className="flex items-center gap-2">
                    {syncingAccounts.includes(account.uid) && (
                      <RefreshCw className="h-4 w-4 animate-spin text-blue-600" />
                    )}
                    <span className="text-sm text-muted-foreground">
                      {formatTime(account.last_sync_at)}
                    </span>
                  </div>
                </TableCell>
                <TableCell>
                  <DropdownMenu>
                    <DropdownMenuTrigger asChild>
                      <Button variant="ghost" size="sm">
                        <MoreHorizontal className="h-4 w-4" />
                      </Button>
                    </DropdownMenuTrigger>
                    <DropdownMenuContent align="end">
                      <DropdownMenuItem onClick={() => onEdit(account)}>
                        <Edit className="mr-2 h-4 w-4" />
                        编辑
                      </DropdownMenuItem>
                      <DropdownMenuItem onClick={() => onSync(account.uid)}>
                        <RefreshCw className="mr-2 h-4 w-4" />
                        同步
                      </DropdownMenuItem>
                      <DropdownMenuSeparator />
                      <DropdownMenuItem onClick={() => onToggleStatus(account.uid, account.status)}>
                        {account.status === 'active' ? (
                          <>
                            <PowerOff className="mr-2 h-4 w-4" />
                            禁用
                          </>
                        ) : (
                          <>
                            <Power className="mr-2 h-4 w-4" />
                            启用
                          </>
                        )}
                      </DropdownMenuItem>
                      {account.status === 'error' && (
                        <DropdownMenuItem onClick={() => onClearError(account.uid)}>
                          <CheckCircle className="mr-2 h-4 w-4" />
                          清除错误
                        </DropdownMenuItem>
                      )}
                      <DropdownMenuSeparator />
                      <DropdownMenuItem 
                        onClick={() => onDelete(account.uid, account.email)}
                        className="text-destructive"
                      >
                        <Trash2 className="mr-2 h-4 w-4" />
                        删除
                      </DropdownMenuItem>
                    </DropdownMenuContent>
                  </DropdownMenu>
                </TableCell>
              </TableRow>
            ))}
          </TableBody>
        </Table>
      </div>

      {/* 分页控件 */}
      <div className="flex items-center justify-between">
        <div className="flex items-center gap-4 text-sm text-muted-foreground">
          <span>
            显示 {startIndex + 1}-{Math.min(endIndex, sortedAccounts.length)} 条，
            共 {sortedAccounts.length} 条记录
          </span>
          <div className="flex items-center gap-2">
            <span>每页显示</span>
            <Select value={pageSize.toString()} onValueChange={handlePageSizeChange}>
              <SelectTrigger className="w-20">
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="10">10</SelectItem>
                <SelectItem value="20">20</SelectItem>
                <SelectItem value="50">50</SelectItem>
                <SelectItem value="100">100</SelectItem>
              </SelectContent>
            </Select>
            <span>条</span>
          </div>
        </div>

        <div className="flex items-center gap-2">
          <Button
            variant="outline"
            size="sm"
            onClick={() => handlePageChange(currentPage - 1)}
            disabled={currentPage <= 1}
          >
            上一页
          </Button>
          
          <div className="flex items-center gap-1">
            {Array.from({ length: Math.min(5, totalPages) }, (_, i) => {
              let pageNum;
              if (totalPages <= 5) {
                pageNum = i + 1;
              } else if (currentPage <= 3) {
                pageNum = i + 1;
              } else if (currentPage >= totalPages - 2) {
                pageNum = totalPages - 4 + i;
              } else {
                pageNum = currentPage - 2 + i;
              }

              return (
                <Button
                  key={pageNum}
                  variant={currentPage === pageNum ? "default" : "outline"}
                  size="sm"
                  onClick={() => handlePageChange(pageNum)}
                  className="w-8 h-8 p-0"
                >
                  {pageNum}
                </Button>
              );
            })}
          </div>

          <Button
            variant="outline"
            size="sm"
            onClick={() => handlePageChange(currentPage + 1)}
            disabled={currentPage >= totalPages}
          >
            下一页
          </Button>
        </div>
      </div>
    </div>
  );
};