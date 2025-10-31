-- 添加通用邮箱提供商的配置字段
-- Migration: 003_add_generic_provider_fields
-- Description: 为 accounts 表添加通用邮箱服务器配置字段

-- 添加 IMAP 配置字段
ALTER TABLE accounts ADD COLUMN IF NOT EXISTS imap_host VARCHAR(255) DEFAULT '';
ALTER TABLE accounts ADD COLUMN IF NOT EXISTS imap_port INTEGER DEFAULT 993;

-- 添加 POP3 配置字段  
ALTER TABLE accounts ADD COLUMN IF NOT EXISTS pop3_host VARCHAR(255) DEFAULT '';
ALTER TABLE accounts ADD COLUMN IF NOT EXISTS pop3_port INTEGER DEFAULT 995;

-- 添加加密方式字段
ALTER TABLE accounts ADD COLUMN IF NOT EXISTS encryption VARCHAR(20) DEFAULT 'ssl';

-- 添加注释
COMMENT ON COLUMN accounts.imap_host IS 'IMAP 服务器地址（仅用于 generic 提供商）';
COMMENT ON COLUMN accounts.imap_port IS 'IMAP 端口（仅用于 generic 提供商）';
COMMENT ON COLUMN accounts.pop3_host IS 'POP3 服务器地址（仅用于 generic 提供商）';
COMMENT ON COLUMN accounts.pop3_port IS 'POP3 端口（仅用于 generic 提供商）';
COMMENT ON COLUMN accounts.encryption IS '加密方式：ssl/starttls/none（仅用于 generic 提供商）';