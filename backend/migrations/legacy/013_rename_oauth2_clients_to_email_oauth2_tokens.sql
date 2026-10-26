-- 重命名 o_auth2_clients 表为 email_oauth2_tokens
-- 注意：GORM 自动将 OAuth2Client 结构体转换为 o_auth2_clients 表名（蛇形命名）
-- 理由：
-- 1. 语义更准确：表存储的是 OAuth2 令牌，而非客户端配置
-- 2. 命名一致：与项目中的 emails、accounts 表风格一致
-- 3. 清晰明确：一看就知道是邮件相关的 OAuth2 令牌

-- 重命名表
ALTER TABLE o_auth2_clients RENAME TO email_oauth2_tokens;

-- 重命名索引（PostgreSQL 会自动保留索引，但建议重命名以保持一致性）
ALTER INDEX IF EXISTS idx_o_auth2_clients_provider_id RENAME TO idx_email_oauth2_tokens_provider_id;
ALTER INDEX IF EXISTS idx_o_auth2_clients_enabled RENAME TO idx_email_oauth2_tokens_enabled;
ALTER INDEX IF EXISTS idx_o_auth2_clients_is_default RENAME TO idx_email_oauth2_tokens_is_default;
ALTER INDEX IF EXISTS idx_oauth2_clients_provider_id RENAME TO idx_email_oauth2_tokens_provider_id;

-- 重命名外键约束
ALTER TABLE email_oauth2_tokens 
DROP CONSTRAINT IF EXISTS fk_o_auth2_clients_provider;

ALTER TABLE email_oauth2_tokens 
DROP CONSTRAINT IF EXISTS fk_oauth2_clients_provider;

ALTER TABLE email_oauth2_tokens 
ADD CONSTRAINT fk_email_oauth2_tokens_provider 
FOREIGN KEY (provider_id) REFERENCES providers(id) ON DELETE CASCADE;

-- 更新表注释
COMMENT ON TABLE email_oauth2_tokens IS 'OAuth2 令牌配置表 - 存储邮件账户的 OAuth2 认证令牌';
