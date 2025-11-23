-- OAuth2 客户端配置表
-- 用于管理多个 OAuth2 客户端配置，支持配额管理和智能切换

CREATE TABLE oauth2_clients (
    id BIGSERIAL PRIMARY KEY,
    provider_name VARCHAR(50) NOT NULL, -- 邮箱提供商：gmail, outlook 等
    name VARCHAR(100) NOT NULL, -- 配置名称：如"开发环境", "生产环境"
    client_id VARCHAR(255) NOT NULL, -- OAuth2 客户端 ID
    client_secret_encrypted TEXT NOT NULL, -- 加密存储的客户端密钥
    redirect_uri VARCHAR(500) NOT NULL, -- 重定向 URI
    enabled BOOLEAN DEFAULT TRUE, -- 是否启用
    is_default BOOLEAN DEFAULT FALSE, -- 是否为默认客户端
    usage_count INTEGER DEFAULT 0, -- 使用次数统计
    quota_daily INTEGER DEFAULT -1, -- 日配额限制，-1表示无限制
    quota_monthly INTEGER DEFAULT -1, -- 月配额限制，-1表示无限制
    last_used_at TIMESTAMP NULL, -- 最后使用时间
    metadata TEXT, -- 元数据（JSON格式），存储额外配置
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- 创建索引
CREATE INDEX idx_oauth2_clients_provider ON oauth2_clients(provider_name);
CREATE INDEX idx_oauth2_clients_enabled ON oauth2_clients(enabled);
CREATE INDEX idx_oauth2_clients_default ON oauth2_clients(is_default) WHERE is_default = TRUE;

-- 唯一约束：每个提供商只能有一个默认客户端
CREATE UNIQUE INDEX uk_oauth2_clients_provider_default
ON oauth2_clients(provider_name)
WHERE is_default = TRUE AND enabled = TRUE;

-- 唯一约束：客户端名称在同一个提供商下唯一
CREATE UNIQUE INDEX uk_oauth2_clients_provider_name
ON oauth2_clients(provider_name, name);

-- 注释
COMMENT ON TABLE oauth2_clients IS 'OAuth2 客户端配置表 - 支持多客户端配额管理';
COMMENT ON COLUMN oauth2_clients.provider_name IS '邮箱提供商名称（如 gmail, outlook）';
COMMENT ON COLUMN oauth2_clients.name IS '配置名称（如开发环境、生产环境）';
COMMENT ON COLUMN oauth2_clients.client_id IS 'OAuth2 客户端 ID';
COMMENT ON COLUMN oauth2_clients.client_secret_encrypted IS '加密存储的客户端密钥';
COMMENT ON COLUMN oauth2_clients.redirect_uri IS 'OAuth2 重定向 URI';
COMMENT ON COLUMN oauth2_clients.enabled IS '是否启用该配置';
COMMENT ON COLUMN oauth2_clients.is_default IS '是否为该提供商的默认配置';
COMMENT ON COLUMN oauth2_clients.usage_count IS '使用次数统计';
COMMENT ON COLUMN oauth2_clients.quota_daily IS '日配额限制（-1为无限制）';
COMMENT ON COLUMN oauth2_clients.quota_monthly IS '月配额限制（-1为无限制）';
COMMENT ON COLUMN oauth2_clients.last_used_at IS '最后使用时间';
COMMENT ON COLUMN oauth2_clients.metadata IS '元数据（JSON格式）';
