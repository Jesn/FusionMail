import { useState } from 'react';
import { AccountTablePaginated } from '../components/account/AccountTablePaginated';
import { AccountToolbar } from '../components/account/AccountToolbar';
import { Account } from '../types';

// 生成测试数据
const generateTestAccounts = (count: number): Account[] => {
  const providers = ['gmail', 'outlook', 'imap', 'pop3'];
  const statuses = ['active', 'disabled', 'error'];
  const domains = ['gmail.com', 'outlook.com', 'qq.com', '163.com', 'company.com'];
  
  return Array.from({ length: count }, (_, i) => ({
    id: i + 1,
    uid: `test-account-${i + 1}`,
    email: `user${i + 1}@${domains[i % domains.length]}`,
    provider: providers[i % providers.length],
    protocol: providers[i % providers.length],
    auth_type: 'oauth2',
    status: statuses[i % statuses.length],
    sync_enabled: true,
    sync_interval: 5,
    unread_count: Math.floor(Math.random() * 100),
    last_sync_at: new Date(Date.now() - Math.random() * 7 * 24 * 60 * 60 * 1000).toISOString(),
    last_sync_status: Math.random() > 0.8 ? 'failed' : 'success',
    created_at: new Date(Date.now() - Math.random() * 30 * 24 * 60 * 60 * 1000).toISOString(),
    updated_at: new Date().toISOString(),
  } as Account));
};

export const AccountTableTestPage = () => {
  const [accounts] = useState(() => generateTestAccounts(75));
  
  // 筛选状态
  const [searchQuery, setSearchQuery] = useState('');
  const [statusFilter, setStatusFilter] = useState<'all' | 'active' | 'disabled' | 'error'>('all');
  const [providerFilter, setProviderFilter] = useState<'all' | 'gmail' | 'outlook' | 'imap' | 'pop3'>('all');
  const [syncStatusFilter, setSyncStatusFilter] = useState<'all' | 'success' | 'failed' | 'running' | 'never'>('all');
  
  // 选择状态
  const [selectedAccounts, setSelectedAccounts] = useState<string[]>([]);
  const [showSelection, setShowSelection] = useState(false);

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
    // Batch sync action
    setSelectedAccounts([]);
  };

  const handleBatchToggleStatus = () => {
    // Batch toggle status action
    setSelectedAccounts([]);
  };

  const handleSync = (_uid: string) => {
    // Sync account action
  };

  const handleDelete = (_uid: string, _email: string) => {
    // Delete account action
  };

  const handleEdit = (_account: Account) => {
    // Edit account action
  };

  const handleToggleStatus = (_uid: string, _status: string) => {
    // Toggle status action
  };

  const handleClearError = (_uid: string) => {
    // Clear error action
  };

  return (
    <div className="h-full overflow-auto">
      <div className="mx-auto max-w-7xl p-6">
        {/* 头部 */}
        <div className="mb-6">
          <h1 className="text-3xl font-bold mb-2">账户表格测试页面</h1>
          <p className="text-muted-foreground">
            测试新的表格分页功能 - 共 {accounts.length} 个测试账户
          </p>
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
            density="compact"
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
          syncingAccounts={[]}
        />

        {/* 功能说明 */}
        <div className="mt-8 p-4 bg-muted/50 rounded-lg">
          <h3 className="font-medium mb-2">测试功能：</h3>
          <ul className="text-sm text-muted-foreground space-y-1">
            <li>• 搜索功能：在搜索框中输入邮箱地址或域名</li>
            <li>• 筛选功能：使用状态、服务商、同步状态筛选器</li>
            <li>• 排序功能：点击表格列标题进行排序</li>
            <li>• 分页功能：使用底部分页控件导航</li>
            <li>• 批量操作：选择多个账户进行批量操作</li>
            <li>• 操作菜单：点击每行的操作按钮查看可用操作</li>
          </ul>
        </div>
      </div>
    </div>
  );
};