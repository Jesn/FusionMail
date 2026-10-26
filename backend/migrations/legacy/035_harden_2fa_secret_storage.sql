-- 放宽 2FA 密钥字段长度，支持 AES-GCM 密文前缀
ALTER TABLE users ALTER COLUMN two_factor_secret TYPE TEXT;

COMMENT ON COLUMN users.two_factor_secret IS 'TOTP 密钥（加密存储）';
COMMENT ON COLUMN users.two_factor_backup IS '恢复码（哈希存储，JSON 数组）';
