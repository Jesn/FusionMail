-- 回滚邮件搜索索引。
--
-- 注意：不 DROP EXTENSION pg_trgm —— 其他对象可能依赖它，
-- 且扩展本身无害。只回滚本迁移创建的索引。
DROP INDEX IF EXISTS idx_emails_account_deleted_sent_at;
DROP INDEX IF EXISTS idx_emails_account_sent_at;
DROP INDEX IF EXISTS idx_emails_text_body_trgm;
DROP INDEX IF EXISTS idx_emails_from_address_trgm;
DROP INDEX IF EXISTS idx_emails_from_name_trgm;
DROP INDEX IF EXISTS idx_emails_subject_trgm;
