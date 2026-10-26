-- ============================================================
-- 邮件搜索：中文场景的可用性与性能修复
--
-- 背景（2026-10 架构体检）：
--   repository/email.go 的 Search 先用
--     to_tsvector('english', subject || from_name || text_body)
--       @@ plainto_tsquery('english', query)
--   做全文搜索。'english' 配置对中文整段只产生一个 token，中文查询几乎必然
--   匹配不到 → 总是走降级路径 ILIKE '%q%' 四列全表扫描，且无相关性排序。
--   而 migrations/001_create_tables.sql:110 的 GIN 索引是被注释掉的，
--   所以这条降级路径连索引都没有。
--
-- 方案：
--   1. 启用 pg_trgm —— trigram 索引对任意字符（含中文）生效，
--      能让 ILIKE '%中文%' 走索引而不是全表扫描。
--      （zhparser / pgroonga 未在目标环境提供，故不采用分词方案。）
--   2. 为四个可搜索列建 GIN(gin_trgm_ops) 索引。
--   3. 应用层（repository/email.go）对非 ASCII 查询直接跳过无用的
--      'english' 全文搜索，避免每次中文搜索都白跑一次全表 tsvector 计算。
--
-- 注意：pg_trgm 在 PostgreSQL 13+ 是 trusted extension，
--       数据库 owner 即可 CREATE EXTENSION，无需超级用户。
-- ============================================================

CREATE EXTENSION IF NOT EXISTS pg_trgm;

-- 主题：中文搜索的主要目标
CREATE INDEX IF NOT EXISTS idx_emails_subject_trgm
    ON emails USING gin (subject gin_trgm_ops);

-- 发件人名称
CREATE INDEX IF NOT EXISTS idx_emails_from_name_trgm
    ON emails USING gin (from_name gin_trgm_ops);

-- 发件人地址（邮箱地址本身是 ASCII，trigram 同样有效）
CREATE INDEX IF NOT EXISTS idx_emails_from_address_trgm
    ON emails USING gin (from_address gin_trgm_ops);

-- 正文：数据量最大，trigram 索引也最大；但中文搜索命中正文是常见需求
CREATE INDEX IF NOT EXISTS idx_emails_text_body_trgm
    ON emails USING gin (text_body gin_trgm_ops);

-- 保留原有的英文全文搜索能力（英文查询仍走 tsvector 路径）。
-- 原 migrations/001 里这条索引被注释掉了，这里补上，让英文路径也有索引。
CREATE INDEX IF NOT EXISTS idx_emails_fulltext_en
    ON emails USING gin (
        to_tsvector('english',
            coalesce(subject, '') || ' ' ||
            coalesce(from_name, '') || ' ' ||
            coalesce(text_body, ''))
    );
