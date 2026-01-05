import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen } from '@testing-library/react';
import { MemoryRouter } from 'react-router-dom';
import { Sidebar } from './Sidebar';

// Mock stores
vi.mock('../../stores/uiStore', () => ({
  useUIStore: () => ({
    sidebarCollapsed: false,
  }),
}));

vi.mock('../../stores/emailStore', () => ({
  useEmailStore: () => ({
    filter: {},
    setFilter: vi.fn(),
    unreadCount: 5,
    starredCount: 2,
    archivedCount: 1,
    deletedCount: 0,
    spamCount: 3,
  }),
}));

vi.mock('../../stores/groupStore', () => ({
  useGroupStore: () => ({
    groups: [],
    selectedGroupId: -1,
    fetchGroups: vi.fn(),
    setSelectedGroupId: vi.fn(),
  }),
  ALL_ACCOUNTS_GROUP_ID: -1,
  UNGROUPED_GROUP_ID: -2,
}));

vi.mock('../../hooks/useAccounts', () => ({
  useAccounts: () => ({
    accounts: [],
  }),
}));

vi.mock('../email/ComposeEmail', () => ({
  ComposeEmail: () => null,
}));

describe('Sidebar', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  describe('邮件视图路由', () => {
    it('在 /inbox 路由应该显示 MailMenu', () => {
      render(
        <MemoryRouter initialEntries={['/inbox']}>
          <Sidebar />
        </MemoryRouter>
      );

      // MailMenu 应该显示写邮件按钮
      expect(screen.getByText('写邮件')).toBeInTheDocument();
      // MailMenu 应该显示搜索邮件按钮
      expect(screen.getByText('搜索邮件')).toBeInTheDocument();
      // 不应该显示返回按钮（AdminMenu 特有）
      expect(screen.queryByText('返回')).not.toBeInTheDocument();
    });

    it('在 /sent 路由应该显示 MailMenu', () => {
      render(
        <MemoryRouter initialEntries={['/sent']}>
          <Sidebar />
        </MemoryRouter>
      );

      expect(screen.getByText('写邮件')).toBeInTheDocument();
      expect(screen.queryByText('返回')).not.toBeInTheDocument();
    });

    it('在 /spam 路由应该显示 MailMenu', () => {
      render(
        <MemoryRouter initialEntries={['/spam']}>
          <Sidebar />
        </MemoryRouter>
      );

      expect(screen.getByText('写邮件')).toBeInTheDocument();
      expect(screen.queryByText('返回')).not.toBeInTheDocument();
    });

    it('在 /search 路由应该显示 MailMenu', () => {
      render(
        <MemoryRouter initialEntries={['/search']}>
          <Sidebar />
        </MemoryRouter>
      );

      expect(screen.getByText('写邮件')).toBeInTheDocument();
      expect(screen.queryByText('返回')).not.toBeInTheDocument();
    });
  });

  describe('设置视图路由', () => {
    it('在 /settings 路由应该显示 AdminMenu', () => {
      render(
        <MemoryRouter initialEntries={['/settings']}>
          <Sidebar />
        </MemoryRouter>
      );

      // AdminMenu 应该显示返回按钮
      expect(screen.getByText('返回')).toBeInTheDocument();
      // AdminMenu 应该显示管理菜单项
      expect(screen.getByText('账户管理')).toBeInTheDocument();
      expect(screen.getByText('个人设置')).toBeInTheDocument();
      // 不应该显示写邮件按钮（MailMenu 特有）
      expect(screen.queryByText('写邮件')).not.toBeInTheDocument();
    });

    it('在 /accounts 路由应该显示 AdminMenu', () => {
      render(
        <MemoryRouter initialEntries={['/accounts']}>
          <Sidebar />
        </MemoryRouter>
      );

      expect(screen.getByText('返回')).toBeInTheDocument();
      expect(screen.getByText('账户管理')).toBeInTheDocument();
      expect(screen.queryByText('写邮件')).not.toBeInTheDocument();
    });

    it('在 /rules 路由应该显示 AdminMenu', () => {
      render(
        <MemoryRouter initialEntries={['/rules']}>
          <Sidebar />
        </MemoryRouter>
      );

      expect(screen.getByText('返回')).toBeInTheDocument();
      expect(screen.getByText('邮件规则')).toBeInTheDocument();
      expect(screen.queryByText('写邮件')).not.toBeInTheDocument();
    });

    it('在 /webhooks 路由应该显示 AdminMenu', () => {
      render(
        <MemoryRouter initialEntries={['/webhooks']}>
          <Sidebar />
        </MemoryRouter>
      );

      expect(screen.getByText('返回')).toBeInTheDocument();
      expect(screen.getByText('Webhook')).toBeInTheDocument();
      expect(screen.queryByText('写邮件')).not.toBeInTheDocument();
    });

    it('在 /api-keys 路由应该显示 AdminMenu', () => {
      render(
        <MemoryRouter initialEntries={['/api-keys']}>
          <Sidebar />
        </MemoryRouter>
      );

      expect(screen.getByText('返回')).toBeInTheDocument();
      expect(screen.queryByText('写邮件')).not.toBeInTheDocument();
    });
  });
});
