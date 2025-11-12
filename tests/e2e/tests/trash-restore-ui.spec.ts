import { test, expect, API_BASE_URL, getAuthToken } from './setup';

const FRONTEND_URL = process.env.FRONTEND_URL || 'http://localhost:4444';
const TARGET_EMAIL = '15026732619@163.com';

/**
 * 目标验证：
 * 1) 收件箱中不应出现已删除邮件
 * 2) 垃圾箱中打开某封已删除邮件的详情时，只显示“恢复”按钮，不显示“删除”按钮
 */

async function loginAndGoTo(page: any, folder: 'inbox' | 'trash') {
  await page.goto(FRONTEND_URL, { waitUntil: 'networkidle' });
  const pwd = page.locator('input[type="password"]');
  if (await pwd.count()) {
    await pwd.fill(process.env.MASTER_PASSWORD || 'admin123');
    const loginBtn = page.locator('button:has-text("登录"), button:has-text("Login")').first();
    await loginBtn.click();
    await page.waitForLoadState('networkidle');
  }
  if (!page.url().includes('/inbox')) {
    await page.goto(FRONTEND_URL + '/inbox', { waitUntil: 'networkidle' });
  }
  // 尝试选择指定邮箱（如果列表暂未出现，不阻塞后续断言）
  try {
    const accountBtn = page.locator(`button:has-text("${TARGET_EMAIL}")`).first();
    await accountBtn.click({ timeout: 20000 });
  } catch (e) {
    // 忽略：仍然可以在“所有邮箱”视图下完成断言
  }
  // 保持在 /inbox 页面即可，后续测试如需垃圾箱会直接通过 URL 访问
  await page.waitForLoadState('networkidle');
}

test.describe('垃圾箱/收件箱 UI 行为校验（指定邮箱）', () => {
  let authToken: string;
  let accountUid: string | null = null;
  let oneDeletedEmail: { id: number; subject: string } | null = null;

  test.beforeAll(async ({ request }) => {
    // 获取后端 token
    authToken = await getAuthToken(request);

    // 查询账户 UID
    const accResp = await request.get(`${API_BASE_URL}/accounts`, {
      headers: { Authorization: `Bearer ${authToken}` },
    });
    if (!accResp.ok()) throw new Error('获取账户列表失败');
    const accounts = await accResp.json();
    const acc = (Array.isArray(accounts) ? accounts : accounts.data || [])
      .find((a: any) => a.email === TARGET_EMAIL);
    if (!acc) throw new Error(`未找到测试账户: ${TARGET_EMAIL}`);
    accountUid = acc.uid;

    // 查询该账户的已删除邮件，取一封用于后续断言
    const delResp = await request.get(`${API_BASE_URL}/emails`, {
      headers: { Authorization: `Bearer ${authToken}` },
      params: { account_uid: accountUid!, is_deleted: 'true', page: '1', page_size: '10' },
    });
    if (delResp.ok()) {
      const body = await delResp.json();
      const list = body.emails || (body.data && body.data.emails) || [];
      if (list.length > 0) {
        oneDeletedEmail = { id: list[0].id, subject: list[0].subject || '' };
      }
    }
  });

  test('登录并选择指定邮箱', async ({ page }) => {
    await page.goto(FRONTEND_URL, { waitUntil: 'networkidle' });

    // 登录（单字段密码登录）
    const pwd = page.locator('input[type="password"]');
    if (await pwd.count()) {
      await pwd.fill(process.env.MASTER_PASSWORD || 'admin123');
      const loginBtn = page.locator('button:has-text("登录"), button:has-text("Login")').first();
      await loginBtn.click();
      await page.waitForLoadState('networkidle');
    }

    // 确保到 /inbox
    if (!page.url().includes('/inbox')) {
      await page.goto(FRONTEND_URL + '/inbox', { waitUntil: 'networkidle' });
    }

    // 侧边栏选择目标账户（按钮 title 为邮箱地址）
    const accountBtn = page.locator(`button[title="${TARGET_EMAIL}"]`).first();
    await expect(accountBtn).toBeVisible();
    await accountBtn.click();

    // 切到“收件箱”
    await page.locator('button:has-text("收件箱")').click();
    await page.waitForLoadState('networkidle');
  });

  test('收件箱中不应包含已删除邮件（使用后端抽样的已删除主题）', async ({ page }) => {
    await loginAndGoTo(page, 'inbox');
    test.skip(!oneDeletedEmail, '无已删除邮件数据，跳过此断言');

    // 在收件箱页面，断言“已删除的那封邮件主题”不可见
    // 注意：主题可能被截断，使用包含匹配
    const subject = oneDeletedEmail!.subject;
    if (subject && subject.trim().length > 0) {
      await expect(page.getByText(subject, { exact: false })).toHaveCount(0);
    } else {
      test.skip(true, '该已删除邮件缺少可断言的主题，跳过');
    }
  });

  test('垃圾箱详情：存在“恢复”按钮且无“删除”按钮', async ({ page }) => {
    test.skip(!oneDeletedEmail, '无已删除邮件数据，跳过此断言');

    // 通过 UI 进入垃圾箱并点击该邮件，避免整页刷新导致的 token 重新注入时机问题
    await loginAndGoTo(page, 'inbox');
    await page.locator('button:has-text("垃圾箱")').first().click();
    await page.waitForLoadState('networkidle');

    // 在垃圾箱列表中点击该已删除邮件（用主题匹配）
    const subject = oneDeletedEmail!.subject;
    await expect(page.getByText(subject, { exact: false })).toBeVisible();
    await page.getByText(subject, { exact: false }).first().click();

    // 断言：应有“恢复”按钮，无“删除”按钮
    await expect(page.locator('button[title="恢复"]').first()).toBeVisible();
    await expect(page.locator('button[title="删除"]').first()).toHaveCount(0);
  });
});

