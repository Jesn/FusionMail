-- 添加 emails 表的 deleted_at 字段用于软删除
ALTER TABLE emails ADD COLUMN IF NOT EXISTS deleted_at TIMESTAMP WITH TIME ZONE;

-- 添加索引以提高软删除查询性能
CREATE INDEX IF NOT EXISTS idx_emails_deleted_at ON emails(deleted_at);
