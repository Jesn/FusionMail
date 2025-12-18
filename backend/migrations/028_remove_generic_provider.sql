-- 迁移脚本：删除 generic 提供商
-- 说明：generic 提供商已不再需要，用户可以通过「提供商管理」自己添加任何邮箱服务商

-- 1. 检查是否有账户使用 generic 提供商
-- 如果有账户使用，需要先处理这些账户
DO $$
DECLARE
    account_count INTEGER;
BEGIN
    SELECT COUNT(*) INTO account_count 
    FROM email_accounts 
    WHERE provider = 'generic' AND deleted_at IS NULL;
    
    IF account_count > 0 THEN
        RAISE EXCEPTION '存在 % 个账户使用 generic 提供商，请先迁移这些账户', account_count;
    END IF;
END $$;

-- 2. 删除 provider_adapters 关联记录
DELETE FROM provider_adapters 
WHERE provider_id IN (SELECT id FROM providers WHERE name = 'generic');

-- 3. 删除 generic 提供商
DELETE FROM providers WHERE name = 'generic';

-- 回滚脚本（如需回滚，手动执行以下 SQL）：
-- INSERT INTO providers (name, display_name, supported_protocols, recommended_protocol, requires_o_auth, imap_host, imap_port, pop3_host, pop3_port, smtp_host, smtp_port, imap_encryption, pop3_encryption, smtp_encryption, enabled, sort_order, description, created_at, updated_at)
-- VALUES ('generic', '通用邮箱 (IMAP/POP3)', 'imap,pop3', 'imap', false, '', 993, '', 995, '', 587, 'ssl', 'ssl', 'starttls', true, 999, '支持任何 IMAP/POP3 邮箱服务器', NOW(), NOW());
