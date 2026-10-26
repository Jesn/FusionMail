-- 迁移脚本：清理 email_accounts 表的冗余字段
-- 说明：这些字段现在从 Provider 表继承，不再需要在 Account 表中存储

-- 1. 删除 IMAP/POP3 服务器配置字段（从 Provider 继承）
ALTER TABLE email_accounts DROP COLUMN IF EXISTS imap_host;
ALTER TABLE email_accounts DROP COLUMN IF EXISTS imap_port;
ALTER TABLE email_accounts DROP COLUMN IF EXISTS pop3_host;
ALTER TABLE email_accounts DROP COLUMN IF EXISTS pop3_port;
ALTER TABLE email_accounts DROP COLUMN IF EXISTS encryption;

-- 2. 删除 SMTP 服务器配置字段（从 Provider 继承）
ALTER TABLE email_accounts DROP COLUMN IF EXISTS smtp_host;
ALTER TABLE email_accounts DROP COLUMN IF EXISTS smtp_port;
ALTER TABLE email_accounts DROP COLUMN IF EXISTS smtp_encryption;

-- 3. 删除 SMTP 凭证字段（使用 email 和 encrypted_credentials）
ALTER TABLE email_accounts DROP COLUMN IF EXISTS smtp_username;
ALTER TABLE email_accounts DROP COLUMN IF EXISTS encrypted_smtp_password;

-- 4. 删除协议和认证类型字段（从 Adapter 推导）
ALTER TABLE email_accounts DROP COLUMN IF EXISTS protocol;
ALTER TABLE email_accounts DROP COLUMN IF EXISTS auth_type;

-- 5. 删除旧的 provider 字符串字段（改用 provider_id 外键）
ALTER TABLE email_accounts DROP COLUMN IF EXISTS provider;

-- 注意：保留以下字段
-- - smtp_enabled: 账户级别的发送功能开关
-- - provider_id: 关联到 providers 表的外键
-- - adapter_id: 关联到 adapters 表的外键

-- 回滚脚本（如需回滚，手动执行以下 SQL）：
-- ALTER TABLE email_accounts ADD COLUMN imap_host VARCHAR(255);
-- ALTER TABLE email_accounts ADD COLUMN imap_port BIGINT;
-- ALTER TABLE email_accounts ADD COLUMN pop3_host VARCHAR(255);
-- ALTER TABLE email_accounts ADD COLUMN pop3_port BIGINT;
-- ALTER TABLE email_accounts ADD COLUMN encryption VARCHAR(20);
-- ALTER TABLE email_accounts ADD COLUMN smtp_host VARCHAR(255);
-- ALTER TABLE email_accounts ADD COLUMN smtp_port BIGINT;
-- ALTER TABLE email_accounts ADD COLUMN smtp_encryption VARCHAR(20);
-- ALTER TABLE email_accounts ADD COLUMN smtp_username VARCHAR(255);
-- ALTER TABLE email_accounts ADD COLUMN encrypted_smtp_password TEXT;
-- ALTER TABLE email_accounts ADD COLUMN protocol VARCHAR(20);
-- ALTER TABLE email_accounts ADD COLUMN auth_type VARCHAR(20);
-- ALTER TABLE email_accounts ADD COLUMN provider VARCHAR(50);
