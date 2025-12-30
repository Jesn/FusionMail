-- 添加 WebAPI 适配器和预置 Provider
-- Migration: 033_add_webapi_adapter
-- Description: 添加 webapi 适配器，以及 cloudflare_temp_email 和 cloud_mail 两个预置 Provider
-- Author: FusionMail Team
-- Date: 2024-12-30
-- Related: webapi-adapter spec

-- ============================================
-- 阶段 1：添加 webapi 适配器
-- 预计执行时间：< 1 分钟
-- 风险等级：低（仅添加新数据，不影响现有功能）
-- ============================================

-- 插入 webapi 适配器
INSERT INTO adapters (name, display_name, auth_type, description, is_enabled) VALUES
('webapi', 'Web API', 'token', 
 '通用 Web API 适配器，支持通过 HTTP API 接入第三方邮箱服务，使用 Token 认证', 
 true)
ON CONFLICT (name) DO UPDATE SET
    display_name = EXCLUDED.display_name,
    auth_type = EXCLUDED.auth_type,
    description = EXCLUDED.description,
    updated_at = CURRENT_TIMESTAMP;

-- ============================================
-- 阶段 2：添加预置 WebAPI Provider
-- ============================================

-- 插入 Cloudflare Temp Email Provider
INSERT INTO providers (
    name, 
    display_name, 
    supported_protocols, 
    recommended_protocol,
    requires_oauth, 
    enabled, 
    sort_order, 
    description,
    metadata
) VALUES (
    'cloudflare_temp_email',
    'Cloudflare Temp Email',
    '["webapi"]',
    'webapi',
    false,
    true,
    100,
    'Cloudflare Workers 临时邮箱服务，支持 Single 模式（单邮箱）和 Admin 模式（域名管理）',
    '{"service_type": "cloudflare_temp_email", "access_modes": ["single", "admin"], "api_docs": "https://github.com/user/cloudflare-temp-email"}'
)
ON CONFLICT (name) DO UPDATE SET
    display_name = EXCLUDED.display_name,
    supported_protocols = EXCLUDED.supported_protocols,
    recommended_protocol = EXCLUDED.recommended_protocol,
    description = EXCLUDED.description,
    metadata = EXCLUDED.metadata,
    updated_at = CURRENT_TIMESTAMP;

-- 插入 Cloud Mail Provider
INSERT INTO providers (
    name, 
    display_name, 
    supported_protocols, 
    recommended_protocol,
    requires_oauth, 
    enabled, 
    sort_order, 
    description,
    metadata
) VALUES (
    'cloud_mail',
    'Cloud Mail',
    '["webapi"]',
    'webapi',
    false,
    true,
    101,
    'Cloud Mail 邮箱服务，支持多账户管理，通过 JWT Token 认证',
    '{"service_type": "cloud_mail", "access_modes": ["multi_account"], "api_docs": "https://github.com/user/cloud-mail"}'
)
ON CONFLICT (name) DO UPDATE SET
    display_name = EXCLUDED.display_name,
    supported_protocols = EXCLUDED.supported_protocols,
    recommended_protocol = EXCLUDED.recommended_protocol,
    description = EXCLUDED.description,
    metadata = EXCLUDED.metadata,
    updated_at = CURRENT_TIMESTAMP;

-- ============================================
-- 阶段 3：创建 Provider-Adapter 关联
-- ============================================

-- 获取 webapi adapter ID 并关联到 cloudflare_temp_email
INSERT INTO provider_adapters (provider_id, adapter_id, priority)
SELECT p.id, a.id, 0
FROM providers p, adapters a
WHERE p.name = 'cloudflare_temp_email' AND a.name = 'webapi'
ON CONFLICT (provider_id, adapter_id) DO NOTHING;

-- 获取 webapi adapter ID 并关联到 cloud_mail
INSERT INTO provider_adapters (provider_id, adapter_id, priority)
SELECT p.id, a.id, 0
FROM providers p, adapters a
WHERE p.name = 'cloud_mail' AND a.name = 'webapi'
ON CONFLICT (provider_id, adapter_id) DO NOTHING;

-- ============================================
-- 阶段 4：更新 Provider 的 default_adapter_id
-- ============================================

-- 设置 cloudflare_temp_email 的默认适配器为 webapi
UPDATE providers 
SET default_adapter_id = (SELECT id FROM adapters WHERE name = 'webapi')
WHERE name = 'cloudflare_temp_email';

-- 设置 cloud_mail 的默认适配器为 webapi
UPDATE providers 
SET default_adapter_id = (SELECT id FROM adapters WHERE name = 'webapi')
WHERE name = 'cloud_mail';

-- ============================================
-- 验证数据
-- ============================================

DO $
DECLARE
    webapi_adapter_count INTEGER;
    webapi_provider_count INTEGER;
    provider_adapter_count INTEGER;
BEGIN
    -- 验证 webapi adapter
    SELECT COUNT(*) INTO webapi_adapter_count 
    FROM adapters WHERE name = 'webapi';
    
    IF webapi_adapter_count = 0 THEN
        RAISE EXCEPTION 'webapi 适配器插入失败';
    END IF;
    
    -- 验证 webapi providers
    SELECT COUNT(*) INTO webapi_provider_count 
    FROM providers WHERE name IN ('cloudflare_temp_email', 'cloud_mail');
    
    IF webapi_provider_count < 2 THEN
        RAISE EXCEPTION 'WebAPI Provider 插入失败，期望 2 条记录，实际 % 条', webapi_provider_count;
    END IF;
    
    -- 验证 provider_adapters 关联
    SELECT COUNT(*) INTO provider_adapter_count 
    FROM provider_adapters pa
    JOIN providers p ON pa.provider_id = p.id
    JOIN adapters a ON pa.adapter_id = a.id
    WHERE p.name IN ('cloudflare_temp_email', 'cloud_mail') AND a.name = 'webapi';
    
    IF provider_adapter_count < 2 THEN
        RAISE EXCEPTION 'Provider-Adapter 关联失败，期望 2 条记录，实际 % 条', provider_adapter_count;
    END IF;
    
    RAISE NOTICE '✓ WebAPI 适配器和 Provider 创建成功';
    RAISE NOTICE '  - webapi 适配器: 1 条';
    RAISE NOTICE '  - WebAPI Provider: % 条', webapi_provider_count;
    RAISE NOTICE '  - Provider-Adapter 关联: % 条', provider_adapter_count;
END $;

-- 更新表的统计信息
ANALYZE adapters;
ANALYZE providers;
ANALYZE provider_adapters;
