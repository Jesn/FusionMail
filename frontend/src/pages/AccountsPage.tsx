import { Plus } from 'lucide-react';
import { Button } from '../components/ui/button';
import { AccountCard } from '../components/account/AccountCard';
import { AccountForm } from '../components/account/AccountForm';
import { useAccounts } from '../hooks/useAccounts';
import { useUIStore } from '../stores/uiStore';

export const AccountsPage = () => {
  const {
    accounts,
    isLoading,
    createAccount,
    deleteAccount,
    testConnection,
    syncAccount,
  } = useAccounts();
  const { isAccountDialogOpen, setAccountDialogOpen } = useUIStore();

  const handleDelete = async (uid: string, email: string) => {
    if (confirm(`确定要删除账户 ${email} 吗？`)) {
      await deleteAccount(uid);
    }
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
                onDelete={() => handleDelete(account.uid, account.email)}
                onTest={() => testConnection(account.uid)}
              />
            ))}
          </div>
        )}

        {/* 添加账户对话框 */}
        <AccountForm
          open={isAccountDialogOpen}
          onClose={() => setAccountDialogOpen(false)}
          onSubmit={createAccount}
        />
      </div>
    </div>
  );
};
