-- 038_cleanup_duplicate_system_settings.sql
-- 修复：PostgreSQL 中 NULL != NULL，ON CONFLICT (user_id, category, key) 对 user_id IS NULL 的行不触发冲突，
-- 导致每次 Set 系统配置时 INSERT 新行而非 UPDATE，产生重复数据。
-- 此 migration 清理已产生的重复行，保留每组 (category, key) 中 updated_at 最新的那条。

-- 删除系统级配置（user_id IS NULL）的重复行，保留 updated_at 最新（或 id 最大）的那条
DELETE FROM settings
WHERE user_id IS NULL
  AND id NOT IN (
    SELECT DISTINCT ON (category, key)
      id
    FROM settings
    WHERE user_id IS NULL
    ORDER BY category, key, updated_at DESC, id DESC
  );