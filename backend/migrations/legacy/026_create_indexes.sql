-- 创建索引和设置约束
-- Migration: 026_create_indexes
-- Description: 创建所有新增索引，设置 NOT NULL 约束
-- Author: FusionMail Team
-- Date: 2024-12-14
-- Related: provider-account-refactor spec

-- ============================================
-- 阶段 5：数据校验
-- 预计执行时间：< 1 分钟
-- 风险等级：无（只读操作）
-- ============================================

-- 5.1 检查 providers 数据完整性
DO $$
DECLARE
    providers_without_adapter INTEGER;
    providers_without_domains INTEGER;
    accounts_without_provider INTEGER;
    accounts_without_adapter INTEGER;
    invalid_provider_refs INTEGER;
    invalid_adapter_refs INTEGER;
BEGIN
    -- 检查 providers
    SELECT COUNT(*) INTO providers_without_adapter 
    FROM providers WHERE default_adapter_id IS NULL;
    
    SELECT COUNT(*) INTO providers_without_domains 
    FROM providers WHERE email_domains IS NULL OR array_length(email_domains, 1) IS NULL;
    
    -- 检查 email_accounts
    SELECT COUNT(*) INTO accounts_without_provider 
    FROM email_accounts WHERE provider_id IS NULL;
    
    SELECT COUNT(*) INTO accounts_without_adapter 
    FROM email_accounts WHERE adapter_id IS NULL;
    
    -- 检查外键有效性
    SELECT COUNT(*) INTO invalid_provider_refs
    FROM email_accounts ea
    WHERE ea.provider_id IS NOT NULL 
      AND NOT EXISTS (SELECT 1 FROM providers p WHERE p.id = ea.provider_id);
    
    SELECT COUNT(*) INTO invalid_adapter_refs
    FROM email_accounts ea
    WHERE ea.adapter_id IS NOT NULL 
      AND NOT EXISTS (SELECT 1 FROM adapters a WHERE a.id = ea.adapter_id);
    
    RAISE NOTICE '========================================';
    RAISE NOTICE '数据完整性校验:';
    RAISE NOTICE '  未设置 default_adapter_id 的 Provider: %', providers_without_adapter;
    RAISE NOTICE '  未设置 email_domains 的 Provider: %', providers_without_domains;
    RAISE NOTICE '  未设置 provider_id 的账户: %', accounts_without_provider;
    RAISE NOTICE '  未设置 adapter_id 的账户: %', accounts_without_adapter;
    RAISE NOTICE '  无效 provider_id 引用: %', invalid_provider_refs;
    RAISE NOTICE '  无效 adapter_id 引用: %', invalid_adapter_refs;
    RAISE NOTICE '========================================';
    
    -- 如果有数据问题，抛出警告但不阻止迁移
    IF accounts_without_provider > 0 OR accounts_without_adapter > 0 THEN
        RAISE WARNING '⚠ 存在未完成迁移的账户数据，请先执行 025_migrate_account_data.sql';
    END IF;
    
    IF invalid_provider_refs > 0 OR invalid_adapter_refs > 0 THEN
        RAISE EXCEPTION '❌ 存在无效的外键引用，请先修复数据';
    END IF;
END $$;

-- ============================================
-- 阶段 6：设置约束和索引
-- 预计执行时间：< 1 分钟
-- 风险等级：中（如果数据不完整会失败）
-- 前置条件：阶段 5 校验全部通过
-- ============================================

-- 6.1 确保所有索引存在
-- 注意：adapters.name 的唯一索引已由 UNIQUE 约束自动创建 (adapters_name_key)
-- 不需要再手动创建 idx_adapters_name，避免重复索引
CREATE INDEX IF NOT EXISTS idx_adapters_is_enabled ON adapters(is_enabled);
CREATE INDEX IF NOT EXISTS idx_providers_default_adapter_id ON providers(default_adapter_id);
CREATE INDEX IF NOT EXISTS idx_providers_email_domains ON providers USING GIN(email_domains);
CREATE INDEX IF NOT EXISTS idx_provider_adapters_provider_id ON provider_adapters(provider_id);
CREATE INDEX IF NOT EXISTS idx_provider_adapters_adapter_id ON provider_adapters(adapter_id);
CREATE INDEX IF NOT EXISTS idx_provider_adapters_priority ON provider_adapters(priority);
CREATE INDEX IF NOT EXISTS idx_email_accounts_provider_id ON email_accounts(provider_id);
CREATE INDEX IF NOT EXISTS idx_email_accounts_adapter_id ON email_accounts(adapter_id);
CREATE INDEX IF NOT EXISTS idx_email_accounts_status ON email_accounts(status);

-- 6.2 设置 NOT NULL 约束（仅在数据完整时执行）
-- 注意：这些约束需要在确认所有数据都已迁移后才能执行
-- 如果有未迁移的数据，这些语句会失败

-- 检查是否可以设置 NOT NULL 约束
DO $$
DECLARE
    can_set_not_null BOOLEAN := TRUE;
    accounts_without_provider INTEGER;
    accounts_without_adapter INTEGER;
BEGIN
    SELECT COUNT(*) INTO accounts_without_provider 
    FROM email_accounts WHERE provider_id IS NULL;
    
    SELECT COUNT(*) INTO accounts_without_adapter 
    FROM email_accounts WHERE adapter_id IS NULL;
    
    IF accounts_without_provider > 0 OR accounts_without_adapter > 0 THEN
        can_set_not_null := FALSE;
        RAISE NOTICE '⚠ 跳过 NOT NULL 约束设置：存在未迁移的账户数据';
        RAISE NOTICE '  请先完成数据迁移，然后手动执行以下 SQL:';
        RAISE NOTICE '  ALTER TABLE email_accounts ALTER COLUMN provider_id SET NOT NULL;';
        RAISE NOTICE '  ALTER TABLE email_accounts ALTER COLUMN adapter_id SET NOT NULL;';
    ELSE
        -- 设置 NOT NULL 约束
        ALTER TABLE email_accounts ALTER COLUMN provider_id SET NOT NULL;
        ALTER TABLE email_accounts ALTER COLUMN adapter_id SET NOT NULL;
        RAISE NOTICE '✓ email_accounts 表 NOT NULL 约束设置成功';
    END IF;
END $$;

-- 6.3 验证索引创建
SELECT 
    schemaname,
    tablename,
    indexname,
    indexdef
FROM pg_indexes 
WHERE tablename IN ('providers', 'email_accounts', 'provider_adapters', 'adapters')
ORDER BY tablename, indexname;

-- 6.4 显示最终表结构
RAISE NOTICE '========================================';
RAISE NOTICE '迁移完成，表结构如下:';
RAISE NOTICE '========================================';

-- adapters 表结构
SELECT column_name, data_type, is_nullable, column_default
FROM information_schema.columns
WHERE table_name = 'adapters'
ORDER BY ordinal_position;

-- providers 新增字段
SELECT column_name, data_type, is_nullable, column_default
FROM information_schema.columns
WHERE table_name = 'providers' 
  AND column_name IN ('default_adapter_id', 'email_domains')
ORDER BY ordinal_position;

-- email_accounts 新增字段
SELECT column_name, data_type, is_nullable, column_default
FROM information_schema.columns
WHERE table_name = 'email_accounts' 
  AND column_name IN ('provider_id', 'adapter_id')
ORDER BY ordinal_position;
