import { useState, useEffect } from 'react';
import { Trash2, Zap, AlertCircle } from 'lucide-react';
import { Input } from '../components/ui/input';
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
import { AccountCard } from '../components/account/AccountCard';
import { useAccounts } from '../hooks/useAccounts';
import { Account } from '../types';

export const TrashPage = () => {
  const {
    isLoading,
    loadTrashAccounts,
    restoreAccount,
    forceDeleteAccount,
  } = useAccounts();

  const [trashAccounts, setTrashAccounts] = useState<Account[]>([]);
  const [deletingAccount, setDeletingAccount] = useState<{ uid: string; email: string } | null>(null);
  const [deleteConfirmEmail, setDeleteConfirmEmail] = useState('');
  const [restoringAccount, setRestoringAccount] = useState<{ uid: string; email: string } | null>(null);

  // 加载回收站账号
  const loadTrash = async () => {
    const accounts = await loadTrashAccounts();
    setTrashAccounts(accounts);
  };

  useEffect(() => {
    loadTrash();
  }, []);

  // 恢复账号 - 显示确认对话框
  const handleRestoreClick = (uid: string, email: string) => {
    setRestoringAccount({ uid, email });
  };

  // 确认恢复
  const handleRestoreConfirm = async () => {
    if (restoringAccount) {
      try {
        await restoreAccount(restoringAccount.uid);
        // 重新加载回收站列表
        await loadTrash();
        setRestoringAccount(null);
      } catch (err) {
        console.error('Failed to restore account:', err);
      }
    }
  };

  // 取消恢复
  const handleRestoreCancel = () => {
    setRestoringAccount(null);
  };

  // 永久删除账号
  const handleForceDeleteClick = (uid: string, email: string) => {
    setDeletingAccount({ uid, email });
  };

  const handleForceDeleteConfirm = async () => {
    if (deletingAccount && deleteConfirmEmail === deletingAccount.email) {
      try {
        await forceDeleteAccount(deletingAccount.uid);
        // 重新加载回收站列表
        await loadTrash();
        setDeletingAccount(null);
        setDeleteConfirmEmail('');
      } catch (err) {
        console.error('Failed to force delete account:', err);
      }
    }
  };

  const handleDeleteCancel = () => {
    setDeletingAccount(null);
    setDeleteConfirmEmail('');
  };

  return (
    <div className="h-full overflow-auto">
      <div className="mx-auto max-w-7xl p-6">
        {/* 头部 */}
        <div className="mb-6">
          <div className="flex items-center gap-2 mb-2">
            <Trash2 className="h-8 w-8 text-muted-foreground" />
            <h1 className="text-3xl font-bold">回收站</h1>
          </div>
          <p className="text-muted-foreground">
            已删除的邮箱账户，可以恢复或永久删除
          </p>
        </div>

        {/* 提示信息 */}
        {trashAccounts.length > 0 && (
          <div className="mb-6 rounded-lg border border-orange-200 bg-orange-50 p-4 dark:border-orange-900 dark:bg-orange-950/20">
            <div className="flex items-start gap-3">
              <AlertCircle className="h-5 w-5 text-orange-600 dark:text-orange-400 mt-0.5" />
              <div className="flex-1">
                <h3 className="font-medium text-orange-900 dark:text-orange-100">
                  回收站说明
                </h3>
                <p className="mt-1 text-sm text-orange-700 dark:text-orange-300">
                  回收站中的账号会根据系统设置自动清理。永久删除后，该账号的所有邮件、附件等数据将无法恢复。
                </p>
              </div>
            </div>
          </div>
        )}

        {/* 账号列表 */}
        {isLoading ? (
          <div className="flex items-center justify-center py-12">
            <div className="text-muted-foreground">加载中...</div>
          </div>
        ) : trashAccounts.length === 0 ? (
          <div className="flex flex-col items-center justify-center rounded-lg border border-dashed py-12">
            <Trash2 className="h-12 w-12 text-muted-foreground mb-4" />
            <p className="text-lg text-muted-foreground">
              回收站是空的
            </p>
            <p className="text-sm text-muted-foreground mt-2">
              删除的账号会出现在这里
            </p>
          </div>
        ) : (
          <div className="space-y-4">
            {trashAccounts.map((account) => (
              <AccountCard
                key={account.uid}
                account={account}
                onRestore={() => handleRestoreClick(account.uid, account.email)}
                onForceDelete={() => handleForceDeleteClick(account.uid, account.email)}
                density="compact"
              />
            ))}
          </div>
        )}

        {/* 恢复确认对话框 */}
        <AlertDialog open={!!restoringAccount} onOpenChange={(open) => !open && handleRestoreCancel()}>
          <AlertDialogContent>
            <AlertDialogHeader>
              <AlertDialogTitle>确认恢复账号</AlertDialogTitle>
              <AlertDialogDescription>
                确定要恢复账号 <span className="font-semibold text-foreground">{restoringAccount?.email}</span> 吗？
                <br />
                <br />
                恢复后，该账号将重新出现在账号列表中，并可以继续使用。
              </AlertDialogDescription>
            </AlertDialogHeader>
            <AlertDialogFooter>
              <AlertDialogCancel onClick={handleRestoreCancel}>取消</AlertDialogCancel>
              <AlertDialogAction
                onClick={handleRestoreConfirm}
                className="bg-blue-600 hover:bg-blue-700 focus:ring-blue-600"
              >
                恢复账号
              </AlertDialogAction>
            </AlertDialogFooter>
          </AlertDialogContent>
        </AlertDialog>

        {/* 永久删除确认对话框 */}
        <AlertDialog open={!!deletingAccount} onOpenChange={(open) => !open && handleDeleteCancel()}>
          <AlertDialogContent>
            <AlertDialogHeader>
              <AlertDialogTitle className="flex items-center gap-2 text-red-600">
                <Zap className="h-5 w-5" />
                永久删除账号
              </AlertDialogTitle>
              <AlertDialogDescription className="space-y-4">
                <div>
                  确定要永久删除账号 <span className="font-semibold text-foreground">{deletingAccount?.email}</span> 吗？
                  <br />
                  <br />
                  <span className="text-red-600 font-medium">
                    此操作将删除该账号的所有邮件、附件等数据，且无法恢复！
                  </span>
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
                onClick={handleForceDeleteConfirm}
                disabled={deleteConfirmEmail !== deletingAccount?.email}
                className="bg-red-600 hover:bg-red-700 focus:ring-red-600 disabled:opacity-50 disabled:cursor-not-allowed"
              >
                <Zap className="mr-2 h-4 w-4" />
                永久删除
              </AlertDialogAction>
            </AlertDialogFooter>
          </AlertDialogContent>
        </AlertDialog>
      </div>
    </div>
  );
};
