import { useState } from 'react';
import { Plus, Trash2, Mail, Globe, Shield, ShieldOff, Search, RefreshCw } from 'lucide-react';
import { Button } from '../components/ui/button';
import { Input } from '../components/ui/input';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '../components/ui/card';
import { Tabs, TabsContent, TabsList, TabsTrigger } from '../components/ui/tabs';
import { Dialog, DialogContent, DialogDescription, DialogFooter, DialogHeader, DialogTitle, DialogTrigger } from '../components/ui/dialog';
import { AlertDialog, AlertDialogAction, AlertDialogCancel, AlertDialogContent, AlertDialogDescription, AlertDialogFooter, AlertDialogHeader, AlertDialogTitle } from '../components/ui/alert-dialog';
import { Label } from '../components/ui/label';
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '../components/ui/select';
import { Textarea } from '../components/ui/textarea';
import { useEmailList } from '../hooks/useEmailList';
import type { AddEmailListRequest } from '../services/emailListService';

// 白名单/黑名单管理页面
export function EmailListPage() {
  const [activeTab, setActiveTab] = useState<'whitelist' | 'blacklist'>('whitelist');
  const [searchQuery, setSearchQuery] = useState('');
  const [isAddDialogOpen, setIsAddDialogOpen] = useState(false);
  const [newEntry, setNewEntry] = useState<AddEmailListRequest>({ target: '', reason: '' });
  const [targetType, setTargetType] = useState<'email' | 'domain'>('email');
  
  // 删除确认弹框状态
  const [deleteDialogOpen, setDeleteDialogOpen] = useState(false);
  const [deleteTarget, setDeleteTarget] = useState<{ id: number; target: string } | null>(null);

  const whitelist = useEmailList('whitelist');
  const blacklist = useEmailList('blacklist');

  const currentList = activeTab === 'whitelist' ? whitelist : blacklist;

  // 过滤列表
  const filteredLists = currentList.lists.filter(item =>
    item.target.toLowerCase().includes(searchQuery.toLowerCase()) ||
    (item.reason && item.reason.toLowerCase().includes(searchQuery.toLowerCase()))
  );

  // 添加条目
  const handleAdd = async () => {
    if (!newEntry.target.trim()) return;

    await currentList.addToList(newEntry);
    setNewEntry({ target: '', reason: '' });
    setIsAddDialogOpen(false);
  };

  // 打开删除确认弹框
  const handleDeleteClick = (id: number, target: string) => {
    setDeleteTarget({ id, target });
    setDeleteDialogOpen(true);
  };

  // 确认删除
  const handleConfirmDelete = async () => {
    if (deleteTarget) {
      await currentList.deleteFromList(deleteTarget.id);
      setDeleteDialogOpen(false);
      setDeleteTarget(null);
    }
  };

  // 验证输入
  const validateTarget = (value: string): boolean => {
    if (targetType === 'email') {
      return /^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(value);
    } else {
      return /^[a-zA-Z0-9][a-zA-Z0-9-]*\.[a-zA-Z]{2,}$/.test(value);
    }
  };

  const isValidTarget = newEntry.target.trim() === '' || validateTarget(newEntry.target);

  return (
    <div className="container mx-auto px-4 py-6 space-y-6">
      {/* 页面标题 */}
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-3xl font-bold">白名单/黑名单管理</h1>
          <p className="text-muted-foreground mt-2">管理邮件发件人的白名单和黑名单</p>
        </div>
      </div>

      {/* 标签页 */}
      <Tabs value={activeTab} onValueChange={(v) => setActiveTab(v as 'whitelist' | 'blacklist')}>
        <div className="flex items-center justify-between">
          <TabsList>
            <TabsTrigger value="whitelist" className="flex items-center gap-2">
              <Shield className="h-4 w-4" />
              白名单 ({whitelist.total})
            </TabsTrigger>
            <TabsTrigger value="blacklist" className="flex items-center gap-2">
              <ShieldOff className="h-4 w-4" />
              黑名单 ({blacklist.total})
            </TabsTrigger>
          </TabsList>

          <div className="flex items-center gap-2">
            {/* 搜索框 */}
            <div className="relative">
              <Search className="absolute left-3 top-1/2 -translate-y-1/2 h-4 w-4 text-muted-foreground" />
              <Input
                placeholder="搜索..."
                value={searchQuery}
                onChange={(e) => setSearchQuery(e.target.value)}
                className="pl-9 w-64"
              />
            </div>

            {/* 刷新按钮 */}
            <Button
              variant="outline"
              size="icon"
              onClick={() => currentList.refresh()}
              disabled={currentList.isLoading}
            >
              <RefreshCw className={`h-4 w-4 ${currentList.isLoading ? 'animate-spin' : ''}`} />
            </Button>

            {/* 添加按钮 */}
            <Dialog open={isAddDialogOpen} onOpenChange={setIsAddDialogOpen}>
              <DialogTrigger asChild>
                <Button>
                  <Plus className="h-4 w-4 mr-2" />
                  添加
                </Button>
              </DialogTrigger>
              <DialogContent>
                <DialogHeader>
                  <DialogTitle>
                    添加到{activeTab === 'whitelist' ? '白名单' : '黑名单'}
                  </DialogTitle>
                  <DialogDescription>
                    {activeTab === 'whitelist'
                      ? '白名单中的发件人将直接放行，不进行垃圾邮件检测'
                      : '黑名单中的发件人将直接标记为垃圾邮件'}
                  </DialogDescription>
                </DialogHeader>

                <div className="space-y-4 py-4">
                  {/* 类型选择 */}
                  <div className="space-y-2">
                    <Label>类型</Label>
                    <Select value={targetType} onValueChange={(v) => setTargetType(v as 'email' | 'domain')}>
                      <SelectTrigger>
                        <SelectValue />
                      </SelectTrigger>
                      <SelectContent>
                        <SelectItem value="email">
                          <div className="flex items-center gap-2">
                            <Mail className="h-4 w-4" />
                            邮箱地址
                          </div>
                        </SelectItem>
                        <SelectItem value="domain">
                          <div className="flex items-center gap-2">
                            <Globe className="h-4 w-4" />
                            域名
                          </div>
                        </SelectItem>
                      </SelectContent>
                    </Select>
                  </div>

                  {/* 目标输入 */}
                  <div className="space-y-2">
                    <Label>{targetType === 'email' ? '邮箱地址' : '域名'}</Label>
                    <Input
                      placeholder={targetType === 'email' ? 'example@domain.com' : 'domain.com'}
                      value={newEntry.target}
                      onChange={(e) => setNewEntry({ ...newEntry, target: e.target.value })}
                      className={!isValidTarget ? 'border-red-500' : ''}
                    />
                    {!isValidTarget && (
                      <p className="text-sm text-red-500">
                        请输入有效的{targetType === 'email' ? '邮箱地址' : '域名'}
                      </p>
                    )}
                  </div>

                  {/* 原因输入 */}
                  <div className="space-y-2">
                    <Label>原因（可选）</Label>
                    <Textarea
                      placeholder="添加原因..."
                      value={newEntry.reason || ''}
                      onChange={(e) => setNewEntry({ ...newEntry, reason: e.target.value })}
                      rows={3}
                    />
                  </div>
                </div>

                <DialogFooter>
                  <Button variant="outline" onClick={() => setIsAddDialogOpen(false)}>
                    取消
                  </Button>
                  <Button
                    onClick={handleAdd}
                    disabled={!newEntry.target.trim() || !isValidTarget}
                  >
                    添加
                  </Button>
                </DialogFooter>
              </DialogContent>
            </Dialog>
          </div>
        </div>

        {/* 白名单内容 */}
        <TabsContent value="whitelist" className="mt-4">
          <EmailListContent
            lists={filteredLists}
            isLoading={whitelist.isLoading}
            onDelete={handleDeleteClick}
            type="whitelist"
          />
        </TabsContent>

        {/* 黑名单内容 */}
        <TabsContent value="blacklist" className="mt-4">
          <EmailListContent
            lists={filteredLists}
            isLoading={blacklist.isLoading}
            onDelete={handleDeleteClick}
            type="blacklist"
          />
        </TabsContent>
      </Tabs>

      {/* 删除确认弹框 */}
      <AlertDialog open={deleteDialogOpen} onOpenChange={setDeleteDialogOpen}>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>确认删除</AlertDialogTitle>
            <AlertDialogDescription>
              确定要从{activeTab === 'whitelist' ? '白名单' : '黑名单'}中删除 <span className="font-medium text-foreground">{deleteTarget?.target}</span> 吗？此操作无法撤销。
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel>取消</AlertDialogCancel>
            <AlertDialogAction onClick={handleConfirmDelete} className="bg-destructive text-destructive-foreground hover:bg-destructive/90">
              删除
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </div>
  );
}


// 列表内容组件
interface EmailListContentProps {
  lists: Array<{
    id: number;
    target: string;
    target_type: 'email' | 'domain';
    reason?: string;
    created_at: string;
  }>;
  isLoading: boolean;
  onDelete: (id: number, target: string) => void;
  type: 'whitelist' | 'blacklist';
}

function EmailListContent({ lists, isLoading, onDelete, type }: EmailListContentProps) {
  if (isLoading) {
    return (
      <Card>
        <CardContent className="py-8">
          <div className="flex items-center justify-center">
            <RefreshCw className="h-6 w-6 animate-spin text-muted-foreground" />
            <span className="ml-2 text-muted-foreground">加载中...</span>
          </div>
        </CardContent>
      </Card>
    );
  }

  if (lists.length === 0) {
    return (
      <Card>
        <CardContent className="py-12">
          <div className="text-center">
            {type === 'whitelist' ? (
              <Shield className="h-12 w-12 mx-auto text-muted-foreground mb-4" />
            ) : (
              <ShieldOff className="h-12 w-12 mx-auto text-muted-foreground mb-4" />
            )}
            <h3 className="text-lg font-medium">
              {type === 'whitelist' ? '白名单为空' : '黑名单为空'}
            </h3>
            <p className="text-muted-foreground mt-1">
              {type === 'whitelist'
                ? '添加发件人到白名单，他们的邮件将直接放行'
                : '添加发件人到黑名单，他们的邮件将被标记为垃圾邮件'}
            </p>
          </div>
        </CardContent>
      </Card>
    );
  }

  return (
    <Card>
      <CardHeader>
        <CardTitle className="text-lg">
          {type === 'whitelist' ? '白名单列表' : '黑名单列表'}
        </CardTitle>
        <CardDescription>
          共 {lists.length} 条记录
        </CardDescription>
      </CardHeader>
      <CardContent>
        <div className="space-y-2">
          {lists.map((item) => (
            <div
              key={item.id}
              className="flex items-center justify-between p-3 rounded-lg border hover:bg-muted/50 transition-colors"
            >
              <div className="flex items-center gap-3">
                {item.target_type === 'email' ? (
                  <Mail className="h-5 w-5 text-muted-foreground" />
                ) : (
                  <Globe className="h-5 w-5 text-muted-foreground" />
                )}
                <div>
                  <div className="font-medium">{item.target}</div>
                  {item.reason && (
                    <div className="text-sm text-muted-foreground">{item.reason}</div>
                  )}
                  <div className="text-xs text-muted-foreground">
                    添加于 {new Date(item.created_at).toLocaleString()}
                  </div>
                </div>
              </div>
              <Button
                variant="ghost"
                size="icon"
                onClick={() => onDelete(item.id, item.target)}
                className="text-destructive hover:text-destructive hover:bg-destructive/10"
              >
                <Trash2 className="h-4 w-4" />
              </Button>
            </div>
          ))}
        </div>
      </CardContent>
    </Card>
  );
}
