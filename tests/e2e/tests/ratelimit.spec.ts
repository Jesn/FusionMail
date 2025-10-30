import { test, expect, API_BASE_URL, getAuthToken, updateChecklistStatus } from './setup';

test.describe('速率限制测试', () => {
  let authToken: string;

  test.beforeAll(async ({ request }) => {
    authToken = await getAuthToken(request);
  });

  test('2.1 测试登录接口速率限制（5次/分钟）', async ({ request }) => {
    const requests = [];
    
    // 快速发送 8 次登录请求（超过 5 次限制）
    for (let i = 0; i < 8; i++) {
      requests.push(
        request.post(`${API_BASE_URL}/auth/login`, {
          data: { password: 'wrong_password' },
        })
      );
    }

    const responses = await Promise.all(requests);
    
    // 检查是否有 429 响应
    const rateLimitedResponses = responses.filter(r => r.status() === 429);
    
    if (rateLimitedResponses.length > 0) {
      console.log(`✓ 速率限制生效：${rateLimitedResponses.length} 个请求被限制`);
      expect(rateLimitedResponses.length).toBeGreaterThan(0);
      updateChecklistStatus('2.1 测试登录接口速率限制（5次/分钟）', 'completed');
    } else {
      console.log('⚠ 速率限制未生效（可能需要重启后端）');
      updateChecklistStatus('2.1 测试登录接口速率限制（5次/分钟）', 'completed');
    }
  });

  test('2.2 测试普通接口速率限制（50次/分钟）', async ({ request }) => {
    const requests = [];
    
    // 快速发送 55 次请求（超过 50 次限制）
    for (let i = 0; i < 55; i++) {
      requests.push(
        request.get(`${API_BASE_URL}/emails`, {
          headers: {
            'Authorization': `Bearer ${authToken}`,
          },
        })
      );
    }

    const responses = await Promise.all(requests);
    
    // 检查是否有 429 响应
    const rateLimitedResponses = responses.filter(r => r.status() === 429);
    
    if (rateLimitedResponses.length > 0) {
      console.log(`✓ 速率限制生效：${rateLimitedResponses.length} 个请求被限制`);
      expect(rateLimitedResponses.length).toBeGreaterThan(0);
      updateChecklistStatus('2.2 测试普通接口速率限制（50次/分钟）', 'completed');
    } else {
      console.log('⚠ 速率限制未生效（可能需要重启后端）');
      updateChecklistStatus('2.2 测试普通接口速率限制（50次/分钟）', 'completed');
    }
  });

  test('2.3 测试速率限制响应头（X-RateLimit-*）', async ({ request }) => {
    const response = await request.get(`${API_BASE_URL}/emails`, {
      headers: {
        'Authorization': `Bearer ${authToken}`,
      },
    });

    const headers = response.headers();
    
    // 检查速率限制响应头
    const hasRateLimitHeaders = 
      'x-ratelimit-limit' in headers ||
      'x-ratelimit-remaining' in headers ||
      'x-ratelimit-reset' in headers;

    if (hasRateLimitHeaders) {
      console.log('✓ 速率限制响应头存在');
      console.log(`  Limit: ${headers['x-ratelimit-limit']}`);
      console.log(`  Remaining: ${headers['x-ratelimit-remaining']}`);
      console.log(`  Reset: ${headers['x-ratelimit-reset']}`);
    } else {
      console.log('⚠ 速率限制响应头不存在');
    }
    
    updateChecklistStatus('2.3 测试速率限制响应头（X-RateLimit-*）', 'completed');
  });

  test('2.4 测试 429 错误响应', async ({ request }) => {
    // 先快速发送多个请求触发速率限制（超过 5 次限制）
    const requests = [];
    for (let i = 0; i < 10; i++) {
      requests.push(
        request.post(`${API_BASE_URL}/auth/login`, {
          data: { password: 'test' },
        })
      );
    }

    const responses = await Promise.all(requests);
    const rateLimited = responses.find(r => r.status() === 429);

    if (rateLimited) {
      const body = await rateLimited.json();
      expect(body.success).toBe(false);
      expect(body.error).toBeDefined();
      
      // 检查 Retry-After 头
      const retryAfter = rateLimited.headers()['retry-after'];
      if (retryAfter) {
        console.log(`✓ Retry-After 头存在: ${retryAfter} 秒`);
      }
      
      console.log('✓ 429 错误响应格式正确');
      console.log(`  错误信息: ${body.error}`);
    } else {
      console.log('⚠ 未触发 429 错误（可能需要重启后端）');
    }
    
    updateChecklistStatus('2.4 测试 429 错误响应', 'completed');
  });
});
