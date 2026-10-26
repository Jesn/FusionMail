-- 填充 providers 数据和 provider_adapters 关联关系
-- Migration: 024_migrate_provider_data
-- Description: 为现有 providers 记录填充 default_adapter_id 和 email_domains
-- Author: FusionMail Team
-- Date: 2024-12-14
-- Related: provider-account-refactor spec

-- ============================================
-- 阶段 2：填充 Providers 数据
-- 预计执行时间：< 1 分钟
-- 风险等级：低（仅填充新字段，不修改现有字段）
-- ============================================

-- 2.1 填充 Gmail Provider
UPDATE providers SET 
    default_adapter_id = (SELECT id FROM adapters WHERE name = 'gmail'),
    email_domains = ARRAY['gmail.com', 'googlemail.com']
WHERE LOWER(name) = 'gmail'
  AND (
      default_adapter_id IS NULL
      OR default_adapter_id = 0
      OR NOT EXISTS (SELECT 1 FROM adapters WHERE adapters.id = providers.default_adapter_id)
  );

-- 2.2 填充 Outlook/Hotmail Provider
UPDATE providers SET 
    default_adapter_id = (SELECT id FROM adapters WHERE name = 'graph'),
    email_domains = ARRAY['outlook.com', 'hotmail.com', 'live.com', 'msn.com', 'outlook.cn']
WHERE LOWER(name) = 'outlook'
  AND (
      default_adapter_id IS NULL
      OR default_adapter_id = 0
      OR NOT EXISTS (SELECT 1 FROM adapters WHERE adapters.id = providers.default_adapter_id)
  );

-- 2.3 填充 QQ 邮箱 Provider
UPDATE providers SET 
    default_adapter_id = (SELECT id FROM adapters WHERE name = 'imap'),
    email_domains = ARRAY['qq.com', 'foxmail.com', 'vip.qq.com']
WHERE LOWER(name) = 'qq'
  AND (
      default_adapter_id IS NULL
      OR default_adapter_id = 0
      OR NOT EXISTS (SELECT 1 FROM adapters WHERE adapters.id = providers.default_adapter_id)
  );

-- 2.4 填充 163/网易邮箱 Provider
UPDATE providers SET 
    default_adapter_id = (SELECT id FROM adapters WHERE name = 'imap'),
    email_domains = ARRAY['163.com', '126.com', 'yeah.net', 'vip.163.com', 'vip.126.com']
WHERE LOWER(name) = '163'
  AND (
      default_adapter_id IS NULL
      OR default_adapter_id = 0
      OR NOT EXISTS (SELECT 1 FROM adapters WHERE adapters.id = providers.default_adapter_id)
  );

-- 2.5 填充 iCloud Provider
UPDATE providers SET 
    default_adapter_id = (SELECT id FROM adapters WHERE name = 'imap'),
    email_domains = ARRAY['icloud.com', 'me.com', 'mac.com']
WHERE LOWER(name) = 'icloud'
  AND (
      default_adapter_id IS NULL
      OR default_adapter_id = 0
      OR NOT EXISTS (SELECT 1 FROM adapters WHERE adapters.id = providers.default_adapter_id)
  );

-- 2.6 填充 139 邮箱 Provider
UPDATE providers SET 
    default_adapter_id = (SELECT id FROM adapters WHERE name = 'imap'),
    email_domains = ARRAY['139.com', '189.cn', '10086.cn']
WHERE LOWER(name) = '139'
  AND (
      default_adapter_id IS NULL
      OR default_adapter_id = 0
      OR NOT EXISTS (SELECT 1 FROM adapters WHERE adapters.id = providers.default_adapter_id)
  );

-- 2.7 填充 generic 通用邮箱 Provider
UPDATE providers SET 
    default_adapter_id = (SELECT id FROM adapters WHERE name = 'imap'),
    email_domains = ARRAY[]::TEXT[]  -- 通用邮箱不预设域名
WHERE LOWER(name) = 'generic'
  AND (
      default_adapter_id IS NULL
      OR default_adapter_id = 0
      OR NOT EXISTS (SELECT 1 FROM adapters WHERE adapters.id = providers.default_adapter_id)
  );

-- 2.8 确保 WebAPI Adapter 存在
INSERT INTO adapters (name, display_name, auth_type, description, is_enabled) VALUES
('webapi', 'Web API', 'token', '通用 Web API 邮箱适配器', true)
ON CONFLICT (name) DO UPDATE SET
    display_name = COALESCE(NULLIF(adapters.display_name, ''), EXCLUDED.display_name),
    auth_type = COALESCE(NULLIF(adapters.auth_type, ''), EXCLUDED.auth_type),
    description = COALESCE(NULLIF(adapters.description, ''), EXCLUDED.description),
    updated_at = CURRENT_TIMESTAMP;

-- 2.9 填充 WebAPI Provider
UPDATE providers SET
    default_adapter_id = (SELECT id FROM adapters WHERE name = 'webapi')
WHERE (name LIKE 'webapi_%' OR name IN ('cloudflare_temp_email', 'cloud_mail'))
  AND (
      default_adapter_id IS NULL
      OR default_adapter_id = 0
      OR default_adapter_id = (SELECT id FROM adapters WHERE name = 'imap')
      OR NOT EXISTS (SELECT 1 FROM adapters WHERE adapters.id = providers.default_adapter_id)
  );

-- 2.10 其他提供商默认使用 IMAP
UPDATE providers SET
    default_adapter_id = (SELECT id FROM adapters WHERE name = 'imap')
WHERE (default_adapter_id IS NULL
   OR default_adapter_id = 0
   OR NOT EXISTS (SELECT 1 FROM adapters WHERE adapters.id = providers.default_adapter_id))
  AND NOT (name LIKE 'webapi_%' OR name IN ('cloudflare_temp_email', 'cloud_mail'));

-- ============================================
-- 阶段 3：填充 Provider-Adapter 关联关系
-- ============================================

-- 3.1 Gmail 支持 gmail（OAuth2，默认）和 imap（备选）
INSERT INTO provider_adapters (provider_id, adapter_id, priority)
SELECT p.id, a.id, 0 
FROM providers p, adapters a 
WHERE LOWER(p.name) = 'gmail' AND a.name = 'gmail'
ON CONFLICT (provider_id, adapter_id) DO NOTHING;

INSERT INTO provider_adapters (provider_id, adapter_id, priority)
SELECT p.id, a.id, 1 
FROM providers p, adapters a 
WHERE LOWER(p.name) = 'gmail' AND a.name = 'imap'
ON CONFLICT (provider_id, adapter_id) DO NOTHING;

-- 3.2 Outlook 支持 graph（OAuth2，默认）和 imap（备选）
INSERT INTO provider_adapters (provider_id, adapter_id, priority)
SELECT p.id, a.id, 0 
FROM providers p, adapters a 
WHERE LOWER(p.name) = 'outlook' AND a.name = 'graph'
ON CONFLICT (provider_id, adapter_id) DO NOTHING;

INSERT INTO provider_adapters (provider_id, adapter_id, priority)
SELECT p.id, a.id, 1 
FROM providers p, adapters a 
WHERE LOWER(p.name) = 'outlook' AND a.name = 'imap'
ON CONFLICT (provider_id, adapter_id) DO NOTHING;

-- 3.3 WebAPI 提供商只支持 WebAPI
DELETE FROM provider_adapters pa
USING providers p, adapters a
WHERE pa.provider_id = p.id
  AND pa.adapter_id = a.id
  AND (p.name LIKE 'webapi_%' OR p.name IN ('cloudflare_temp_email', 'cloud_mail'))
  AND a.name <> 'webapi';

INSERT INTO provider_adapters (provider_id, adapter_id, priority)
SELECT p.id, a.id, 0
FROM providers p, adapters a
WHERE (p.name LIKE 'webapi_%' OR p.name IN ('cloudflare_temp_email', 'cloud_mail'))
  AND a.name = 'webapi'
ON CONFLICT (provider_id, adapter_id) DO UPDATE SET
    priority = EXCLUDED.priority;

-- 3.4 其他 IMAP 提供商只支持 IMAP
INSERT INTO provider_adapters (provider_id, adapter_id, priority)
SELECT p.id, a.id, 0
FROM providers p, adapters a
WHERE p.id NOT IN (SELECT DISTINCT provider_id FROM provider_adapters)
  AND a.name = 'imap'
  AND NOT (p.name LIKE 'webapi_%' OR p.name IN ('cloudflare_temp_email', 'cloud_mail'))
ON CONFLICT (provider_id, adapter_id) DO NOTHING;

-- 验证数据填充
DO $$
DECLARE
    providers_without_adapter INTEGER;
    provider_adapters_count INTEGER;
BEGIN
    SELECT COUNT(*) INTO providers_without_adapter 
    FROM providers p
    WHERE p.default_adapter_id IS NULL
       OR p.default_adapter_id = 0
       OR NOT EXISTS (
           SELECT 1 FROM adapters a WHERE a.id = p.default_adapter_id
       );
    
    SELECT COUNT(*) INTO provider_adapters_count 
    FROM provider_adapters;
    
    IF providers_without_adapter > 0 THEN
        RAISE WARNING '⚠ 有 % 个 Provider 未设置 default_adapter_id', providers_without_adapter;
    ELSE
        RAISE NOTICE '✓ 所有 Provider 已设置 default_adapter_id';
    END IF;
    
    RAISE NOTICE '✓ provider_adapters 关联记录数: %', provider_adapters_count;
END $$;

-- 显示填充结果
SELECT p.id, p.name, p.display_name, 
       a.name as default_adapter, 
       p.email_domains,
       (SELECT COUNT(*) FROM provider_adapters pa WHERE pa.provider_id = p.id) as adapter_count
FROM providers p
LEFT JOIN adapters a ON p.default_adapter_id = a.id
ORDER BY p.sort_order, p.id;
