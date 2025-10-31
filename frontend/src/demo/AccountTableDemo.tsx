import { useState } from 'react';
import { AccountTablePaginated } from '../components/account/AccountTablePaginated';
import { AccountToolbar } from '../components/account/AccountToolbar';
import { Account } from '../types';

// 生成演示数据
const generateDemoAccounts = (count: number): Account[] => {
  const providers = ['gmail', 'outlook', 'imap', 'pop3'];
  const statuses = ['active', 'disabled', 'error'];
  const domains = ['gmail.com', 'outlook.com', 'qq.com', '163.com', 'company.com', 'example.org'];
  
  return Array.from({ length: count }, (_, i) => {
    const provider = providers[i % providers.length];
    const status = statuses[i % statuses.length];
    const domain = domains[i % domains.length];
    const email = `user${i + 1}@${domain}`;
    
    return {
      uid: `account-${i + 1}`,
      email,
      provider,
      status,
      unread_count: Math.floor(Math.random() * 100),
      last_sync_at: new Date(Date.now() - Math.random() * 7 * 24 * 60 * 60 * 1000).toISOString(),
      last_sync_status: Math.random() > 0.8 ? 'failed' : 'success',
      created_at: new Date(Date.now() - Math.random() * 30 * 24 * 60 * 60 * 1000).toISOString(),
      updated_at: new Date().toISOString(),
    } as Account;
  });
};

export const AccountTableDemo = () => {
  // 生成不同数量的演示数据
  const [accountCount, setAccountCount] = useState(75);
  const [accounts] = useState(() => generateDemoAccounts(accountCount));
  
  // 筛选状态
  const [searchQuery, setSearchQuery] = useState('');
  const [statusFilter, setStatusFilter] = useState<'all' | 'active' | 'disabled' | 'error'>('all');
  const [providerFilter, setProviderFilter] = useState<'all' | 'gmail' | 'outlook' | 'imap' | 'pop3'>('all');
  const [syncStatusFilter, setSyncStatusFilter] = useState<'all' | 'success' | 'failed' | 'running' | 'never'>('all');
  
  // 选择状态
  const [selectedAccounts, setSelectedAccounts] = useState<string[]>([]);
  const [showSelection, setShowSelection] = useState(false);
  const [syncingAccounts] = useState<string[]>([]);

  // 统计信息
  const activeCount = accounts.filter(acc => acc.status === 'active').length;
  const errorCount = accounts.filter(acc => acc.status === 'error').length;

  // 事件处理
  const handleAccountSelect = (uid: string, selected: boolean) => {
    if (selected) {
      setSelectedAccounts(prev => [...prev, uid]);
    } else {
      setSelectedAccounts(prev => prev.filter(id => id !== uid));
    }
  };

  const handleSelectAll = () => {
    setSelectedAccounts(accounts.map(acc => acc.uid));
  };

  const handleClearSelection = () => {
    setSelectedAccounts([]);
    setShowSelection(false);
  };

  const handleBatchSync = () => {
    console.log('批量同步:', selectedAccounts);
    setSelectedAccounts([]);
  };

  const handleBatchToggleStatus = () => {
    console.log('批量切换状态:', selectedAccounts);
    setSelectedAccounts([]);
  };

  const handleSync = (uid: string) => {
    console.log('同步账户:', uid);
  };

  const handleDelete = (uid: string, email: string) => {
    console.log('删除账户:', uid, email);
  };

  const handleEdit = (account: Account) => {
    console.log('编辑账户:', account);
  };

  const handleToggleStatus = (uid: string, status: string) => {
    console.log('切换状态:', uid, status);
  };

  const handleClearError = (uid: string) => {
    console.log('清除错误:', uid);
  };

  return (
    <div className="h-full overflow-auto">
      <div className="mx-auto max-w-7xl p-6">
        {/* 头部 */}
        <div className="mb-6">
          <h1 className="text-3xl font-bold mb-2">邮箱账户表格演示</h1>
          <p className="text-muted-foreground mb-4">
            演示表格分页模式 - 当前显示 {accountCount} 个账户
          </p>
          
          {/* 数据量控制 */}
          <div className="flex items-center gap-4 mb-4">
            <span className="text-sm font-medium">演示数据量:</span>
            {[25, 50, 75, 100, 150].map(count => (
              <button
                key={count}
                onClick={() => setAccountCount(count)}
                className={`px-3 py-1 text-sm rounded ${
                  accountCount === count 
                    ? 'bg-primary text-primary-foreground' 
                    : 'bg-secondary text-secondary-foreground hover:bg-secondary/80'
                }`}
              >
                {count}
              </button>
            ))}
          </div>
        </div>

        {/* 工具栏 */}
        <div className="mb-6">
          <AccountToolbar
            searchQuery={searchQuery}
            onSearchChange={setSearchQuery}
            statusFilter={statusFilter}
            onStatusFilterChange={setStatusFilter}
            providerFilter={providerFilter}
            onProviderFilterChange={setProviderFilter}
            syncStatusFilter={syncStatusFilter}
            onSyncStatusFilterChange={setSyncStatusFilter}
            density="compact" // 表格模式不需要密度控制
            onDensityChange={() => {}}
            viewMode="table"
            onViewModeChange={() => {}}
            groupBy="none"
            onGroupByChange={() => {}}
            selectedCount={selectedAccounts.length}
            totalCount={accounts.length}
            onSelectAll={handleSelectAll}
            onClearSelection={handleClearSelection}
            onBatchSync={handleBatchSync}
            onBatchToggleStatus={handleBatchToggleStatus}
            activeCount={activeCount}
            errorCount={errorCount}
          />
        </div>

        {/* 表格 */}
        <AccountTablePaginated
          accounts={accounts}
          searchQuery={searchQuery}
          statusFilter={statusFilter}
          providerFilter={providerFilter}
          selectedAccounts={selectedAccounts}
          onAccountSelect={handleAccountSelect}
          showSelection={showSelection}
          onSync={handleSync}
          onDelete={handleDelete}
          onEdit={handleEdit}
          onToggleStatus={handleToggleStatus}
          onClearError={handleClearError}
          syncingAccounts={syncingAccounts}
        />

        {/* 操作说明 */}
        <div className="mt-8 p-4 bg-muted/50 rounded-lg">
          <h3 className="font-medium mb-2">功能特点：</h3>
          <ul className="text-sm text-muted-foreground space-y-1">
            <li>• <strong>智能分页</strong>：每页 20-100 条记录，性能优秀</li>
            <li>• <strong>多维筛选</strong>：支持搜索、状态、服务商、同步状态筛选</li>
            <li>• <strong>列排序</strong>：点击列标题进行升序/降序排序</li>
            <li>• <strong>批量操作</strong>：支持批量选择和操作</li>
            <li>• <strong>实时搜索</strong>：支持邮箱地址和域名搜索</li>
            <li>• <strong>状态指示</strong>：直观的状态徽章和同步状态显示</li>
          </ul>
        </div>
      </div>
    </div>
  );
};