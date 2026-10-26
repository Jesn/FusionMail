-- 2FA 双因素认证字段迁移
-- 为用户表添加 TOTP 双因素认证支持

-- 添加 2FA 相关字段到 users 表
ALTER TABLE users ADD COLUMN IF NOT EXISTS two_factor_enabled BOOLEAN DEFAULT FALSE;
ALTER TABLE users ADD COLUMN IF NOT EXISTS two_factor_secret VARCHAR(64);
ALTER TABLE users ADD COLUMN IF NOT EXISTS two_factor_backup TEXT;
ALTER TABLE users ADD COLUMN IF NOT EXISTS two_factor_verified BOOLEAN DEFAULT FALSE;
ALTER TABLE users ADD COLUMN IF NOT EXISTS two_factor_enabled_at TIMESTAMP;

-- 添加注释
COMMENT ON COLUMN users.two_factor_enabled IS '是否启用双因素认证';
COMMENT ON COLUMN users.two_factor_secret IS 'TOTP 密钥（加密存储）';
COMMENT ON COLUMN users.two_factor_backup IS '恢复码（加密存储，JSON 数组）';
COMMENT ON COLUMN users.two_factor_verified IS '2FA 是否已验证完成设置';
COMMENT ON COLUMN users.two_factor_enabled_at IS '2FA 启用时间';
