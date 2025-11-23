-- 为 oauth2_clients 表添加 provider_type 字段
-- 用于更可靠地关联到 providers 表

-- 1. 添加 provider_type 字段（使用整数类型，枚举值）
ALTER TABLE oauth2_clients
ADD COLUMN provider_type INTEGER NOT NULL DEFAULT 1;

-- 2. 添加注释
COMMENT ON COLUMN oauth2_clients.provider_type IS '邮箱提供商类型枚举：1=gmail, 2=outlook, 3=icloud, 4=qq, 5=163, 6=generic';

-- 3. 更新现有数据（根据 provider_name 映射到 provider_type）
-- Gmail (provider_type = 1)
UPDATE oauth2_clients
SET provider_type = 1
WHERE provider_name = 'gmail';

-- Outlook (provider_type = 2)
UPDATE oauth2_clients
SET provider_type = 2
WHERE provider_name = 'outlook';

-- 4. 添加外键约束（可选，增强数据完整性）
-- 注意：外键约束需要在 Provider 表也有 provider_type 字段
-- 如果 Provider 表没有此字段，可以先不添加外键约束

-- 5. 创建索引（提高查询性能）
CREATE INDEX idx_oauth2_clients_provider_type ON oauth2_clients(provider_type);

-- 6. 更新唯一约束（使用 provider_type 而不是 provider_name）
-- 删除旧的唯一约束（如果存在）
DROP INDEX IF EXISTS uk_oauth2_clients_provider_name;

-- 创建新的唯一约束：每个 provider_type 只能有一个默认客户端
-- 注意：这还需要考虑 provider_type 和 name 的组合唯一性
CREATE UNIQUE INDEX uk_oauth2_clients_provider_type_default
ON oauth2_clients(provider_type)
WHERE is_default = TRUE AND enabled = TRUE;

-- 保留 provider_name 字段用于向前兼容
-- 建议在未来版本中废弃 provider_name 字段
