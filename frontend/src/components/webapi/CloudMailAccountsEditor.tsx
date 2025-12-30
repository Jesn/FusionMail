// Cloud Mail 账户列表编辑器组件
import React from 'react';
import { Button } from '../ui/button';
import { Input } from '../ui/input';
import { Plus, Trash2 } from 'lucide-react';
import type { CloudMailAccount } from '../../types/webapi';

interface CloudMailAccountsEditorProps {
  accounts: CloudMailAccount[];
  onChange: (accounts: CloudMailAccount[]) => void;
}

/**
 * Cloud Mail 账户列表编辑器
 * 支持添加、删除和编辑多个邮箱账户
 */
export const CloudMailAccountsEditor: React.FC<CloudMailAccountsEditorProps> = ({
  accounts,
  onChange,
}) => {
  // 添加账户
  const handleAdd = () => {
    onChange([...accounts, { email: '', password: '' }]);
  };

  // 删除账户
  const handleRemove = (index: number) => {
    const newAccounts = accounts.filter((_, i) => i !== index);
    onChange(newAccounts);
  };

  // 更新账户
  const handleUpdate = (index: number, field: keyof CloudMailAccount, value: string) => {
    const newAccounts = accounts.map((account, i) => {
      if (i === index) {
        return { ...account, [field]: value };
      }
      return account;
    });
    onChange(newAccounts);
  };

  return (
    <div className="space-y-3">
      {accounts.length === 0 ? (
        <div className="text-sm text-muted-foreground text-center py-4 border border-dashed rounded-md">
          暂无账户，点击下方按钮添加
        </div>
      ) : (
        accounts.map((account, index) => (
          <div key={index} className="flex items-center gap-2">
            <div className="flex-1 grid grid-cols-2 gap-2">
              <Input
                type="email"
                placeholder="邮箱地址"
                value={account.email}
                onChange={(e) => handleUpdate(index, 'email', e.target.value)}
              />
              <Input
                type="password"
                placeholder="密码"
                value={account.password}
                onChange={(e) => handleUpdate(index, 'password', e.target.value)}
              />
            </div>
            <Button
              type="button"
              variant="ghost"
              size="icon"
              onClick={() => handleRemove(index)}
              className="text-destructive hover:text-destructive"
            >
              <Trash2 className="h-4 w-4" />
            </Button>
          </div>
        ))
      )}
      <Button
        type="button"
        variant="outline"
        size="sm"
        onClick={handleAdd}
        className="w-full"
      >
        <Plus className="h-4 w-4 mr-2" />
        添加账户
      </Button>
    </div>
  );
};

export default CloudMailAccountsEditor;
