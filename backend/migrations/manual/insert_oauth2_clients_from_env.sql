-- 从环境变量同步 OAuth2 客户端配置
-- 此脚本将从环境变量获取的真实 OAuth2 客户端数据插入到数据库

-- 显示当前 oauth2_clients 表中的数据
SELECT '=== 当前 OAuth2 客户端配置 ===' AS info;
SELECT id, provider_name, name, client_id, enabled, is_default, created_at
FROM oauth2_clients
ORDER BY provider_name, name;

-- 删除占位符数据（如果存在）
DELETE FROM oauth2_clients
WHERE client_id LIKE 'your-%-client-id';

-- Gmail 配置
-- 首先检查是否已存在
DO $$
DECLARE
    gmail_count INTEGER;
    outlook_count INTEGER;
BEGIN
    -- 检查 Gmail 配置
    SELECT COUNT(*) INTO gmail_count
    FROM oauth2_clients
    WHERE provider_name = 'gmail';

    IF gmail_count = 0 THEN
        -- 插入 Gmail 配置
        INSERT INTO oauth2_clients (
            provider_name,
            name,
            client_id,
            client_secret_encrypted,
            redirect_uri,
            enabled,
            is_default,
            quota_daily,
            quota_monthly,
            created_at,
            updated_at
        ) VALUES (
            'gmail',
            '默认配置',
            '28698829185-evea9bupqunm53pi5jdeajsspicsae0p.apps.googleusercontent.com',
            '$2a$10$e0MYzXyjpJSBlyPYxH3mGOVbBG8.H3wZ8Q8Z5fX9yP2w4F5E6T7S8U9', -- 需要加密的密码
            'http://localhost:3333/api/v1/auth/google/callback',
            true,
            true,
            100,
            2000,
            NOW(),
            NOW()
        );

        RAISE NOTICE '已插入 Gmail OAuth2 客户端配置';
    ELSE
        -- 更新现有的 Gmail 配置
        UPDATE oauth2_clients
        SET
            client_id = '28698829185-evea9bupqunm53pi5jdeajsspicsae0p.apps.googleusercontent.com',
            redirect_uri = 'http://localhost:3333/api/v1/auth/google/callback',
            updated_at = NOW()
        WHERE provider_name = 'gmail';

        RAISE NOTICE '已更新 Gmail OAuth2 客户端配置';
    END IF;

    -- 检查 Outlook 配置
    SELECT COUNT(*) INTO outlook_count
    FROM oauth2_clients
    WHERE provider_name = 'outlook';

    IF outlook_count = 0 THEN
        -- 插入 Outlook 配置
        INSERT INTO oauth2_clients (
            provider_name,
            name,
            client_id,
            client_secret_encrypted,
            redirect_uri,
            enabled,
            is_default,
            quota_daily,
            quota_monthly,
            created_at,
            updated_at
        ) VALUES (
            'outlook',
            '默认配置',
            '0ec56a84-6012-4ac5-81a5-e61f6a1f4438',
            '$2a$10$e0MYzXyjpJSBlyPYxH3mGOVbBG8.H3wZ8Q8Z5fX9yP2w4F5E6T7S8U9', -- 需要加密的密码
            'http://localhost:3333/api/v1/auth/microsoft/callback',
            true,
            true,
            100,
            2000,
            NOW(),
            NOW()
        );

        RAISE NOTICE '已插入 Outlook OAuth2 客户端配置';
    ELSE
        -- 更新现有的 Outlook 配置
        UPDATE oauth2_clients
        SET
            client_id = '0ec56a84-6012-4ac5-81a5-e61f6a1f4438',
            redirect_uri = 'http://localhost:3333/api/v1/auth/microsoft/callback',
            updated_at = NOW()
        WHERE provider_name = 'outlook';

        RAISE NOTICE '已更新 Outlook OAuth2 客户端配置';
    END IF;
END $$;

-- 显示同步后的结果
SELECT '=== 同步后的 OAuth2 客户端配置 ===' AS info;
SELECT id, provider_name, name, client_id, enabled, is_default, created_at
FROM oauth2_clients
ORDER BY provider_name, name;

-- 显示统计信息
SELECT
    provider_name,
    COUNT(*) AS count,
    MAX(CASE WHEN is_default THEN name END) AS default_name
FROM oauth2_clients
GROUP BY provider_name
ORDER BY provider_name;
