import { test, expect } from '@playwright/test';

test.describe('账户状态显示功能', () => {
  test.beforeEach(async ({ page }) => {
    // 导航到账户页面
    await page.goto('/accounts');
    
    // 等待页面加载
    await page.waitForLoadState('networkidle');
  });

  test('应该显示账户状态', async ({ page }) => {
    // 检查是否有账户卡片
    const accountCards = page.locator('[data-testid="account-card"]');
    const cardCount = await accountCards.count();
    
    if (cardCount === 0) {
      console.log('没有账户，跳过测试');
      return;
    }

    // 检查第一个账户卡片
    const firstCard = accountCards.first();
    
    // 验证账户状态显示
    await expect(firstCard.locator('text=账户状态')).toBeVisible();
    
    // 验证状态标签存在
    const statusBadge = firstCard.locator('[data-testid="account-status-badge"]');
    await expect(statusBadge).toBeVisible();
    
    // 验证状态文本
    const statusText = await statusBadge.textContent();
    expect(['正常', '已禁用', '错误']).toContain(statusText);
  });

  test('应该显示启用/禁用按钮', async ({ page }) => {
    const accountCards = page.locator('[data-testid="account-card"]');
    const cardCount = await accountCards.count();
    
    if (cardCount === 0) {
      console.log('没有账户，跳过测试');
      return;
    }

    const firstCard = accountCards.first();
    
    // 验证启用/禁用按钮存在
    const toggleButton = firstCard.locator('[data-testid="toggle-status-button"]');
    await expect(toggleButton).toBeVisible();
    
    // 验证按钮标题
    const buttonTitle = await toggleButton.getAttribute('title');
    expect(['启用账户', '禁用账户']).toContain(buttonTitle);
  });

  test('应该能够切换账户状态', async ({ page }) => {
    const accountCards = page.locator('[data-testid="account-card"]');
    const cardCount = await accountCards.count();
    
    if (cardCount === 0) {
      console.log('没有账户，跳过测试');
      return;
    }

    const firstCard = accountCards.first();
    
    // 获取初始状态
    const initialStatusBadge = firstCard.locator('[data-testid="account-status-badge"]');
    const initialStatus = await initialStatusBadge.textContent();
    
    // 点击切换按钮
    const toggleButton = firstCard.locator('[data-testid="toggle-status-button"]');
    await toggleButton.click();
    
    // 等待状态更新
    await page.waitForTimeout(2000);
    
    // 验证状态已改变
    const updatedStatus = await initialStatusBadge.textContent();
    expect(updatedStatus).not.toBe(initialStatus);
    
    // 再次点击恢复状态
    await toggleButton.click();
    await page.waitForTimeout(2000);
    
    // 验证状态已恢复
    const finalStatus = await initialStatusBadge.textContent();
    expect(finalStatus).toBe(initialStatus);
  });

  test('禁用状态的账户应该显示红色标签', async ({ page }) => {
    const accountCards = page.locator('[data-testid="account-card"]');
    const cardCount = await accountCards.count();
    
    if (cardCount === 0) {
      console.log('没有账户，跳过测试');
      return;
    }

    const firstCard = accountCards.first();
    const statusBadge = firstCard.locator('[data-testid="account-status-badge"]');
    const statusText = await statusBadge.textContent();
    
    if (statusText === '已禁用') {
      // 验证禁用状态的样式
      await expect(statusBadge).toHaveClass(/destructive/);
    }
  });

  test('正常状态的账户应该显示默认标签', async ({ page }) => {
    const accountCards = page.locator('[data-testid="account-card"]');
    const cardCount = await accountCards.count();
    
    if (cardCount === 0) {
      console.log('没有账户，跳过测试');
      return;
    }

    const firstCard = accountCards.first();
    const statusBadge = firstCard.locator('[data-testid="account-status-badge"]');
    const statusText = await statusBadge.textContent();
    
    if (statusText === '正常') {
      // 验证正常状态的样式
      await expect(statusBadge).toHaveClass(/default/);
    }
  });

  test('应该显示状态图标', async ({ page }) => {
    const accountCards = page.locator('[data-testid="account-card"]');
    const cardCount = await accountCards.count();
    
    if (cardCount === 0) {
      console.log('没有账户，跳过测试');
      return;
    }

    const firstCard = accountCards.first();
    
    // 验证状态图标存在
    const statusIcon = firstCard.locator('[data-testid="account-status-icon"]');
    await expect(statusIcon).toBeVisible();
  });
});