import { test, expect } from '@playwright/test';

test.describe('修复QQ邮箱账户问题', () => {
  let authToken: string;

  test.beforeAll(async () => {
    authToken = process.env.TEST_AUTH_TOKEN || '';
    if (!authToken) {
      throw new Error('TEST_AUTH_TOKEN 环境变量未设置');
    }
  });

  test('启用QQ邮箱账户并触发同步', async ({ request }) => {
    console.log('🔧 开始修复QQ邮箱账户问题...');
    
    try {
      // 1. 获取QQ邮箱账户信息
      console.log('📋 获取账户列表...');
      const accountsResponse = await request.get('http://localhost:3333/api/v1/accounts', {
        headers: {
          'Authorization': `Bearer ${authToken}`,
          'Content-Type': 'application/json'
        }
      });
      
      if (!accountsResponse.ok()) {
        throw new Error(`获取账户列表失败: ${accountsResponse.status()}`);
      }
      
      const accountsData = await accountsResponse.json();
      console.log(`找到 ${accountsData.data.length} 个账户`);
      
      // 找到QQ邮箱账户
      const qqAccount = accountsData.data.find((account: any) => 
        account.provider === 'qq' || account.email.includes('qq.com')
      );
      
      if (!qqAccount) {
        throw new Error('未找到QQ邮箱账户');
      }
      
      console.log(`\n🎯 找到QQ邮箱账户:`);
      console.log(`- UID: ${qqAccount.uid}`);
      console.log(`- 邮箱: ${qqAccount.email}`);
      console.log(`- 当前状态: ${qqAccount.enabled ? '启用' : '禁用'}`);
      console.log(`- 同步启用: ${qqAccount.sync_enabled ? '启用' : '禁用'}`);
      
      // 2. 启用账户（如果被禁用）
      if (!qqAccount.enabled) {
        console.log('\n🔄 启用QQ邮箱账户...');
        
        const updateData = {
          ...qqAccount,
          enabled: true,
          sync_enabled: true
        };
        
        const updateResponse = await request.put(`http://localhost:3333/api/v1/accounts/${qqAccount.uid}`, {
          headers: {
            'Authorization': `Bearer ${authToken}`,
            'Content-Type': 'application/json'
          },
          data: updateData
        });
        
        if (updateResponse.ok()) {
          const updateResult = await updateResponse.json();
          console.log('✅ QQ邮箱账户已启用');
          console.log('更新结果:', JSON.stringify(updateResult, null, 2));
        } else {
          console.log(`❌ 启用账户失败: ${updateResponse.status()}`);
          const errorText = await updateResponse.text();
          console.log('错误详情:', errorText);
          throw new Error('启用账户失败');
        }
      } else {
        console.log('✅ QQ邮箱账户已经是启用状态');
      }
      
      // 3. 触发手动同步
      console.log('\n🔄 触发QQ邮箱手动同步...');
      
      const syncResponse = await request.post(`http://localhost:3333/api/v1/accounts/${qqAccount.uid}/sync`, {
        headers: {
          'Authorization': `Bearer ${authToken}`,
          'Content-Type': 'application/json'
        }
      });
      
      if (syncResponse.ok()) {
        const syncResult = await syncResponse.json();
        console.log('✅ 手动同步已触发');
        console.log('同步结果:', syncResult.message);
      } else {
        console.log(`⚠️ 触发同步失败: ${syncResponse.status()}`);
        const errorText = await syncResponse.text();
        console.log('错误详情:', errorText);
      }
      
      // 4. 等待同步开始并检查状态
      console.log('\n⏳ 等待同步开始...');
      await new Promise(resolve => setTimeout(resolve, 5000));
      
      // 检查同步状态
      const statusResponse = await request.get('http://localhost:3333/api/v1/sync/status', {
        headers: {
          'Authorization': `Bearer ${authToken}`,
          'Content-Type': 'application/json'
        }
      });
      
      if (statusResponse.ok()) {
        const statusData = await statusResponse.json();
        const qqStatus = statusData.data.find((account: any) => 
          account.account_uid === qqAccount.uid
        );
        
        if (qqStatus) {
          console.log('\n📊 QQ邮箱当前同步状态:');
          console.log(`- 状态: ${qqStatus.status}`);
          console.log(`- 邮件数量: ${qqStatus.email_count}`);
          console.log(`- 未读数量: ${qqStatus.unread_count}`);
          console.log(`- 上次同步: ${qqStatus.last_sync_time}`);
          console.log(`- 下次同步: ${qqStatus.next_sync_time}`);
          
          if (qqStatus.error_message) {
            console.log(`- 错误信息: ${qqStatus.error_message}`);
          }
        }
      }
      
      // 5. 检查最新的同步日志
      console.log('\n📋 检查最新同步日志...');
      
      const logsResponse = await request.get('http://localhost:3333/api/v1/sync/logs?size=5', {
        headers: {
          'Authorization': `Bearer ${authToken}`,
          'Content-Type': 'application/json'
        }
      });
      
      if (logsResponse.ok()) {
        const logsData = await logsResponse.json();
        const qqLogs = logsData.data.filter((log: any) => 
          log.account_uid === qqAccount.uid
        );
        
        if (qqLogs.length > 0) {
          console.log(`\n📝 QQ邮箱最新同步日志 (${qqLogs.length} 条):`);
          qqLogs.slice(0, 2).forEach((log: any, index: number) => {
            console.log(`\n日志 ${index + 1}:`);
            console.log(`- 状态: ${log.status}`);
            console.log(`- 开始时间: ${log.start_time}`);
            console.log(`- 结束时间: ${log.end_time || '进行中'}`);
            console.log(`- 新增邮件: ${log.emails_added}`);
            console.log(`- 总邮件数: ${log.emails_total}`);
            if (log.error_message) {
              console.log(`- 错误: ${log.error_message}`);
            }
          });
        }
      }
      
      // 6. 检查邮件列表
      console.log('\n📧 检查邮件列表更新...');
      
      const emailsResponse = await request.get('http://localhost:3333/api/v1/emails?size=10', {
        headers: {
          'Authorization': `Bearer ${authToken}`,
          'Content-Type': 'application/json'
        }
      });
      
      if (emailsResponse.ok()) {
        const emailsData = await emailsResponse.json();
        const totalEmails = emailsData.total || emailsData.data.length;
        
        console.log(`📬 当前邮件总数: ${totalEmails}`);
        
        if (emailsData.data && emailsData.data.length > 0) {
          console.log('\n📨 最新邮件:');
          emailsData.data.slice(0, 3).forEach((email: any, index: number) => {
            console.log(`\n邮件 ${index + 1}:`);
            console.log(`- 主题: ${email.subject}`);
            console.log(`- 发件人: ${email.from}`);
            console.log(`- 时间: ${email.received_at || email.created_at}`);
            console.log(`- 账户: ${email.account_uid === qqAccount.uid ? 'QQ邮箱' : '其他'}`);
          });
        }
      }
      
      console.log('\n✅ QQ邮箱账户修复操作完成！');
      console.log('\n💡 后续建议:');
      console.log('1. 等待几分钟让同步完成');
      console.log('2. 检查是否收到新邮件');
      console.log('3. 如果仍然失败，可能需要重新配置邮箱凭证');
      console.log('4. 建议改进前端显示账户状态信息');
      
    } catch (error) {
      console.error('❌ 修复QQ邮箱账户失败:', error);
      throw error;
    }
  });
});