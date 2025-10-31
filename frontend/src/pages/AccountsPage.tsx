import { Plus } from 'lucide-react';
import { useState } from 'react';
import { Button } from '../components/ui/button';
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
import { AccountForm } from '../components/account/AccountForm';
import { useAccounts } from '../hooks/useAccounts';
import { useUIStore } from '../stores/uiStore';
import { Account } from '../types';

export const AccountsPage = () => {
  const {
    accounts,
    isLoading,
    createAccount,
    updateAccount,
    deleteAccount,
    syncAccount,
    toggleAccountStatus,
  } = useAccounts();
  const { isAccountDialogOpen, setAccountDialogOpen } = useUIStore();
  const [editingAccount, setEditingAccount] = useState<Account | null>(null);
  const [deletingAccount, setDeletingAccount] = useState<{ uid: string; email: string } | null>(null);

  const handleDeleteClick = (uid: string, email: string) => {
    setDeletingAccount({ uid, email });
  };

  const handleDeleteConfirm = async () => {
    if (deletingAccount) {
      await deleteAccount(deletingAccount.uid);
      setDeletingAccount(null);
    }
  };

  const handleDeleteCancel = () => {
    setDeletingAccount(null);
  };

  const handleEdit = (account: Account) => {
    setEditingAccount(account);
    setAccountDialogOpen(true);
  };

  const handleCloseDialog = () => {
    setAccountDialogOpen(false);
    setEditingAccount(null);
  };

  const handleSubmit = async (data: any) => {
    if (editingAccount) {
      // 编辑模式
      await updateAccount(editingAccount.uid, data);
    } else {
      // 创建模式
      await createAccount(data);
    }
    handleCloseDialog();
  };

  return (
    <div className="h-full overflow-auto">
      <div className="mx-auto max-w-4xl p-6">
        {/* 头部 */}
        <div className="mb-6 flex items-center justify-between">
          <div>
            <h1 className="text-3xl font-bold">邮箱账户</h1>
            <p className="text-muted-foreground">
              管理您的邮箱账户和同步设置
            </p>
          </div>
          <Button onClick={() => setAccountDialogOpen(true)}>
            <Plus className="mr-2 h-4 w-4" />
            添加账户
          </Button>
        </div>

        {/* 账户列表 */}
        {isLoading ? (
          <div className="flex items-center justify-center py-12">
            <div className="text-muted-foreground">加载中...</div>
          </div>
        ) : accounts.length === 0 ? (
          <div className="flex flex-col items-center justify-center rounded-lg border border-dashed py-12">
            <p className="mb-4 text-lg text-muted-foreground">
              还没有添加邮箱账户
            </p>
            <Button onClick={() => setAccountDialogOpen(true)}>
              <Plus className="mr-2 h-4 w-4" />
              添加第一个账户
            </Button>
          </div>
        ) : (
          <div className="space-y-4">
            {accounts.map((account) => (
              <AccountCard
                key={account.uid}
                account={account}
                onSync={() => syncAccount(account.uid)}
                onDelete={() => handleDeleteClick(account.uid, account.email)}
                onEdit={() => handleEdit(account)}
                onToggleStatus={() => toggleAccountStatus(account.uid, account.status)}
              />
            ))}
          </div>
        )}

        {/* 添加/编辑账户对话框 */}
        <AccountForm
          open={isAccountDialogOpen}
          onClose={handleCloseDialog}
          onSubmit={handleSubmit}
          account={editingAccount}
        />

        {/* 删除确认对话框 */}
        <AlertDialog open={!!deletingAccount} onOpenChange={(open) => !open && handleDeleteCancel()}>
          <AlertDialogContent>
            <AlertDialogHeader>
              <AlertDialogTitle>确认删除账户</AlertDialogTitle>
              <AlertDialogDescription>
                确定要删除账户 <span className="font-semibold text-foreground">{deletingAccount?.email}</span> 吗？
                <br />
                <br />
                此操作将删除该账户的所有邮件数据，且无法恢复。
              </AlertDialogDescription>
            </AlertDialogHeader>
            <AlertDialogFooter>
              <AlertDialogCancel onClick={handleDeleteCancel}>取消</AlertDialogCancel>
              <AlertDialogAction
                onClick={handleDeleteConfirm}
                className="bg-red-600 hover:bg-red-700 focus:ring-red-600"
              >
                删除
              </AlertDialogAction>
            </AlertDialogFooter>
          </AlertDialogContent>
        </AlertDialog>
      </div>
    </div>
  );
};
