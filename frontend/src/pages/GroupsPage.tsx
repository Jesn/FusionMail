import { useState, useEffect, useMemo, useCallback } from 'react';
import { Plus, FolderOpen, Folder, Users, MoreHorizontal, Pencil, Trash2, ArrowRightLeft, Mail, FolderInput, GripVertical, ChevronLeft, ChevronRight, Search, X, Loader2 } from 'lucide-react';
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
  DropdownMenuTrigger,
} from '../components/ui/dropdown-menu';
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '../components/ui/select';
import { GroupDialog, GroupDeleteDialog, GroupBatchAssign } from '../components/group';
import { useGroupStore, ALL_ACCOUNTS_GROUP_ID, UNGROUPED_GROUP_ID } from '../stores/groupStore';
import { useAccounts } from '../hooks/useAccounts';
import { cn } from '../lib/utils';
import type { Account, AccountGroupWithCount } from '../types';
import { groupService } from '../services/groupService';
import { accountService, AccountListResponse } from '../services/accountService';
import toast from 'react-hot-toast';

export const GroupsPage = () => {
  const { groups, selectedGroupId, setSelectedGroupId, fetchGroups, createGroup, editGroup, deleteGroup } = useGroupStore();
  const { accounts, loadAccounts } = useAccounts();
  
  // 对话框状态
  const [groupDialogOpen, setGroupDialogOpen] = useState(false);
  const [editingGroup, setEditingGroup] = useState<AccountGroupWithCount | null>(null);
  const [deleteDialogOpen, setDeleteDialogOpen] = useState(false);
  const [deletingGroup, setDeletingGroup] = useState<AccountGroupWithCount | null>(null);
  const [batchAssignOpen, setBatchAssignOpen] = useState(false);
  const [isSubmitting, setIsSubmitting] = useState(false);
  
  // 表格选择状态
  const [selectedAccountUids, setSelectedAccountUids] = useState<string[]>([]);
  
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

  // 确保数据已加载（用于左侧分组列表的计数）
  const safeAccounts = accounts || [];
  const activeAccounts = safeAccounts.filter((acc) => !acc.deleted_at);

  // 加载分组数据
  useEffect(() => {
    useGroupStore.getState().setCacheTimestamp(0);
    fetchGroups();
    loadAccounts(true); // 仍然加载全部账号用于左侧分组计数
  }, [fetchGroups, loadAccounts]);

  // 计算各分组的账号（用于左侧分组列表）
  const ungroupedAccounts = useMemo(() => 
    activeAccounts.filter((acc) => !acc.group_id), 
    [activeAccounts]
  );

  // 获取所有提供商列表（用于筛选下拉框）
  const providerList = useMemo(() => {
    const providers = new Set(activeAccounts.map((acc) => acc.provider));
    return Array.from(providers).sort();
  }, [activeAccounts]);

  // 后端分页和筛选请求
  const fetchAccountsWithFilter = useCallback(async () => {
    setIsLoadingAccounts(true);
    try {
      // 构建筛选参数
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

      // 分组筛选
      if (selectedGroupId === ALL_ACCOUNTS_GROUP_ID) {
        filter.group_id = -1; // 所有账号
      } else if (selectedGroupId === UNGROUPED_GROUP_ID) {
        filter.group_id = 0; // 未分组
      } else {
        filter.group_id = selectedGroupId; // 具体分组
      }

      // 邮箱搜索
      if (searchEmail.trim()) {
        filter.email = searchEmail.trim();
      }

      // 提供商筛选
      if (filterProvider !== 'all') {
        filter.provider = filterProvider;
      }

      // 状态筛选
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

  // 当筛选条件变化时重新请求
  useEffect(() => {
    fetchAccountsWithFilter();
  }, [fetchAccountsWithFilter]);

  // 切换分组时重置页码
  useEffect(() => {
    setCurrentPage(1);
    setSelectedAccountUids([]);
  }, [selectedGroupId]);

  // 筛选条件变化时重置页码（使用防抖）
  useEffect(() => {
    const timer = setTimeout(() => {
      setCurrentPage(1);
    }, 300);
    return () => clearTimeout(timer);
  }, [searchEmail, filterProvider, filterStatus]);

  // 清除所有筛选
  const clearFilters = () => {
    setSearchEmail('');
    setFilterProvider('all');
    setFilterStatus('all');
  };

  // 是否有筛选条件
  const hasFilters = searchEmail.trim() || filterProvider !== 'all' || filterStatus !== 'all';

  // 当前分组名称
  const currentGroupName = useMemo(() => {
    if (selectedGroupId === ALL_ACCOUNTS_GROUP_ID) return '所有账号';
    if (selectedGroupId === UNGROUPED_GROUP_ID) return '未分组';
    const group = groups.find((g) => g.id === selectedGroupId);
    return group?.name || '未知分组';
  }, [selectedGroupId, groups]);

  // 处理创建分组
  const handleCreateGroup = () => {
    setEditingGroup(null);
    setGroupDialogOpen(true);
  };

  // 处理编辑分组
  const handleEditGroup = (group: AccountGroupWithCount) => {
    setEditingGroup(group);
    setGroupDialogOpen(true);
  };

  // 处理删除分组
  const handleDeleteGroup = (group: AccountGroupWithCount) => {
    setDeletingGroup(group);
    setDeleteDialogOpen(true);
  };

  // 提交分组表单
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
      toast.error(error instanceof Error ? error.message : '操作失败');
    } finally {
      setIsSubmitting(false);
    }
  };

  // 确认删除分组
  const handleConfirmDelete = async () => {
    if (!deletingGroup) return;
    setIsSubmitting(true);
    try {
      await deleteGroup(deletingGroup.id);
      toast.success('分组删除成功');
      setDeleteDialogOpen(false);
      setDeletingGroup(null);
      // 如果删除的是当前选中的分组，切换到所有账号
      if (selectedGroupId === deletingGroup.id) {
        setSelectedGroupId(ALL_ACCOUNTS_GROUP_ID);
      }
    } catch (error) {
      toast.error(error instanceof Error ? error.message : '删除失败');
    } finally {
      setIsSubmitting(false);
    }
  };

  // 批量移动完成后刷新
  const handleBatchComplete = async () => {
    setSelectedAccountUids([]);
    await fetchGroups();
    await loadAccounts(true);
    await fetchAccountsWithFilter(); // 刷新当前列表
  };

  // 将选中账号加入当前分组
  const handleAddToCurrentGroup = async () => {
    if (selectedGroupId <= 0 || selectedAccountUids.length === 0) return;
    
    setIsSubmitting(true);
    try {
      // 批量将账号加入当前分组
      await Promise.all(
        selectedAccountUids.map((uid) => 
          groupService.assignAccountToGroup(uid, selectedGroupId)
        )
      );
      toast.success(`已将 ${selectedAccountUids.length} 个账号加入分组`);
      setSelectedAccountUids([]);
      await fetchGroups();
      await loadAccounts(true);
      await fetchAccountsWithFilter(); // 刷新当前列表
    } catch (error) {
      toast.error(error instanceof Error ? error.message : '操作失败');
    } finally {
      setIsSubmitting(false);
    }
  };

  // 全选/取消全选（当前页）
  const handleSelectAll = (checked: boolean) => {
    if (checked) {
      setSelectedAccountUids(paginatedAccounts.map((acc) => acc.uid));
    } else {
      setSelectedAccountUids([]);
    }
  };

  // 单个选择
  const handleSelectAccount = (uid: string, checked: boolean) => {
    if (checked) {
      setSelectedAccountUids((prev) => [...prev, uid]);
    } else {
      setSelectedAccountUids((prev) => prev.filter((id) => id !== uid));
    }
  };

  // 拖拽开始
  const handleDragStart = useCallback((e: React.DragEvent, accountUid: string) => {
    setDraggedAccountUid(accountUid);
    e.dataTransfer.effectAllowed = 'move';
    e.dataTransfer.setData('text/plain', accountUid);
    // 添加拖拽时的视觉效果
    if (e.currentTarget instanceof HTMLElement) {
      e.currentTarget.style.opacity = '0.5';
    }
  }, []);

  // 拖拽结束
  const handleDragEnd = useCallback((e: React.DragEvent) => {
    setDraggedAccountUid(null);
    setDragOverGroupId(null);
    if (e.currentTarget instanceof HTMLElement) {
      e.currentTarget.style.opacity = '1';
    }
  }, []);

  // 拖拽经过分组
  const handleDragOver = useCallback((e: React.DragEvent, groupId: number) => {
    e.preventDefault();
    e.dataTransfer.dropEffect = 'move';
    setDragOverGroupId(groupId);
  }, []);

  // 拖拽离开分组
  const handleDragLeave = useCallback(() => {
    setDragOverGroupId(null);
  }, []);

  // 放置到分组
  const handleDrop = useCallback(async (e: React.DragEvent, targetGroupId: number) => {
    e.preventDefault();
    const accountUid = e.dataTransfer.getData('text/plain');
    
    if (!accountUid) return;
    
    // 如果是"所有账号"，不允许放置
    if (targetGroupId === ALL_ACCOUNTS_GROUP_ID) {
      toast.error('请选择具体分组或"未分组"');
      setDragOverGroupId(null);
      setDraggedAccountUid(null);
      return;
    }
    
    try {
      // 未分组 = null，具体分组 = groupId
      const newGroupId = targetGroupId === UNGROUPED_GROUP_ID ? null : targetGroupId;
      await groupService.assignAccountToGroup(accountUid, newGroupId);
      
      const targetName = targetGroupId === UNGROUPED_GROUP_ID 
        ? '未分组' 
        : groups.find(g => g.id === targetGroupId)?.name || '分组';
      toast.success(`已移动到「${targetName}」`);
      
      await fetchGroups();
      await loadAccounts(true);
      await fetchAccountsWithFilter(); // 刷新当前列表
    } catch (error) {
      toast.error(error instanceof Error ? error.message : '移动失败');
    } finally {
      setDragOverGroupId(null);
      setDraggedAccountUid(null);
    }
  }, [groups, fetchGroups, loadAccounts, fetchAccountsWithFilter]);

  // 计算分组的实际账号数量（从账号列表中计算，确保实时准确）
  const getGroupAccountCount = useCallback((groupId: number) => {
    if (groupId === ALL_ACCOUNTS_GROUP_ID) {
      return activeAccounts.length;
    }
    if (groupId === UNGROUPED_GROUP_ID) {
      return ungroupedAccounts.length;
    }
    return activeAccounts.filter((acc) => acc.group_id === groupId).length;
  }, [activeAccounts, ungroupedAccounts]);

  // 渲染分组树项（支持拖拽放置）
  const renderGroupItem = (
    id: number,
    name: string,
    isSpecial: boolean = false,
    group?: AccountGroupWithCount
  ) => {
    const isSelected = selectedGroupId === id;
    const Icon = isSelected ? FolderOpen : Folder;
    const isDragOver = dragOverGroupId === id;
    // "所有账号"不能作为放置目标
    const canDrop = id !== ALL_ACCOUNTS_GROUP_ID;
    // 使用实时计算的账号数量
    const count = getGroupAccountCount(id);

    return (
      <div
        key={id}
        className={cn(
          'group flex items-center justify-between px-3 py-2 rounded-md cursor-pointer transition-all duration-200',
          isSelected
            ? 'bg-primary text-primary-foreground'
            : 'hover:bg-muted',
          // 拖拽悬停效果
          isDragOver && canDrop && 'ring-2 ring-primary ring-offset-2 bg-primary/10',
          isDragOver && !canDrop && 'ring-2 ring-destructive ring-offset-2 bg-destructive/10'
        )}
        onClick={() => {
          setSelectedGroupId(id);
          setSelectedAccountUids([]);
        }}
        // 拖拽放置事件
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
          <Badge 
            variant={isSelected ? 'secondary' : 'outline'} 
            className="text-xs min-w-[24px] justify-center"
          >
            {count}
          </Badge>
          {/* 自定义分组显示下拉菜单，特殊分组显示占位符保持对齐 */}
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
            /* 占位符，保持与下拉菜单按钮相同宽度 */
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
        return <Badge variant="secondary">已禁用</Badge>;
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
            <h1 className="text-2xl font-bold">分组管理</h1>
            <p className="text-sm text-muted-foreground">
              管理邮箱账户分组，便于分类和筛选
            </p>
          </div>
          <Button onClick={handleCreateGroup}>
            <Plus className="mr-2 h-4 w-4" />
            创建分组
          </Button>
        </div>
      </div>

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
            {/* 所有账号 */}
            {renderGroupItem(ALL_ACCOUNTS_GROUP_ID, '所有账号', true)}
            
            {/* 未分组 */}
            {renderGroupItem(UNGROUPED_GROUP_ID, '未分组', true)}
            
            {/* 分隔线 */}
            {groups.length > 0 && <div className="border-t my-2" />}
            
            {/* 自定义分组 */}
            {groups.map((group) =>
              renderGroupItem(group.id, group.name, false, group)
            )}
            
            {/* 空状态 */}
            {groups.length === 0 && (
              <div className="px-3 py-4 text-center text-sm text-muted-foreground">
                暂无自定义分组
              </div>
            )}
          </div>
        </div>

        {/* 右侧：账号表格 */}
        <div className="flex-1 flex flex-col overflow-hidden">
          {/* 表格头部 */}
          <div className="px-6 py-3 border-b flex items-center justify-between bg-background">
            <div className="flex items-center gap-2">
              <h2 className="text-lg font-semibold">{currentGroupName}</h2>
              <Badge variant="outline">{totalAccounts} 个账号</Badge>
            </div>
            {selectedAccountUids.length > 0 && (
              <div className="flex items-center gap-2">
                {/* 如果当前选中的是具体分组，显示"加入当前分组"按钮 */}
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
                  移动到其他分组 ({selectedAccountUids.length})
                </Button>
              </div>
            )}
          </div>

          {/* 筛选栏 */}
          <div className="px-6 py-2 border-b flex items-center gap-3 bg-muted/30">
            {/* 邮箱搜索 */}
            <div className="relative flex-1 max-w-xs">
              <Search className="absolute left-2.5 top-1/2 -translate-y-1/2 h-4 w-4 text-muted-foreground" />
              <Input
                placeholder="搜索邮箱..."
                value={searchEmail}
                onChange={(e) => setSearchEmail(e.target.value)}
                className="pl-8 h-8"
              />
            </div>
            
            {/* 提供商筛选 */}
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
            
            {/* 状态筛选 */}
            <Select value={filterStatus} onValueChange={setFilterStatus}>
              <SelectTrigger className="w-[100px] h-8">
                <SelectValue placeholder="状态" />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="all">全部状态</SelectItem>
                <SelectItem value="active">正常</SelectItem>
                <SelectItem value="disabled">已禁用</SelectItem>
                <SelectItem value="error">错误</SelectItem>
              </SelectContent>
            </Select>
            
            {/* 清除筛选 */}
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
              </div>
            ) : (
              <Table>
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
                    <TableHead className="w-48">邮箱地址</TableHead>
                    <TableHead className="w-24">提供商</TableHead>
                    <TableHead className="w-28">所属分组</TableHead>
                    <TableHead className="w-20">状态</TableHead>
                    <TableHead className="w-28">最后同步</TableHead>
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {paginatedAccounts.map((account) => {
                    const accountGroup = account.group_id
                      ? groups.find((g) => g.id === account.group_id)
                      : null;
                    const isDragging = draggedAccountUid === account.uid;
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
                        <TableCell className="font-medium">{account.email}</TableCell>
                        <TableCell>
                          <Badge variant="outline">{account.provider}</Badge>
                        </TableCell>
                        <TableCell>
                          {accountGroup ? (
                            <Badge variant="secondary">{accountGroup.name}</Badge>
                          ) : (
                            <span className="text-muted-foreground">未分组</span>
                          )}
                        </TableCell>
                        <TableCell>{getStatusBadge(account.status)}</TableCell>
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

      {/* 对话框 */}
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
        onConfirm={handleConfirmDelete}
        isDeleting={isSubmitting}
      />

      <GroupBatchAssign
        open={batchAssignOpen}
        onOpenChange={setBatchAssignOpen}
        accountUids={selectedAccountUids}
        onComplete={handleBatchComplete}
      />
    </div>
  );
};
