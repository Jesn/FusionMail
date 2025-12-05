-- 添加首次同步优化相关字段
-- Migration: 016_add_first_sync_optimization_fields
-- Description: 为 email_accounts 表添加首次同步配置字段，为 sync_logs 表添加扩展字段
-- Requirements: 6.1

-- =====================================================
-- Part 1: email_accounts 表新增字段
-- =====================================================

-- 添加首次同步天数配置字段
ALTER TABLE email_accounts ADD COLUMN IF NOT EXISTS first_sync_days INTEGER DEFAULT 7;

-- 添加批次大小配置字段
ALTER TABLE email_accounts ADD COLUMN IF NOT EXISTS batch_size INTEGER DEFAULT 100;

-- 添加单次同步最大邮件数配置字段
ALTER TABLE email_accounts ADD COLUMN IF NOT EXISTS max_emails_per_sync INTEGER DEFAULT 5000;

-- 添加同步游标字段（用于断点续传）
ALTER TABLE email_accounts ADD COLUMN IF NOT EXISTS sync_cursor TEXT;

-- 添加同步进度 JSON 字段（存储详细进度信息）
ALTER TABLE email_accounts ADD COLUMN IF NOT EXISTS sync_progress_json JSONB;

-- 添加字段注释
COMMENT ON COLUMN email_accounts.first_sync_days IS '首次同步天数，0 表示全量同步，默认 7 天';
COMMENT ON COLUMN email_accounts.batch_size IS '每批处理邮件数量，默认 100';
COMMENT ON COLUMN email_accounts.max_emails_per_sync IS '单次同步最大邮件数，默认 5000';
COMMENT ON COLUMN email_accounts.sync_cursor IS '同步游标，用于断点续传';
COMMENT ON COLUMN email_accounts.sync_progress_json IS '同步进度 JSON，存储详细进度信息';

-- =====================================================
-- Part 2: sync_logs 表新增字段
-- =====================================================

-- 添加预估总数字段
ALTER TABLE sync_logs ADD COLUMN IF NOT EXISTS total_estimated INTEGER;

-- 添加当前批次字段
ALTER TABLE sync_logs ADD COLUMN IF NOT EXISTS current_batch INTEGER;

-- 添加总批次数字段
ALTER TABLE sync_logs ADD COLUMN IF NOT EXISTS total_batches INTEGER;

-- 添加同步游标字段
ALTER TABLE sync_logs ADD COLUMN IF NOT EXISTS sync_cursor TEXT;

-- 添加是否首次同步标记字段
ALTER TABLE sync_logs ADD COLUMN IF NOT EXISTS is_first_sync BOOLEAN DEFAULT FALSE;

-- 添加字段注释
COMMENT ON COLUMN sync_logs.total_estimated IS '预估邮件总数';
COMMENT ON COLUMN sync_logs.current_batch IS '当前处理批次';
COMMENT ON COLUMN sync_logs.total_batches IS '总批次数';
COMMENT ON COLUMN sync_logs.sync_cursor IS '同步游标位置';
COMMENT ON COLUMN sync_logs.is_first_sync IS '是否为首次同步';

-- =====================================================
-- Part 3: 索引优化
-- =====================================================

-- 为 sync_logs 添加首次同步查询索引
CREATE INDEX IF NOT EXISTS idx_sync_logs_is_first_sync ON sync_logs(is_first_sync) WHERE is_first_sync = TRUE;

-- 为 email_accounts 添加同步配置查询索引
CREATE INDEX IF NOT EXISTS idx_email_accounts_sync_enabled ON email_accounts(sync_enabled) WHERE sync_enabled = TRUE;
