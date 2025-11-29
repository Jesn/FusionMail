-- 为 Provider 表添加加密方式字段
-- Migration: 015_add_encryption_to_providers
-- Description: 添加 IMAP/POP3/SMTP 加密方式配置字段
-- Author: FusionMail Team
-- Date: 2025-11-29

-- 添加加密方式字段
ALTER TABLE providers 
ADD COLUMN IF NOT EXISTS imap_encryption VARCHAR(20) DEFAULT 'ssl',
ADD COLUMN IF NOT EXISTS pop3_encryption VARCHAR(20) DEFAULT 'ssl',
ADD COLUMN IF NOT EXISTS smtp_encryption VARCHAR(20) DEFAULT 'ssl';

-- 添加字段注释
COMMENT ON COLUMN providers.imap_encryption IS 'IMAP 加密方式: ssl/starttls/none';
COMMENT ON COLUMN providers.pop3_encryption IS 'POP3 加密方式: ssl/starttls/none';
COMMENT ON COLUMN providers.smtp_encryption IS 'SMTP 加密方式: ssl/starttls/none';

-- 根据端口智能设置加密方式
-- IMAP: 993=SSL, 143=STARTTLS
UPDATE providers 
SET imap_encryption = CASE 
    WHEN imap_port = 993 THEN 'ssl'
    WHEN imap_port = 143 THEN 'starttls'
    ELSE 'ssl'
END
WHERE imap_encryption IS NULL OR imap_encryption = '';

-- POP3: 995=SSL, 110=STARTTLS
UPDATE providers 
SET pop3_encryption = CASE 
    WHEN pop3_port = 995 THEN 'ssl'
    WHEN pop3_port = 110 THEN 'starttls'
    ELSE 'ssl'
END
WHERE pop3_encryption IS NULL OR pop3_encryption = '';

-- SMTP: 465=SSL, 587=STARTTLS
UPDATE providers 
SET smtp_encryption = CASE 
    WHEN smtp_port = 465 THEN 'ssl'
    WHEN smtp_port = 587 THEN 'starttls'
    ELSE 'ssl'
END
WHERE smtp_encryption IS NULL OR smtp_encryption = '';

-- 更新统计信息
ANALYZE providers;
