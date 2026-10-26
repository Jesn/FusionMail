-- 添加provider_id字段到o_auth2_clients表
ALTER TABLE o_auth2_clients
ADD COLUMN provider_id BIGINT;

-- 迁移现有数据 - 根据provider_name找到对应的provider_id
UPDATE o_auth2_clients
SET provider_id = p.id
FROM providers p
WHERE (
    (o_auth2_clients.provider_name = 'gmail' AND p.name = 'Gmail') OR
    (o_auth2_clients.provider_name = 'outlook' AND p.name = 'outlook') OR
    (o_auth2_clients.provider_name = 'icloud' AND p.name = 'icloud') OR
    (o_auth2_clients.provider_name = 'qq' AND p.name = 'qq') OR
    (o_auth2_clients.provider_name = '163' AND p.name = '163') OR
    (o_auth2_clients.provider_name = 'generic' AND p.name = 'generic')
);

-- 添加外键约束
ALTER TABLE o_auth2_clients
ADD CONSTRAINT fk_oauth2_clients_provider
FOREIGN KEY (provider_id)
REFERENCES providers(id)
ON DELETE CASCADE;

-- 为provider_id创建索引
CREATE INDEX idx_oauth2_clients_provider_id ON o_auth2_clients(provider_id);

-- 设置provider_id为NOT NULL（确保数据完整性）
ALTER TABLE o_auth2_clients
ALTER COLUMN provider_id SET NOT NULL;

-- 删除旧的provider_name字段
ALTER TABLE o_auth2_clients
DROP COLUMN provider_name;