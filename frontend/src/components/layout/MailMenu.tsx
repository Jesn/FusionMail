/**
 * MailMenu 组件 - 用户菜单（邮件视图）
 * 显示邮件相关的功能入口：写邮件、搜索、文件夹、邮箱账户
 */
import { Inbox, Star, Archive, Trash2, Search, Users, ChevronDown, ChevronRight, ShieldAlert, Folder, FolderOpen, PenSquare, Send } from 'lucide-react';
import { Button } from '../ui/button';
import { ScrollArea } from '../ui/scroll-area';
import { Separator } from '../ui/separator';
import { Badge } from '../ui/badge';
import { Collapsible, CollapsibleContent, CollapsibleTrigger } from '../ui/collapsible';

import { useAccounts } from '../../hooks/useAccounts';
import { useEmailStore } from '../../stores/emailStore';
import { useGroupStore, ALL_ACCOUNTS_GROUP_ID, UNGROUPED_GROUP_ID } from '../../stores/groupStore';

import { cn } from '../../lib/utils';
import { useNavigate, useLocation } from 'react-router-dom';
import { useState, useEffect } from 'react';
import { ComposeEmail } from '../email/ComposeEmail';

// 定义菜单组类型（仅邮件视图相关）
type MailMenuSection = 'folders' | 'accounts';

export const MailMenu = () => {
  const navigate = useNavigate();
  const location = useLocation();
  const { accounts } = useAccounts();
  const activeAccounts = (accounts || []).filter((account) => !account.deleted_at);
  const { filter, setFilter, unreadCount, starredCount, archivedCount, deletedCount, spamCount } = useEmailStore();
  
  // 分组状态
  const {
    groups,
    selectedGroupId,
    fetchGroups,
    setSelectedGroupId,
  } = useGroupStore();

  // 菜单组展开状态（独立控制，默认全部展开）
  const [openSections, setOpenSections] = useState<Record<MailMenuSection, boolean>>(() => {
    const saved = localStorage.getItem('sidebar-mail-open-sections');
    if (saved) {
      try {
        return JSON.parse(saved);
      } catch {
        return { folders: true, accounts: true };
      }
    }
    return { folders: true, accounts: true }; // 默认全部展开
  });

  // 写邮件对话框状态
  const [composeOpen, setComposeOpen] = useState(false);

  // 加载分组列表
  useEffect(() => {
    fetchGroups();
  }, [fetchGroups]);

  // 切换菜单组展开状态（独立控制）
  const handleSectionToggle = (section: MailMenuSection) => {
    const newSections = { ...openSections, [section]: !openSections[section] };
    setOpenSections(newSections);
    localStorage.setItem('sidebar-mail-open-sections', JSON.stringify(newSections));
  };

  const folders = [
    { id: 'inbox', name: '收件箱', icon: Inbox, count: unreadCount, showCount: true },
    { id: 'sent', name: '已发送', icon: Send, count: 0, showCount: false, route: '/sent' },
    { id: 'starred', name: '星标邮件', icon: Star, count: starredCount, showCount: false },
    { id: 'archived', name: '归档', icon: Archive, count: archivedCount, showCount: false },
    { id: 'spam', name: '垃圾邮件', icon: ShieldAlert, count: spamCount, showCount: true, route: '/spam' },
    { id: 'trash', name: '回收站', icon: Trash2, count: deletedCount, showCount: false },
  ];

  const handleFolderClick = (folderId: string, route?: string) => {
    if (route) {
      navigate(route);
      return;
    }

    const newFilter: any = {};
    if (filter.account_uid) {
      newFilter.account_uid = filter.account_uid;
    }

    switch (folderId) {
      case 'inbox':
        newFilter.is_archived = false;
        newFilter.is_deleted = false;
        delete newFilter.is_starred;
        break;
      case 'starred':
        newFilter.is_starred = true;
        newFilter.is_deleted = false;
        break;
      case 'archived':
        newFilter.is_archived = true;
        newFilter.is_deleted = false;
        break;
      case 'trash':
        newFilter.is_deleted = true;
        delete newFilter.is_starred;
        delete newFilter.is_archived;
        break;
    }

    setFilter(newFilter);
    navigate('/inbox');
  };

  // 计算未分组账号数量
  const ungroupedCount = activeAccounts.filter((account) => !account.group_id).length;
  const totalAccountCount = activeAccounts.length;

  // 处理分组选择
  const handleGroupSelect = (groupId: number) => {
    setSelectedGroupId(groupId);
    
    const newFilter = { ...filter };
    delete newFilter.account_uid;
    
    if (groupId === ALL_ACCOUNTS_GROUP_ID) {
      delete newFilter.group_id;
    } else {
      newFilter.group_id = groupId;
    }
    
    if (!newFilter.is_starred && !newFilter.is_archived && !newFilter.is_deleted) {
      newFilter.is_archived = false;
      newFilter.is_deleted = false;
    }
    
    setFilter(newFilter);
    navigate('/inbox');
  };

  return (
    <aside className="flex w-64 flex-col border-r bg-background">
      {/* Logo */}
      <div className="flex h-16 items-center justify-between border-b px-3">
        <div 
          className="flex items-center cursor-pointer hover:opacity-80 transition-opacity"
          onClick={() => {
            if (window.location.pathname === '/inbox') {
              setFilter({ is_archived: false, is_deleted: false });
            } else {
              navigate('/inbox');
            }
          }}
        >
          <img 
            src="/logo.png" 
            alt="FusionMail" 
            className="h-14 w-auto"
          />
        </div>
      </div>

      <ScrollArea className="flex-1">
        <div className="space-y-3 p-3">
          {/* 写邮件按钮 */}
          <div className="space-y-1">
            <Button
              variant="default"
              className="w-full justify-start"
              onClick={() => setComposeOpen(true)}
            >
              <PenSquare className="mr-2 h-4 w-4" />
              写邮件
            </Button>
          </div>

          {/* 搜索 */}
          <div className="space-y-1">
            <Button
              variant="ghost"
              className="w-full justify-start"
              onClick={() => navigate('/search')}
            >
              <Search className="mr-2 h-4 w-4" />
              搜索邮件
            </Button>
          </div>

          <Separator />

          {/* 文件夹列表 */}
          <div className="space-y-1 pr-2">
            <Collapsible open={openSections.folders} onOpenChange={() => handleSectionToggle('folders')}>
              <CollapsibleTrigger asChild>
                <Button variant="ghost" className="w-full justify-start mb-0.5 h-8 px-2">
                  <span className="flex-1 text-left text-xs font-medium text-muted-foreground">文件夹</span>
                  {openSections.folders ? (
                    <ChevronDown className="h-3.5 w-3.5 text-muted-foreground" />
                  ) : (
                    <ChevronRight className="h-3.5 w-3.5 text-muted-foreground" />
                  )}
                </Button>
              </CollapsibleTrigger>
              <CollapsibleContent className="space-y-1">
                {folders.map((folder) => {
                  const Icon = folder.icon;
                  const folderRoute = (folder as any).route;
                  const searchParams = new URLSearchParams(location.search);
                  const fromParam = searchParams.get('from');
                  const isActive = 
                    (folder.id === 'inbox' && !filter.is_starred && !filter.is_archived && !filter.is_deleted && location.pathname === '/inbox') ||
                    (folder.id === 'starred' && filter.is_starred) ||
                    (folder.id === 'archived' && filter.is_archived) ||
                    (folder.id === 'spam' && (location.pathname === '/spam' || fromParam === 'spam')) ||
                    (folder.id === 'trash' && (filter.is_deleted || fromParam === 'trash'));

                  return (
                    <Button
                      key={folder.id}
                      variant={isActive ? 'secondary' : 'ghost'}
                      className={cn('w-full justify-start', isActive && 'bg-secondary')}
                      onClick={() => handleFolderClick(folder.id, folderRoute)}
                    >
                      <Icon className="mr-2 h-4 w-4" />
                      <span className="flex-1 text-left">{folder.name}</span>
                      {folder.showCount && folder.count !== undefined && folder.count > 0 && (
                        <Badge variant="secondary" className="ml-auto">{folder.count}</Badge>
                      )}
                    </Button>
                  );
                })}
              </CollapsibleContent>
            </Collapsible>
          </div>

          <Separator />

          {/* 邮箱账户 */}
          <div className="space-y-1 pr-2">
            <Collapsible open={openSections.accounts} onOpenChange={() => handleSectionToggle('accounts')}>
              <CollapsibleTrigger asChild>
                <Button variant="ghost" className="w-full justify-start mb-1 h-8 px-2">
                  <span className="flex-1 text-left text-xs font-medium text-muted-foreground">邮箱账户</span>
                  {openSections.accounts ? (
                    <ChevronDown className="h-3.5 w-3.5 text-muted-foreground" />
                  ) : (
                    <ChevronRight className="h-3.5 w-3.5 text-muted-foreground" />
                  )}
                </Button>
              </CollapsibleTrigger>
              <CollapsibleContent className="space-y-1">
                {/* 所有邮箱 */}
                <Button
                  variant={selectedGroupId === ALL_ACCOUNTS_GROUP_ID ? 'secondary' : 'ghost'}
                  className={cn('w-full justify-start', selectedGroupId === ALL_ACCOUNTS_GROUP_ID && 'bg-secondary')}
                  onClick={() => handleGroupSelect(ALL_ACCOUNTS_GROUP_ID)}
                >
                  <Users className="mr-2 h-4 w-4" />
                  <span className="flex-1 text-left">所有邮箱</span>
                  <Badge variant="secondary" className="ml-auto text-xs">{totalAccountCount}</Badge>
                </Button>

                {/* 只有存在分组时才显示分组相关内容 */}
                {groups.length > 0 && (
                  <>
                    {/* 未分组（只有存在分组时才显示） */}
                    {ungroupedCount > 0 && (
                      <Button
                        variant={selectedGroupId === UNGROUPED_GROUP_ID ? 'secondary' : 'ghost'}
                        className={cn('w-full justify-start', selectedGroupId === UNGROUPED_GROUP_ID && 'bg-secondary')}
                        onClick={() => handleGroupSelect(UNGROUPED_GROUP_ID)}
                      >
                        <Folder className="mr-2 h-4 w-4" />
                        <span className="flex-1 text-left">未分组</span>
                        <Badge variant="secondary" className="ml-auto text-xs">{ungroupedCount}</Badge>
                      </Button>
                    )}

                    {/* 分组列表 */}
                    {groups.map((group) => {
                      const isSelected = selectedGroupId === group.id;
                      const GroupIcon = isSelected ? FolderOpen : Folder;
                      return (
                        <Button
                          key={group.id}
                          variant={isSelected ? 'secondary' : 'ghost'}
                          className={cn('w-full justify-start h-9 px-2', isSelected && 'bg-secondary')}
                          onClick={() => handleGroupSelect(group.id)}
                        >
                          <GroupIcon className="mr-2 h-4 w-4" />
                          <span className="flex-1 text-left truncate">{group.name}</span>
                          <Badge variant="secondary" className="ml-auto text-xs">{group.account_count}</Badge>
                        </Button>
                      );
                    })}
                  </>
                )}
              </CollapsibleContent>
            </Collapsible>
          </div>
        </div>
      </ScrollArea>

      {/* 写邮件对话框 */}
      <ComposeEmail
        open={composeOpen}
        onOpenChange={setComposeOpen}
        mode="new"
      />
    </aside>
  );
};
