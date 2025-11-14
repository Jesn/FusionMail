import { test, expect } from '@playwright/test';

/**
 * SSE Cookie 鉴权测试
 * 测试目标：
 * 1. 验证 SSE 连接能够正确使用 Cookie 进行鉴权
 * 2. 验证 /emails/stats 接口不会被重复调用
 * 3. 验证 SSE 事件能够正确触发统计更新
 */

test.describe('SSE Cookie 鉴权测试', () => {
  test.beforeEach(async ({ page }) => {
    // 监听所有网络请求
    await page.route('**/*', route => route.continue());
  });

  test('登录后 SSE 应该能够使用 Cookie 鉴权', async ({ page }) => {
    const requests: { url: string; method: string; headers: any }[] = [];
    
    // 监听所有请求
    page.on('request', request => {
      requests.push({
        url: request.url(),
        method: request.method(),
        headers: request.headers(),
      });
    });

    // 1. 访问登录页
    await page.goto('http://localhost:4444/login');
    
    // 2. 填写登录表单
    await page.fill('input[type="email"]', 'admin@example.com');
    await page.fill('input[type="password"]', 'admin123');
    
    // 3. 点击登录按钮
    await page.click('button[type="submit"]');
    
    // 4. 等待跳转到主页
    await page.waitForURL('http://localhost:4444/inbox', { timeout: 10000 });
    
    // 5. 等待一段时间，让 SSE 连接建立
    await page.waitForTimeout(2000);
    
    // 6. 检查 SSE 连接请求
    const sseRequests = requests.filter(r => r.url.includes('/api/v1/events'));
    console.log('SSE 请求数量:', sseRequests.length);
    console.log('SSE 请求详情:', sseRequests);
    
    // 应该有至少一个 SSE 连接请求
    expect(sseRequests.length).toBeGreaterThan(0);
    
    // 7. 检查 SSE 请求是否携带了 Cookie
    const sseRequest = sseRequests[0];
    console.log('SSE 请求 headers:', sseRequest.headers);
    
    // Cookie 应该存在
    expect(sseRequest.headers['cookie']).toBeDefined();
    expect(sseRequest.headers['cookie']).toContain('fm_session');
    
    // 8. 检查页面是否有 401 错误
    const responses = await page.evaluate(() => {
      return (window as any).__networkErrors || [];
    });
    
    const has401 = responses.some((r: any) => r.status === 401);
    expect(has401).toBe(false);
  });

  test('/emails/stats 接口不应该被重复调用', async ({ page }) => {
    const statsRequests: { url: string; timestamp: number }[] = [];
    
    // 监听 /emails/stats 请求
    page.on('request', request => {
      if (request.url().includes('/emails/stats') && !request.url().includes('/stats/')) {
        statsRequests.push({
          url: request.url(),
          timestamp: Date.now(),
        });
      }
    });

    // 1. 登录
    await page.goto('http://localhost:4444/login');
    await page.fill('input[type="email"]', 'admin@example.com');
    await page.fill('input[type="password"]', 'admin123');
    await page.click('button[type="submit"]');
    await page.waitForURL('http://localhost:4444/inbox', { timeout: 10000 });
    
    // 2. 等待初始加载完成
    await page.waitForTimeout(3000);
    
    // 3. 检查请求次数
    console.log('/emails/stats 请求次数:', statsRequests.length);
    console.log('/emails/stats 请求详情:', statsRequests);
    
    // 在初始加载时，应该只有 1-2 次请求（一次是冷启动，一次可能是 SSE open 事件）
    // 但不应该有更多的重复请求
    expect(statsRequests.length).toBeLessThanOrEqual(2);
    
    // 4. 如果有多次请求，检查时间间隔
    if (statsRequests.length > 1) {
      const timeDiff = statsRequests[1].timestamp - statsRequests[0].timestamp;
      console.log('两次请求的时间间隔:', timeDiff, 'ms');
      
      // 时间间隔应该大于防抖时间（400ms）
      expect(timeDiff).toBeGreaterThan(400);
    }
  });

  test('SSE 事件应该能够触发统计更新', async ({ page, context }) => {
    const statsRequests: string[] = [];
    
    // 监听 /emails/stats 请求
    page.on('request', request => {
      if (request.url().includes('/emails/stats') && !request.url().includes('/stats/')) {
        statsRequests.push(request.url());
      }
    });

    // 1. 登录
    await page.goto('http://localhost:4444/login');
    await page.fill('input[type="email"]', 'admin@example.com');
    await page.fill('input[type="password"]', 'admin123');
    await page.click('button[type="submit"]');
    await page.waitForURL('http://localhost:4444/inbox', { timeout: 10000 });
    
    // 2. 等待初始加载
    await page.waitForTimeout(2000);
    
    // 3. 记录初始请求数
    const initialRequestCount = statsRequests.length;
    console.log('初始请求数:', initialRequestCount);
    
    // 4. 触发一个会导致统计变化的操作（例如标记为已读）
    // 首先找到第一封未读邮件
    const firstEmail = page.locator('[data-email-item]').first();
    if (await firstEmail.isVisible()) {
      // 点击邮件进入详情页
      await firstEmail.click();
      await page.waitForTimeout(1000);
      
      // 查找并点击"标记为已读"按钮（如果存在）
      const markReadButton = page.locator('button:has-text("标记为已读")');
      if (await markReadButton.isVisible()) {
        await markReadButton.click();
        
        // 5. 等待 SSE 事件触发和防抖
        await page.waitForTimeout(1000);
        
        // 6. 检查是否有新的统计请求
        const newRequestCount = statsRequests.length;
        console.log('操作后请求数:', newRequestCount);
        
        // 应该有新的请求（由 SSE 事件触发）
        expect(newRequestCount).toBeGreaterThan(initialRequestCount);
      }
    }
  });

  test('检查 Cookie 是否正确设置', async ({ page, context }) => {
    // 1. 登录
    await page.goto('http://localhost:4444/login');
    await page.fill('input[type="email"]', 'admin@example.com');
    await page.fill('input[type="password"]', 'admin123');
    await page.click('button[type="submit"]');
    await page.waitForURL('http://localhost:4444/inbox', { timeout: 10000 });
    
    // 2. 获取所有 Cookie
    const cookies = await context.cookies();
    console.log('所有 Cookie:', cookies);
    
    // 3. 查找 fm_session Cookie
    const sessionCookie = cookies.find(c => c.name === 'fm_session');
    
    // 4. 验证 Cookie 存在
    expect(sessionCookie).toBeDefined();
    
    // 5. 验证 Cookie 属性
    if (sessionCookie) {
      console.log('fm_session Cookie:', sessionCookie);
      
      // HttpOnly 应该为 true
      expect(sessionCookie.httpOnly).toBe(true);
      
      // Path 应该为 /
      expect(sessionCookie.path).toBe('/');
      
      // SameSite 应该为 Lax
      expect(sessionCookie.sameSite).toBe('Lax');
    }
  });

  test('验证 axios withCredentials 配置', async ({ page }) => {
    // 1. 登录
    await page.goto('http://localhost:4444/login');
    await page.fill('input[type="email"]', 'admin@example.com');
    await page.fill('input[type="password"]', 'admin123');
    await page.click('button[type="submit"]');
    await page.waitForURL('http://localhost:4444/inbox', { timeout: 10000 });
    
    // 2. 在页面中执行代码检查 axios 配置
    const axiosConfig = await page.evaluate(() => {
      // 尝试访问 axios 实例的配置
      return {
        // 这里我们无法直接访问 axios 实例，但可以通过发起请求来验证
        message: 'axios 配置需要通过网络请求来验证'
      };
    });
    
    console.log('axios 配置:', axiosConfig);
    
    // 3. 发起一个 API 请求并检查是否携带 Cookie
    const requestPromise = page.waitForRequest(request => 
      request.url().includes('/api/v1/emails/stats')
    );
    
    // 触发一个统计请求
    await page.reload();
    
    const request = await requestPromise;
    const headers = request.headers();
    
    console.log('API 请求 headers:', headers);
    
    // 验证请求携带了 Cookie
    expect(headers['cookie']).toBeDefined();
    expect(headers['cookie']).toContain('fm_session');
  });
});

