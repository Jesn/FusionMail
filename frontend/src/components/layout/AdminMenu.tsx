/**
 * AdminMenu 组件 - 管理员菜单（设置视图）
 * 显示系统配置和管理功能入口
 */
import { Mail, Settings, Zap, Webhook, Key, Shield, Server, ChevronDown, ChevronRight, Trash2, FileText, ArrowLeft } from 'lucide-react';
import { Button } from '../ui/button';
import { ScrollArea } from '../ui/scroll-area';
import { Separator } from '../ui/separator';
import { Collapsible, CollapsibleContent, CollapsibleTrigger } from '../ui/collapsible';

import { cn } from '../../lib/utils';
import { useNavigate, useLocation } from 'react-router-dom';
import { useState } from 'react';

export const AdminMenu = () => {
  const navigate = useNavigate();
  const location = useLocation();

  // 高级配置展开状态
  const [advancedOpen, setAdvancedOpen] = useState<boolean>(() => {
    const saved = localStorage.getItem('sidebar-settings-advanced-open');
    // 默认展开高级配置（如果当前在高级配置页面）
    if (saved === null) {
      return ['/api-keys', '/providers', '/oauth2-clients', '/email-list'].includes(location.pathname);
    }
    return saved === 'true';
  });

  // 切换高级配置展开状态
  const handleAdvancedToggle = () => {
    const newState = !advancedOpen;
    setAdvancedOpen(newState);
    localStorage.setItem('sidebar-settings-advanced-open', String(newState));
  };

  // 返回邮件视图
  const handleBackToMail = () => {
    navigate('/inbox');
  };

  // 管理菜单项
  const managementItems = [
    { path: '/accounts', name: '账户管理', icon: Mail },
    { path: '/trash', name: '已删除账户', icon: Trash2 },
    { path: '/rules', name: '邮件规则', icon: Zap },
    { path: '/webhooks', name: 'Webhook', icon: Webhook },
    { path: '/settings', name: '个人设置', icon: Settings },
    { path: '/settings/system', name: '系统设置', icon: Server },
    { path: '/logs', name: '系统日志', icon: FileText },
  ];

  // 高级配置菜单项
  const advancedItems = [
    { path: '/api-keys', name: 'API Key', icon: Key },
    { path: '/providers', name: '邮箱提供商', icon: Server },
    { path: '/oauth2-clients', name: 'OAuth2 客户端', icon: Shield },
    { path: '/email-list', name: '白名单/黑名单', icon: Shield },
  ];

  return (
    <aside className="flex w-64 flex-col border-r bg-background">
      {/* Logo */}
      <div className="flex h-16 items-center justify-between border-b px-3">
        <div 
          className="flex items-center cursor-pointer hover:opacity-80 transition-opacity"
          onClick={handleBackToMail}
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
          {/* 返回按钮 */}
          <div className="space-y-1">
            <Button
              variant="ghost"
              className="w-full justify-start"
              onClick={handleBackToMail}
            >
              <ArrowLeft className="mr-2 h-4 w-4" />
              返回
            </Button>
          </div>

          <Separator />

          {/* 管理菜单项 */}
          <div className="space-y-1">
            {managementItems.map((item) => {
              const Icon = item.icon;
              const isActive = location.pathname === item.path;
              return (
                <Button
                  key={item.path}
                  variant={isActive ? 'secondary' : 'ghost'}
                  className={cn('w-full justify-start', isActive && 'bg-secondary')}
                  onClick={() => navigate(item.path)}
                >
                  <Icon className="mr-2 h-4 w-4" />
                  {item.name}
                </Button>
              );
            })}
          </div>

          <Separator />

          {/* 高级配置（可折叠） */}
          <div className="space-y-1 pr-2">
            <Collapsible open={advancedOpen} onOpenChange={handleAdvancedToggle}>
              <CollapsibleTrigger asChild>
                <Button variant="ghost" className="w-full justify-start mb-1 h-8 px-2">
                  <span className="flex-1 text-left text-xs font-medium text-muted-foreground">高级配置</span>
                  {advancedOpen ? (
                    <ChevronDown className="h-3.5 w-3.5 text-muted-foreground" />
                  ) : (
                    <ChevronRight className="h-3.5 w-3.5 text-muted-foreground" />
                  )}
                </Button>
              </CollapsibleTrigger>
              <CollapsibleContent className="space-y-1">
                {advancedItems.map((item) => {
                  const Icon = item.icon;
                  const isActive = location.pathname === item.path;
                  return (
                    <Button
                      key={item.path}
                      variant={isActive ? 'secondary' : 'ghost'}
                      className={cn('w-full justify-start', isActive && 'bg-secondary')}
                      onClick={() => navigate(item.path)}
                    >
                      <Icon className="mr-2 h-4 w-4" />
                      {item.name}
                    </Button>
                  );
                })}
              </CollapsibleContent>
            </Collapsible>
          </div>
        </div>
      </ScrollArea>
    </aside>
  );
};
