-- 创建适配器表
-- Migration: 020_create_adapters_table
-- Description: 创建 adapters 表管理邮箱协议适配器元数据
-- Author: FusionMail Team
-- Date: 2024-12-14
-- Related: provider-account-refactor spec

-- ============================================
-- 阶段 1.1：创建 adapters 表
-- 预计执行时间：< 1 分钟
-- 风险等级：低（仅添加新表，不影响现有功能）
-- ============================================

-- 创建适配器表
CREATE TABLE IF NOT EXISTS adapters (
    id BIGSERIAL PRIMARY KEY,
    name VARCHAR(50) UNIQUE NOT NULL,           -- 适配器唯一标识 (gmail/graph/imap)
    display_name VARCHAR(100) NOT NULL,         -- 显示名称 (Gmail API / Microsoft Graph / IMAP 协议)
    auth_type VARCHAR(20) NOT NULL,             -- 认证类型 (oauth2/password)
    description TEXT,                           -- 描述信息
    is_enabled BOOLEAN DEFAULT TRUE,            -- 是否启用
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- 创建索引
-- 注意：name 字段的唯一索引已由 UNIQUE 约束自动创建 (adapters_name_key)
-- 不需要再手动创建 idx_adapters_name，避免重复索引
CREATE INDEX IF NOT EXISTS idx_adapters_is_enabled ON adapters(is_enabled);

-- 添加注释
COMMENT ON TABLE adapters IS '邮箱协议适配器表，管理不同的邮箱协议实现';
COMMENT ON COLUMN adapters.name IS '适配器唯一标识符，用于代码中选择适配器实现';
COMMENT ON COLUMN adapters.display_name IS '用户界面显示的适配器名称';
COMMENT ON COLUMN adapters.auth_type IS '认证类型：oauth2（OAuth2授权）或 password（密码/授权码）';
COMMENT ON COLUMN adapters.description IS '适配器的详细描述';
COMMENT ON COLUMN adapters.is_enabled IS '是否启用此适配器';

-- 插入预置适配器数据
INSERT INTO adapters (name, display_name, auth_type, description, is_enabled) VALUES
-- Gmail API 适配器（OAuth2）
('gmail', 'Gmail API', 'oauth2', 
 'Google Gmail API 适配器，使用 OAuth2 授权，支持完整的 Gmail 功能', 
 true),

-- Microsoft Graph 适配器（OAuth2）
('graph', 'Microsoft Graph', 'oauth2', 
 'Microsoft Graph API 适配器，支持 Outlook/Hotmail/Live 邮箱，使用 OAuth2 授权', 
 true),

-- 通用 IMAP 适配器（密码/授权码）
('imap', 'IMAP 协议', 'password', 
 '通用 IMAP 协议适配器，使用密码或授权码认证，支持所有标准 IMAP 邮箱服务', 
 true)

ON CONFLICT (name) DO UPDATE SET
    display_name = EXCLUDED.display_name,
    auth_type = EXCLUDED.auth_type,
    description = EXCLUDED.description,
    updated_at = CURRENT_TIMESTAMP;

-- 验证数据插入
DO $$
DECLARE
    adapter_count INTEGER;
BEGIN
    SELECT COUNT(*) INTO adapter_count FROM adapters;
    IF adapter_count < 3 THEN
        RAISE EXCEPTION '适配器数据插入失败，期望至少 3 条记录，实际 % 条', adapter_count;
    END IF;
    RAISE NOTICE '✓ adapters 表创建成功，共 % 条记录', adapter_count;
END $$;

-- 更新表的统计信息
ANALYZE adapters;
