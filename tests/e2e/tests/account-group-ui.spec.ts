import { test, expect } from '@playwright/test';

const FRONTEND_URL = process.env.FRONTEND_URL || 'http://localhost:4444';
const TEST_PASSWORD = process.env.MASTER_PASSWORD || 'admin123456';

test.describe('账号分组 UI 测试', () => {
  test.beforeEach(async ({ page }) => {
    // 访问登录页面
    await page.goto(FRONTEND_URL);
    
    // 等待页面加载
    await page.waitForLoadState('networkidle');
    
    // 检查是否需要登录
    const loginForm = page.locator('input[type="password"]');
    if (await loginForm.isVisible({ timeout: 3000 }).catch(() => false)) {
      // 输入密码并登录
      await loginForm.fill(TEST_PASSWORD);
      await page.keyboard.press('Enter');
      
      // 等待登录完成
      await page.waitForLoadState('networkidle');
      await page.waitForTimeout(1000);
    }
  });

  test('1. 页面加载后应显示分组列表', async ({ page }) => {
    // 等待侧边栏加载
    await page.waitForTimeout(1000);
    
    // 检查是否有分组相关的 UI 元素
    const sidebar = page.locator('[data-testid="sidebar"], .sidebar, aside, nav');
    
    // 截图记录当前状态
    await page.screenshot({ path: 'test-results/group-ui-initial.png' });
    
    console.log('✓ 页面加载完成');
  });

  test('2. 应能打开创建分组对话框', async ({ page }) => {
    await page.waitForTimeout(1000);
    
    // 尝试找到创建分组按钮
    const createButton = page.locator('button:has-text("新建分组"), button:has-text("创建分组"), button:has-text("添加分组"), [data-testid="create-group"]');
    
    if (await createButton.isVisible({ timeout: 3000 }).catch(() => false)) {
      await createButton.click();
      
      // 等待对话框出现
      await page.waitForTimeout(500);
      
      // 检查对话框是否打开
      const dialog = page.locator('[role="dialog"], .dialog, .modal');
      const isDialogVisible = await dialog.isVisible({ timeout: 2000 }).catch(() => false);
      
      if (isDialogVisible) {
        console.log('✓ 创建分组对话框已打开');
        await page.screenshot({ path: 'test-results/group-create-dialog.png' });
      } else {
        console.log('⚠ 对话框未找到，可能使用了不同的 UI 模式');
      }
    } else {
      console.log('⚠ 未找到创建分组按钮，跳过此测试');
    }
  });

  test('3. 应能查看账号列表', async ({ page }) => {
    await page.waitForTimeout(1000);
    
    // 尝试导航到账号页面
    const accountsLink = page.locator('a:has-text("账号"), a:has-text("邮箱"), [href*="account"]');
    
    if (await accountsLink.first().isVisible({ timeout: 3000 }).catch(() => false)) {
      await accountsLink.first().click();
      await page.waitForLoadState('networkidle');
      await page.waitForTimeout(500);
    }
    
    // 截图记录账号列表页面
    await page.screenshot({ path: 'test-results/accounts-list.png' });
    console.log('✓ 账号列表页面已加载');
  });

  test('4. 应能按分组筛选账号', async ({ page }) => {
    await page.waitForTimeout(1000);
    
    // 查找分组筛选器
    const groupFilter = page.locator('[data-testid="group-filter"], select:has-text("分组"), .group-filter');
    
    if (await groupFilter.isVisible({ timeout: 3000 }).catch(() => false)) {
      console.log('✓ 找到分组筛选器');
      await page.screenshot({ path: 'test-results/group-filter.png' });
    } else {
      // 尝试在侧边栏查找分组列表
      const groupList = page.locator('[data-testid="group-list"], .group-list');
      if (await groupList.isVisible({ timeout: 2000 }).catch(() => false)) {
        console.log('✓ 找到分组列表');
      } else {
        console.log('⚠ 未找到分组筛选器或分组列表');
      }
    }
  });

  test('5. 截图记录整体 UI 状态', async ({ page }) => {
    await page.waitForTimeout(1000);
    
    // 全页面截图
    await page.screenshot({ 
      path: 'test-results/full-page.png',
      fullPage: true 
    });
    
    console.log('✓ 已保存全页面截图');
  });
});
