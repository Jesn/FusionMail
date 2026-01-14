import { useState, useEffect, useMemo, useCallback } from 'react';
import { 
  Plus, FolderOpen, Folder, Users, MoreHorizontal, Pencil, Trash2, 
  ArrowRightLeft, Mail, FolderInput, GripVertical, ChevronLeft, ChevronRight, 
  Search, X, Loader2, RefreshCw, Power, AlertCircle, Square, Copy, Upload,
  Cloud, Globe, Inbox
} from 'lucide-react';
import { Button } from '../components/ui/button';
import { Badge } from '../components/ui/badge';
import { Input } from '../components/ui/input';
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '../components/ui/table';
import { Checkbox } from '../components/ui/checkbox';
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from '../components/ui/dropdown-menu';
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '../components/ui/select';
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
} from '../components/ui/alert-dialog';
import {
  Tooltip,
  TooltipContent,
  TooltipProvider,
  TooltipTrigger,
} from '../components/ui/tooltip';
import { GroupDialog, GroupDeleteDialog, GroupBatchAssign } from '../components/group';
import { AccountForm } from '../components/account/AccountForm';
import { BatchImportDialog, BatchImportResult, BatchImportConfig } from '../components/account/BatchImportDialog';
import { WebAPIProviderDialog, WebAPIAccountEditDialog, ChildAccountsDialog } from '../components/webapi';
import { useGroupStore, ALL_ACCOUNTS_GROUP_ID, UNGROUPED_GROUP_ID } from '../stores/groupStore';
import { useAccounts } from '../hooks/useAccounts';
import { useUIStore } from '../stores/uiStore';
import { cn } from '../lib/utils';
import type { Account, AccountGroupWithCount } from '../types';
import { groupService } from '../services/groupService';
import { accountService, AccountListResponse } from '../services/accountService';
import { toast } from 'sonner';


export const AccountsPage = () => {
  const { groups, totalCount, ungroupedCount, selectedGroupId, setSelectedGroupId, fetchGroups, createGroup, editGroup, deleteGroup } = useGroupStore();
  const { 
    accounts, 
    loadAccounts, 
    createAccount,
    updateAccount,
    deleteAccount,
    syncAccount,
    toggleAccountStatus,
    clearSyncError,
    cancelSync,
    syncProgressMap,
  } = useAccounts();
  const { isAccountDialogOpen, setAccountDialogOpen } = useUIStore();
  
  // 账户编辑/删除状态
  const [editingAccount, setEditingAccount] = useState<Account | null>(null);
  const [deletingAccount, setDeletingAccount] = useState<{ uid: string; email: string } | null>(null);
  const [deleteConfirmEmail, setDeleteConfirmEmail] = useState('');
  

  
  // 分组对话框状态
  const [groupDialogOpen, setGroupDialogOpen] = useState(false);
  const [editingGroup, setEditingGroup] = useState<AccountGroupWithCount | null>(null);
  const [deleteDialogOpen, setDeleteDialogOpen] = useState(false);
  const [deletingGroup, setDeletingGroup] = useState<AccountGroupWithCount | null>(null);
  const [batchAssignOpen, setBatchAssignOpen] = useState(false);
  const [isSubmitting, setIsSubmitting] = useState(false);
  
  // 批量导入 Outlook 对话框状态
  const [batchImportOpen, setBatchImportOpen] = useState(false);
  
  // WebAPI Provider 对话框状态
  const [webAPIDialogOpen, setWebAPIDialogOpen] = useState(false);
  const [webAPIServiceType, setWebAPIServiceType] = useState<'cloudflare_temp_email' | 'cloud_mail' | null>(null);
  
  // WebAPI 账户编辑对话框状态
  const [webAPIEditDialogOpen, setWebAPIEditDialogOpen] = useState(false);
  const [editingWebAPIAccount, setEditingWebAPIAccount] = useState<Account | null>(null);
  
  // 子邮箱列表对话框状态
  const [childAccountsDialogOpen, setChildAccountsDialogOpen] = useState(false);
  const [viewingChildAccountsAccount, setViewingChildAccountsAccount] = useState<Account | null>(null);
  
  // 表格选择状态
  const [selectedAccountUids, setSelectedAccountUids] = useState<string[]>([]);
  
  // 同步状态
  const [syncingAccounts, setSyncingAccounts] = useState<string[]>([]);
  
  // 拖拽状态
  const [draggedAccountUid, setDraggedAccountUid] = useState<string | null>(null);
  const [dragOverGroupId, setDragOverGroupId] = useState<number | null>(null);
  
  // 分页状态
  const [currentPage, setCurrentPage] = useState(1);
  const [pageSize, setPageSize] = useState(10);

  // 筛选状态
  const [searchEmail, setSearchEmail] = useState('');
  const [filterProvider, setFilterProvider] = useState<string>('all');
  const [filterStatus, setFilterStatus] = useState<string>('all');

  // 后端分页数据
  const [paginatedAccounts, setPaginatedAccounts] = useState<Account[]>([]);
  const [totalAccounts, setTotalAccounts] = useState(0);
  const [totalPages, setTotalPages] = useState(0);
  const [isLoadingAccounts, setIsLoadingAccounts] = useState(false);

  // 确保数据已加载（仅用于提供商筛选下拉框）
  const safeAccounts = accounts || [];
  const activeAccounts = safeAccounts.filter((acc) => !acc.deleted_at);

  // 加载分组数据
  useEffect(() => {
    useGroupStore.getState().setCacheTimestamp(0);
    fetchGroups();
    loadAccounts(true); // 仅用于获取提供商列表
  }, [fetchGroups, loadAccounts]);

  // 获取所有提供商列表（用于筛选下拉框）
  const providerList = useMemo(() => {
    const providers = new Set(activeAccounts.map((acc) => acc.provider).filter(Boolean));
    return Array.from(providers).sort();
  }, [activeAccounts]);

  // 计算高风险账号数量（失败次数 > 3）
  const highRiskAccounts = useMemo(() => {
    return activeAccounts.filter(acc => acc.consecutive_auth_failures > 3);
  }, [activeAccounts]);

  // 计算每个分组的异常账号数量（有同步错误或失败次数 > 0）
  const groupErrorCounts = useMemo(() => {
    const counts: Record<number, number> = {};
    
    activeAccounts.forEach(acc => {
      const hasError = acc.last_sync_error || acc.consecutive_auth_failures > 0;
      if (!hasError) return;
      
      if (acc.group_id) {
        counts[acc.group_id] = (counts[acc.group_id] || 0) + 1;
      } else {
        // 未分组账号
        counts[UNGROUPED_GROUP_ID] = (counts[UNGROUPED_GROUP_ID] || 0) + 1;
      }
    });
    
    // 计算"所有账号"分组的总异常数（仅在没有自定义分组时才计算）
    if (groups.length === 0) {
      counts[ALL_ACCOUNTS_GROUP_ID] = Object.values(counts).reduce((sum, count) => sum + count, 0);
    }
    
    return counts;
  }, [activeAccounts, groups.length]);

  // 获取分组数字 Badge 的警告背景色（根据异常账号数量）
  const getGroupBadgeWarningColor = useCallback((groupId: number) => {
    const errorCount = groupErrorCounts[groupId] || 0;
    if (errorCount === 0) return '';
    
    // 根据异常数量返回不同深浅的橙色（更醒目）
    if (errorCount >= 10) return 'bg-orange-200 border-orange-300 text-orange-900'; // 深橙色
    if (errorCount >= 5) return 'bg-orange-100 border-orange-200 text-orange-800';  // 中橙色
    if (errorCount >= 2) return 'bg-orange-50 border-orange-200 text-orange-700';   // 浅橙色
    return 'bg-orange-50 border-orange-100 text-orange-700'; // 极浅橙色
  }, [groupErrorCounts]);

  // 后端分页和筛选请求
  const fetchAccountsWithFilter = useCallback(async () => {
    setIsLoadingAccounts(true);
    try {
      const filter: {
        page: number;
        page_size: number;
        group_id?: number;
        email?: string;
        provider?: string;
        status?: string;
      } = {
        page: currentPage,
        page_size: pageSize,
      };

      if (selectedGroupId === ALL_ACCOUNTS_GROUP_ID) {
        filter.group_id = -1;
      } else if (selectedGroupId === UNGROUPED_GROUP_ID) {
        filter.group_id = 0;
      } else {
        filter.group_id = selectedGroupId;
      }

      if (searchEmail.trim()) {
        filter.email = searchEmail.trim();
      }
      if (filterProvider !== 'all') {
        filter.provider = filterProvider;
      }
      if (filterStatus !== 'all') {
        filter.status = filterStatus;
      }

      const result: AccountListResponse = await accountService.getListWithFilter(filter);
      setPaginatedAccounts(result.accounts || []);
      setTotalAccounts(result.total);
      setTotalPages(result.total_pages);
    } catch (error) {
      console.error('Failed to fetch accounts:', error);
      toast.error('获取账号列表失败');
    } finally {
      setIsLoadingAccounts(false);
    }
  }, [currentPage, pageSize, selectedGroupId, searchEmail, filterProvider, filterStatus]);

  // 统一的数据刷新函数，避免遗漏和重复代码
  // 注意：分组计数现在由后端返回，无需加载全量账号数据
  const refreshAllData = useCallback(async () => {
    // 清除缓存时间戳，强制重新获取分组数据（包含统计信息）
    useGroupStore.getState().setCacheTimestamp(0);
    await Promise.all([
      fetchGroups(),           // 刷新分组列表（含统计数据）
      fetchAccountsWithFilter() // 刷新表格数据
    ]);
  }, [fetchGroups, fetchAccountsWithFilter]);

  useEffect(() => {
    fetchAccountsWithFilter();
  }, [fetchAccountsWithFilter]);

  useEffect(() => {
    setCurrentPage(1);
    setSelectedAccountUids([]);
  }, [selectedGroupId]);

  useEffect(() => {
    const timer = setTimeout(() => {
      setCurrentPage(1);
    }, 300);
    return () => clearTimeout(timer);
  }, [searchEmail, filterProvider, filterStatus]);

  const clearFilters = () => {
    setSearchEmail('');
    setFilterProvider('all');
    setFilterStatus('all');
  };

  const hasFilters = searchEmail.trim() || filterProvider !== 'all' || filterStatus !== 'all';

  const currentGroupName = useMemo(() => {
    if (selectedGroupId === ALL_ACCOUNTS_GROUP_ID) return '所有账号';
    if (selectedGroupId === UNGROUPED_GROUP_ID) return '未分组';
    const group = groups.find((g) => g.id === selectedGroupId);
    return group?.name || '未知分组';
  }, [selectedGroupId, groups]);


  // 分组操作处理
  const handleCreateGroup = () => {
    setEditingGroup(null);
    setGroupDialogOpen(true);
  };

  const handleEditGroup = (group: AccountGroupWithCount) => {
    setEditingGroup(group);
    setGroupDialogOpen(true);
  };

  const handleDeleteGroup = (group: AccountGroupWithCount) => {
    setDeletingGroup(group);
    setDeleteDialogOpen(true);
  };

  const handleGroupSubmit = async (name: string, description: string) => {
    setIsSubmitting(true);
    try {
      if (editingGroup) {
        await editGroup(editingGroup.id, name, description);
        toast.success('分组更新成功');
      } else {
        await createGroup(name, description);
        toast.success('分组创建成功');
      }
      setGroupDialogOpen(false);
    } catch (error) {
      // 重新抛出错误，让 GroupDialog 组件在表单中显示错误信息
      throw error;
    } finally {
      setIsSubmitting(false);
    }
  };

  const handleConfirmDeleteGroup = async () => {
    if (!deletingGroup) return;
    setIsSubmitting(true);
    try {
      await deleteGroup(deletingGroup.id);
      toast.success('分组删除成功');
      setDeleteDialogOpen(false);
      setDeletingGroup(null);
      if (selectedGroupId === deletingGroup.id) {
        setSelectedGroupId(ALL_ACCOUNTS_GROUP_ID);
      }
    } catch (error) {
      toast.error(error instanceof Error ? error.message : '删除失败');
    } finally {
      setIsSubmitting(false);
    }
  };

  // 账户操作处理
  const handleAddAccount = () => {
    setEditingAccount(null);
    setAccountDialogOpen(true);
  };

  const handleEditAccount = (account: Account) => {
    // 检查是否是 WebAPI 账户
    const isWebAPIAccount = account.protocol === 'webapi' || 
      (account.email && /^(cloudflare|cloud_mail|webapi)[-_]/.test(account.email));
    
    if (isWebAPIAccount) {
      // WebAPI 账户使用专用编辑对话框
      setEditingWebAPIAccount(account);
      setWebAPIEditDialogOpen(true);
    } else {
      // 传统账户使用原有编辑对话框
      setEditingAccount(account);
      setAccountDialogOpen(true);
    }
  };

  const handleCloseAccountDialog = () => {
    setAccountDialogOpen(false);
    setEditingAccount(null);
  };

  const handleAccountSubmit = async (data: any) => {
    if (editingAccount) {
      await updateAccount(editingAccount.uid, data);
    } else {
      await createAccount(data);
    }
    handleCloseAccountDialog();
    await refreshAllData();
  };

  // 批量导入 Outlook 账户
  const handleBatchImport = async (config: BatchImportConfig): Promise<BatchImportResult> => {
    try {
      const result = await accountService.batchImport(
        config.accounts, 
        config.syncEnabled, 
        config.syncInterval,
        config.groupId,
        config.firstSyncDays
      );
      // 导入完成后刷新数据
      await refreshAllData();
      return {
        success: result.success,
        failed: result.failed,
        results: result.results.map(r => ({
          email: r.email,
          status: r.status as 'success' | 'failed',
          error: r.error,
        })),
      };
    } catch (error) {
      console.error('批量导入失败:', error);
      throw error;
    }
  };

  const handleDeleteClick = (uid: string, email: string) => {
    setDeletingAccount({ uid, email });
  };

  const handleDeleteConfirm = async () => {
    if (deletingAccount && deleteConfirmEmail === deletingAccount.email) {
      await deleteAccount(deletingAccount.uid);
      setDeletingAccount(null);
      setDeleteConfirmEmail('');
      await refreshAllData();
    }
  };

  const handleDeleteCancel = () => {
    setDeletingAccount(null);
    setDeleteConfirmEmail('');
  };


  // 同步操作
  const handleSyncAccount = async (uid: string) => {
    setSyncingAccounts(prev => [...prev, uid]);
    try {
      await syncAccount(uid);
    } finally {
      setSyncingAccounts(prev => prev.filter(id => id !== uid));
    }
  };

  // 监听同步进度变化，同步完成后刷新数据
  useEffect(() => {
    const completedSyncs = Object.entries(syncProgressMap).filter(
      ([_, progress]) => 
        progress.status === 'completed' || 
        progress.status === 'failed' || 
        progress.status === 'cancelled'
    );

    if (completedSyncs.length > 0) {
      // 延迟刷新，等待后端数据更新
      const timer = setTimeout(() => {
        fetchAccountsWithFilter();
      }, 1000);
      return () => clearTimeout(timer);
    }
  }, [syncProgressMap, fetchAccountsWithFilter]);

  const handleCancelSync = async (uid: string) => {
    try {
      await cancelSync(uid);
    } catch (err) {
      // 错误已在 hook 中处理
    }
  };

  // 重新授权处理
  const handleReauthorize = async (uid: string, provider: string) => {
    try {
      // 根据 provider 确定 OAuth2 端点
      let authEndpoint = '';
      if (provider === 'gmail') {
        authEndpoint = `/api/v1/auth/google/reauthorize/${uid}`;
      } else if (provider === 'outlook') {
        authEndpoint = `/api/v1/auth/microsoft/reauthorize/${uid}`;
      } else {
        toast.error('该账户类型不支持重新授权');
        return;
      }

      // 获取重新授权 URL
      const response = await fetch(authEndpoint);
      const result = await response.json();
      
      if (!result.success || !result.data?.auth_url) {
        toast.error(result.message || '获取授权链接失败');
        return;
      }

      // 打开授权窗口
      const authWindow = window.open(
        result.data.auth_url,
        'oauth2_reauthorize',
        'width=600,height=700,scrollbars=yes'
      );

      // 监听授权结果
      const checkClosed = setInterval(() => {
        if (authWindow?.closed) {
          clearInterval(checkClosed);
          // 刷新账户列表
          setTimeout(async () => {
            await refreshAllData();
            toast.success('授权流程已完成，请检查账户状态');
          }, 1000);
        }
      }, 500);

      // 监听 postMessage
      const handleMessage = (event: MessageEvent) => {
        if (event.data?.type === 'oauth2_result') {
          window.removeEventListener('message', handleMessage);
          clearInterval(checkClosed);
          
          if (event.data.data?.success) {
            toast.success('重新授权成功');
            refreshAllData();
          } else {
            toast.error(event.data.data?.error || '授权失败');
          }
        }
      };
      window.addEventListener('message', handleMessage);

      // 30秒后清理监听器
      setTimeout(() => {
        window.removeEventListener('message', handleMessage);
        clearInterval(checkClosed);
      }, 30000);

    } catch (error) {
      console.error('Reauthorize error:', error);
      toast.error('重新授权失败');
    }
  };

  // 批量操作
  const handleBatchComplete = async () => {
    setSelectedAccountUids([]);
    await refreshAllData();
  };

  const handleAddToCurrentGroup = async () => {
    if (selectedGroupId <= 0 || selectedAccountUids.length === 0) return;
    
    setIsSubmitting(true);
    try {
      await Promise.all(
        selectedAccountUids.map((uid) => 
          groupService.assignAccountToGroup(uid, selectedGroupId)
        )
      );
      toast.success(`已将 ${selectedAccountUids.length} 个账号加入分组`);
      setSelectedAccountUids([]);
      await refreshAllData();
    } catch (error) {
      toast.error(error instanceof Error ? error.message : '操作失败');
    } finally {
      setIsSubmitting(false);
    }
  };

  const handleBatchSync = async () => {
    setSyncingAccounts(selectedAccountUids);
    try {
      for (const uid of selectedAccountUids) {
        await syncAccount(uid);
      }
    } finally {
      setSyncingAccounts([]);
      setSelectedAccountUids([]);
    }
  };

  // 批量启用账户
  const handleBatchEnable = async () => {
    if (selectedAccountUids.length === 0) return;
    
    setIsSubmitting(true);
    try {
      const result = await accountService.batchEnable(selectedAccountUids);
      
      if (result.failed > 0) {
        // 有失败的情况，显示详细信息
        const failedEmails = result.failed_items.map(item => item.email || item.uid).join(', ');
        toast.warning(
          `批量启用完成：成功 ${result.success} 个，失败 ${result.failed} 个`,
          {
            description: `失败账号：${failedEmails}`,
            duration: 5000,
          }
        );
      } else {
        // 全部成功
        toast.success(`已成功启用 ${result.success} 个账号`);
      }
      
      // 清空选择并刷新数据
      setSelectedAccountUids([]);
      await refreshAllData();
    } catch (error) {
      console.error('批量启用失败:', error);
      toast.error('批量启用失败');
    } finally {
      setIsSubmitting(false);
    }
  };

  // 批量禁用账户
  const handleBatchDisable = async () => {
    if (selectedAccountUids.length === 0) return;
    
    setIsSubmitting(true);
    try {
      const result = await accountService.batchDisable(selectedAccountUids);
      
      if (result.failed > 0) {
        // 有失败的情况，显示详细信息
        const failedEmails = result.failed_items.map(item => item.email || item.uid).join(', ');
        toast.warning(
          `批量禁用完成：成功 ${result.success} 个，失败 ${result.failed} 个`,
          {
            description: `失败账号：${failedEmails}`,
            duration: 5000,
          }
        );
      } else {
        // 全部成功
        toast.success(`已成功禁用 ${result.success} 个账号`);
      }
      
      // 清空选择并刷新数据
      setSelectedAccountUids([]);
      await refreshAllData();
    } catch (error) {
      console.error('批量禁用失败:', error);
      toast.error('批量禁用失败');
    } finally {
      setIsSubmitting(false);
    }
  };

  // 选择操作
  const handleSelectAll = (checked: boolean) => {
    if (checked) {
      setSelectedAccountUids(paginatedAccounts.map((acc) => acc.uid));
    } else {
      setSelectedAccountUids([]);
    }
  };

  const handleSelectAccount = (uid: string, checked: boolean) => {
    if (checked) {
      setSelectedAccountUids((prev) => [...prev, uid]);
    } else {
      setSelectedAccountUids((prev) => prev.filter((id) => id !== uid));
    }
  };


  // 拖拽操作
  const handleDragStart = useCallback((e: React.DragEvent, accountUid: string) => {
    setDraggedAccountUid(accountUid);
    e.dataTransfer.effectAllowed = 'move';
    e.dataTransfer.setData('text/plain', accountUid);
    if (e.currentTarget instanceof HTMLElement) {
      e.currentTarget.style.opacity = '0.5';
    }
  }, []);

  const handleDragEnd = useCallback((e: React.DragEvent) => {
    setDraggedAccountUid(null);
    setDragOverGroupId(null);
    if (e.currentTarget instanceof HTMLElement) {
      e.currentTarget.style.opacity = '1';
    }
  }, []);

  const handleDragOver = useCallback((e: React.DragEvent, groupId: number) => {
    e.preventDefault();
    e.dataTransfer.dropEffect = 'move';
    setDragOverGroupId(groupId);
  }, []);

  const handleDragLeave = useCallback(() => {
    setDragOverGroupId(null);
  }, []);

  const handleDrop = useCallback(async (e: React.DragEvent, targetGroupId: number) => {
    e.preventDefault();
    const accountUid = e.dataTransfer.getData('text/plain');
    
    if (!accountUid) return;
    
    if (targetGroupId === ALL_ACCOUNTS_GROUP_ID) {
      toast.error('请选择具体分组或"未分组"');
      setDragOverGroupId(null);
      setDraggedAccountUid(null);
      return;
    }
    
    try {
      const newGroupId = targetGroupId === UNGROUPED_GROUP_ID ? null : targetGroupId;
      await groupService.assignAccountToGroup(accountUid, newGroupId);
      
      const targetName = targetGroupId === UNGROUPED_GROUP_ID 
        ? '未分组' 
        : groups.find(g => g.id === targetGroupId)?.name || '分组';
      toast.success(`已移动到「${targetName}」`);
      
      await refreshAllData();
    } catch (error) {
      toast.error(error instanceof Error ? error.message : '移动失败');
    } finally {
      setDragOverGroupId(null);
      setDraggedAccountUid(null);
    }
  }, [groups, fetchGroups, loadAccounts, fetchAccountsWithFilter]);

  // 获取分组账号数量（直接使用后端返回的统计数据，无需前端计算）
  const getGroupAccountCount = useCallback((groupId: number) => {
    if (groupId === ALL_ACCOUNTS_GROUP_ID) {
      return totalCount; // 使用后端返回的总数
    }
    if (groupId === UNGROUPED_GROUP_ID) {
      return ungroupedCount; // 使用后端返回的未分组数
    }
    // 从 groups 中获取对应分组的 account_count
    const group = groups.find(g => g.id === groupId);
    return group?.account_count || 0;
  }, [totalCount, ungroupedCount, groups]);


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
          setSelectedGroupId(id);
          setSelectedAccountUids([]);
        }}
        onDragOver={(e) => canDrop && handleDragOver(e, id)}
        onDragLeave={handleDragLeave}
        onDrop={(e) => canDrop && handleDrop(e, id)}
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
                    {count}
                  </Badge>
                </TooltipTrigger>
                <TooltipContent side="right">
                  <p className="text-xs">{errorCount} 个账号存在同步异常</p>
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
                <DropdownMenuItem onClick={() => handleEditGroup(group)}>
                  <Pencil className="h-4 w-4 mr-2" />
                  编辑
                </DropdownMenuItem>
                <DropdownMenuItem
                  className="text-destructive"
                  onClick={() => handleDeleteGroup(group)}
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


  return (
    <div className="h-full overflow-hidden flex flex-col">
      {/* 页面头部 */}
      <div className="border-b px-6 py-4">
        <div className="flex items-center justify-between">
          <div>
            <h1 className="text-2xl font-bold">账户管理</h1>
            <p className="text-sm text-muted-foreground">
              管理邮箱账户和分组，便于分类和筛选
            </p>
          </div>
          <div className="flex items-center gap-2">
            <Button variant="outline" onClick={handleCreateGroup}>
              <Plus className="mr-2 h-4 w-4" />
              创建分组
            </Button>
            <Button variant="outline" onClick={() => setBatchImportOpen(true)}>
              <Upload className="mr-2 h-4 w-4" />
              批量导入 Outlook
            </Button>
            <DropdownMenu>
              <DropdownMenuTrigger asChild>
                <Button variant="outline">
                  <Globe className="mr-2 h-4 w-4" />
                  添加 WebAPI 账户
                </Button>
              </DropdownMenuTrigger>
              <DropdownMenuContent align="end">
                <DropdownMenuItem onClick={() => {
                  setWebAPIServiceType('cloudflare_temp_email');
                  setWebAPIDialogOpen(true);
                }}>
                  <Cloud className="mr-2 h-4 w-4" />
                  Cloudflare Temp Email
                </DropdownMenuItem>
                <DropdownMenuItem onClick={() => {
                  setWebAPIServiceType('cloud_mail');
                  setWebAPIDialogOpen(true);
                }}>
                  <Mail className="mr-2 h-4 w-4" />
                  Cloud Mail
                </DropdownMenuItem>
              </DropdownMenuContent>
            </DropdownMenu>
            <Button onClick={handleAddAccount}>
              <Plus className="mr-2 h-4 w-4" />
              添加账户
            </Button>
          </div>
        </div>
      </div>

      {/* 高风险账号警告提示 */}
      {highRiskAccounts.length > 0 && (
        <div className="bg-orange-50 border-b border-orange-200 px-6 py-3">
          <div className="flex items-center gap-3">
            <AlertCircle className="h-5 w-5 text-orange-600 flex-shrink-0" />
            <div className="flex-1">
              <p className="text-sm font-medium text-orange-900">
                检测到 {highRiskAccounts.length} 个账号连续同步失败超过 3 次
              </p>
              <p className="text-xs text-orange-700 mt-0.5">
                这些账号可能存在认证问题，建议检查账号配置或重新授权
              </p>
            </div>
            <Button 
              variant="outline" 
              size="sm"
              className="border-orange-300 text-orange-700 hover:bg-orange-100"
              onClick={() => {
                // 筛选出高风险账号
                setFilterStatus('error');
                toast.info('已筛选出异常账号');
              }}
            >
              查看详情
            </Button>
          </div>
        </div>
      )}

      {/* 主体内容：左右分栏 */}
      <div className="flex-1 flex overflow-hidden">
        {/* 左侧：分组树 */}
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

        {/* 右侧：账号表格 */}
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
                      onClick={handleAddToCurrentGroup}
                      disabled={isSubmitting}
                    >
                      <FolderInput className="mr-2 h-4 w-4" />
                      加入当前分组 ({selectedAccountUids.length})
                    </Button>
                  )}
                  <Button 
                    variant="outline" 
                    size="sm"
                    onClick={() => setBatchAssignOpen(true)}
                  >
                    <ArrowRightLeft className="mr-2 h-4 w-4" />
                    移动到分组 ({selectedAccountUids.length})
                  </Button>
                  <Button 
                    variant="outline" 
                    size="sm"
                    onClick={handleBatchSync}
                    disabled={syncingAccounts.length > 0}
                  >
                    <RefreshCw className="mr-2 h-4 w-4" />
                    批量同步
                  </Button>
                  <Button 
                    variant="outline" 
                    size="sm"
                    onClick={handleBatchEnable}
                    disabled={isSubmitting}
                  >
                    <Power className="mr-2 h-4 w-4" />
                    批量启用 ({selectedAccountUids.length})
                  </Button>
                  <Button 
                    variant="outline" 
                    size="sm"
                    onClick={handleBatchDisable}
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
                onChange={(e) => setSearchEmail(e.target.value)}
                className="pl-8 h-8"
              />
            </div>
            
            <Select value={filterProvider} onValueChange={setFilterProvider}>
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
            
            <Select value={filterStatus} onValueChange={setFilterStatus}>
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
              <Button variant="ghost" size="sm" onClick={clearFilters} className="h-8">
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
            ) : paginatedAccounts.length === 0 ? (
              <div className="flex flex-col items-center justify-center h-full text-muted-foreground">
                <Mail className="h-12 w-12 mb-4 opacity-50" />
                <p>{hasFilters ? '没有符合条件的账号' : '该分组暂无账号'}</p>
                {!hasFilters && (
                  <Button variant="outline" className="mt-4" onClick={handleAddAccount}>
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
                            paginatedAccounts.length > 0 &&
                            paginatedAccounts.every((acc) => selectedAccountUids.includes(acc.uid))
                          }
                          onCheckedChange={handleSelectAll}
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
                  {paginatedAccounts.map((account) => {
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
                        onDragStart={(e) => handleDragStart(e, account.uid)}
                        onDragEnd={handleDragEnd}
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
                                handleSelectAccount(account.uid, checked as boolean)
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
                                <DropdownMenuItem onClick={() => handleCancelSync(account.uid)}>
                                  <Square className="h-4 w-4 mr-2" />
                                  取消同步
                                </DropdownMenuItem>
                              ) : (
                                // 仅在账号未禁用时显示"立即同步"
                                account.status !== 'disabled' && (
                                  <DropdownMenuItem onClick={() => handleSyncAccount(account.uid)}>
                                    <RefreshCw className="h-4 w-4 mr-2" />
                                    立即同步
                                  </DropdownMenuItem>
                                )
                              )}
                              <DropdownMenuItem onClick={() => handleEditAccount(account)}>
                                <Pencil className="h-4 w-4 mr-2" />
                                编辑
                              </DropdownMenuItem>
                              {/* 查看子邮箱（仅 WebAPI 账户且未禁用时显示） */}
                              {account.status !== 'disabled' &&
                               (account.protocol === 'webapi' || 
                                (account.email && /^(cloudflare|cloud_mail|webapi)[-_]/.test(account.email)) ||
                                account.provider === 'Cloud Mail' ||
                                account.provider === 'Cloudflare Temp Email') && (
                                <DropdownMenuItem onClick={() => {
                                  setViewingChildAccountsAccount(account);
                                  setChildAccountsDialogOpen(true);
                                }}>
                                  <Inbox className="h-4 w-4 mr-2" />
                                  查看子邮箱
                                </DropdownMenuItem>
                              )}
                              <DropdownMenuItem 
                                onClick={async () => {
                                  await toggleAccountStatus(account.uid, account.status);
                                  await fetchAccountsWithFilter();
                                }}
                              >
                                <Power className="h-4 w-4 mr-2" />
                                {account.status === 'active' ? '禁用' : '启用'}
                              </DropdownMenuItem>
                              {account.last_sync_error && (
                                <DropdownMenuItem onClick={async () => {
                                  await clearSyncError(account.uid);
                                  await fetchAccountsWithFilter();
                                }}>
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
                                <DropdownMenuItem onClick={() => handleReauthorize(account.uid, account.provider)}>
                                  <RefreshCw className="h-4 w-4 mr-2" />
                                  重新授权
                                </DropdownMenuItem>
                              )}
                              <DropdownMenuSeparator />
                              <DropdownMenuItem 
                                className="text-destructive"
                                onClick={() => handleDeleteClick(account.uid, account.email)}
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
                  onValueChange={(value) => {
                    setPageSize(Number(value));
                    setCurrentPage(1);
                  }}
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
                  onClick={() => setCurrentPage((p) => Math.max(1, p - 1))}
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
                  onClick={() => setCurrentPage((p) => Math.min(totalPages, p + 1))}
                  disabled={currentPage >= totalPages}
                >
                  <ChevronRight className="h-4 w-4" />
                </Button>
              </div>
            </div>
          )}
        </div>
      </div>

      {/* 分组对话框 */}
      <GroupDialog
        open={groupDialogOpen}
        onOpenChange={setGroupDialogOpen}
        group={editingGroup}
        onSubmit={handleGroupSubmit}
        isSubmitting={isSubmitting}
      />

      <GroupDeleteDialog
        open={deleteDialogOpen}
        onOpenChange={setDeleteDialogOpen}
        group={deletingGroup}
        onConfirm={handleConfirmDeleteGroup}
        isDeleting={isSubmitting}
      />

      <GroupBatchAssign
        open={batchAssignOpen}
        onOpenChange={setBatchAssignOpen}
        accountUids={selectedAccountUids}
        onComplete={handleBatchComplete}
      />

      {/* 账户表单对话框 */}
      <AccountForm
        open={isAccountDialogOpen}
        onClose={handleCloseAccountDialog}
        onSubmit={handleAccountSubmit}
        onOAuth2Success={refreshAllData}
        account={editingAccount}
      />

      {/* 批量导入 Outlook 对话框 */}
      <BatchImportDialog
        open={batchImportOpen}
        onClose={() => setBatchImportOpen(false)}
        onImport={handleBatchImport}
      />

      {/* WebAPI Provider 对话框 */}
      <WebAPIProviderDialog
        open={webAPIDialogOpen}
        onOpenChange={(open) => {
          setWebAPIDialogOpen(open);
          // 关闭对话框时重置预选服务类型
          if (!open) {
            setWebAPIServiceType(null);
          }
        }}
        onSuccess={refreshAllData}
        preselectedServiceType={webAPIServiceType || undefined}
      />

      {/* WebAPI 账户编辑对话框 */}
      <WebAPIAccountEditDialog
        open={webAPIEditDialogOpen}
        onOpenChange={(open) => {
          setWebAPIEditDialogOpen(open);
          if (!open) setEditingWebAPIAccount(null);
        }}
        account={editingWebAPIAccount}
        onSuccess={refreshAllData}
      />

      {/* 子邮箱列表对话框 */}
      <ChildAccountsDialog
        open={childAccountsDialogOpen}
        onOpenChange={(open) => {
          setChildAccountsDialogOpen(open);
          if (!open) setViewingChildAccountsAccount(null);
        }}
        account={viewingChildAccountsAccount}
      />

      {/* 删除确认对话框 */}
      <AlertDialog open={!!deletingAccount} onOpenChange={(open) => !open && handleDeleteCancel()}>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>确认删除账户</AlertDialogTitle>
            <AlertDialogDescription className="space-y-4">
              <div>
                确定要删除账户 <span className="font-semibold text-foreground">{deletingAccount?.email}</span> 吗？
                <br /><br />
                账户将被移入回收站，您可以在回收站中恢复或永久删除。
              </div>
              <div className="space-y-2">
                <label htmlFor="confirm-email" className="text-sm font-medium text-foreground">
                  请输入邮箱地址以确认删除：
                </label>
                <Input
                  id="confirm-email"
                  type="email"
                  placeholder={deletingAccount?.email}
                  value={deleteConfirmEmail}
                  onChange={(e) => setDeleteConfirmEmail(e.target.value)}
                  className="w-full text-foreground"
                  autoComplete="off"
                />
              </div>
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel onClick={handleDeleteCancel}>取消</AlertDialogCancel>
            <AlertDialogAction
              onClick={handleDeleteConfirm}
              disabled={deleteConfirmEmail !== deletingAccount?.email}
              className="bg-red-600 hover:bg-red-700 focus:ring-red-600 disabled:opacity-50 disabled:cursor-not-allowed"
            >
              删除
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </div>
  );
};
