-- 添加短期邮箱账号过期处理相关字段
-- Migration: 004_add_quick_account_expiry_fields
-- Description: 为 accounts 表添加自动禁用相关字段，用于短期邮箱账号过期检测

-- 添加连续认证失败计数字段
ALTER TABLE accounts ADD COLUMN IF NOT EXISTS consecutive_auth_failures INTEGER DEFAULT 0 NOT NULL;

-- 添加自动禁用时间字段
ALTER TABLE accounts ADD COLUMN IF NOT EXISTS auto_disabled_at TIMESTAMP NULL;

-- 添加禁用原因字段
ALTER TABLE accounts ADD COLUMN IF NOT EXISTS disable_reason VARCHAR(100) NULL;

-- 添加索引以优化查询性能
-- 用于快速查询特定类型和状态的账号
CREATE INDEX IF NOT EXISTS idx_accounts_auth_type_status ON accounts(auth_type, status);

-- 用于快速查询有失败记录的短期账号
CREATE INDEX IF NOT EXISTS idx_accounts_consecutive_failures ON accounts(consecutive_auth_failures) 
  WHERE auth_type = 'quick' AND consecutive_auth_failures > 0;

-- 为现有的 quick 账号初始化字段（确保不为 NULL）
UPDATE accounts 
SET consecutive_auth_failures = 0 
WHERE auth_type = 'quick' AND consecutive_auth_failures IS NULL;

-- 添加字段注释
COMMENT ON COLUMN accounts.consecutive_auth_failures IS '连续认证失败次数（仅用于 quick 类型账号）';
COMMENT ON COLUMN accounts.auto_disabled_at IS '自动禁用时间戳';
COMMENT ON COLUMN accounts.disable_reason IS '禁用原因（如：auto_disabled_auth_failure）';
