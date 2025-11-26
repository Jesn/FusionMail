import { Inbox, Star, Archive, Trash2, Mail, Settings, Zap, Webhook, Search, Users, Key, Shield, Server, ChevronDown, ChevronRight, User } from 'lucide-react';
import { Button } from '../ui/button';
import { ScrollArea } from '../ui/scroll-area';
import { Separator } from '../ui/separator';
import { Badge } from '../ui/badge';
import { Collapsible, CollapsibleContent, CollapsibleTrigger } from '../ui/collapsible';
import { useUIStore } from '../../stores/uiStore';
import { useAccounts } from '../../hooks/useAccounts';
import { useEmailStore } from '../../stores/emailStore';
import { cn } from '../../lib/utils';
import { useNavigate, useLocation } from 'react-router-dom';
import { useState, useEffect } from 'react';

// 定义菜单组类型
type OpenSection = 'folders' | 'accounts' | 'management' | null;

export const Sidebar = () => {
  const navigate = useNavigate();
  const location = useLocation();
  const { sidebarCollapsed } = useUIStore();
  const { accounts } = useAccounts();
  const activeAccounts = (accounts || []).filter((account) => !account.deleted_at);
  const { filter, setFilter, unreadCount, starredCount, archivedCount, deletedCount } = useEmailStore();
  
  // 菜单组展开状态（手风琴模式：同一时间只能展开一个）
  const [openSection, setOpenSection] = useState<OpenSection>(() => {
    const saved = localStorage.getItem('sidebar-open-section');
    return (saved as OpenSection) || 'folders'; // 默认展开文件夹
  });

  // 设置子菜单展开状态
  const [settingsOpen, setSettingsOpen] = useState(() => {
    const saved = localStorage.getItem('sidebar-settings-open');
    return saved !== null ? saved === 'true' : true;
  });

  // 如果当前在设置页面，自动展开管理菜单和设置子菜单
  useEffect(() => {
    if (location.pathname.startsWith('/settings') || location.pathname.startsWith('/admin/settings')) {
      setOpenSection('management');
      setSettingsOpen(true);
    }
  }, [location.pathname]);

  // 切换菜单组展开状态
  const handleSectionToggle = (section: OpenSection) => {
    const newSection = openSection === section ? null : section;
    setOpenSection(newSection);
    localStorage.setItem('sidebar-open-section', newSection || '');
  };

  // 保存设置子菜单展开状态到 localStorage
  const handleSettingsToggle = (open: boolean) => {
    setSettingsOpen(open);
    localStorage.setItem('sidebar-settings-open', String(open));
  };

  const folders = [
    { id: 'inbox', name: '收件箱', icon: Inbox, count: unreadCount, showCount: true },
    { id: 'starred', name: '星标邮件', icon: Star, count: starredCount, showCount: false },
    { id: 'archived', name: '归档', icon: Archive, count: archivedCount, showCount: false },
    { id: 'trash', name: '垃圾箱', icon: Trash2, count: deletedCount, showCount: false },
  ];

  const handleFolderClick = (folderId: string) => {
    // 保留当前已选择的账户（如果有）；仅切换文件夹筛选
    const newFilter: any = {};

    // 保留账户筛选，避免切换文件夹后回到“所有邮箱”
    if (filter.account_uid) {
      newFilter.account_uid = filter.account_uid;
    }

    switch (folderId) {
      case 'inbox':
        // 收件箱：清除归档/删除/星标状态
        newFilter.is_archived = false;
        newFilter.is_deleted = false;
        delete newFilter.is_starred;
        break;
      case 'starred':
        // 星标：仅设置星标，且不显示已删除
        newFilter.is_starred = true;
        newFilter.is_deleted = false;
        break;
      case 'archived':
        // 归档：显示已归档，且不显示已删除
        newFilter.is_archived = true;
        newFilter.is_deleted = false;
        break;
      case 'trash':
        // 垃圾箱：仅显示已删除
        newFilter.is_deleted = true;
        // 同时清除与文件夹无关的标记
        delete newFilter.is_starred;
        delete newFilter.is_archived;
        break;
    }

    setFilter(newFilter);
    // 跳转到收件箱页面
    navigate('/inbox');
  };

  const handleAccountClick = (accountUid: string) => {
    // 切换账户时，保持当前文件夹筛选，添加账户筛选
    const newFilter = { ...filter };
    newFilter.account_uid = accountUid;
    
    // 如果当前没有文件夹筛选，默认显示收件箱
    if (!newFilter.is_starred && !newFilter.is_archived && !newFilter.is_deleted) {
      newFilter.is_archived = false;
      newFilter.is_deleted = false;
    }
    
    setFilter(newFilter);
    // 跳转到收件箱页面
    navigate('/inbox');
  };

  const handleAllAccountsClick = () => {
    // 保持当前文件夹筛选，只清除账户筛选
    const newFilter = { ...filter };
    delete newFilter.account_uid;
    
    // 如果当前没有文件夹筛选，默认显示收件箱
    if (!newFilter.is_starred && !newFilter.is_archived && !newFilter.is_deleted) {
      newFilter.is_archived = false;
      newFilter.is_deleted = false;
    }
    
    setFilter(newFilter);
    // 跳转到收件箱页面
    navigate('/inbox');
  };

  if (sidebarCollapsed) {
    return null;
  }

  return (
    <aside className="flex w-64 flex-col border-r bg-background">
      {/* Logo 和新建按钮 */}
      <div className="flex h-16 items-center justify-between border-b px-4">
        <div 
          className="flex items-center gap-2 cursor-pointer hover:opacity-80 transition-opacity"
          onClick={() => {
            // 如果已经在 inbox 页面，清除筛选条件回到默认视图
            if (window.location.pathname === '/inbox') {
              setFilter({
                is_archived: false,
                is_deleted: false,
              });
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

          {/* 文件夹列表（可折叠） */}
          <div className="space-y-1 pr-2">
            <Collapsible open={openSection === 'folders'} onOpenChange={() => handleSectionToggle('folders')}>
              <CollapsibleTrigger asChild>
                <Button
                  variant="ghost"
                  className="w-full justify-start mb-0.5 h-8 px-2"
                >
                  <span className="flex-1 text-left text-xs font-medium text-muted-foreground">
                    文件夹
                  </span>
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
                  const isActive = 
                    (folder.id === 'inbox' && !filter.is_starred && !filter.is_archived && !filter.is_deleted) ||
                    (folder.id === 'starred' && filter.is_starred) ||
                    (folder.id === 'archived' && filter.is_archived) ||
                    (folder.id === 'trash' && filter.is_deleted);

                  return (
                    <Button
                      key={folder.id}
                      variant={isActive ? 'secondary' : 'ghost'}
                      className={cn(
                        'w-full justify-start',
                        isActive && 'bg-secondary'
                      )}
                      onClick={() => handleFolderClick(folder.id)}
                    >
                      <Icon className="mr-2 h-4 w-4" />
                      <span className="flex-1 text-left">{folder.name}</span>
                      {folder.showCount && folder.count !== undefined && folder.count > 0 && (
                        <Badge variant="secondary" className="ml-auto">
                          {folder.count}
                        </Badge>
                      )}
                    </Button>
                  );
                })}
              </CollapsibleContent>
            </Collapsible>
          </div>

          <Separator />

          {/* 账户列表（可折叠） */}
          <div className="space-y-1 pr-2">
            <Collapsible open={openSection === 'accounts'} onOpenChange={() => handleSectionToggle('accounts')}>
              <CollapsibleTrigger asChild>
                <Button
                  variant="ghost"
                  className="w-full justify-start mb-1 h-8 px-2"
                >
                  <span className="flex-1 text-left text-xs font-medium text-muted-foreground">
                    邮箱账户
                  </span>
                  {openSection === 'accounts' ? (
                    <ChevronDown className="h-3.5 w-3.5 text-muted-foreground" />
                  ) : (
                    <ChevronRight className="h-3.5 w-3.5 text-muted-foreground" />
                  )}
                </Button>
              </CollapsibleTrigger>
              <CollapsibleContent className="space-y-1">
                {activeAccounts.length === 0 ? (
                  <div className="px-2 py-4 text-center text-sm text-muted-foreground">
                    暂无账户
                  </div>
                ) : (
                  <>
                    {/* 所有邮箱选项 - 永远在第一行 */}
                    <Button
                      variant={!filter.account_uid ? 'secondary' : 'ghost'}
                      className={cn(
                        'w-full justify-start',
                        !filter.account_uid && 'bg-secondary'
                      )}
                      onClick={handleAllAccountsClick}
                    >
                      <Users className="mr-2 h-4 w-4" />
                      <span className="flex-1 text-left">所有邮箱</span>
                    </Button>

                    {/* 具体账户列表 */}
                    {activeAccounts.map((account) => {
                      const isActive = filter.account_uid === account.uid;
                      const isDisabled = account.status === 'disabled';
                      const hasUnread = account.unread_count && account.unread_count > 0;
                      return (
                        <Button
                          key={account.uid}
                          variant={isActive ? 'secondary' : 'ghost'}
                          className={cn(
                            'w-full justify-start',
                            isActive && 'bg-secondary'
                          )}
                          onClick={() => handleAccountClick(account.uid)}
                          title={account.email}
                        >
                          <div className="flex items-center w-full min-w-0">
                            <Mail className={cn(
                              "mr-2 h-4 w-4 flex-shrink-0",
                              isDisabled && "text-red-500",
                              hasUnread && "font-bold stroke-[2.5]"
                            )} />
                            <span className={cn(
                              "truncate text-left",
                              hasUnread && "font-semibold"
                            )}>
                              {account.email}
                            </span>
                          </div>
                        </Button>
                      );
                    })}
                  </>
                )}
              </CollapsibleContent>
            </Collapsible>
          </div>

          <Separator />

          {/* 管理功能（可折叠） */}
          <div className="space-y-1 pr-2">
            <Collapsible open={openSection === 'management'} onOpenChange={() => handleSectionToggle('management')}>
              <CollapsibleTrigger asChild>
                <Button
                  variant="ghost"
                  className="w-full justify-start mb-1 h-8 px-2"
                >
                  <span className="flex-1 text-left text-xs font-medium text-muted-foreground">
                    管理
                  </span>
                  {openSection === 'management' ? (
                    <ChevronDown className="h-3.5 w-3.5 text-muted-foreground" />
                  ) : (
                    <ChevronRight className="h-3.5 w-3.5 text-muted-foreground" />
                  )}
                </Button>
              </CollapsibleTrigger>
              <CollapsibleContent className="space-y-1">
                <Button
                  variant="ghost"
                  className="w-full justify-start"
                  onClick={() => navigate('/accounts')}
                >
                  <Mail className="mr-2 h-4 w-4" />
                  邮箱账户
                </Button>
                <Button
                  variant="ghost"
                  className="w-full justify-start"
                  onClick={() => navigate('/trash')}
                >
                  <Trash2 className="mr-2 h-4 w-4" />
                  回收站
                </Button>
                <Button
                  variant="ghost"
                  className="w-full justify-start"
                  onClick={() => navigate('/rules')}
                >
                  <Zap className="mr-2 h-4 w-4" />
                  邮件规则
                </Button>
                <Button
                  variant="ghost"
                  className="w-full justify-start"
                  onClick={() => navigate('/webhooks')}
                >
                  <Webhook className="mr-2 h-4 w-4" />
                  Webhook
                </Button>
                <Button
                  variant="ghost"
                  className="w-full justify-start"
                  onClick={() => navigate('/api-keys')}
                >
                  <Key className="mr-2 h-4 w-4" />
                  API Key
                </Button>
                <Button
                  variant="ghost"
                  className="w-full justify-start"
                  onClick={() => navigate('/providers')}
                >
                  <Server className="mr-2 h-4 w-4" />
                  邮箱提供商
                </Button>
                <Button
                  variant="ghost"
                  className="w-full justify-start"
                  onClick={() => navigate('/oauth2-clients')}
                >
                  <Shield className="mr-2 h-4 w-4" />
                  OAuth2 客户端
                </Button>
                {/* 设置子菜单（可折叠） */}
                <Collapsible open={settingsOpen} onOpenChange={handleSettingsToggle}>
                  <CollapsibleTrigger asChild>
                    <Button
                      variant="ghost"
                      className="w-full justify-start"
                    >
                      <Settings className="mr-2 h-4 w-4" />
                      <span className="flex-1 text-left">设置</span>
                      {settingsOpen ? (
                        <ChevronDown className="h-4 w-4" />
                      ) : (
                        <ChevronRight className="h-4 w-4" />
                      )}
                    </Button>
                  </CollapsibleTrigger>
                  <CollapsibleContent className="space-y-1 pl-4 mt-1">
                    <Button
                      variant={location.pathname === '/settings' ? 'secondary' : 'ghost'}
                      className={cn(
                        'w-full justify-start',
                        location.pathname === '/settings' && 'bg-secondary'
                      )}
                      onClick={() => navigate('/settings')}
                    >
                      <User className="mr-2 h-4 w-4" />
                      个人设置
                    </Button>
                    <Button
                      variant={location.pathname === '/settings/system' ? 'secondary' : 'ghost'}
                      className={cn(
                        'w-full justify-start',
                        location.pathname === '/settings/system' && 'bg-secondary'
                      )}
                      onClick={() => navigate('/settings/system')}
                    >
                      <Server className="mr-2 h-4 w-4" />
                      系统设置
                    </Button>
                  </CollapsibleContent>
                </Collapsible>
              </CollapsibleContent>
            </Collapsible>
          </div>
        </div>
      </ScrollArea>
    </aside>
  );
};