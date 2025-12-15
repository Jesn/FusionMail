-- 填充 email_accounts 的 provider_id 和 adapter_id
-- Migration: 025_migrate_account_data
-- Description: 根据现有 provider 字段值填充 provider_id 和 adapter_id
-- Author: FusionMail Team
-- Date: 2024-12-14
-- Related: provider-account-refactor spec

-- ============================================
-- 阶段 4：填充 EmailAccounts 数据（关键步骤）
-- 预计执行时间：取决于数据量，可能需要几分钟
-- 风险等级：中（核心数据迁移）
-- ============================================

-- 4.1 方法1：通过 provider 字段名称精确匹配
UPDATE email_accounts ea
SET provider_id = p.id
FROM providers p
WHERE LOWER(TRIM(ea.provider)) = LOWER(TRIM(p.name))
  AND ea.provider_id IS NULL;

-- 4.2 方法2：通过邮箱域名匹配（处理名称不完全匹配的情况）
UPDATE email_accounts ea
SET provider_id = p.id
FROM providers p
WHERE ea.provider_id IS NULL
  AND p.email_domains IS NOT NULL
  AND array_length(p.email_domains, 1) > 0
  AND LOWER(SPLIT_PART(ea.email, '@', 2)) = ANY(
      SELECT LOWER(unnest(p.email_domains))
  );

-- 4.3 方法3：模糊匹配（处理 provider 字段值与 name 略有差异的情况）
UPDATE email_accounts ea
SET provider_id = p.id
FROM providers p
WHERE ea.provider_id IS NULL
  AND (
      LOWER(ea.provider) LIKE '%' || LOWER(p.name) || '%'
      OR LOWER(p.name) LIKE '%' || LOWER(ea.provider) || '%'
  );

-- 4.4 方法4：未匹配的账户使用 generic 提供商
UPDATE email_accounts ea
SET provider_id = (SELECT id FROM providers WHERE LOWER(name) = 'generic' LIMIT 1)
WHERE ea.provider_id IS NULL;

-- 4.5 设置 adapter_id 为 provider 的默认适配器
UPDATE email_accounts ea
SET adapter_id = p.default_adapter_id
FROM providers p
WHERE ea.provider_id = p.id
  AND ea.adapter_id IS NULL;

-- 4.6 检查未匹配的账户（需要手动处理）
DO $$
DECLARE
    unmatched_provider_count INTEGER;
    unmatched_adapter_count INTEGER;
    total_accounts INTEGER;
    migrated_accounts INTEGER;
BEGIN
    SELECT COUNT(*) INTO total_accounts FROM email_accounts;
    
    SELECT COUNT(*) INTO unmatched_provider_count 
    FROM email_accounts WHERE provider_id IS NULL;
    
    SELECT COUNT(*) INTO unmatched_adapter_count 
    FROM email_accounts WHERE adapter_id IS NULL;
    
    SELECT COUNT(*) INTO migrated_accounts 
    FROM email_accounts WHERE provider_id IS NOT NULL AND adapter_id IS NOT NULL;
    
    RAISE NOTICE '========================================';
    RAISE NOTICE '账户数据迁移统计:';
    RAISE NOTICE '  总账户数: %', total_accounts;
    RAISE NOTICE '  已迁移: %', migrated_accounts;
    RAISE NOTICE '  未匹配 provider_id: %', unmatched_provider_count;
    RAISE NOTICE '  未匹配 adapter_id: %', unmatched_adapter_count;
    RAISE NOTICE '========================================';
    
    IF unmatched_provider_count > 0 THEN
        RAISE WARNING '⚠ 有 % 个账户未匹配到 provider_id，请手动处理', unmatched_provider_count;
    END IF;
    
    IF unmatched_adapter_count > 0 THEN
        RAISE WARNING '⚠ 有 % 个账户未匹配到 adapter_id，请手动处理', unmatched_adapter_count;
    END IF;
END $$;

-- 显示未匹配的账户（如果有）
SELECT id, uid, email, provider, provider_id, adapter_id
FROM email_accounts 
WHERE provider_id IS NULL OR adapter_id IS NULL
ORDER BY id;

-- 显示迁移结果统计
SELECT 
    p.name as provider_name,
    a.name as adapter_name,
    COUNT(ea.id) as account_count
FROM email_accounts ea
LEFT JOIN providers p ON ea.provider_id = p.id
LEFT JOIN adapters a ON ea.adapter_id = a.id
GROUP BY p.name, a.name
ORDER BY account_count DESC;
