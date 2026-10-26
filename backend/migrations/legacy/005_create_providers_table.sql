-- 创建邮箱提供商配置表
-- Migration: 005_create_providers_table
-- Description: 将邮箱提供商配置从代码迁移到数据库
-- Author: FusionMail Team
-- Date: 2025-11-21

-- 创建邮箱提供商配置表
CREATE TABLE IF NOT EXISTS providers (
    id BIGSERIAL PRIMARY KEY,
    name VARCHAR(50) UNIQUE NOT NULL,           -- 提供商唯一标识 (gmail/outlook/qq/163/icloud/generic)
    display_name VARCHAR(100) NOT NULL,         -- 显示名称 (Gmail / Outlook / Hotmail / QQ 邮箱等)
    supported_protocols TEXT NOT NULL,          -- 支持的协议 (JSON数组: ["oauth2","imap","pop3"])
    recommended_protocol VARCHAR(20) NOT NULL,  -- 推荐协议 (oauth2/imap/pop3)
    requires_oauth BOOLEAN DEFAULT FALSE,       -- 是否强制OAuth认证

    -- 服务器配置
    imap_host VARCHAR(255),                     -- IMAP服务器地址
    imap_port INTEGER DEFAULT 993,              -- IMAP端口
    pop3_host VARCHAR(255),                     -- POP3服务器地址
    pop3_port INTEGER DEFAULT 995,              -- POP3端口
    smtp_host VARCHAR(255),                     -- SMTP服务器地址（预留）
    smtp_port INTEGER DEFAULT 587,              -- SMTP端口（预留）

    -- 管理字段
    enabled BOOLEAN DEFAULT TRUE,               -- 是否启用
    sort_order INTEGER DEFAULT 0,               -- 排序顺序（越小越靠前）
    description TEXT,                           -- 描述信息
    metadata TEXT,                              -- JSON格式的额外配置（如特殊设置）

    -- 时间戳
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- 创建索引
CREATE INDEX IF NOT EXISTS idx_providers_name ON providers(name);
CREATE INDEX IF NOT EXISTS idx_providers_enabled ON providers(enabled);
CREATE INDEX IF NOT EXISTS idx_providers_sort_order ON providers(sort_order);

-- 添加注释
COMMENT ON TABLE providers IS '邮箱提供商配置表';
COMMENT ON COLUMN providers.name IS '提供商唯一标识符';
COMMENT ON COLUMN providers.display_name IS '用户界面显示的提供商名称';
COMMENT ON COLUMN providers.supported_protocols IS '支持的协议列表，JSON数组格式';
COMMENT ON COLUMN providers.recommended_protocol IS '推荐的默认协议';
COMMENT ON COLUMN providers.requires_oauth IS '是否强制使用OAuth2认证';
COMMENT ON COLUMN providers.enabled IS '是否启用此提供商';
COMMENT ON COLUMN providers.sort_order IS '显示排序，数值越小越靠前';
COMMENT ON COLUMN providers.metadata IS 'JSON格式的额外配置信息';

-- 初始化数据：包含现有的所有邮箱提供商配置
INSERT INTO providers (name, display_name, supported_protocols, recommended_protocol,
                       requires_oauth, imap_host, imap_port, pop3_host, pop3_port,
                       sort_order, description) VALUES

-- Gmail (支持 OAuth2 和 IMAP)
('gmail', 'Gmail', '["oauth2","imap"]', 'oauth2', true,
 'imap.gmail.com', 993, '', 0, 1, 'Google Gmail 邮箱服务'),

-- Outlook (支持 OAuth2 和 IMAP)
('outlook', 'Outlook / Hotmail', '["oauth2","imap"]', 'oauth2', true,
 'outlook.office365.com', 993, '', 0, 2, 'Microsoft Outlook / Hotmail 邮箱服务'),

-- iCloud (仅支持 IMAP)
('icloud', 'iCloud Mail', '["imap"]', 'imap', false,
 'imap.mail.me.com', 993, '', 0, 3, 'Apple iCloud 邮箱服务'),

-- QQ邮箱 (支持 IMAP 和 POP3)
('qq', 'QQ 邮箱', '["imap","pop3"]', 'imap', false,
 'imap.qq.com', 993, 'pop.qq.com', 995, 4, '腾讯 QQ 邮箱服务'),

-- 163邮箱 (支持 IMAP 和 POP3)
('163', '163 邮箱', '["imap","pop3"]', 'imap', false,
 'imap.163.com', 993, 'pop.163.com', 995, 5, '网易 163 邮箱服务'),

-- 通用邮箱 (支持 IMAP 和 POP3)
('generic', '通用邮箱 (IMAP/POP3)', '["imap","pop3"]', 'imap', false,
 '', 993, '', 995, 99, '支持标准 IMAP/POP3 协议的通用邮箱')

ON CONFLICT (name) DO NOTHING; -- 如果数据已存在则跳过

-- 更新表的统计信息
ANALYZE providers;
