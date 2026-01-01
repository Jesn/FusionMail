-- WebAPI Provider 初始化脚本
-- 用于本地开发环境初始化 WebAPI 相关数据

-- 1. 插入 WebAPI 适配器（如果不存在）
INSERT INTO adapters (name, display_name, auth_type, description, is_enabled, created_at, updated_at)
VALUES ('webapi', 'Web API 适配器', 'token', '通用 Web API 适配器，支持 Cloudflare Temp Email、Cloud Mail 等服务', true, NOW(), NOW())
ON CONFLICT (name) DO NOTHING;

-- 2. 插入 WebAPI Provider（Cloudflare Temp Email）
INSERT INTO providers (name, display_name, default_adapter_id, email_domains, supported_protocols, recommended_protocol, requires_o_auth, imap_port, pop3_port, smtp_port, imap_encryption, pop3_encryption, smtp_encryption, enabled, sort_order, description, metadata, created_at, updated_at)
SELECT 'webapi_cloudflare_temp_email', 'Cloudflare Temp Email', a.id, '{}', '["webapi"]', 'webapi', false, 0, 0, 0, 'ssl', 'ssl', 'ssl', true, 100, 'Cloudflare Workers 临时邮箱服务', '{"service_type":"cloudflare_temp_email","access_modes":["single","admin"]}', NOW(), NOW()
FROM adapters a WHERE a.name = 'webapi'
ON CONFLICT (name) DO NOTHING;

-- 3. 插入 WebAPI Provider（Cloud Mail）
INSERT INTO providers (name, display_name, default_adapter_id, email_domains, supported_protocols, recommended_protocol, requires_o_auth, imap_port, pop3_port, smtp_port, imap_encryption, pop3_encryption, smtp_encryption, enabled, sort_order, description, metadata, created_at, updated_at)
SELECT 'webapi_cloud_mail', 'Cloud Mail', a.id, '{}', '["webapi"]', 'webapi', false, 0, 0, 0, 'ssl', 'ssl', 'ssl', true, 101, 'Cloud Mail 邮箱服务 (如 mail.hema.edu.kg)', '{"service_type":"cloud_mail","access_modes":["single"]}', NOW(), NOW()
FROM adapters a WHERE a.name = 'webapi'
ON CONFLICT (name) DO NOTHING;

-- 4. 插入 WebAPI Provider（自定义）
INSERT INTO providers (name, display_name, default_adapter_id, email_domains, supported_protocols, recommended_protocol, requires_o_auth, imap_port, pop3_port, smtp_port, imap_encryption, pop3_encryption, smtp_encryption, enabled, sort_order, description, metadata, created_at, updated_at)
SELECT 'webapi_custom', '自定义 Web API', a.id, '{}', '["webapi"]', 'webapi', false, 0, 0, 0, 'ssl', 'ssl', 'ssl', true, 102, '自定义 Web API 邮箱服务', '{"service_type":"custom","access_modes":["single","admin"]}', NOW(), NOW()
FROM adapters a WHERE a.name = 'webapi'
ON CONFLICT (name) DO NOTHING;

-- 5. 验证结果
SELECT id, name, display_name, default_adapter_id, supported_protocols FROM providers WHERE name LIKE 'webapi%';
