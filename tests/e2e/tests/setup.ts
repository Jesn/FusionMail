import { test as base } from '@playwright/test';
import * as fs from 'fs';
import * as path from 'path';

// 测试清单路径
const CHECKLIST_PATH = path.join(__dirname, '../../../.kiro/specs/fusionmail/test-checklist.md');

// 更新测试清单状态
export function updateChecklistStatus(taskId: string, status: 'completed' | 'failed') {
  try {
    let content = fs.readFileSync(CHECKLIST_PATH, 'utf-8');
    
    // 查找任务行
    const taskPattern = new RegExp(`- \\[ \\] ${taskId.replace(/\./g, '\\.')}`, 'g');
    const statusMark = status === 'completed' ? 'x' : '!';
    
    content = content.replace(taskPattern, `- [${statusMark}] ${taskId}`);
    
    fs.writeFileSync(CHECKLIST_PATH, content, 'utf-8');
    console.log(`✓ Updated checklist: ${taskId} -> ${status}`);
  } catch (error) {
    console.error(`Failed to update checklist for ${taskId}:`, error);
  }
}

// 扩展测试上下文
export const test = base.extend({
  // 自动更新测试清单
  autoUpdateChecklist: async ({}, use, testInfo) => {
    await use(async (taskId: string) => {
      const status = testInfo.status === 'passed' ? 'completed' : 'failed';
      updateChecklistStatus(taskId, status);
    });
  },
});

export { expect } from '@playwright/test';

// API 基础 URL
export const API_BASE_URL = process.env.API_BASE_URL || 'http://localhost:8080/api/v1';

// 测试用户凭证
export const TEST_CREDENTIALS = {
  password: process.env.MASTER_PASSWORD || 'admin123',
};

// 辅助函数：获取认证 Token（从全局 setup 或重新登录）
export async function getAuthToken(request: any): Promise<string> {
  // 优先使用全局 setup 中缓存的 token
  if (process.env.TEST_AUTH_TOKEN) {
    return process.env.TEST_AUTH_TOKEN;
  }
  
  // 如果没有全局 token，则重新登录（fallback）
  console.warn('⚠️  未找到全局 token，重新登录...');
  const response = await request.post(`${API_BASE_URL}/auth/login`, {
    data: {
      password: TEST_CREDENTIALS.password,
    },
  });
  
  if (!response.ok()) {
    throw new Error(`Login failed with status ${response.status()}`);
  }
  
  const body = await response.json();
  
  if (!body.data || !body.data.token) {
    throw new Error('Invalid login response: missing token');
  }
  
  // 缓存到环境变量
  process.env.TEST_AUTH_TOKEN = body.data.token;
  
  return body.data.token;
}
