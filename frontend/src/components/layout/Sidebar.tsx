import { Inbox, Star, Archive, Trash2, Mail, Settings, Zap, Webhook, Search, Users, Key, Shield, Server } from 'lucide-react';
import { Button } from '../ui/button';
import { ScrollArea } from '../ui/scroll-area';
import { Separator } from '../ui/separator';
import { Badge } from '../ui/badge';
import { useUIStore } from '../../stores/uiStore';
import { useAccounts } from '../../hooks/useAccounts';
import { useEmailStore } from '../../stores/emailStore';
import { cn } from '../../lib/utils';
import { useNavigate } from 'react-router-dom';

export const Sidebar = () => {
  const navigate = useNavigate();
  const { sidebarCollapsed } = useUIStore();
  const { accounts } = useAccounts();
  const activeAccounts = (accounts || []).filter((account) => !account.deleted_at);
  const { filter, setFilter, unreadCount, starredCount, archivedCount, deletedCount } = useEmailStore();

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
        <div className="space-y-4 p-4">
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
            <h3 className="mb-2 px-2 text-xs font-medium text-muted-foreground">
              文件夹
            </h3>
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
          </div>

          <Separator />

          {/* 账户列表 */}
          <div className="space-y-1 pr-2">
            <div className="px-2">
              <h3 className="text-xs font-medium text-muted-foreground truncate">
                邮箱账户
              </h3>
            </div>
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
          </div>

          <Separator />

          {/* 管理功能 */}
          <div className="space-y-1 pr-2">
            <h3 className="mb-2 px-2 text-xs font-medium text-muted-foreground">
              管理
            </h3>
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
            <Button
              variant="ghost"
              className="w-full justify-start"
              onClick={() => navigate('/settings')}
            >
              <Settings className="mr-2 h-4 w-4" />
              用户设置
            </Button>
            <Button
              variant="ghost"
              className="w-full justify-start"
              onClick={() => navigate('/settings/system')}
            >
              <Server className="mr-2 h-4 w-4" />
              系统设置
            </Button>
          </div>
        </div>
      </ScrollArea>
    </aside>
  );
};