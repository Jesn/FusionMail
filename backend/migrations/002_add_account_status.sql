-- 添加账户状态字段
-- 迁移时间: 2025-10-31

-- 为 accounts 表添加 status 字段
ALTER TABLE accounts ADD COLUMN IF NOT EXISTS status VARCHAR(20) DEFAULT 'active';

-- 创建索引以提高查询性能
CREATE INDEX IF NOT EXISTS idx_accounts_status ON accounts(status);

-- 更新现有记录的状态为 'active'
UPDATE accounts SET status = 'active' WHERE status IS NULL;

-- 添加注释
COMMENT ON COLUMN accounts.status IS '账户状态: active(正常), disabled(已禁用), error(错误)';