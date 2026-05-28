import type { InternalAxiosRequestConfig } from 'axios';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import apiClient from './api';
import { useAuthStore, type User } from '../stores/authStore';

const user: User = {
  id: 1,
  username: 'alice',
  email: 'alice@example.com',
  role: 'admin',
};

describe('apiClient', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    useAuthStore.setState({
      user: null,
      token: null,
      expiresAt: null,
      isAuthenticated: false,
      isLoading: false,
    });
    apiClient.defaults.adapter = undefined;
  });

  it('用户会话请求只依赖 Cookie，不从 store 注入 Bearer token', async () => {
    useAuthStore.getState().login(user, 'store-token', new Date(Date.now() + 60_000).toISOString());

    let capturedConfig: InternalAxiosRequestConfig | undefined;
    apiClient.defaults.adapter = async (config) => {
      capturedConfig = config;
      return {
        data: {},
        status: 200,
        statusText: 'OK',
        headers: {},
        config,
      };
    };

    await apiClient.get('/probe');

    expect(capturedConfig?.withCredentials).toBe(true);
    expect(capturedConfig?.headers.Authorization).toBeUndefined();
  });

  it('用户会话请求不会从旧 localStorage 快照恢复 Bearer token', async () => {
    vi.mocked(localStorage.getItem).mockImplementation((key) => {
      if (key === 'fusionmail-auth') {
        return JSON.stringify({ state: { token: 'legacy-local-storage-token' } });
      }
      return null;
    });

    let capturedConfig: InternalAxiosRequestConfig | undefined;
    apiClient.defaults.adapter = async (config) => {
      capturedConfig = config;
      return {
        data: {},
        status: 200,
        statusText: 'OK',
        headers: {},
        config,
      };
    };

    await apiClient.get('/probe');

    expect(capturedConfig?.withCredentials).toBe(true);
    expect(capturedConfig?.headers.Authorization).toBeUndefined();
  });
});
