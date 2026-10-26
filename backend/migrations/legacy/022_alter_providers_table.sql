-- 修改 providers 表，添加适配器关联和邮箱域名字段
-- Migration: 022_alter_providers_table
-- Description: 为 providers 表添加 default_adapter_id 和 email_domains 字段
-- Author: FusionMail Team
-- Date: 2024-12-14
-- Related: provider-account-refactor spec

-- ============================================
-- 阶段 1.3：修改 providers 表
-- 预计执行时间：< 1 分钟
-- 风险等级：低（仅添加新字段，允许 NULL，不影响现有功能）
-- ============================================

-- 添加 default_adapter_id 字段（默认适配器）
ALTER TABLE providers 
ADD COLUMN IF NOT EXISTS default_adapter_id BIGINT REFERENCES adapters(id) ON DELETE RESTRICT;

-- 添加 email_domains 字段（支持的邮箱域名列表）
ALTER TABLE providers 
ADD COLUMN IF NOT EXISTS email_domains TEXT[];

-- 创建索引
CREATE INDEX IF NOT EXISTS idx_providers_default_adapter_id ON providers(default_adapter_id);
CREATE INDEX IF NOT EXISTS idx_providers_email_domains ON providers USING GIN(email_domains);

-- 添加注释
COMMENT ON COLUMN providers.default_adapter_id IS '默认适配器 ID，关联 adapters 表';
COMMENT ON COLUMN providers.email_domains IS '支持的邮箱域名列表，用于自动匹配提供商';

-- 添加 provider_adapters 表的外键约束（现在 providers 表已存在）
ALTER TABLE provider_adapters 
DROP CONSTRAINT IF EXISTS fk_provider_adapters_provider;

ALTER TABLE provider_adapters 
ADD CONSTRAINT fk_provider_adapters_provider 
FOREIGN KEY (provider_id) REFERENCES providers(id) ON DELETE CASCADE;

-- 验证字段添加
DO $$
BEGIN
    IF EXISTS (
        SELECT 1 FROM information_schema.columns 
        WHERE table_name = 'providers' AND column_name = 'default_adapter_id'
    ) AND EXISTS (
        SELECT 1 FROM information_schema.columns 
        WHERE table_name = 'providers' AND column_name = 'email_domains'
    ) THEN
        RAISE NOTICE '✓ providers 表字段添加成功';
    ELSE
        RAISE EXCEPTION 'providers 表字段添加失败';
    END IF;
END $$;
