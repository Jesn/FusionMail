import { test, expect } from '@playwright/test';

test.describe('SSE 统计接口测试', () => {
  test.beforeEach(async ({ page }) => {
    // 导航到登录页
    await page.goto('http://localhost:4444/login');
    
    // 登录（使用默认密码 admin123）
    await page.fill('input[type="email"]', 'test@example.com');
    await page.fill('input[type="password"]', 'admin123');
    await page.click('button[type="submit"]');
    
    // 等待登录成功并跳转到主页
    await page.waitForURL('http://localhost:4444/**', { timeout: 10000 });
  });

  test('SSE 连接应该成功建立（200 状态）', async ({ page }) => {
    // 监听 SSE 请求
    const sseRequest = page.waitForRequest(
      request => request.url().includes('/api/v1/events'),
      { timeout: 5000 }
    );

    // 等待 SSE 请求
    const request = await sseRequest;
    
    // 验证请求 URL 正确（不应该有重复的 /api/v1）
    expect(request.url()).toMatch(/\/api\/v1\/events$/);
    expect(request.url()).not.toContain('/api/v1/api/v1');
    
    // 等待响应
    const response = await request.response();
    expect(response).not.toBeNull();
    
    // 验证状态码（应该是 200，不是 401）
    expect(response!.status()).toBe(200);
    
    // 验证响应头
    const headers = response!.headers();
    expect(headers['content-type']).toContain('text/event-stream');
  });

  test('统计接口应该只被请求一次（冷启动）', async ({ page }) => {
    let statsRequestCount = 0;
    
    // 监听所有统计接口请求
    page.on('request', request => {
      if (request.url().includes('/api/v1/emails/stats') || 
          request.url().includes('/emails/stats')) {
        statsRequestCount++;
        console.log(`统计请求 #${statsRequestCount}: ${request.url()}`);
      }
    });

    // 等待页面完全加载
    await page.waitForLoadState('networkidle');
    
    // 等待 1 秒确保没有额外请求
    await page.waitForTimeout(1000);
    
    // 验证统计接口只被调用一次
    expect(statsRequestCount).toBe(1);
  });

  test('标记邮件已读后应该触发 SSE 事件并刷新统计', async ({ page }) => {
    let statsRequestCount = 0;
    let sseEventReceived = false;

    // 监听统计接口请求
    page.on('request', request => {
      if (request.url().includes('/emails/stats')) {
        statsRequestCount++;
        console.log(`统计请求 #${statsRequestCount}: ${request.url()}`);
      }
    });

    // 等待初始加载完成
    await page.waitForLoadState('networkidle');
    const initialCount = statsRequestCount;
    
    // 查找第一封未读邮件并标记为已读
    const firstEmail = page.locator('[data-testid="email-item"]').first();
    if (await firstEmail.isVisible()) {
      await firstEmail.click();
      
      // 点击"标记已读"按钮
      const markReadBtn = page.locator('button:has-text("标记已读")');
      if (await markReadBtn.isVisible()) {
        await markReadBtn.click();
        
        // 等待 SSE 事件触发和去抖（400ms + 余量）
        await page.waitForTimeout(1000);
        
        // 验证统计接口被再次调用
        expect(statsRequestCount).toBeGreaterThan(initialCount);
      }
    }
  });

  test('验证 Cookie 正确设置', async ({ page, context }) => {
    // 获取所有 cookies
    const cookies = await context.cookies();
    
    // 查找 fm_session cookie
    const sessionCookie = cookies.find(c => c.name === 'fm_session');
    
    // 验证 cookie 存在
    expect(sessionCookie).toBeDefined();
    
    // 验证 cookie 属性
    expect(sessionCookie!.httpOnly).toBe(true);
    expect(sessionCookie!.path).toBe('/');
    
    console.log('Session Cookie:', {
      name: sessionCookie!.name,
      httpOnly: sessionCookie!.httpOnly,
      secure: sessionCookie!.secure,
      sameSite: sessionCookie!.sameSite,
      path: sessionCookie!.path,
    });
  });

  test('验证统计数据正确显示', async ({ page }) => {
    // 等待侧边栏加载
    await page.waitForSelector('[data-testid="sidebar"]', { timeout: 5000 });
    
    // 查找未读数徽标
    const unreadBadge = page.locator('[data-testid="unread-badge"]');
    
    if (await unreadBadge.isVisible()) {
      const unreadText = await unreadBadge.textContent();
      console.log('未读数显示:', unreadText);
      
      // 验证未读数是数字
      expect(unreadText).toMatch(/^\d+$/);
    }
  });
});

