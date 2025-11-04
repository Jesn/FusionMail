import { test, expect } from '@playwright/test';

const FRONTEND_URL = 'http://localhost:4444';
const API_BASE_URL = 'http://localhost:3333/api/v1';

test.describe('登录功能调试测试', () => {
  test('详细测试登录流程', async ({ page }) => {
    console.log('🔍 开始登录测试...');
    
    // 1. 访问登录页面
    console.log('1️⃣ 访问登录页面:', FRONTEND_URL + '/login');
    await page.goto(FRONTEND_URL + '/login', { waitUntil: 'networkidle' });
    
    // 截图
    await page.screenshot({ path: 'test-results/login-page.png', fullPage: true });
    console.log('📸 登录页面截图已保存');
    
    // 2. 检查页面标题
    const title = await page.title();
    console.log('📄 页面标题:', title);
    
    // 3. 查找所有输入框
    const allInputs = await page.locator('input').all();
    console.log('🔍 找到', allInputs.length, '个输入框');
    
    for (let i = 0; i < allInputs.length; i++) {
      const input = allInputs[i];
      const type = await input.getAttribute('type');
      const name = await input.getAttribute('name');
      const placeholder = await input.getAttribute('placeholder');
      const id = await input.getAttribute('id');
      console.log(`  输入框 ${i + 1}:`, { type, name, placeholder, id });
    }
    
    // 4. 查找所有按钮
    const allButtons = await page.locator('button').all();
    console.log('🔍 找到', allButtons.length, '个按钮');
    
    for (let i = 0; i < allButtons.length; i++) {
      const button = allButtons[i];
      const text = await button.textContent();
      const type = await button.getAttribute('type');
      console.log(`  按钮 ${i + 1}:`, { text: text?.trim(), type });
    }
    
    // 5. 尝试填写密码
    console.log('5️⃣ 尝试填写密码...');
    const passwordInput = page.locator('input[type="password"]').first();
    const passwordCount = await page.locator('input[type="password"]').count();
    console.log('🔍 找到', passwordCount, '个密码输入框');
    
    if (passwordCount > 0) {
      await passwordInput.fill('admin123');
      console.log('✅ 密码已填写');
      
      // 检查输入值
      const value = await passwordInput.inputValue();
      console.log('🔍 输入框值:', value ? '***' : '(空)');
    } else {
      console.log('❌ 未找到密码输入框');
    }
    
    // 6. 尝试点击登录按钮
    console.log('6️⃣ 尝试点击登录按钮...');
    const loginButton = page.locator('button[type="submit"]').first();
    const loginButtonCount = await page.locator('button[type="submit"]').count();
    console.log('🔍 找到', loginButtonCount, '个提交按钮');
    
    if (loginButtonCount > 0) {
      // 监听网络请求
      page.on('request', request => {
        if (request.url().includes('/api/')) {
          console.log('📤 API 请求:', request.method(), request.url());
        }
      });
      
      page.on('response', async response => {
        if (response.url().includes('/api/')) {
          console.log('📥 API 响应:', response.status(), response.url());
          try {
            const body = await response.json();
            console.log('📦 响应内容:', JSON.stringify(body, null, 2));
          } catch (e) {
            console.log('⚠️ 无法解析响应内容');
          }
        }
      });
      
      // 点击登录按钮
      await loginButton.click();
      console.log('✅ 登录按钮已点击');
      
      // 等待响应
      await page.waitForTimeout(3000);
      
      // 检查 URL 变化
      const currentUrl = page.url();
      console.log('🔍 当前 URL:', currentUrl);
      
      // 检查是否有错误提示
      const errorElements = await page.locator('.error, .alert, [role="alert"], .text-red-500, .text-destructive').all();
      console.log('🔍 找到', errorElements.length, '个错误提示元素');
      
      for (let i = 0; i < errorElements.length; i++) {
        const text = await errorElements[i].textContent();
        if (text && text.trim()) {
          console.log(`  错误 ${i + 1}:`, text.trim());
        }
      }
      
      // 截图
      await page.screenshot({ path: 'test-results/after-login.png', fullPage: true });
      console.log('📸 登录后截图已保存');
      
      // 检查 localStorage
      const token = await page.evaluate(() => localStorage.getItem('token'));
      console.log('🔍 localStorage token:', token ? '存在' : '不存在');
      
      // 检查 cookies
      const cookies = await page.context().cookies();
      console.log('🔍 Cookies:', cookies.length, '个');
      cookies.forEach(cookie => {
        console.log(`  - ${cookie.name}: ${cookie.value.substring(0, 20)}...`);
      });
      
    } else {
      console.log('❌ 未找到登录按钮');
    }
    
    // 7. 直接测试 API
    console.log('7️⃣ 直接测试登录 API...');
    const apiResponse = await page.request.post(API_BASE_URL + '/auth/login', {
      data: {
        password: 'admin123'
      }
    });
    
    console.log('📥 API 响应状态:', apiResponse.status());
    const apiBody = await apiResponse.json();
    console.log('📦 API 响应内容:', JSON.stringify(apiBody, null, 2));
  });
});
