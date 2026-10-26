-- 邮件同步优化迁移
-- 实现基于 UID 的增量同步和稳定的去重机制
-- Requirements: 1.1, 1.2, 1.3, 2.1, 2.2, 2.3, 3.1, 3.2, 3.3, 3.4, 6.1

-- ============================================
-- 1. emails 表新增 dedupe_key 字段
-- ============================================
-- dedupe_key 格式：
--   有 Message-ID: "mid:{message_id}"
--   无 Message-ID: "hash:{sha256(from|subject|sent_at)[:32]}"

ALTER TABLE emails ADD COLUMN IF NOT EXISTS dedupe_key VARCHAR(128);

-- 创建唯一索引（account_uid + dedupe_key），仅对非空值生效
-- 这确保同一账户下不会有重复的 dedupe_key
CREATE UNIQUE INDEX IF NOT EXISTS idx_emails_dedupe_account 
    ON emails(account_uid, dedupe_key) 
    WHERE dedupe_key IS NOT NULL;

-- 为 dedupe_key 创建普通索引，用于快速查找
CREATE INDEX IF NOT EXISTS idx_emails_dedupe_key ON emails(dedupe_key) WHERE dedupe_key IS NOT NULL;

-- ============================================
-- 2. email_accounts 表新增同步状态字段
-- ============================================
-- uid_validity: IMAP 邮箱有效性标识，变化时需要全量同步
-- last_uid: 上次同步的最大 UID，用于增量同步

ALTER TABLE email_accounts ADD COLUMN IF NOT EXISTS uid_validity BIGINT DEFAULT 0;
ALTER TABLE email_accounts ADD COLUMN IF NOT EXISTS last_uid BIGINT DEFAULT 0;

-- 添加注释
COMMENT ON COLUMN email_accounts.uid_validity IS 'IMAP UIDVALIDITY 值，变化时需要全量同步';
COMMENT ON COLUMN email_accounts.last_uid IS '上次同步的最大 UID，用于增量同步';

-- ============================================
-- 3. 创建 deleted_email_keys 表
-- ============================================
-- 用于记录已物理删除邮件的 dedupe_key，防止重新同步时重复创建

CREATE TABLE IF NOT EXISTS deleted_email_keys (
    id BIGSERIAL PRIMARY KEY,
    account_uid UUID NOT NULL,
    dedupe_key VARCHAR(128) NOT NULL,
    deleted_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    
    -- 唯一约束：同一账户下的 dedupe_key 只能记录一次
    CONSTRAINT uk_deleted_email_keys UNIQUE(account_uid, dedupe_key)
);

-- 创建索引
-- 用于清理过期记录（90天前）
CREATE INDEX IF NOT EXISTS idx_deleted_email_keys_cleanup ON deleted_email_keys(deleted_at);
-- 用于按账户查询
CREATE INDEX IF NOT EXISTS idx_deleted_email_keys_account ON deleted_email_keys(account_uid);

-- 添加表注释
COMMENT ON TABLE deleted_email_keys IS '已删除邮件的去重标识记录，防止同步时重复创建';
COMMENT ON COLUMN deleted_email_keys.account_uid IS '邮箱账户 UID';
COMMENT ON COLUMN deleted_email_keys.dedupe_key IS '邮件去重标识';
COMMENT ON COLUMN deleted_email_keys.deleted_at IS '删除时间，用于 90 天后清理';
