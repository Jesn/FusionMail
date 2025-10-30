import { test, expect, API_BASE_URL, TEST_CREDENTIALS, updateChecklistStatus } from './setup';

test.describe('认证与授权测试', () => {
  
  test('1.1 测试登录功能（正确密码）', async ({ request }) => {
    const response = await request.post(`${API_BASE_URL}/auth/login`, {
      data: {
        password: TEST_CREDENTIALS.password,
      },
    });

    expect(response.ok()).toBeTruthy();
    const body = await response.json();
    expect(body.success).toBe(true);
    expect(body.data.token).toBeDefined();
    expect(body.data.expiresAt).toBeDefined();
    
    updateChecklistStatus('1.1 测试登录功能（正确密码）', 'completed');
  });

  test('1.2 测试登录功能（错误密码）', async ({ request }) => {
    const response = await request.post(`${API_BASE_URL}/auth/login`, {
      data: {
        password: 'wrong_password_123',
      },
    });

    expect(response.status()).toBe(401);
    const body = await response.json();
    expect(body.error).toBeDefined();
    
    updateChecklistStatus('1.2 测试登录功能（错误密码）', 'completed');
  });

  test('1.3 测试 Token 验证', async ({ request }) => {
    // 先登录获取 token
    const loginResponse = await request.post(`${API_BASE_URL}/auth/login`, {
      data: { password: TEST_CREDENTIALS.password },
    });
    const loginBody = await loginResponse.json();
    const token = loginBody.data.token;

    // 验证 token
    const verifyResponse = await request.get(`${API_BASE_URL}/auth/verify`, {
      headers: {
        'Authorization': `Bearer ${token}`,
      },
    });

    expect(verifyResponse.ok()).toBeTruthy();
    const verifyBody = await verifyResponse.json();
    expect(verifyBody.valid).toBe(true);
    
    updateChecklistStatus('1.3 测试 Token 验证', 'completed');
  });

  test('1.4 测试 Token 刷新', async ({ request }) => {
    // 先登录获取 token
    const loginResponse = await request.post(`${API_BASE_URL}/auth/login`, {
      data: { password: TEST_CREDENTIALS.password },
    });
    const loginBody = await loginResponse.json();
    const oldToken = loginBody.data.token;

    // 刷新 token
    const refreshResponse = await request.post(`${API_BASE_URL}/auth/refresh`, {
      data: {
        token: oldToken,
      },
    });

    expect(refreshResponse.ok()).toBeTruthy();
    const refreshBody = await refreshResponse.json();
    expect(refreshBody.success).toBe(true);
    expect(refreshBody.data.token).toBeDefined();
    expect(refreshBody.data.token).not.toBe(oldToken);
    
    updateChecklistStatus('1.4 测试 Token 刷新', 'completed');
  });

  test('1.5 测试登出功能', async ({ request }) => {
    const response = await request.post(`${API_BASE_URL}/auth/logout`);

    expect(response.ok()).toBeTruthy();
    const body = await response.json();
    expect(body.message).toBeDefined();
    
    updateChecklistStatus('1.5 测试登出功能', 'completed');
  });

  test('1.6 测试未认证访问受保护接口', async ({ request }) => {
    const response = await request.get(`${API_BASE_URL}/accounts`);

    expect(response.status()).toBe(401);
    const body = await response.json();
    expect(body.success).toBe(false);
    expect(body.error).toBeDefined();
    
    updateChecklistStatus('1.6 测试未认证访问受保护接口', 'completed');
  });
});
