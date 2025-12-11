import { test, expect, API_BASE_URL, getAuthToken } from './setup';

test.describe('账号分组功能测试', () => {
  let authToken: string;
  let createdGroupId: number | null = null;

  test.beforeAll(async ({ request }) => {
    authToken = await getAuthToken(request);
  });

  test.afterAll(async ({ request }) => {
    // 清理测试创建的分组
    if (createdGroupId) {
      try {
        await request.delete(`${API_BASE_URL}/groups/${createdGroupId}`, {
          headers: { 'Authorization': `Bearer ${authToken}` },
        });
      } catch (e) {
        console.log('清理分组失败:', e);
      }
    }
  });

  test.describe('分组 CRUD 操作', () => {
    test('1.1 创建分组 - 有效名称', async ({ request }) => {
      const groupName = `测试分组_${Date.now()}`;
      const response = await request.post(`${API_BASE_URL}/groups`, {
        headers: { 'Authorization': `Bearer ${authToken}` },
        data: {
          name: groupName,
          description: '这是一个测试分组',
        },
      });

      expect(response.ok()).toBeTruthy();
      const body = await response.json();
      expect(body.success).toBe(true);
      expect(body.data.id).toBeDefined();
      expect(body.data.name).toBe(groupName);
      expect(body.data.description).toBe('这是一个测试分组');
      
      createdGroupId = body.data.id;
      console.log(`✓ 创建分组成功: ${groupName} (ID: ${createdGroupId})`);
    });

    test('1.2 创建分组 - 空名称应被拒绝', async ({ request }) => {
      const response = await request.post(`${API_BASE_URL}/groups`, {
        headers: { 'Authorization': `Bearer ${authToken}` },
        data: {
          name: '',
          description: '空名称测试',
        },
      });

      expect(response.ok()).toBeFalsy();
      expect(response.status()).toBe(400);
      console.log('✓ 空名称被正确拒绝');
    });

    test('1.3 创建分组 - 重复名称应被拒绝', async ({ request }) => {
      // 先创建一个分组
      const groupName = `重复测试_${Date.now()}`;
      const firstResponse = await request.post(`${API_BASE_URL}/groups`, {
        headers: { 'Authorization': `Bearer ${authToken}` },
        data: { name: groupName },
      });
      expect(firstResponse.ok()).toBeTruthy();
      const firstGroup = await firstResponse.json();

      // 尝试创建同名分组
      const secondResponse = await request.post(`${API_BASE_URL}/groups`, {
        headers: { 'Authorization': `Bearer ${authToken}` },
        data: { name: groupName },
      });

      expect(secondResponse.ok()).toBeFalsy();
      expect(secondResponse.status()).toBe(409);
      console.log('✓ 重复名称被正确拒绝');

      // 清理
      await request.delete(`${API_BASE_URL}/groups/${firstGroup.data.id}`, {
        headers: { 'Authorization': `Bearer ${authToken}` },
      });
    });

    test('1.4 获取分组列表', async ({ request }) => {
      const response = await request.get(`${API_BASE_URL}/groups`, {
        headers: { 'Authorization': `Bearer ${authToken}` },
      });

      expect(response.ok()).toBeTruthy();
      const body = await response.json();
      expect(body.success).toBe(true);
      expect(Array.isArray(body.data)).toBe(true);
      console.log(`✓ 获取分组列表成功，共 ${body.data.length} 个分组`);
    });

    test('1.5 获取单个分组详情', async ({ request }) => {
      if (!createdGroupId) {
        test.skip();
        return;
      }

      const response = await request.get(`${API_BASE_URL}/groups/${createdGroupId}`, {
        headers: { 'Authorization': `Bearer ${authToken}` },
      });

      expect(response.ok()).toBeTruthy();
      const body = await response.json();
      expect(body.success).toBe(true);
      expect(body.data.id).toBe(createdGroupId);
      console.log(`✓ 获取分组详情成功: ${body.data.name}`);
    });

    test('1.6 更新分组', async ({ request }) => {
      if (!createdGroupId) {
        test.skip();
        return;
      }

      const newName = `更新后的分组_${Date.now()}`;
      const response = await request.put(`${API_BASE_URL}/groups/${createdGroupId}`, {
        headers: { 'Authorization': `Bearer ${authToken}` },
        data: {
          name: newName,
          description: '更新后的描述',
        },
      });

      expect(response.ok()).toBeTruthy();
      const body = await response.json();
      expect(body.success).toBe(true);
      expect(body.data.name).toBe(newName);
      console.log(`✓ 更新分组成功: ${newName}`);
    });
  });

  test.describe('账号分组分配', () => {
    let testGroupId: number;

    test.beforeAll(async ({ request }) => {
      // 创建测试用分组
      const response = await request.post(`${API_BASE_URL}/groups`, {
        headers: { 'Authorization': `Bearer ${authToken}` },
        data: { name: `账号分配测试_${Date.now()}` },
      });
      const body = await response.json();
      testGroupId = body.data.id;
    });

    test.afterAll(async ({ request }) => {
      // 清理测试分组
      if (testGroupId) {
        await request.delete(`${API_BASE_URL}/groups/${testGroupId}`, {
          headers: { 'Authorization': `Bearer ${authToken}` },
        });
      }
    });

    test('2.1 获取账号列表', async ({ request }) => {
      const response = await request.get(`${API_BASE_URL}/accounts`, {
        headers: { 'Authorization': `Bearer ${authToken}` },
      });

      expect(response.ok()).toBeTruthy();
      const body = await response.json();
      expect(body.success).toBe(true);
      console.log(`✓ 获取账号列表成功，共 ${body.data?.length || 0} 个账号`);
    });

    test('2.2 按分组筛选账号 - 未分组', async ({ request }) => {
      const response = await request.get(`${API_BASE_URL}/accounts?group_id=ungrouped`, {
        headers: { 'Authorization': `Bearer ${authToken}` },
      });

      expect(response.ok()).toBeTruthy();
      const body = await response.json();
      expect(body.success).toBe(true);
      console.log(`✓ 获取未分组账号成功，共 ${body.data?.length || 0} 个`);
    });

    test('2.3 按分组筛选账号 - 指定分组', async ({ request }) => {
      const response = await request.get(`${API_BASE_URL}/accounts?group_id=${testGroupId}`, {
        headers: { 'Authorization': `Bearer ${authToken}` },
      });

      expect(response.ok()).toBeTruthy();
      const body = await response.json();
      expect(body.success).toBe(true);
      console.log(`✓ 按分组筛选账号成功，共 ${body.data?.length || 0} 个`);
    });
  });

  test.describe('分组排序', () => {
    let group1Id: number;
    let group2Id: number;

    test.beforeAll(async ({ request }) => {
      // 创建两个测试分组
      const res1 = await request.post(`${API_BASE_URL}/groups`, {
        headers: { 'Authorization': `Bearer ${authToken}` },
        data: { name: `排序测试1_${Date.now()}` },
      });
      group1Id = (await res1.json()).data.id;

      const res2 = await request.post(`${API_BASE_URL}/groups`, {
        headers: { 'Authorization': `Bearer ${authToken}` },
        data: { name: `排序测试2_${Date.now()}` },
      });
      group2Id = (await res2.json()).data.id;
    });

    test.afterAll(async ({ request }) => {
      // 清理测试分组
      if (group1Id) {
        await request.delete(`${API_BASE_URL}/groups/${group1Id}`, {
          headers: { 'Authorization': `Bearer ${authToken}` },
        });
      }
      if (group2Id) {
        await request.delete(`${API_BASE_URL}/groups/${group2Id}`, {
          headers: { 'Authorization': `Bearer ${authToken}` },
        });
      }
    });

    test('3.1 重新排序分组', async ({ request }) => {
      const response = await request.put(`${API_BASE_URL}/groups/reorder`, {
        headers: { 'Authorization': `Bearer ${authToken}` },
        data: {
          group_ids: [group2Id, group1Id],
        },
      });

      expect(response.ok()).toBeTruthy();
      console.log('✓ 分组重新排序成功');

      // 验证排序结果
      const listResponse = await request.get(`${API_BASE_URL}/groups`, {
        headers: { 'Authorization': `Bearer ${authToken}` },
      });
      const groups = (await listResponse.json()).data;
      
      const group1 = groups.find((g: any) => g.id === group1Id);
      const group2 = groups.find((g: any) => g.id === group2Id);
      
      if (group1 && group2) {
        expect(group2.display_order).toBeLessThan(group1.display_order);
        console.log('✓ 排序顺序验证通过');
      }
    });
  });

  test.describe('分组删除', () => {
    test('4.1 删除分组', async ({ request }) => {
      // 创建一个用于删除的分组
      const createResponse = await request.post(`${API_BASE_URL}/groups`, {
        headers: { 'Authorization': `Bearer ${authToken}` },
        data: { name: `删除测试_${Date.now()}` },
      });
      const groupId = (await createResponse.json()).data.id;

      // 删除分组
      const deleteResponse = await request.delete(`${API_BASE_URL}/groups/${groupId}`, {
        headers: { 'Authorization': `Bearer ${authToken}` },
      });

      expect(deleteResponse.ok()).toBeTruthy();
      console.log('✓ 删除分组成功');

      // 验证分组已被删除
      const getResponse = await request.get(`${API_BASE_URL}/groups/${groupId}`, {
        headers: { 'Authorization': `Bearer ${authToken}` },
      });
      expect(getResponse.status()).toBe(404);
      console.log('✓ 验证分组已被删除');
    });

    test('4.2 删除不存在的分组应返回 404', async ({ request }) => {
      const response = await request.delete(`${API_BASE_URL}/groups/999999`, {
        headers: { 'Authorization': `Bearer ${authToken}` },
      });

      expect(response.status()).toBe(404);
      console.log('✓ 删除不存在的分组正确返回 404');
    });
  });
});
