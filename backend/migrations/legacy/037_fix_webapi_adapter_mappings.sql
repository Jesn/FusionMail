-- 修复 WebAPI Provider 的默认 Adapter 和账户 Adapter
-- Migration: 037_fix_webapi_adapter_mappings
-- Description: 纠正旧迁移可能写入的 WebAPI Provider -> IMAP 错误映射
-- Date: 2026-05-18

-- 确保 webapi adapter 存在
INSERT INTO adapters (name, display_name, auth_type, description, is_enabled) VALUES
('webapi', 'Web API', 'token', '通用 Web API 邮箱适配器', true)
ON CONFLICT (name) DO UPDATE SET
    display_name = COALESCE(NULLIF(adapters.display_name, ''), EXCLUDED.display_name),
    auth_type = COALESCE(NULLIF(adapters.auth_type, ''), EXCLUDED.auth_type),
    description = COALESCE(NULLIF(adapters.description, ''), EXCLUDED.description),
    updated_at = CURRENT_TIMESTAMP;

-- 修复 WebAPI Provider 被回填成 IMAP、0、NULL 或无效 Adapter 的情况
UPDATE providers
SET default_adapter_id = (SELECT id FROM adapters WHERE name = 'webapi')
WHERE (name LIKE 'webapi_%' OR name IN ('cloudflare_temp_email', 'cloud_mail') OR recommended_protocol = 'webapi')
  AND (
      default_adapter_id IS NULL
      OR default_adapter_id = 0
      OR default_adapter_id = (SELECT id FROM adapters WHERE name = 'imap')
      OR NOT EXISTS (SELECT 1 FROM adapters WHERE adapters.id = providers.default_adapter_id)
  );

-- 清理 WebAPI Provider 上旧迁移错误插入的非 WebAPI 关联
DELETE FROM provider_adapters pa
USING providers p, adapters a
WHERE pa.provider_id = p.id
  AND pa.adapter_id = a.id
  AND (p.name LIKE 'webapi_%' OR p.name IN ('cloudflare_temp_email', 'cloud_mail') OR p.recommended_protocol = 'webapi')
  AND a.name <> 'webapi';

-- 确保 WebAPI Provider 只关联 webapi adapter，且优先级最高
INSERT INTO provider_adapters (provider_id, adapter_id, priority)
SELECT p.id, a.id, 0
FROM providers p, adapters a
WHERE (p.name LIKE 'webapi_%' OR p.name IN ('cloudflare_temp_email', 'cloud_mail') OR p.recommended_protocol = 'webapi')
  AND a.name = 'webapi'
ON CONFLICT (provider_id, adapter_id) DO UPDATE SET
    priority = EXCLUDED.priority;

-- 修复旧 025 迁移可能写入的 WebAPI 账户 IMAP Adapter
UPDATE email_accounts ea
SET adapter_id = webapi.id
FROM providers p, adapters webapi
WHERE ea.provider_id = p.id
  AND webapi.name = 'webapi'
  AND (p.name LIKE 'webapi_%' OR p.name IN ('cloudflare_temp_email', 'cloud_mail') OR p.recommended_protocol = 'webapi')
  AND (
      ea.adapter_id IS NULL
      OR ea.adapter_id = 0
      OR ea.adapter_id = (SELECT id FROM adapters WHERE name = 'imap')
      OR NOT EXISTS (SELECT 1 FROM adapters a WHERE a.id = ea.adapter_id)
  );

DO $$
DECLARE
    wrong_provider_count INTEGER;
    wrong_account_count INTEGER;
BEGIN
    SELECT COUNT(*) INTO wrong_provider_count
    FROM providers p
    LEFT JOIN adapters a ON a.id = p.default_adapter_id
    WHERE (p.name LIKE 'webapi_%' OR p.name IN ('cloudflare_temp_email', 'cloud_mail') OR p.recommended_protocol = 'webapi')
      AND COALESCE(a.name, '') <> 'webapi';

    SELECT COUNT(*) INTO wrong_account_count
    FROM email_accounts ea
    JOIN providers p ON p.id = ea.provider_id
    LEFT JOIN adapters a ON a.id = ea.adapter_id
    WHERE (p.name LIKE 'webapi_%' OR p.name IN ('cloudflare_temp_email', 'cloud_mail') OR p.recommended_protocol = 'webapi')
      AND COALESCE(a.name, '') <> 'webapi';

    IF wrong_provider_count > 0 THEN
        RAISE WARNING '仍有 % 个 WebAPI Provider 未使用 webapi adapter', wrong_provider_count;
    END IF;

    IF wrong_account_count > 0 THEN
        RAISE WARNING '仍有 % 个 WebAPI 账户未使用 webapi adapter', wrong_account_count;
    END IF;
END $$;
