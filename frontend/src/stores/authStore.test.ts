import { beforeEach, describe, expect, it, vi } from 'vitest';
import { useAuthStore, type User } from './authStore';

const user: User = {
  id: 1,
  username: 'alice',
  email: 'alice@example.com',
  role: 'admin',
};

describe('authStore', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    useAuthStore.setState({
      user: null,
      token: null,
      expiresAt: null,
      isAuthenticated: false,
      isLoading: false,
    });
  });

  it('持久化登录状态时不写入 JWT', () => {
    const expiresAt = new Date(Date.now() + 60_000).toISOString();

    useAuthStore.getState().login(user, 'frontend-visible-token', expiresAt);

    const persistCall = vi
      .mocked(localStorage.setItem)
      .mock.calls.find(([key]) => key === 'fusionmail-auth');

    expect(persistCall).toBeDefined();
    expect(persistCall?.[1]).not.toContain('frontend-visible-token');
    expect(persistCall?.[1]).not.toContain('"token"');
  });
});
