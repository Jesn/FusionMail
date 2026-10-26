-- 028_create_sent_emails_table.sql
-- 创建已发送邮件表
-- 用于存储通过 FusionMail 发送的邮件记录

-- ============================================
-- 创建 sent_emails 表
-- ============================================

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
    sent_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    retry_count INTEGER DEFAULT 0,
    last_retry_at TIMESTAMP,
    
    -- 发送器信息
    sender_type VARCHAR(20),
    
    -- 元数据
    size_bytes BIGINT DEFAULT 0,
    
    -- 时间戳
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP
);

-- ============================================
-- 创建索引
-- ============================================

-- 账户 UID 索引（查询某账户的发送记录）
CREATE INDEX IF NOT EXISTS idx_sent_emails_account_uid ON sent_emails(account_uid);

-- Message-ID 索引（用于查找特定邮件）
CREATE INDEX IF NOT EXISTS idx_sent_emails_message_id ON sent_emails(message_id);

-- 发送状态索引
CREATE INDEX IF NOT EXISTS idx_sent_emails_status ON sent_emails(status);

-- 发送时间索引（按时间排序）
CREATE INDEX IF NOT EXISTS idx_sent_emails_sent_at ON sent_emails(sent_at DESC);

-- 回复邮件 ID 索引
CREATE INDEX IF NOT EXISTS idx_sent_emails_reply_to ON sent_emails(reply_to_email_id);

-- 转发邮件 ID 索引
CREATE INDEX IF NOT EXISTS idx_sent_emails_forward_from ON sent_emails(forward_from_id);

-- 软删除索引
CREATE INDEX IF NOT EXISTS idx_sent_emails_deleted_at ON sent_emails(deleted_at);

-- ============================================
-- 添加注释
-- ============================================

COMMENT ON TABLE sent_emails IS '已发送邮件记录表';
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

-- ============================================
-- 回滚脚本（如需回滚，执行以下语句）
-- ============================================
-- DROP TABLE IF EXISTS sent_emails;
