import { test, expect } from '@playwright/test';

test.describe('检查邮件同步状态', () => {
  let authToken: string;

  test.beforeAll(async () => {
    authToken = process.env.TEST_AUTH_TOKEN || '';
    if (!authToken) {
      throw new Error('TEST_AUTH_TOKEN 环境变量未设置');
    }
  });

  test('检查当前同步状态和触发同步', async ({ request }) => {
    console.log('🔍 检查当前邮件同步状态...');
    
    try {
      // 1. 检查同步状态
      const statusResponse = await request.get('http://localhost:3333/api/v1/sync/status', {
        headers: {
          'Authorization': `Bearer ${authToken}`,
          'Content-Type': 'application/json'
        }
      });
      
      if (statusResponse.ok()) {
        const statusData = await statusResponse.json();
        console.log('\n📊 当前同步状态:');
        
        statusData.data.forEach((account: any, index: number) => {
          console.log(`\n账户 ${index + 1}:`);
          console.log(`- 邮箱: ${account.account_name}`);
          console.log(`- 提供商: ${account.provider}`);
          console.log(`- 状态: ${account.status}`);
          console.log(`- 邮件数量: ${account.email_count}`);
          console.log(`- 未读数量: ${account.unread_count}`);
          console.log(`- 上次同步: ${account.last_sync_time}`);
          console.log(`- 下次同步: ${account.next_sync_time}`);
          console.log(`- 同步间隔: ${account.sync_interval} 分钟`);
          if (account.error_message) {
            console.log(`- 错误信息: ${account.error_message}`);
          }
        });
        
        // 找到QQ邮箱账户
        const qqAccount = statusData.data.find((account: any) => 
          account.provider === 'qq' || account.account_name.includes('qq.com')
        );
        
        if (qqAccount) {
          console.log(`\n🎯 QQ邮箱账户详情:`);
          console.log(`- 账户UID: ${qqAccount.account_uid}`);
          console.log(`- 状态: ${qqAccount.status}`);
          console.log(`- 邮件总数: ${qqAccount.email_count}`);
          console.log(`- 未读数量: ${qqAccount.unread_count}`);
          
          if (qqAccount.status === 'failed') {
            console.log('⚠️ QQ邮箱同步状态为失败，这可能是收不到新邮件的原因');
          }
          
          // 2. 尝试手动触发QQ邮箱同步
          console.log('\n🔄 尝试手动触发QQ邮箱同步...');
          
          const syncResponse = await request.post(`http://localhost:3333/api/v1/accounts/${qqAccount.account_uid}/sync`, {
            headers: {
              'Authorization': `Bearer ${authToken}`,
              'Content-Type': 'application/json'
            }
          });
          
          if (syncResponse.ok()) {
            const syncData = await syncResponse.json();
            console.log('✅ 手动同步触发成功:', syncData.message);
            
            // 等待几秒钟让同步开始
            console.log('⏳ 等待同步开始...');
            await new Promise(resolve => setTimeout(resolve, 3000));
            
            // 3. 检查最新的同步日志
            const logsResponse = await request.get('http://localhost:3333/api/v1/sync/logs?size=5', {
              headers: {
                'Authorization': `Bearer ${authToken}`,
                'Content-Type': 'application/json'
              }
            });
            
            if (logsResponse.ok()) {
              const logsData = await logsResponse.json();
              const qqLogs = logsData.data.filter((log: any) => 
                log.account_uid === qqAccount.account_uid
              );
              
              console.log(`\n📋 QQ邮箱最近的同步日志 (${qqLogs.length} 条):`);
              qqLogs.slice(0, 3).forEach((log: any, index: number) => {
                console.log(`\n日志 ${index + 1}:`);
                console.log(`- ID: ${log.id}`);
                console.log(`- 状态: ${log.status}`);
                console.log(`- 开始时间: ${log.start_time}`);
                console.log(`- 结束时间: ${log.end_time || '进行中'}`);
                console.log(`- 新增邮件: ${log.emails_added}`);
                console.log(`- 总邮件数: ${log.emails_total}`);
                if (log.error_message) {
                  console.log(`- 错误信息: ${log.error_message}`);
                }
              });
            }
            
            // 4. 检查邮件列表是否有更新
            console.log('\n📧 检查邮件列表...');
            const emailsResponse = await request.get('http://localhost:3333/api/v1/emails?size=10', {
              headers: {
                'Authorization': `Bearer ${authToken}`,
                'Content-Type': 'application/json'
              }
            });
            
            if (emailsResponse.ok()) {
              const emailsData = await emailsResponse.json();
              console.log(`📬 当前邮件总数: ${emailsData.total || emailsData.data.length}`);
              
              if (emailsData.data && emailsData.data.length > 0) {
                console.log('\n📨 最新的几封邮件:');
                emailsData.data.slice(0, 3).forEach((email: any, index: number) => {
                  console.log(`\n邮件 ${index + 1}:`);
                  console.log(`- 主题: ${email.subject}`);
                  console.log(`- 发件人: ${email.from}`);
                  console.log(`- 接收时间: ${email.received_at || email.created_at}`);
                  console.log(`- 账户: ${email.account_uid}`);
                });
              } else {
                console.log('📭 当前没有邮件');
              }
            }
            
          } else {
            console.log(`❌ 手动同步触发失败: ${syncResponse.status()}`);
            const errorText = await syncResponse.text();
            console.log('错误详情:', errorText);
          }
          
        } else {
          console.log('⚠️ 未找到QQ邮箱账户');
        }
        
      } else {
        console.log(`❌ 获取同步状态失败: ${statusResponse.status()}`);
      }
      
      // 5. 检查账户配置
      console.log('\n🔧 检查账户配置...');
      const accountsResponse = await request.get('http://localhost:3333/api/v1/accounts', {
        headers: {
          'Authorization': `Bearer ${authToken}`,
          'Content-Type': 'application/json'
        }
      });
      
      if (accountsResponse.ok()) {
        const accountsData = await accountsResponse.json();
        console.log(`\n📋 账户配置 (${accountsData.data.length} 个账户):`);
        
        accountsData.data.forEach((account: any, index: number) => {
          console.log(`\n账户 ${index + 1}:`);
          console.log(`- 名称: ${account.name}`);
          console.log(`- 邮箱: ${account.email}`);
          console.log(`- 提供商: ${account.provider}`);
          console.log(`- 启用状态: ${account.enabled ? '启用' : '禁用'}`);
          console.log(`- 同步启用: ${account.sync_enabled ? '启用' : '禁用'}`);
          console.log(`- 同步间隔: ${account.sync_interval} 分钟`);
          console.log(`- 创建时间: ${account.created_at}`);
        });
      }
      
      console.log('\n✅ 邮件同步状态检查完成');
      
    } catch (error) {
      console.error('❌ 检查邮件同步状态失败:', error);
      throw error;
    }
  });
});