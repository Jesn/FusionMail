import { test } from '@playwright/test';

const FRONTEND_URL = 'http://localhost:4444';

test('逐步测试登录流程', async ({ page }) => {
  console.log('\n=== 步骤 1: 访问登录页面 ===');
  await page.goto(FRONTEND_URL + '/login');
  await page.waitForLoadState('networkidle');
  console.log('✅ 页面加载完成');
  console.log('URL:', page.url());
  
  console.log('\n=== 步骤 2: 检查初始状态 ===');
  let token = await page.evaluate(() => localStorage.getItem('auth_token'));
  console.log('初始 token:', token ? '存在' : '不存在');
  
  console.log('\n=== 步骤 3: 填写密码 ===');
  await page.fill('input[type="password"]', 'admin123');
  console.log('✅ 密码已填写');
  
  console.log('\n=== 步骤 4: 监听导航事件 ===');
  page.on('framenavigated', frame => {
    if (frame === page.mainFrame()) {
      console.log('🔄 页面导航到:', frame.url());
    }
  });
  
  console.log('\n=== 步骤 5: 点击登录按钮 ===');
  await page.click('button[type="submit"]');
  console.log('✅ 按钮已点击');
  
  console.log('\n=== 步骤 6: 等待 API 响应 ===');
  await page.waitForTimeout(2000);
  
  console.log('\n=== 步骤 7: 检查登录后状态 ===');
  token = await page.evaluate(() => localStorage.getItem('auth_token'));
  console.log('登录后 token:', token ? '存在' : '不存在');
  if (token) {
    console.log('Token 前30个字符:', token.substring(0, 30));
  }
  
  const expires = await page.evaluate(() => localStorage.getItem('auth_expires'));
  console.log('过期时间:', expires);
  
  console.log('\n=== 步骤 8: 检查 URL ===');
  console.log('当前 URL:', page.url());
  
  console.log('\n=== 步骤 9: 等待可能的跳转 ===');
  await page.waitForTimeout(3000);
  console.log('3秒后 URL:', page.url());
  
  console.log('\n=== 步骤 10: 手动导航到 inbox ===');
  await page.goto(FRONTEND_URL + '/inbox');
  await page.waitForTimeout(2000);
  console.log('手动导航后 URL:', page.url());
  
  console.log('\n=== 步骤 11: 检查页面内容 ===');
  const bodyText = await page.textContent('body');
  if (bodyText?.includes('收件箱') || bodyText?.includes('Inbox')) {
    console.log('✅ 找到收件箱内容');
  } else if (bodyText?.includes('登录') || bodyText?.includes('Login')) {
    console.log('❌ 仍在登录页面');
  } else {
    console.log('⚠️ 未知页面状态');
  }
  
  // 截图
  await page.screenshot({ path: 'test-results/final-state.png', fullPage: true });
  console.log('\n📸 最终状态截图已保存');
});
