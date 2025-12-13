-- 添加邮件发送功能支持
-- Migration: 019_add_email_sending_support
-- Description: 创建 sent_emails 表，为 email_accounts 表添加 SMTP 配置字段
-- Requirements: 1.4, 3.1, 7.1, 7.2

-- =====================================================
-- Part 1: 为 email_accounts 表添加 SMTP 配置字段
-- Requirements: 3.1
-- =====================================================

-- 添加 SMTP 服务器地址
ALTER TABLE email_accounts 
    ADD COLUMN IF NOT EXISTS smtp_host VARCHAR(255);

-- 添加 SMTP 端口
ALTER TABLE email_accounts 
    ADD COLUMN IF NOT EXISTS smtp_port INTEGER DEFAULT 0;

-- 添加 SMTP 加密方式
ALTER TABLE email_accounts 
    ADD COLUMN IF NOT EXISTS smtp_encryption VARCHAR(20);

-- 添加 SMTP 用户名
ALTER TABLE email_accounts 
    ADD COLUMN IF NOT EXISTS smtp_username VARCHAR(255);

-- 添加 SMTP 密码（AES-256 加密存储）
ALTER TABLE email_accounts 
    ADD COLUMN IF NOT EXISTS encrypted_smtp_password TEXT;

-- 添加 SMTP 启用标志
ALTER TABLE email_accounts 
    ADD COLUMN IF NOT EXISTS smtp_enabled BOOLEAN DEFAULT FALSE;

-- 添加字段注释
COMMENT ON COLUMN email_accounts.smtp_host IS 'SMTP 服务器地址';
COMMENT ON COLUMN email_accounts.smtp_port IS 'SMTP 端口';
COMMENT ON COLUMN email_accounts.smtp_encryption IS 'SMTP 加密方式 (none/tls/starttls)';
COMMENT ON COLUMN email_accounts.smtp_username IS 'SMTP 用户名（通常是邮箱地址）';
COMMENT ON COLUMN email_accounts.encrypted_smtp_password IS 'SMTP 密码（AES-256 加密存储）';
COMMENT ON COLUMN email_accounts.smtp_enabled IS '是否启用 SMTP 发送';

-- =====================================================
-- Part 2: 创建 sent_emails 表
-- Requirements: 1.4, 7.1, 7.2
-- =====================================================

CREATE TABLE IF NOT EXISTS sent_emails (
    id BIGSERIAL PRIMARY KEY,
    account_uid VARCHAR(64) NOT NULL,
    provider_msg_id VARCHAR(255),
    message_id VARCHAR(255),
    
    -- 邮件内容
    subject TEXT,
    from_address VARCHAR(255),
    from_name VARCHAR(255),
    to_addresses TEXT,
    cc_addresses TEXT,
    bcc_addresses TEXT,
    text_body TEXT,
    html_body TEXT,
    
    -- 附件信息
    has_attachments BOOLEAN DEFAULT FALSE,
    attachment_count INTEGER DEFAULT 0,
    attachment_info TEXT,
    
    -- 关联信息（回复/转发）
    reply_to_email_id BIGINT,
    forward_from_id BIGINT,
    in_reply_to VARCHAR(255),
    "references" TEXT,
    
    -- 发送状态
    status VARCHAR(20) DEFAULT 'sent',
    error_message TEXT,
    sent_at TIMESTAMP WITH TIME ZONE NOT NULL,
    retry_count INTEGER DEFAULT 0,
    last_retry_at TIMESTAMP WITH TIME ZONE,
    
    -- 发送器信息
    sender_type VARCHAR(20),
    
    -- 元数据
    size_bytes BIGINT DEFAULT 0,
    
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP WITH TIME ZONE
);

-- 添加索引
CREATE INDEX IF NOT EXISTS idx_sent_emails_account_uid 
    ON sent_emails(account_uid);

CREATE INDEX IF NOT EXISTS idx_sent_emails_message_id 
    ON sent_emails(message_id);

CREATE INDEX IF NOT EXISTS idx_sent_emails_sent_at 
    ON sent_emails(sent_at DESC);

CREATE INDEX IF NOT EXISTS idx_sent_emails_status 
    ON sent_emails(status);

CREATE INDEX IF NOT EXISTS idx_sent_emails_reply_to_email_id 
    ON sent_emails(reply_to_email_id);

CREATE INDEX IF NOT EXISTS idx_sent_emails_forward_from_id 
    ON sent_emails(forward_from_id);

CREATE INDEX IF NOT EXISTS idx_sent_emails_deleted_at 
    ON sent_emails(deleted_at);

-- 添加表注释
COMMENT ON TABLE sent_emails IS '已发送邮件表';
COMMENT ON COLUMN sent_emails.id IS '主键ID';
COMMENT ON COLUMN sent_emails.account_uid IS '发送账户 UID';
COMMENT ON COLUMN sent_emails.provider_msg_id IS '服务商返回的消息 ID';
COMMENT ON COLUMN sent_emails.message_id IS '邮件 Message-ID（RFC 2822）';
COMMENT ON COLUMN sent_emails.subject IS '邮件主题';
COMMENT ON COLUMN sent_emails.from_address IS '发件人地址';
COMMENT ON COLUMN sent_emails.from_name IS '发件人名称';
COMMENT ON COLUMN sent_emails.to_addresses IS '收件人列表（JSON 数组）';
COMMENT ON COLUMN sent_emails.cc_addresses IS '抄送列表（JSON 数组）';
COMMENT ON COLUMN sent_emails.bcc_addresses IS '密送列表（JSON 数组）';
COMMENT ON COLUMN sent_emails.text_body IS '纯文本正文';
COMMENT ON COLUMN sent_emails.html_body IS 'HTML 正文';
COMMENT ON COLUMN sent_emails.has_attachments IS '是否有附件';
COMMENT ON COLUMN sent_emails.attachment_count IS '附件数量';
COMMENT ON COLUMN sent_emails.attachment_info IS '附件信息（JSON 数组）';
COMMENT ON COLUMN sent_emails.reply_to_email_id IS '回复的邮件 ID（本地 Email 表 ID）';
COMMENT ON COLUMN sent_emails.forward_from_id IS '转发的原邮件 ID（本地 Email 表 ID）';
COMMENT ON COLUMN sent_emails.in_reply_to IS 'In-Reply-To 邮件头';
COMMENT ON COLUMN sent_emails."references" IS 'References 邮件头';
COMMENT ON COLUMN sent_emails.status IS '发送状态：sent/failed';
COMMENT ON COLUMN sent_emails.error_message IS '错误信息（如果失败）';
COMMENT ON COLUMN sent_emails.sent_at IS '发送时间';
COMMENT ON COLUMN sent_emails.retry_count IS '重试次数';
COMMENT ON COLUMN sent_emails.last_retry_at IS '最后重试时间';
COMMENT ON COLUMN sent_emails.sender_type IS '发送器类型：gmail_api/graph_api/smtp';
COMMENT ON COLUMN sent_emails.size_bytes IS '邮件大小（字节）';
COMMENT ON COLUMN sent_emails.created_at IS '创建时间';
COMMENT ON COLUMN sent_emails.updated_at IS '更新时间';
COMMENT ON COLUMN sent_emails.deleted_at IS '软删除时间';
