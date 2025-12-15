-- 修改 email_accounts 表，添加外键关联字段
-- Migration: 023_alter_email_accounts_table
-- Description: 为 email_accounts 表添加 provider_id 和 adapter_id 外键字段
-- Author: FusionMail Team
-- Date: 2024-12-14
-- Related: provider-account-refactor spec

-- ============================================
-- 阶段 1.4：修改 email_accounts 表
-- 预计执行时间：< 1 分钟
-- 风险等级：低（仅添加新字段，允许 NULL，不影响现有功能）
-- ============================================

-- 添加 provider_id 字段（关联 providers 表）
ALTER TABLE email_accounts 
ADD COLUMN IF NOT EXISTS provider_id BIGINT REFERENCES providers(id) ON DELETE RESTRICT;

-- 添加 adapter_id 字段（关联 adapters 表，用户选择的适配器）
ALTER TABLE email_accounts 
ADD COLUMN IF NOT EXISTS adapter_id BIGINT REFERENCES adapters(id) ON DELETE RESTRICT;

-- 创建索引
CREATE INDEX IF NOT EXISTS idx_email_accounts_provider_id ON email_accounts(provider_id);
CREATE INDEX IF NOT EXISTS idx_email_accounts_adapter_id ON email_accounts(adapter_id);

-- 添加注释
COMMENT ON COLUMN email_accounts.provider_id IS '关联的提供商 ID，外键关联 providers 表';
COMMENT ON COLUMN email_accounts.adapter_id IS '用户选择的适配器 ID，外键关联 adapters 表';

-- 验证字段添加
DO $$
BEGIN
    IF EXISTS (
        SELECT 1 FROM information_schema.columns 
        WHERE table_name = 'email_accounts' AND column_name = 'provider_id'
    ) AND EXISTS (
        SELECT 1 FROM information_schema.columns 
        WHERE table_name = 'email_accounts' AND column_name = 'adapter_id'
    ) THEN
        RAISE NOTICE '✓ email_accounts 表字段添加成功';
    ELSE
        RAISE EXCEPTION 'email_accounts 表字段添加失败';
    END IF;
END $$;
