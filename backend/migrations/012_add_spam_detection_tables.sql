-- 垃圾邮件检测功能数据库迁移
-- 创建时间: 2025-11-26

-- 1. 扩展 emails 表，添加垃圾邮件相关字段
ALTER TABLE emails ADD COLUMN IF NOT EXISTS is_spam BOOLEAN DEFAULT FALSE;
ALTER TABLE emails ADD COLUMN IF NOT EXISTS spam_score DOUBLE PRECISION DEFAULT 0;
ALTER TABLE emails ADD COLUMN IF NOT EXISTS spam_confidence DOUBLE PRECISION DEFAULT 0;
ALTER TABLE emails ADD COLUMN IF NOT EXISTS spam_reason TEXT;
ALTER TABLE emails ADD COLUMN IF NOT EXISTS spam_detected_at TIMESTAMP;
ALTER TABLE emails ADD COLUMN IF NOT EXISTS spam_detected_by VARCHAR(50);
ALTER TABLE emails ADD COLUMN IF NOT EXISTS user_marked_spam BOOLEAN DEFAULT FALSE;
ALTER TABLE emails ADD COLUMN IF NOT EXISTS user_marked_at TIMESTAMP;

-- 为垃圾邮件字段创建索引
CREATE INDEX IF NOT EXISTS idx_emails_is_spam ON emails(is_spam);
CREATE INDEX IF NOT EXISTS idx_emails_spam_score ON emails(spam_score);

-- 2. 创建 email_lists 表（白名单/黑名单）
CREATE TABLE IF NOT EXISTS email_lists (
    id BIGSERIAL PRIMARY KEY,
    user_uid VARCHAR(64) NOT NULL,
    type VARCHAR(20) NOT NULL CHECK (type IN ('whitelist', 'blacklist')),
    target VARCHAR(255) NOT NULL,
    target_type VARCHAR(20) NOT NULL CHECK (target_type IN ('email', 'domain')),
    reason TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_email_lists_user_uid ON email_lists(user_uid);
CREATE INDEX IF NOT EXISTS idx_email_lists_type ON email_lists(type);
CREATE INDEX IF NOT EXISTS idx_email_lists_target ON email_lists(target);
CREATE UNIQUE INDEX IF NOT EXISTS idx_email_lists_unique ON email_lists(user_uid, type, target);

-- 3. 创建 sender_reputations 表（发件人信誉）
CREATE TABLE IF NOT EXISTS sender_reputations (
    id BIGSERIAL PRIMARY KEY,
    email VARCHAR(255) UNIQUE NOT NULL,
    domain VARCHAR(255),
    reputation_score DOUBLE PRECISION DEFAULT 50,
    trust_level VARCHAR(20) CHECK (trust_level IN ('trusted', 'neutral', 'suspicious', 'blocked')),
    total_emails BIGINT DEFAULT 0,
    spam_count BIGINT DEFAULT 0,
    ham_count BIGINT DEFAULT 0,
    rbl_status VARCHAR(20) CHECK (rbl_status IN ('clean', 'listed', 'unknown')),
    rbl_checked_at TIMESTAMP,
    rbl_lists TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_sender_reputations_email ON sender_reputations(email);
CREATE INDEX IF NOT EXISTS idx_sender_reputations_domain ON sender_reputations(domain);
CREATE INDEX IF NOT EXISTS idx_sender_reputations_score ON sender_reputations(reputation_score);
CREATE INDEX IF NOT EXISTS idx_sender_reputations_trust_level ON sender_reputations(trust_level);

-- 4. 创建 spam_rules 表（垃圾邮件规则）
CREATE TABLE IF NOT EXISTS spam_rules (
    id BIGSERIAL PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    description TEXT,
    category VARCHAR(50) CHECK (category IN ('keyword', 'pattern', 'header', 'content', 'url', 'attachment')),
    pattern TEXT NOT NULL,
    score INTEGER DEFAULT 10,
    enabled BOOLEAN DEFAULT TRUE,
    is_builtin BOOLEAN DEFAULT FALSE,
    hit_count BIGINT DEFAULT 0,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_spam_rules_enabled ON spam_rules(enabled);
CREATE INDEX IF NOT EXISTS idx_spam_rules_category ON spam_rules(category);
CREATE INDEX IF NOT EXISTS idx_spam_rules_is_builtin ON spam_rules(is_builtin);

-- 5. 创建 bayesian_trainings 表（贝叶斯训练数据）
CREATE TABLE IF NOT EXISTS bayesian_trainings (
    id BIGSERIAL PRIMARY KEY,
    user_uid VARCHAR(64) NOT NULL,
    email_id VARCHAR(255),
    is_spam BOOLEAN NOT NULL,
    tokens TEXT NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_bayesian_trainings_user_uid ON bayesian_trainings(user_uid);
CREATE INDEX IF NOT EXISTS idx_bayesian_trainings_email_id ON bayesian_trainings(email_id);
CREATE INDEX IF NOT EXISTS idx_bayesian_trainings_is_spam ON bayesian_trainings(is_spam);

-- 6. 创建 spam_detection_logs 表（垃圾邮件检测日志）
CREATE TABLE IF NOT EXISTS spam_detection_logs (
    id BIGSERIAL PRIMARY KEY,
    email_id VARCHAR(255),
    is_spam BOOLEAN NOT NULL,
    final_score DOUBLE PRECISION NOT NULL,
    pre_filter_score DOUBLE PRECISION DEFAULT 0,
    rule_score DOUBLE PRECISION DEFAULT 0,
    bayesian_score DOUBLE PRECISION DEFAULT 0,
    detection_details TEXT,
    processing_time_ms BIGINT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_spam_detection_logs_email_id ON spam_detection_logs(email_id);
CREATE INDEX IF NOT EXISTS idx_spam_detection_logs_is_spam ON spam_detection_logs(is_spam);
CREATE INDEX IF NOT EXISTS idx_spam_detection_logs_created_at ON spam_detection_logs(created_at DESC);

-- 7. 创建系统设置表（如果不存在）
CREATE TABLE IF NOT EXISTS settings (
    id BIGSERIAL PRIMARY KEY,
    key VARCHAR(255) UNIQUE NOT NULL,
    value TEXT NOT NULL,
    category VARCHAR(50),
    description TEXT,
    is_public BOOLEAN DEFAULT FALSE,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_settings_key ON settings(key);
CREATE INDEX IF NOT EXISTS idx_settings_category ON settings(category);

-- 8. 插入垃圾邮件检测的默认系统设置
INSERT INTO settings (key, value, category, description, is_public)
VALUES 
    ('spam_detection_enabled', 'true', 'spam', '垃圾邮件检测总开关', false),
    ('spam_threshold', '60', 'spam', '垃圾邮件评分阈值', false),
    ('spam_auto_cleanup_days', '30', 'spam', '垃圾邮件自动清理天数', false)
ON CONFLICT (key) DO NOTHING;
