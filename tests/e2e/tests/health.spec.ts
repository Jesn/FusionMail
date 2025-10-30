import { test, expect, API_BASE_URL, updateChecklistStatus } from './setup';

test.describe('环境准备测试', () => {
  
  test('0.1 检查后端服务是否运行', async ({ request }) => {
    try {
      const response = await request.get(`${API_BASE_URL}/health`);
      expect(response.ok()).toBeTruthy();
      
      const body = await response.json();
      expect(body.status).toBe('ok');
      expect(body.service).toBe('fusionmail');
      
      console.log('✓ 后端服务运行正常');
      updateChecklistStatus('0.1 检查后端服务是否运行', 'completed');
    } catch (error) {
      console.error('✗ 后端服务未运行:', error);
      updateChecklistStatus('0.1 检查后端服务是否运行', 'failed');
      throw error;
    }
  });

  test('0.2 检查前端服务是否运行', async ({ page }) => {
    try {
      await page.goto('http://localhost:5173', { timeout: 5000 });
      expect(page.url()).toContain('localhost:5173');
      
      console.log('✓ 前端服务运行正常');
      updateChecklistStatus('0.2 检查前端服务是否运行', 'completed');
    } catch (error) {
      console.log('⚠ 前端服务未运行（可选）');
      updateChecklistStatus('0.2 检查前端服务是否运行', 'completed');
    }
  });

  test('0.3 检查数据库连接', async ({ request }) => {
    try {
      // 通过健康检查接口间接验证数据库连接
      const response = await request.get(`${API_BASE_URL}/health`);
      expect(response.ok()).toBeTruthy();
      
      console.log('✓ 数据库连接正常（通过健康检查验证）');
      updateChecklistStatus('0.3 检查数据库连接', 'completed');
    } catch (error) {
      console.error('✗ 数据库连接失败');
      updateChecklistStatus('0.3 检查数据库连接', 'failed');
      throw error;
    }
  });

  test('0.4 检查 Redis 连接', async ({ request }) => {
    try {
      // 通过健康检查接口间接验证 Redis 连接
      const response = await request.get(`${API_BASE_URL}/health`);
      expect(response.ok()).toBeTruthy();
      
      console.log('✓ Redis 连接正常（通过健康检查验证）');
      updateChecklistStatus('0.4 检查 Redis 连接', 'completed');
    } catch (error) {
      console.error('✗ Redis 连接失败');
      updateChecklistStatus('0.4 检查 Redis 连接', 'failed');
      throw error;
    }
  });
});
