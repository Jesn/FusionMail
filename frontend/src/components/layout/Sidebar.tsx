import { Inbox, Star, Archive, Trash2, Plus, Mail, Settings, Zap, Webhook, Search, Users } from 'lucide-react';
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
  const { sidebarCollapsed, setAccountDialogOpen } = useUIStore();
  const { accounts } = useAccounts();
  const { filter, setFilter, unreadCount, starredCount, archivedCount, deletedCount } = useEmailStore();

  const folders = [
    { id: 'inbox', name: '收件箱', icon: Inbox, count: unreadCount, showCount: true },
    { id: 'starred', name: '星标邮件', icon: Star, count: starredCount, showCount: false },
    { id: 'archived', name: '归档', icon: Archive, count: archivedCount, showCount: false },
    { id: 'trash', name: '垃圾箱', icon: Trash2, count: deletedCount, showCount: false },
  ];

  const handleFolderClick = (folderId: string) => {
    const newFilter: any = {};
    
    switch (folderId) {
      case 'inbox':
        newFilter.is_archived = false;
        newFilter.is_deleted = false;
        // 清除账户过滤器，显示所有账户的收件箱
        break;
      case 'starred':
        newFilter.is_starred = true;
        newFilter.is_deleted = false;
        // 清除账户过滤器，显示所有账户的星标邮件
        break;
      case 'archived':
        newFilter.is_archived = true;
        newFilter.is_deleted = false;
        // 清除账户过滤器，显示所有账户的归档邮件
        break;
      case 'trash':
        newFilter.is_deleted = true;
        // 清除账户过滤器，显示所有账户的垃圾箱邮件
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
        <div className="flex items-center gap-2">
          <Mail className="h-6 w-6 text-primary" />
          <span className="text-lg font-semibold">FusionMail</span>
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
          <div className="space-y-1">
            <h3 className="mb-2 px-2 text-xs font-semibold text-muted-foreground">
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
          <div className="space-y-1">
            <div className="flex items-center justify-between px-2">
              <h3 className="text-xs font-semibold text-muted-foreground">
                邮箱账户
              </h3>
              <Button
                variant="ghost"
                size="icon"
                className="h-6 w-6"
                onClick={() => {
                  navigate('/accounts');
                  setAccountDialogOpen(true);
                }}
                title="添加账户"
              >
                <Plus className="h-4 w-4" />
              </Button>
            </div>
{accounts.length === 0 ? (
              <div className="px-2 py-4 text-center text-sm text-muted-foreground">
                暂无账户
                <br />
                <Button
                  variant="link"
                  size="sm"
                  onClick={() => {
                    navigate('/accounts');
                    setAccountDialogOpen(true);
                  }}
                  className="mt-2"
                >
                  添加账户
                </Button>
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
                {accounts.map((account) => {
                  const isActive = filter.account_uid === account.uid;
                  return (
                    <Button
                      key={account.uid}
                      variant={isActive ? 'secondary' : 'ghost'}
                      className={cn(
                        'w-full justify-start',
                        isActive && 'bg-secondary'
                      )}
                      onClick={() => handleAccountClick(account.uid)}
                    >
                      <Mail className="mr-2 h-4 w-4" />
                      <span className="flex-1 truncate text-left">
                        {account.email}
                      </span>
                    </Button>
                  );
                })}
              </>
            )}
          </div>

          <Separator />

          {/* 管理功能 */}
          <div className="space-y-1">
            <h3 className="mb-2 px-2 text-xs font-semibold text-muted-foreground">
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
              onClick={() => navigate('/settings')}
            >
              <Settings className="mr-2 h-4 w-4" />
              设置
            </Button>
          </div>
        </div>
      </ScrollArea>
    </aside>
  );
};