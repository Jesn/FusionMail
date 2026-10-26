-- 清理 OAuth2 客户端占位符数据
-- 此脚本删除所有使用占位符 client_id 的 OAuth2 客户端配置

-- 显示将要删除的数据
SELECT
    id,
    provider_name,
    name,
    client_id,
    redirect_uri,
    enabled,
    is_default,
    created_at
FROM oauth2_clients
WHERE client_id LIKE 'your-%-client-id'
ORDER BY provider_name, name;

-- 删除占位符数据
DELETE FROM oauth2_clients
WHERE client_id LIKE 'your-%-client-id';

-- 显示删除结果
SELECT COUNT(*) AS deleted_count
FROM (
    SELECT 1
    FROM oauth2_clients
    WHERE client_id LIKE 'your-%-client-id'
) AS deleted_rows;
