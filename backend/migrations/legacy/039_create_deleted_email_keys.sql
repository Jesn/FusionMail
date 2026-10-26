-- 039_create_deleted_email_keys.sql
-- 补建 deleted_email_keys：清理服务与同步去重依赖此表
-- 说明：
-- 1) 历史 SQL 030 使用 account_uid UUID，与实际账户 UID（含 webhook_* 字符串）不兼容
-- 2) 线上 migrate 工具走 GORM AutoMigrate，此前未注册该模型导致表缺失
-- 3) 此处使用 VARCHAR(64)，与 model.DeletedEmailKey / email_accounts.uid 一致

CREATE TABLE IF NOT EXISTS deleted_email_keys (
    id BIGSERIAL PRIMARY KEY,
    account_uid VARCHAR(64) NOT NULL,
    dedupe_key VARCHAR(128) NOT NULL,
    deleted_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    CONSTRAINT uk_deleted_email_keys UNIQUE (account_uid, dedupe_key)
);

CREATE INDEX IF NOT EXISTS idx_deleted_email_keys_cleanup ON deleted_email_keys (deleted_at);
CREATE INDEX IF NOT EXISTS idx_deleted_email_keys_account ON deleted_email_keys (account_uid);

COMMENT ON TABLE deleted_email_keys IS '已删除邮件的去重标识记录，防止同步时重复创建';
COMMENT ON COLUMN deleted_email_keys.account_uid IS '邮箱账户 UID（与 email_accounts.uid 同型，非强制 UUID）';
COMMENT ON COLUMN deleted_email_keys.dedupe_key IS '邮件去重标识';
COMMENT ON COLUMN deleted_email_keys.deleted_at IS '删除时间，用于 90 天后清理';
