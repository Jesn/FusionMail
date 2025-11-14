import { defineConfig, devices } from '@playwright/test';

/**
 * Playwright 配置文件
 * 用于 E2E 测试
 */
export default defineConfig({
  testDir: './tests/e2e',
  
  // 测试超时时间
  timeout: 30 * 1000,
  
  // 每个测试的重试次数
  retries: 0,
  
  // 并行执行的 worker 数量
  workers: 1,
  
  // 报告配置
  reporter: [
    ['html', { outputFolder: 'playwright-report' }],
    ['list'],
  ],
  
  // 全局配置
  use: {
    // 基础 URL
    baseURL: 'http://localhost:4444',
    
    // 截图配置
    screenshot: 'only-on-failure',
    
    // 视频配置
    video: 'retain-on-failure',
    
    // 追踪配置
    trace: 'on-first-retry',
    
    // 浏览器上下文选项
    viewport: { width: 1280, height: 720 },
    
    // 忽略 HTTPS 错误
    ignoreHTTPSErrors: true,
  },

  // 项目配置
  projects: [
    {
      name: 'chromium',
      use: { ...devices['Desktop Chrome'] },
    },
  ],

  // Web 服务器配置（可选）
  // webServer: {
  //   command: 'npm run dev',
  //   port: 4444,
  //   reuseExistingServer: !process.env.CI,
  // },
});

