-- 创建 Provider-Adapter 多对多关联表
-- Migration: 021_create_provider_adapters_table
-- Description: 创建 provider_adapters 表支持一个 Provider 关联多个 Adapter
-- Author: FusionMail Team
-- Date: 2024-12-14
-- Related: provider-account-refactor spec

-- ============================================
-- 阶段 1.2：创建 provider_adapters 关联表
-- 预计执行时间：< 1 分钟
-- 风险等级：低（仅添加新表，不影响现有功能）
-- ============================================

-- 创建 Provider-Adapter 多对多关联表
CREATE TABLE IF NOT EXISTS provider_adapters (
    provider_id BIGINT NOT NULL,
    adapter_id BIGINT NOT NULL REFERENCES adapters(id) ON DELETE RESTRICT,
    priority INTEGER DEFAULT 0,                 -- 优先级，0 为最高（默认推荐）
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (provider_id, adapter_id)
);

-- 创建索引
CREATE INDEX IF NOT EXISTS idx_provider_adapters_provider_id ON provider_adapters(provider_id);
CREATE INDEX IF NOT EXISTS idx_provider_adapters_adapter_id ON provider_adapters(adapter_id);
CREATE INDEX IF NOT EXISTS idx_provider_adapters_priority ON provider_adapters(priority);

-- 添加注释
COMMENT ON TABLE provider_adapters IS 'Provider 与 Adapter 的多对多关联表，支持一个提供商配置多个适配器';
COMMENT ON COLUMN provider_adapters.provider_id IS '关联的提供商 ID';
COMMENT ON COLUMN provider_adapters.adapter_id IS '关联的适配器 ID';
COMMENT ON COLUMN provider_adapters.priority IS '适配器优先级，0 为最高优先级（默认推荐），数值越大优先级越低';

-- 注意：provider_id 的外键约束将在 022 迁移中添加（需要先修改 providers 表）

-- 验证表创建
DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM information_schema.tables WHERE table_name = 'provider_adapters') THEN
        RAISE NOTICE '✓ provider_adapters 表创建成功';
    ELSE
        RAISE EXCEPTION 'provider_adapters 表创建失败';
    END IF;
END $$;
