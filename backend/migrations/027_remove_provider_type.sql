-- 027_remove_provider_type.sql
-- 移除 providers 表中的 provider_type 字段
-- 该字段已被 default_adapter_id 和 name 字段替代

-- ============================================
-- 阶段 1: 删除依赖 provider_type 的索引
-- ============================================

-- 删除 provider_type 索引（如果存在）
DROP INDEX IF EXISTS idx_providers_provider_type;

-- ============================================
-- 阶段 2: 删除 provider_type 字段
-- ============================================

-- 删除 providers 表的 provider_type 字段
ALTER TABLE providers DROP COLUMN IF EXISTS provider_type;

-- ============================================
-- 验证
-- ============================================

-- 验证 provider_type 字段已被删除
-- SELECT column_name FROM information_schema.columns 
-- WHERE table_name = 'providers' AND column_name = 'provider_type';
-- 应返回空结果

-- ============================================
-- 回滚脚本（如需回滚，手动执行）
-- ============================================
-- ALTER TABLE providers ADD COLUMN provider_type INTEGER;
-- CREATE INDEX idx_providers_provider_type ON providers(provider_type);
-- 
-- 注意：回滚后需要手动填充 provider_type 数据：
-- UPDATE providers SET provider_type = 1 WHERE name = 'gmail';
-- UPDATE providers SET provider_type = 2 WHERE name = 'outlook';
-- UPDATE providers SET provider_type = 3 WHERE name = 'icloud';
-- UPDATE providers SET provider_type = 4 WHERE name = 'qq';
-- UPDATE providers SET provider_type = 5 WHERE name = '163';
-- UPDATE providers SET provider_type = 6 WHERE name = 'generic';
