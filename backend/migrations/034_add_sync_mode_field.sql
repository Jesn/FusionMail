-- 034_add_sync_mode_field.sql
-- 添加 sync_mode 字段到 email_accounts 表
-- 用于区分轮询模式（polling）和 Webhook 推送模式（webhook）

-- 添加 sync_mode 字段
ALTER TABLE email_accounts 
ADD COLUMN IF NOT EXISTS sync_mode VARCHAR(20) DEFAULT 'polling';

-- 添加注释
COMMENT ON COLUMN email_accounts.sync_mode IS '同步模式：polling（轮询，默认）或 webhook（推送）';

-- 创建索引（用于按同步模式筛选账户）
CREATE INDEX IF NOT EXISTS idx_email_accounts_sync_mode ON email_accounts(sync_mode);
