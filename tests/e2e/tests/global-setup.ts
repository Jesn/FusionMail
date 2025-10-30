import { chromium, FullConfig } from '@playwright/test';
import { API_BASE_URL, TEST_CREDENTIALS } from './setup';

async function globalSetup(config: FullConfig) {
  // 如果已经有 token，跳过登录
  if (process.env.TEST_AUTH_TOKEN) {
    console.log('✅ 使用已缓存的 token');
    return;
  }
  
  console.log('🔐 执行全局登录...');
  
  const browser = await chromium.launch();
  const context = await browser.newContext();
  const page = await context.newPage();
  
  try {
    // 登录获取 token
    const response = await page.request.post(`${API_BASE_URL}/auth/login`, {
      data: {
        password: TEST_CREDENTIALS.password,
      },
    });
    
    if (!response.ok()) {
      const status = response.status();
      if (status === 429) {
        console.warn('⚠️  触发速率限制，等待 60 秒后重试...');
        await new Promise(resolve => setTimeout(resolve, 60000));
        
        // 重试登录
        const retryResponse = await page.request.post(`${API_BASE_URL}/auth/login`, {
          data: {
            password: TEST_CREDENTIALS.password,
          },
        });
        
        if (!retryResponse.ok()) {
          throw new Error(`Login retry failed with status ${retryResponse.status()}`);
        }
        
        const retryBody = await retryResponse.json();
        if (!retryBody.data || !retryBody.data.token) {
          throw new Error('Invalid login response: missing token');
        }
        
        process.env.TEST_AUTH_TOKEN = retryBody.data.token;
        console.log('✅ 重试登录成功，token 已缓存');
        return;
      }
      
      throw new Error(`Login failed with status ${status}`);
    }
    
    const body = await response.json();
    
    if (!body.data || !body.data.token) {
      throw new Error('Invalid login response: missing token');
    }
    
    // 将 token 保存到环境变量中
    process.env.TEST_AUTH_TOKEN = body.data.token;
    
    console.log('✅ 全局登录成功，token 已缓存');
  } catch (error) {
    console.error('❌ 全局登录失败:', error);
    throw error;
  } finally {
    await context.close();
    await browser.close();
  }
}

export default globalSetup;
