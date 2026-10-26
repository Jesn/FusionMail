-- 添加 139 邮箱（中国移动）提供商配置
-- Migration: 014_add_139_provider
-- Description: 添加 139 邮箱到提供商列表
-- Author: FusionMail Team
-- Date: 2025-11-29

-- 添加 139 邮箱配置
INSERT INTO providers (
    name, 
    display_name, 
    provider_type,
    supported_protocols, 
    recommended_protocol,
    requires_o_auth, 
    imap_host, 
    imap_port, 
    pop3_host, 
    pop3_port,
    smtp_host,
    smtp_port,
    sort_order, 
    description, 
    enabled
) VALUES (
    '139', 
    '139 邮箱 (中国移动)', 
    1,  -- Generic 类型
    '["imap","pop3"]', 
    'imap', 
    false,
    'imap.139.com', 
    993, 
    'pop.139.com', 
    995,
    'smtp.139.com',
    465,
    6, 
    '中国移动 139 邮箱服务，需要使用授权码登录', 
    true
)
ON CONFLICT (name) DO UPDATE SET
    display_name = EXCLUDED.display_name,
    imap_host = EXCLUDED.imap_host,
    imap_port = EXCLUDED.imap_port,
    pop3_host = EXCLUDED.pop3_host,
    pop3_port = EXCLUDED.pop3_port,
    smtp_host = EXCLUDED.smtp_host,
    smtp_port = EXCLUDED.smtp_port,
    description = EXCLUDED.description,
    updated_at = CURRENT_TIMESTAMP;

-- 添加 126 邮箱（网易）
INSERT INTO providers (
    name, 
    display_name, 
    provider_type,
    supported_protocols, 
    recommended_protocol,
    requires_o_auth, 
    imap_host, 
    imap_port, 
    pop3_host, 
    pop3_port,
    smtp_host,
    smtp_port,
    sort_order, 
    description, 
    enabled
) VALUES (
    '126', 
    '126 邮箱 (网易)', 
    1,
    '["imap","pop3"]', 
    'imap', 
    false,
    'imap.126.com', 
    993, 
    'pop.126.com', 
    995,
    'smtp.126.com',
    465,
    7, 
    '网易 126 邮箱服务，需要使用授权码登录', 
    true
)
ON CONFLICT (name) DO UPDATE SET
    display_name = EXCLUDED.display_name,
    imap_host = EXCLUDED.imap_host,
    imap_port = EXCLUDED.imap_port,
    pop3_host = EXCLUDED.pop3_host,
    pop3_port = EXCLUDED.pop3_port,
    smtp_host = EXCLUDED.smtp_host,
    smtp_port = EXCLUDED.smtp_port,
    description = EXCLUDED.description,
    updated_at = CURRENT_TIMESTAMP;

-- 添加 189 邮箱（中国电信）
INSERT INTO providers (
    name, 
    display_name, 
    provider_type,
    supported_protocols, 
    recommended_protocol,
    requires_o_auth, 
    imap_host, 
    imap_port, 
    pop3_host, 
    pop3_port,
    smtp_host,
    smtp_port,
    sort_order, 
    description, 
    enabled
) VALUES (
    '189', 
    '189 邮箱 (中国电信)', 
    1,
    '["imap","pop3"]', 
    'imap', 
    false,
    'imap.189.cn', 
    993, 
    'pop.189.cn', 
    995,
    'smtp.189.cn',
    465,
    8, 
    '中国电信 189 邮箱服务', 
    true
)
ON CONFLICT (name) DO UPDATE SET
    display_name = EXCLUDED.display_name,
    imap_host = EXCLUDED.imap_host,
    imap_port = EXCLUDED.imap_port,
    pop3_host = EXCLUDED.pop3_host,
    pop3_port = EXCLUDED.pop3_port,
    smtp_host = EXCLUDED.smtp_host,
    smtp_port = EXCLUDED.smtp_port,
    description = EXCLUDED.description,
    updated_at = CURRENT_TIMESTAMP;

-- 修复已存在的 139 邮箱记录（如果名称不对）
UPDATE providers SET 
    name = '139',
    display_name = '139 邮箱 (中国移动)',
    imap_host = 'imap.139.com',
    imap_port = 993,
    pop3_host = 'pop.139.com',
    pop3_port = 995,
    smtp_host = 'smtp.139.com',
    smtp_port = 465,
    description = '中国移动 139 邮箱服务，需要使用授权码登录',
    updated_at = CURRENT_TIMESTAMP
WHERE name = '139 邮箱';

-- 更新统计信息
ANALYZE providers;
