import { Plus, Trash2, Mail, Globe } from 'lucide-react';
import { useState } from 'react';
import { Button } from '../components/ui/button';
import { Input } from '../components/ui/input';
import { Textarea } from '../components/ui/textarea';
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '../components/ui/dialog';
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
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '../components/ui/table';
import { Tabs, TabsContent, TabsList, TabsTrigger } from '../components/ui/tabs';
import { useEmailList } from '../hooks/useEmailList';
import type { EmailList } from '../services/emailListService';

export const EmailListPage = () => {
  const [activeTab, setActiveTab] = useState<'whitelist' | 'blacklist'>('whitelist');
  const [isAddDialogOpen, setIsAddDialogOpen] = useState(false);
  const [deletingItem, setDeletingItem] = useState<EmailList | null>(null);
  const [formData, setFormData] = useState({ target: '', reason: '' });
  const [formErrors, setFormErrors] = useState<{ target?: string }>({});

  const {
    lists,
    isLoading,
    total,
    page,
    pageSize,
    setPage,
    addToList,
    deleteFromList,
  } = useEmailList(activeTab);

  const handleAddClick = () => {
    setFormData({ target: '', reason: '' });
    setFormErrors({});
    setIsAddDialogOpen(true);
  };

  const validateEmail = (email: string): boolean => {
    const emailRegex = /^[a-z0-9._%+\-]+@[a-z0-9.\-]+\.[a-z]{2,}$/i;
    const domainRegex = /^[a-z0-9.\-]+\.[a-z]{2,}$/i;
    return emailRegex.test(email) || domainRegex.test(email);
  };

  const handleSubmit = async () => {
    // 验证
    const errors: { target?: string } = {};
    if (!formData.target.trim()) {
      errors.target = '请输入邮箱地址或域名';
    } else if (!validateEmail(formData.target.trim())) {
      errors.target = '请输入有效的邮箱地址或域名';
    }

    if (Object.keys(errors).length > 0) {
      setFormErrors(errors);
      return;
    }

    try {
      await addToList({
        target: formData.target.trim().toLowerCase(),
        reason: formData.reason.trim() || undefined,
      });
      setIsAddDialogOpen(false);
      setFormData({ target: '', reason: '' });
    } catch (error) {
      // 错误已在 hook 中处理
    }
  };

  const handleDeleteClick = (item: EmailList) => {
    setDeletingItem(item);
  };

  const handleDeleteConfirm = async () => {
    if (deletingItem) {
      await deleteFromList(deletingItem.id);
      setDeletingItem(null);
    }
  };

  const formatDate = (dateString: string) => {
    return new Date(dateString).toLocaleString('zh-CN');
  };

  const getTargetIcon = (targetType: string) => {
    return targetType === 'email' ? <Mail className="h-4 w-4" /> : <Globe className="h-4 w-4" />;
  };

  const totalPages = Math.ceil(total / pageSize);

  return (
    <div className="container mx-auto px-4 py-6">
      <div className="flex items-center justify-between mb-6">
        <div>
          <h1 className="text-2xl font-bold text-gray-900 dark:text-white">
            白名单/黑名单管理
          </h1>
          <p className="text-sm text-gray-500 dark:text-gray-400 mt-1">
            管理邮件发件人的白名单和黑名单
          </p>
        </div>
        <Button onClick={handleAddClick}>
          <Plus className="h-4 w-4 mr-2" />
          添加
        </Button>
      </div>

      <Tabs value={activeTab} onValueChange={(v) => setActiveTab(v as 'whitelist' | 'blacklist')}>
        <TabsList className="mb-4">
          <TabsTrigger value="whitelist">白名单</TabsTrigger>
          <TabsTrigger value="blacklist">黑名单</TabsTrigger>
        </TabsList>

        <TabsContent value="whitelist">
          <div className="bg-white dark:bg-gray-800 rounded-lg shadow">
            {isLoading ? (
              <div className="p-8 text-center text-gray-500">加载中...</div>
            ) : lists.length === 0 ? (
              <div className="p-8 text-center text-gray-500">
                暂无白名单条目
              </div>
            ) : (
              <>
                <Table>
                  <TableHeader>
                    <TableRow>
                      <TableHead>类型</TableHead>
                      <TableHead>目标</TableHead>
                      <TableHead>原因</TableHead>
                      <TableHead>添加时间</TableHead>
                      <TableHead className="text-right">操作</TableHead>
                    </TableRow>
                  </TableHeader>
                  <TableBody>
                    {lists.map((item) => (
                      <TableRow key={item.id}>
                        <TableCell>
                          <div className="flex items-center gap-2">
                            {getTargetIcon(item.target_type)}
                            <span className="text-sm">
                              {item.target_type === 'email' ? '邮箱' : '域名'}
                            </span>
                          </div>
                        </TableCell>
                        <TableCell className="font-mono text-sm">{item.target}</TableCell>
                        <TableCell className="text-sm text-gray-600 dark:text-gray-400">
                          {item.reason || '-'}
                        </TableCell>
                        <TableCell className="text-sm text-gray-500">
                          {formatDate(item.created_at)}
                        </TableCell>
                        <TableCell className="text-right">
                          <Button
                            variant="ghost"
                            size="sm"
                            onClick={() => handleDeleteClick(item)}
                          >
                            <Trash2 className="h-4 w-4 text-red-500" />
                          </Button>
                        </TableCell>
                      </TableRow>
                    ))}
                  </TableBody>
                </Table>

                {totalPages > 1 && (
                  <div className="flex items-center justify-between px-4 py-3 border-t">
                    <div className="text-sm text-gray-500">
                      共 {total} 条，第 {page} / {totalPages} 页
                    </div>
                    <div className="flex gap-2">
                      <Button
                        variant="outline"
                        size="sm"
                        disabled={page === 1}
                        onClick={() => setPage(page - 1)}
                      >
                        上一页
                      </Button>
                      <Button
                        variant="outline"
                        size="sm"
                        disabled={page === totalPages}
                        onClick={() => setPage(page + 1)}
                      >
                        下一页
                      </Button>
                    </div>
                  </div>
                )}
              </>
            )}
          </div>
        </TabsContent>

        <TabsContent value="blacklist">
          <div className="bg-white dark:bg-gray-800 rounded-lg shadow">
            {isLoading ? (
              <div className="p-8 text-center text-gray-500">加载中...</div>
            ) : lists.length === 0 ? (
              <div className="p-8 text-center text-gray-500">
                暂无黑名单条目
              </div>
            ) : (
              <>
                <Table>
                  <TableHeader>
                    <TableRow>
                      <TableHead>类型</TableHead>
                      <TableHead>目标</TableHead>
                      <TableHead>原因</TableHead>
                      <TableHead>添加时间</TableHead>
                      <TableHead className="text-right">操作</TableHead>
                    </TableRow>
                  </TableHeader>
                  <TableBody>
                    {lists.map((item) => (
                      <TableRow key={item.id}>
                        <TableCell>
                          <div className="flex items-center gap-2">
                            {getTargetIcon(item.target_type)}
                            <span className="text-sm">
                              {item.target_type === 'email' ? '邮箱' : '域名'}
                            </span>
                          </div>
                        </TableCell>
                        <TableCell className="font-mono text-sm">{item.target}</TableCell>
                        <TableCell className="text-sm text-gray-600 dark:text-gray-400">
                          {item.reason || '-'}
                        </TableCell>
                        <TableCell className="text-sm text-gray-500">
                          {formatDate(item.created_at)}
                        </TableCell>
                        <TableCell className="text-right">
                          <Button
                            variant="ghost"
                            size="sm"
                            onClick={() => handleDeleteClick(item)}
                          >
                            <Trash2 className="h-4 w-4 text-red-500" />
                          </Button>
                        </TableCell>
                      </TableRow>
                    ))}
                  </TableBody>
                </Table>

                {totalPages > 1 && (
                  <div className="flex items-center justify-between px-4 py-3 border-t">
                    <div className="text-sm text-gray-500">
                      共 {total} 条，第 {page} / {totalPages} 页
                    </div>
                    <div className="flex gap-2">
                      <Button
                        variant="outline"
                        size="sm"
                        disabled={page === 1}
                        onClick={() => setPage(page - 1)}
                      >
                        上一页
                      </Button>
                      <Button
                        variant="outline"
                        size="sm"
                        disabled={page === totalPages}
                        onClick={() => setPage(page + 1)}
                      >
                        下一页
                      </Button>
                    </div>
                  </div>
                )}
              </>
            )}
          </div>
        </TabsContent>
      </Tabs>

      {/* 添加对话框 */}
      <Dialog open={isAddDialogOpen} onOpenChange={setIsAddDialogOpen}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>
              添加到{activeTab === 'whitelist' ? '白名单' : '黑名单'}
            </DialogTitle>
            <DialogDescription>
              输入邮箱地址（如 user@example.com）或域名（如 example.com）
            </DialogDescription>
          </DialogHeader>

          <div className="space-y-4 py-4">
            <div className="space-y-2">
              <label className="text-sm font-medium">邮箱地址或域名 *</label>
              <Input
                placeholder="user@example.com 或 example.com"
                value={formData.target}
                onChange={(e) => {
                  setFormData({ ...formData, target: e.target.value });
                  setFormErrors({ ...formErrors, target: undefined });
                }}
              />
              {formErrors.target && (
                <p className="text-sm text-red-500">{formErrors.target}</p>
              )}
            </div>

            <div className="space-y-2">
              <label className="text-sm font-medium">原因（可选）</label>
              <Textarea
                placeholder="添加原因..."
                value={formData.reason}
                onChange={(e) => setFormData({ ...formData, reason: e.target.value })}
                rows={3}
              />
            </div>
          </div>

          <DialogFooter>
            <Button variant="outline" onClick={() => setIsAddDialogOpen(false)}>
              取消
            </Button>
            <Button onClick={handleSubmit}>确定</Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      {/* 删除确认对话框 */}
      <AlertDialog open={!!deletingItem} onOpenChange={() => setDeletingItem(null)}>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>确认删除</AlertDialogTitle>
            <AlertDialogDescription>
              确定要删除 <span className="font-mono font-semibold">{deletingItem?.target}</span> 吗？
              此操作无法撤销。
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel>取消</AlertDialogCancel>
            <AlertDialogAction onClick={handleDeleteConfirm}>删除</AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </div>
  );
};
