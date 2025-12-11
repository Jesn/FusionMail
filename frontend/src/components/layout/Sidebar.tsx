import { Inbox, Star, Archive, Trash2, Mail, Settings, Zap, Webhook, Search, Users, Key, Shield, Server, ChevronDown, ChevronRight, ShieldAlert, Folder, FolderOpen } from 'lucide-react';
import { Button } from '../ui/button';
import { ScrollArea } from '../ui/scroll-area';
import { Separator } from '../ui/separator';
import { Badge } from '../ui/badge';
import { Collapsible, CollapsibleContent, CollapsibleTrigger } from '../ui/collapsible';

import { useUIStore } from '../../stores/uiStore';
import { useAccounts } from '../../hooks/useAccounts';
import { useEmailStore } from '../../stores/emailStore';
import { useGroupStore, ALL_ACCOUNTS_GROUP_ID, UNGROUPED_GROUP_ID } from '../../stores/groupStore';

import { cn } from '../../lib/utils';
import { useNavigate, useLocation } from 'react-router-dom';
import { useState, useEffect } from 'react';

// 定义菜单组类型
type OpenSection = 'folders' | 'accounts' | 'management' | 'advanced' | null;

export const Sidebar = () => {
  const navigate = useNavigate();
  const location = useLocation();
  const { sidebarCollapsed } = useUIStore();
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



  // 菜单组展开状态（手风琴模式：同一时间只能展开一个）
  const [openSection, setOpenSection] = useState<OpenSection>(() => {
    const saved = localStorage.getItem('sidebar-open-section');
    return (saved as OpenSection) || 'folders'; // 默认展开文件夹
  });

  // 加载分组列表
  useEffect(() => {
    fetchGroups();
  }, [fetchGroups]);

  // 如果当前在设置或高级配置页面，自动展开对应菜单
  useEffect(() => {
    if (location.pathname.startsWith('/settings')) {
      setOpenSection('management');
    } else if (['/api-keys', '/providers', '/oauth2-clients', '/email-list'].includes(location.pathname)) {
      setOpenSection('advanced');
    }
  }, [location.pathname]);

  // 切换菜单组展开状态
  const handleSectionToggle = (section: OpenSection) => {
    const newSection = openSection === section ? null : section;
    setOpenSection(newSection);
    localStorage.setItem('sidebar-open-section', newSection || '');
  };

  const folders = [
    { id: 'inbox', name: '收件箱', icon: Inbox, count: unreadCount, showCount: true },
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



  if (sidebarCollapsed) {
    return null;
  }


  return (
    <aside className="flex w-64 flex-col border-r bg-background">
      {/* Logo */}
      <div className="flex h-16 items-center justify-between border-b px-4">
        <div 
          className="flex items-center gap-2 cursor-pointer hover:opacity-80 transition-opacity"
          onClick={() => {
            if (window.location.pathname === '/inbox') {
              setFilter({ is_archived: false, is_deleted: false });
            } else {
              navigate('/inbox');
            }
          }}
        >
          <Mail className="h-6 w-6 text-primary" />
          <span className="text-lg font-medium">FusionMail</span>
        </div>
      </div>

      <ScrollArea className="flex-1">
        <div className="space-y-3 p-3">
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
            <Collapsible open={openSection === 'folders'} onOpenChange={() => handleSectionToggle('folders')}>
              <CollapsibleTrigger asChild>
                <Button variant="ghost" className="w-full justify-start mb-0.5 h-8 px-2">
                  <span className="flex-1 text-left text-xs font-medium text-muted-foreground">文件夹</span>
                  {openSection === 'folders' ? (
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

          {/* 邮箱账户（方案C：只显示分组，不显示账户列表） */}
          <div className="space-y-1 pr-2">
            <Collapsible open={openSection === 'accounts'} onOpenChange={() => handleSectionToggle('accounts')}>
              <CollapsibleTrigger asChild>
                <Button variant="ghost" className="w-full justify-start mb-1 h-8 px-2">
                  <span className="flex-1 text-left text-xs font-medium text-muted-foreground">邮箱账户</span>
                  {openSection === 'accounts' ? (
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
                    <div className="border-t my-2 pt-2">
                      <div className="px-2 mb-1">
                        <span className="text-xs text-muted-foreground">分组</span>
                      </div>
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
                    </div>
                  </>
                )}


              </CollapsibleContent>
            </Collapsible>
          </div>

          <Separator />


          {/* 管理（常用功能） */}
          <div className="space-y-1 pr-2">
            <Collapsible open={openSection === 'management'} onOpenChange={() => handleSectionToggle('management')}>
              <CollapsibleTrigger asChild>
                <Button variant="ghost" className="w-full justify-start mb-1 h-8 px-2">
                  <span className="flex-1 text-left text-xs font-medium text-muted-foreground">管理</span>
                  {openSection === 'management' ? (
                    <ChevronDown className="h-3.5 w-3.5 text-muted-foreground" />
                  ) : (
                    <ChevronRight className="h-3.5 w-3.5 text-muted-foreground" />
                  )}
                </Button>
              </CollapsibleTrigger>
              <CollapsibleContent className="space-y-1">
                <Button
                  variant={location.pathname === '/accounts' ? 'secondary' : 'ghost'}
                  className={cn('w-full justify-start', location.pathname === '/accounts' && 'bg-secondary')}
                  onClick={() => navigate('/accounts')}
                >
                  <Mail className="mr-2 h-4 w-4" />
                  账户管理
                </Button>
                <Button
                  variant={location.pathname === '/groups' ? 'secondary' : 'ghost'}
                  className={cn('w-full justify-start', location.pathname === '/groups' && 'bg-secondary')}
                  onClick={() => navigate('/groups')}
                >
                  <Folder className="mr-2 h-4 w-4" />
                  分组管理
                </Button>
                <Button
                  variant={location.pathname === '/trash' ? 'secondary' : 'ghost'}
                  className={cn('w-full justify-start', location.pathname === '/trash' && 'bg-secondary')}
                  onClick={() => navigate('/trash')}
                >
                  <Trash2 className="mr-2 h-4 w-4" />
                  已删除账户
                </Button>
                <Button
                  variant={location.pathname === '/rules' ? 'secondary' : 'ghost'}
                  className={cn('w-full justify-start', location.pathname === '/rules' && 'bg-secondary')}
                  onClick={() => navigate('/rules')}
                >
                  <Zap className="mr-2 h-4 w-4" />
                  邮件规则
                </Button>
                <Button
                  variant={location.pathname === '/webhooks' ? 'secondary' : 'ghost'}
                  className={cn('w-full justify-start', location.pathname === '/webhooks' && 'bg-secondary')}
                  onClick={() => navigate('/webhooks')}
                >
                  <Webhook className="mr-2 h-4 w-4" />
                  Webhook
                </Button>
                <Button
                  variant={location.pathname === '/settings' ? 'secondary' : 'ghost'}
                  className={cn('w-full justify-start', location.pathname === '/settings' && 'bg-secondary')}
                  onClick={() => navigate('/settings')}
                >
                  <Settings className="mr-2 h-4 w-4" />
                  个人设置
                </Button>
                <Button
                  variant={location.pathname === '/settings/system' ? 'secondary' : 'ghost'}
                  className={cn('w-full justify-start', location.pathname === '/settings/system' && 'bg-secondary')}
                  onClick={() => navigate('/settings/system')}
                >
                  <Server className="mr-2 h-4 w-4" />
                  系统设置
                </Button>
              </CollapsibleContent>
            </Collapsible>
          </div>

          <Separator />

          {/* 高级配置（开发者/高级用户） */}
          <div className="space-y-1 pr-2">
            <Collapsible open={openSection === 'advanced'} onOpenChange={() => handleSectionToggle('advanced')}>
              <CollapsibleTrigger asChild>
                <Button variant="ghost" className="w-full justify-start mb-1 h-8 px-2">
                  <span className="flex-1 text-left text-xs font-medium text-muted-foreground">高级配置</span>
                  {openSection === 'advanced' ? (
                    <ChevronDown className="h-3.5 w-3.5 text-muted-foreground" />
                  ) : (
                    <ChevronRight className="h-3.5 w-3.5 text-muted-foreground" />
                  )}
                </Button>
              </CollapsibleTrigger>
              <CollapsibleContent className="space-y-1">
                <Button
                  variant={location.pathname === '/api-keys' ? 'secondary' : 'ghost'}
                  className={cn('w-full justify-start', location.pathname === '/api-keys' && 'bg-secondary')}
                  onClick={() => navigate('/api-keys')}
                >
                  <Key className="mr-2 h-4 w-4" />
                  API Key
                </Button>
                <Button
                  variant={location.pathname === '/providers' ? 'secondary' : 'ghost'}
                  className={cn('w-full justify-start', location.pathname === '/providers' && 'bg-secondary')}
                  onClick={() => navigate('/providers')}
                >
                  <Server className="mr-2 h-4 w-4" />
                  邮箱提供商
                </Button>
                <Button
                  variant={location.pathname === '/oauth2-clients' ? 'secondary' : 'ghost'}
                  className={cn('w-full justify-start', location.pathname === '/oauth2-clients' && 'bg-secondary')}
                  onClick={() => navigate('/oauth2-clients')}
                >
                  <Shield className="mr-2 h-4 w-4" />
                  OAuth2 客户端
                </Button>
                <Button
                  variant={location.pathname === '/email-list' ? 'secondary' : 'ghost'}
                  className={cn('w-full justify-start', location.pathname === '/email-list' && 'bg-secondary')}
                  onClick={() => navigate('/email-list')}
                >
                  <Shield className="mr-2 h-4 w-4" />
                  白名单/黑名单
                </Button>
              </CollapsibleContent>
            </Collapsible>
          </div>
        </div>
      </ScrollArea>


    </aside>
  );
};
