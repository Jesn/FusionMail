-- 回填现有邮件的 dedupe_key
-- Requirements: 3.3 - 为所有现有邮件生成 dedupe_key
-- 
-- 注意：此迁移需要在应用层执行，因为 dedupe_key 的生成逻辑涉及：
-- 1. 系统通知域名检测（@139.com, @10086.cn 等）
-- 2. SHA256 哈希计算
-- 
-- 此 SQL 文件仅作为占位符，实际迁移通过 Go 代码执行

-- 为没有 dedupe_key 的邮件添加临时标记（用于追踪迁移进度）
-- 实际的 dedupe_key 生成在应用层完成

-- 验证迁移完成的查询：
-- SELECT COUNT(*) FROM emails WHERE dedupe_key IS NULL;
-- 期望结果：0

-- 如果需要手动生成简单的 dedupe_key（不推荐，仅用于紧急情况）：
-- UPDATE emails 
-- SET dedupe_key = CASE 
--     WHEN message_id IS NOT NULL AND message_id != '' THEN 'mid:' || LEFT(message_id, 60)
--     ELSE 'hash:' || LEFT(MD5(COALESCE(from_address, '') || '|' || COALESCE(subject, '') || '|' || TO_CHAR(sent_at, 'YYYY-MM-DD"T"HH24:MI')), 32)
-- END
-- WHERE dedupe_key IS NULL;
