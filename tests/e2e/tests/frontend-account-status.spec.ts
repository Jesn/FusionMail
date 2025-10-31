import { test, expect } from '@playwright/test';

test.describe('前端账户状态显示测试', () => {
  test.beforeEach(async ({ page }) => {
    // 设置认证 token
    const token = process.env.TEST_AUTH_TOKEN;
    if (token) {
      await page.addInitScript((token) => {
        localStorage.setItem('auth_token', token);
      }, token);
    }
  });

  test('检查账户管理页面是否显示账户状态', async ({ page }) => {
    console.log('🧪 检查前端账户状态显示...');
    
    try {
      // 访问账户管理页面
      await page.goto('http://localhost:3000/accounts');
      await page.waitForLoadState('networkidle');
      
      // 检查页面是否加载成功
      const title = await page.title();
      expect(title).toContain('FusionMail');
      
      console.log('✅ 账户管理页面加载成功');
      
      // 等待账户数据加载
      await page.waitForTimeout(2000);
      
      // 查找账户卡片或列表项
      const accountElements = [
        '.account-card',
        '.account-item',
        '[data-testid*="account"]',
        '.account',
        '.card'
      ];
      
      let accountsFound = false;
      let accountCount = 0;
      
      for (const selector of accountElements) {
        const elements = page.locator(selector);
        const count = await elements.count();
        if (count > 0) {
          console.log(`✅ 找到 ${count} 个账户元素: ${selector}`);
          accountsFound = true;
          accountCount = count;
          
          // 检查每个账户元素的内容
          for (let i = 0; i < Math.min(count, 3); i++) {
            const element = elements.nth(i);
            const text = await element.textContent();
            console.log(`\n账户 ${i + 1} 显示内容:`);
            console.log(text);
            
            // 检查是否显示了状态相关信息
            const statusIndicators = [
              '启用', '禁用', 'enabled', 'disabled',
              '正常', '失败', 'active', 'inactive',
              '同步中', '已停止', 'syncing', 'stopped',
              '连接成功', '连接失败', 'connected', 'disconnected'
            ];
            
            let hasStatusInfo = false;
            for (const indicator of statusIndicators) {
              if (text && text.toLowerCase().includes(indicator.toLowerCase())) {
                console.log(`✅ 发现状态指示器: ${indicator}`);
                hasStatusInfo = true;
                break;
              }
            }
            
            if (!hasStatusInfo) {
              console.log('⚠️ 未发现明显的状态指示器');
            }
          }
          break;
        }
      }
      
      if (!accountsFound) {
        console.log('⚠️ 未找到账户显示元素，可能页面结构不同');
        
        // 尝试查找其他可能的元素
        const alternativeSelectors = [
          'div', 'li', 'tr', 'section'
        ];
        
        for (const selector of alternativeSelectors) {
          const elements = page.locator(selector);
          const count = await elements.count();
          
          if (count > 0 && count < 50) { // 避免选择太多元素
            for (let i = 0; i < Math.min(count, 10); i++) {
              const element = elements.nth(i);
              const text = await element.textContent();
              
              if (text && (
                text.includes('qq.com') || 
                text.includes('163.com') ||
                text.includes('794382693') ||
                text.includes('15026732619')
              )) {
                console.log(`\n找到可能的账户信息 (${selector}):`);
                console.log(text);
                accountsFound = true;
              }
            }
          }
        }
      }
      
      // 检查是否有状态切换按钮或开关
      const toggleElements = [
        'button:has-text("启用")',
        'button:has-text("禁用")',
        'button:has-text("Enable")',
        'button:has-text("Disable")',
        '[type="checkbox"]',
        '.toggle',
        '.switch',
        '[data-testid*="toggle"]',
        '[data-testid*="enable"]'
      ];
      
      let toggleFound = false;
      for (const selector of toggleElements) {
        const elements = page.locator(selector);
        const count = await elements.count();
        if (count > 0) {
          console.log(`✅ 找到 ${count} 个状态切换元素: ${selector}`);
          toggleFound = true;
          break;
        }
      }
      
      if (!toggleFound) {
        console.log('⚠️ 未找到状态切换控件');
      }
      
      // 检查是否有同步状态显示
      const syncStatusElements = [
        'text=同步中',
        'text=已停止',
        'text=失败',
        'text=成功',
        'text=Syncing',
        'text=Stopped',
        'text=Failed',
        'text=Success',
        '.status',
        '.sync-status',
        '[data-testid*="status"]'
      ];
      
      let syncStatusFound = false;
      for (const selector of syncStatusElements) {
        const elements = page.locator(selector);
        const count = await elements.count();
        if (count > 0) {
          console.log(`✅ 找到 ${count} 个同步状态元素: ${selector}`);
          syncStatusFound = true;
          break;
        }
      }
      
      if (!syncStatusFound) {
        console.log('⚠️ 未找到同步状态显示');
      }
      
      // 总结检查结果
      console.log('\n📊 前端状态显示检查结果:');
      console.log(`- 账户元素: ${accountsFound ? '✅ 找到' : '❌ 未找到'}`);
      console.log(`- 状态切换控件: ${toggleFound ? '✅ 找到' : '❌ 未找到'}`);
      console.log(`- 同步状态显示: ${syncStatusFound ? '✅ 找到' : '❌ 未找到'}`);
      
      if (accountsFound && (toggleFound || syncStatusFound)) {
        console.log('✅ 前端账户状态显示基本正常');
      } else {
        console.log('⚠️ 前端可能缺少清晰的账户状态显示');
        console.log('💡 建议改进:');
        console.log('  - 添加明显的启用/禁用状态指示器');
        console.log('  - 显示同步状态（正常/失败/同步中）');
        console.log('  - 提供状态切换按钮');
        console.log('  - 显示最后同步时间');
        console.log('  - 显示错误信息（如果有）');
      }
      
    } catch (error) {
      console.error('❌ 前端账户状态显示检查失败:', error);
      throw error;
    }
  });

  test('检查是否有账户启用/禁用功能', async ({ page }) => {
    console.log('🧪 检查账户启用/禁用功能...');
    
    try {
      // 访问账户管理页面
      await page.goto('http://localhost:3000/accounts');
      await page.waitForLoadState('networkidle');
      await page.waitForTimeout(2000);
      
      // 查找可能的启用/禁用按钮
      const enableDisableButtons = [
        'button:has-text("启用")',
        'button:has-text("禁用")',
        'button:has-text("Enable")',
        'button:has-text("Disable")',
        'button:has-text("激活")',
        'button:has-text("停用")'
      ];
      
      let buttonFound = false;
      for (const selector of enableDisableButtons) {
        const button = page.locator(selector);
        if (await button.count() > 0) {
          console.log(`✅ 找到启用/禁用按钮: ${selector}`);
          buttonFound = true;
          
          // 尝试点击按钮（但不实际执行，只是测试是否可点击）
          const isEnabled = await button.isEnabled();
          console.log(`- 按钮可点击: ${isEnabled ? '是' : '否'}`);
          break;
        }
      }
      
      // 查找切换开关
      const toggleSwitches = [
        'input[type="checkbox"]',
        '.toggle-switch',
        '.switch',
        '[role="switch"]',
        '[data-testid*="toggle"]'
      ];
      
      let switchFound = false;
      for (const selector of toggleSwitches) {
        const toggle = page.locator(selector);
        if (await toggle.count() > 0) {
          console.log(`✅ 找到切换开关: ${selector}`);
          switchFound = true;
          
          const isEnabled = await toggle.isEnabled();
          console.log(`- 开关可操作: ${isEnabled ? '是' : '否'}`);
          break;
        }
      }
      
      // 查找下拉菜单或操作菜单
      const actionMenus = [
        'button:has-text("操作")',
        'button:has-text("Actions")',
        'button:has-text("更多")',
        'button:has-text("More")',
        '.dropdown-toggle',
        '.menu-button',
        '[data-testid*="menu"]',
        '[data-testid*="action"]'
      ];
      
      let menuFound = false;
      for (const selector of actionMenus) {
        const menu = page.locator(selector);
        if (await menu.count() > 0) {
          console.log(`✅ 找到操作菜单: ${selector}`);
          menuFound = true;
          break;
        }
      }
      
      console.log('\n📊 账户控制功能检查结果:');
      console.log(`- 启用/禁用按钮: ${buttonFound ? '✅ 找到' : '❌ 未找到'}`);
      console.log(`- 切换开关: ${switchFound ? '✅ 找到' : '❌ 未找到'}`);
      console.log(`- 操作菜单: ${menuFound ? '✅ 找到' : '❌ 未找到'}`);
      
      if (!buttonFound && !switchFound && !menuFound) {
        console.log('\n⚠️ 重要发现: 前端缺少账户启用/禁用控制功能！');
        console.log('💡 这解释了为什么用户不知道账户被禁用了');
        console.log('🔧 建议添加以下功能:');
        console.log('  1. 在账户卡片上显示启用/禁用状态');
        console.log('  2. 提供切换开关或按钮来启用/禁用账户');
        console.log('  3. 显示禁用账户的警告信息');
        console.log('  4. 在禁用状态下显示"点击启用"的提示');
      } else {
        console.log('✅ 前端具有账户控制功能');
      }
      
    } catch (error) {
      console.error('❌ 账户启用/禁用功能检查失败:', error);
      throw error;
    }
  });

  test('检查同步状态和错误信息显示', async ({ page }) => {
    console.log('🧪 检查同步状态和错误信息显示...');
    
    try {
      // 访问账户管理页面
      await page.goto('http://localhost:3000/accounts');
      await page.waitForLoadState('networkidle');
      await page.waitForTimeout(2000);
      
      // 查找同步状态指示器
      const statusIndicators = [
        '.status-indicator',
        '.sync-status',
        '.connection-status',
        '[data-testid*="status"]',
        'text=失败',
        'text=Failed',
        'text=错误',
        'text=Error',
        'text=正常',
        'text=Success',
        'text=同步中',
        'text=Syncing'
      ];
      
      let statusFound = false;
      for (const selector of statusIndicators) {
        const elements = page.locator(selector);
        if (await elements.count() > 0) {
          console.log(`✅ 找到状态指示器: ${selector}`);
          statusFound = true;
          
          // 获取状态文本
          const statusText = await elements.first().textContent();
          console.log(`- 状态内容: ${statusText}`);
          break;
        }
      }
      
      // 查找错误信息显示
      const errorElements = [
        '.error-message',
        '.alert-error',
        '.warning',
        '[data-testid*="error"]',
        'text=连接失败',
        'text=Connection failed',
        'text=授权失败',
        'text=Authentication failed',
        '.text-red',
        '.text-danger'
      ];
      
      let errorFound = false;
      for (const selector of errorElements) {
        const elements = page.locator(selector);
        if (await elements.count() > 0) {
          console.log(`✅ 找到错误信息显示: ${selector}`);
          errorFound = true;
          
          const errorText = await elements.first().textContent();
          console.log(`- 错误内容: ${errorText}`);
          break;
        }
      }
      
      // 查找最后同步时间显示
      const timeElements = [
        'text=最后同步',
        'text=Last sync',
        'text=上次同步',
        'text=同步时间',
        '.last-sync',
        '.sync-time',
        '[data-testid*="sync-time"]'
      ];
      
      let timeFound = false;
      for (const selector of timeElements) {
        const elements = page.locator(selector);
        if (await elements.count() > 0) {
          console.log(`✅ 找到同步时间显示: ${selector}`);
          timeFound = true;
          
          const timeText = await elements.first().textContent();
          console.log(`- 时间内容: ${timeText}`);
          break;
        }
      }
      
      console.log('\n📊 同步状态信息检查结果:');
      console.log(`- 同步状态显示: ${statusFound ? '✅ 找到' : '❌ 未找到'}`);
      console.log(`- 错误信息显示: ${errorFound ? '✅ 找到' : '❌ 未找到'}`);
      console.log(`- 同步时间显示: ${timeFound ? '✅ 找到' : '❌ 未找到'}`);
      
      if (!statusFound && !errorFound) {
        console.log('\n⚠️ 重要发现: 前端缺少同步状态和错误信息显示！');
        console.log('💡 这解释了为什么用户不知道同步失败了');
        console.log('🔧 建议添加以下显示信息:');
        console.log('  1. 同步状态指示器（成功/失败/进行中）');
        console.log('  2. 错误信息提示（连接失败、认证失败等）');
        console.log('  3. 最后同步时间');
        console.log('  4. 下次同步时间');
        console.log('  5. 邮件数量统计');
      } else {
        console.log('✅ 前端具有基本的状态信息显示');
      }
      
    } catch (error) {
      console.error('❌ 同步状态信息检查失败:', error);
      throw error;
    }
  });
});