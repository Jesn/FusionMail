-- maintenance_optimize_log_tables.sql
-- 优化日志表：添加索引和清理历史数据
-- 
-- 背景：
--   1. sync_logs 和 spam_detection_logs 表数据量过大
--   2. 定时清理任务因 SQL 语法错误未能正常执行
--   3. 需要添加索引以提升清理性能
--
-- 修复的 Bug：
--   - sync_log.go: DeleteOldLogs 方法的 INTERVAL 语法错误
--   - spam_detection_log.go: DeleteOldLogs 方法字段名 checked_at 应为 created_at
--   - sync_service.go: 垃圾检测在邮件保存前执行导致 email_id=0
--
-- Author: FusionMail Team
-- Date: 2024-12-17

-- ============================================
-- 阶段 1：创建索引（提升清理性能）
-- 预计执行时间：< 1 分钟
-- 风险等级：低
-- ============================================

-- 1.1 为 sync_logs 表的 started_at 字段创建索引
-- 用于加速按时间范围删除旧日志
CREATE INDEX IF NOT EXISTS idx_sync_logs_started_at ON sync_logs(started_at);

-- 1.2 为 spam_detection_logs 表的 created_at 字段创建索引
-- 用于加速按时间范围删除旧日志
CREATE INDEX IF NOT EXISTS idx_spam_detection_logs_created_at ON spam_detection_logs(created_at);

-- ============================================
-- 阶段 2：清理历史数据
-- 预计执行时间：取决于数据量（可能需要几分钟）
-- 风险等级：中（删除数据操作）
-- 注意：此操作不可逆，请确保已备份重要数据
-- ============================================

-- 2.1 清理超过 7 天的 sync_logs
-- 配置项 sync_logs_retention_days 默认为 7 天
DELETE FROM sync_logs WHERE started_at < NOW() - INTERVAL '7 days';

-- 2.2 清理超过 7 天的 spam_detection_logs
-- 配置项 spam_detection_logs_retention_days 默认为 7 天
DELETE FROM spam_detection_logs WHERE created_at < NOW() - INTERVAL '7 days';

-- 2.3 清理 email_id='0' 的无效日志
-- 这些日志是由于垃圾检测在邮件保存前执行导致的
-- Bug 已在 sync_service.go 中修复
DELETE FROM spam_detection_logs WHERE email_id = '0';

-- ============================================
-- 阶段 3：验证清理结果
-- ============================================

-- 3.1 显示清理后的数据统计
DO $$
DECLARE
    sync_logs_count INTEGER;
    spam_logs_count INTEGER;
    sync_logs_oldest TIMESTAMP;
    spam_logs_oldest TIMESTAMP;
BEGIN
    SELECT COUNT(*), MIN(started_at) INTO sync_logs_count, sync_logs_oldest FROM sync_logs;
    SELECT COUNT(*), MIN(created_at) INTO spam_logs_count, spam_logs_oldest FROM spam_detection_logs;
    
    RAISE NOTICE '========================================';
    RAISE NOTICE '日志表清理完成:';
    RAISE NOTICE '  sync_logs: % 条记录, 最早记录: %', sync_logs_count, sync_logs_oldest;
    RAISE NOTICE '  spam_detection_logs: % 条记录, 最早记录: %', spam_logs_count, spam_logs_oldest;
    RAISE NOTICE '========================================';
END $$;

-- 3.2 验证索引创建
SELECT 
    tablename,
    indexname,
    indexdef
FROM pg_indexes 
WHERE tablename IN ('sync_logs', 'spam_detection_logs')
  AND indexname IN ('idx_sync_logs_started_at', 'idx_spam_detection_logs_created_at')
ORDER BY tablename, indexname;

-- ============================================
-- 回滚脚本（如需回滚索引，执行以下语句）
-- 注意：删除的数据无法恢复
-- ============================================
-- DROP INDEX IF EXISTS idx_sync_logs_started_at;
-- DROP INDEX IF EXISTS idx_spam_detection_logs_created_at;
