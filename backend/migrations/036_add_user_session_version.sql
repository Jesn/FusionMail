ALTER TABLE users ADD COLUMN IF NOT EXISTS session_version BIGINT NOT NULL DEFAULT 0;

COMMENT ON COLUMN users.session_version IS '会话版本号，用于服务端撤销 JWT';
