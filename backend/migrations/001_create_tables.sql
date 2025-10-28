-- FusionMail 数据库表结构
-- 此文件仅作为文档和参考，实际迁移使用 GORM AutoMigrate
-- 生成时间: 2025-10-28

-- 用户表
CREATE TABLE IF NOT EXISTS users (
    id BIGSERIAL PRIMARY KEY,
    username VARCHAR(100) UNIQUE NOT NULL,
    email VARCHAR(255) UNIQUE NOT NULL,
    password_hash VARCHAR(255) NOT NULL,
    display_name VARCHAR(255),
    avatar_url TEXT,
    timezone VARCHAR(50) DEFAULT 'UTC',
    language VARCHAR(10) DEFAULT 'en',
    theme VARCHAR(20) DEFAULT 'light',
    is_active BOOLEAN DEFAULT TRUE,
    email_verified BOOLEAN DEFAULT FALSE,
    last_login_at TIMESTAMP,
    last_login_ip VARCHAR(45),
    failed_login_attempts INTEGER DEFAULT 0,
    locked_until TIMESTAMP,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_users_username ON users(username);
CREATE INDEX idx_users_email ON users(email);

-- 邮箱账户表
CREATE TABLE IF NOT EXISTS accounts (
    id BIGSERIAL PRIMARY KEY,
    uid VARCHAR(64) UNIQUE NOT NULL,
    email VARCHAR(255) NOT NULL,
    provider VARCHAR(50) NOT NULL,
    protocol VARCHAR(20) NOT NULL,
    auth_type VARCHAR(20) NOT NULL,
    encrypted_credentials TEXT NOT NULL,
    proxy_enabled BOOLEAN DEFAULT FALSE,
    proxy_type VARCHAR(20),
    proxy_host VARCHAR(255),
    proxy_port INTEGER,
    proxy_username VARCHAR(255),
    encrypted_proxy_password TEXT,
    sync_enabled BOOLEAN DEFAULT TRUE,
    sync_interval INTEGER DEFAULT 5,
    last_sync_at TIMESTAMP,
    last_sync_status VARCHAR(20),
    last_sync_error TEXT,
    total_emails INTEGER DEFAULT 0,
    unread_count INTEGER DEFAULT 0,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP
);

CREATE INDEX idx_accounts_email ON accounts(email);
CREATE INDEX idx_accounts_provider ON accounts(provider);
CREATE INDEX idx_accounts_sync_enabled ON accounts(sync_enabled);
CREATE INDEX idx_accounts_deleted_at ON accounts(deleted_at);

-- 邮件主表
CREATE TABLE IF NOT EXISTS emails (
    id BIGSERIAL PRIMARY KEY,
    provider_id VARCHAR(255) NOT NULL,
    account_uid VARCHAR(64) NOT NULL,
    message_id VARCHAR(255),
    subject TEXT NOT NULL,
    from_address VARCHAR(255) NOT NULL,
    from_name VARCHAR(255),
    to_addresses TEXT,
    cc_addresses TEXT,
    bcc_addresses TEXT,
    reply_to VARCHAR(255),
    text_body TEXT,
    html_body TEXT,
    snippet TEXT,
    is_read BOOLEAN DEFAULT FALSE,
    is_starred BOOLEAN DEFAULT FALSE,
    is_archived BOOLEAN DEFAULT FALSE,
    is_deleted BOOLEAN DEFAULT FALSE,
    local_labels TEXT,
    source_is_read BOOLEAN,
    source_labels TEXT,
    source_folder VARCHAR(255),
    has_attachments BOOLEAN DEFAULT FALSE,
    attachments_count INTEGER DEFAULT 0,
    sent_at TIMESTAMP NOT NULL,
    received_at TIMESTAMP NOT NULL,
    synced_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    size_bytes BIGINT,
    thread_id VARCHAR(255),
    in_reply_to VARCHAR(255),
    references TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    UNIQUE (provider_id, account_uid)
);

CREATE INDEX idx_emails_account_uid ON emails(account_uid);
CREATE INDEX idx_emails_message_id ON emails(message_id);
CREATE INDEX idx_emails_from_address ON emails(from_address);
CREATE INDEX idx_emails_sent_at ON emails(sent_at DESC);
CREATE INDEX idx_emails_is_read ON emails(is_read);
CREATE INDEX idx_emails_is_starred ON emails(is_starred);
CREATE INDEX idx_emails_is_archived ON emails(is_archived);
CREATE INDEX idx_emails_is_deleted ON emails(is_deleted);

-- 全文搜索索引（将在 AutoMigrate 后创建）
-- CREATE INDEX idx_emails_fulltext_search ON emails 
-- USING gin(to_tsvector('english', coalesce(subject, '') || ' ' || coalesce(from_name, '') || ' ' || coalesce(text_body, '')));

-- 邮件附件表
CREATE TABLE IF NOT EXISTS email_attachments (
    id BIGSERIAL PRIMARY KEY,
    email_id BIGINT NOT NULL,
    filename VARCHAR(255) NOT NULL,
    content_type VARCHAR(100),
    size_bytes BIGINT NOT NULL,
    storage_type VARCHAR(20) DEFAULT 'local',
    storage_path TEXT NOT NULL,
    is_inline BOOLEAN DEFAULT FALSE,
    content_id VARCHAR(255),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (email_id) REFERENCES emails(id) ON DELETE CASCADE
);

CREATE INDEX idx_email_attachments_email_id ON email_attachments(email_id);
CREATE INDEX idx_email_attachments_filename ON email_attachments(filename);

-- 邮件标签表
CREATE TABLE IF NOT EXISTS email_labels (
    id BIGSERIAL PRIMARY KEY,
    name VARCHAR(100) UNIQUE NOT NULL,
    color VARCHAR(20),
    description TEXT,
    email_count INTEGER DEFAULT 0,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- 邮件-标签关联表
CREATE TABLE IF NOT EXISTS email_label_relations (
    email_id BIGINT NOT NULL,
    label_id BIGINT NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (email_id, label_id),
    FOREIGN KEY (email_id) REFERENCES emails(id) ON DELETE CASCADE,
    FOREIGN KEY (label_id) REFERENCES email_labels(id) ON DELETE CASCADE
);

-- 邮件规则表
CREATE TABLE IF NOT EXISTS email_rules (
    id BIGSERIAL PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    description TEXT,
    enabled BOOLEAN DEFAULT TRUE,
    priority INTEGER DEFAULT 0,
    conditions TEXT NOT NULL,
    actions TEXT NOT NULL,
    matched_count INTEGER DEFAULT 0,
    last_matched_at TIMESTAMP,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_email_rules_enabled ON email_rules(enabled);
CREATE INDEX idx_email_rules_priority ON email_rules(priority);

-- Webhook 配置表
CREATE TABLE IF NOT EXISTS webhooks (
    id BIGSERIAL PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    description TEXT,
    url TEXT NOT NULL,
    method VARCHAR(10) DEFAULT 'POST',
    headers TEXT,
    enabled BOOLEAN DEFAULT TRUE,
    events TEXT NOT NULL,
    filters TEXT,
    retry_enabled BOOLEAN DEFAULT TRUE,
    max_retries INTEGER DEFAULT 3,
    retry_intervals TEXT DEFAULT '[10, 30, 60]',
    total_calls INTEGER DEFAULT 0,
    success_calls INTEGER DEFAULT 0,
    failed_calls INTEGER DEFAULT 0,
    last_called_at TIMESTAMP,
    last_status VARCHAR(20),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_webhooks_enabled ON webhooks(enabled);

-- Webhook 调用日志表
CREATE TABLE IF NOT EXISTS webhook_logs (
    id BIGSERIAL PRIMARY KEY,
    webhook_id BIGINT NOT NULL,
    request_url TEXT NOT NULL,
    request_method VARCHAR(10),
    request_headers TEXT,
    request_body TEXT,
    response_status INTEGER,
    response_body TEXT,
    response_time_ms INTEGER,
    success BOOLEAN,
    error_message TEXT,
    retry_count INTEGER DEFAULT 0,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (webhook_id) REFERENCES webhooks(id) ON DELETE CASCADE
);

CREATE INDEX idx_webhook_logs_webhook_id ON webhook_logs(webhook_id);
CREATE INDEX idx_webhook_logs_created_at ON webhook_logs(created_at DESC);

-- 同步日志表
CREATE TABLE IF NOT EXISTS sync_logs (
    id BIGSERIAL PRIMARY KEY,
    account_uid VARCHAR(64) NOT NULL,
    sync_type VARCHAR(20) NOT NULL,
    status VARCHAR(20) NOT NULL,
    emails_fetched INTEGER DEFAULT 0,
    emails_new INTEGER DEFAULT 0,
    emails_updated INTEGER DEFAULT 0,
    started_at TIMESTAMP NOT NULL,
    completed_at TIMESTAMP,
    duration_ms INTEGER,
    error_message TEXT,
    error_stack TEXT,
    FOREIGN KEY (account_uid) REFERENCES accounts(uid) ON DELETE CASCADE
);

CREATE INDEX idx_sync_logs_account_uid ON sync_logs(account_uid);
CREATE INDEX idx_sync_logs_status ON sync_logs(status);
CREATE INDEX idx_sync_logs_started_at ON sync_logs(started_at DESC);

-- API 密钥表
CREATE TABLE IF NOT EXISTS api_keys (
    id BIGSERIAL PRIMARY KEY,
    key_hash VARCHAR(64) UNIQUE NOT NULL,
    name VARCHAR(255) NOT NULL,
    description TEXT,
    permissions TEXT,
    rate_limit INTEGER DEFAULT 100,
    enabled BOOLEAN DEFAULT TRUE,
    expires_at TIMESTAMP,
    total_requests INTEGER DEFAULT 0,
    last_used_at TIMESTAMP,
    last_ip VARCHAR(45),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_api_keys_key_hash ON api_keys(key_hash);
CREATE INDEX idx_api_keys_enabled ON api_keys(enabled);
