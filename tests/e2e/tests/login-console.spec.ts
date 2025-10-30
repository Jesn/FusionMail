import { test } from '@playwright/test';

const FRONTEND_URL = 'http://localhost:3000';

test('检查登录过程中的控制台错误', async ({ page }) => {
  // 监听控制台消息
  const consoleMessages: string[] = [];
  const errors: string[] = [];
  
  page.on('console', msg => {
    const text = msg.text();
    consoleMessages.push(`[${msg.type()}] ${text}`);
    console.log(`[浏览器控制台 ${msg.type()}]`, text);
  });
  
  page.on('pageerror', error => {
    errors.push(error.message);
    console.log('[页面错误]', error.message);
  });
  
  // 访问登录页面
  console.log('访问登录页面...');
  await page.goto(FRONTEND_URL + '/login', { waitUntil: 'networkidle' });
  
  // 填写密码
  console.log('填写密码...');
  await page.fill('input[type="password"]', 'admin123');
  
  // 点击登录
  console.log('点击登录按钮...');
  await page.click('button[type="submit"]');
  
  // 等待一段时间
  await page.waitForTimeout(5000);
  
  // 输出所有控制台消息
  console.log('\n=== 控制台消息汇总 ===');
  consoleMessages.forEach(msg => console.log(msg));
  
  console.log('\n=== 页面错误汇总 ===');
  if (errors.length > 0) {
    errors.forEach(err => console.log(err));
  } else {
    console.log('无页面错误');
  }
  
  // 检查最终状态
  console.log('\n=== 最终状态 ===');
  console.log('当前 URL:', page.url());
  
  const token = await page.evaluate(() => localStorage.getItem('auth_token'));
  console.log('localStorage token:', token ? '存在' : '不存在');
  
  if (token) {
    console.log('Token 前20个字符:', token.substring(0, 20));
  }
});
