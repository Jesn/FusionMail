-- 创建账号分组功能相关表和字段
-- Migration: 018_create_account_groups
-- Description: 创建 account_groups 表，为 email_accounts 表添加 group_id 字段
-- Requirements: 1.1, 1.4, 4.1

-- =====================================================
-- Part 1: 创建 account_groups 表
-- =====================================================

CREATE TABLE IF NOT EXISTS account_groups (
    id BIGSERIAL PRIMARY KEY,
    name VARCHAR(100) NOT NULL,
    description TEXT,
    display_order INTEGER NOT NULL DEFAULT 0,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP WITH TIME ZONE
);

-- 添加唯一约束（仅对未删除的记录生效）
CREATE UNIQUE INDEX IF NOT EXISTS uk_account_groups_name 
    ON account_groups(name) 
    WHERE deleted_at IS NULL;

-- 添加显示顺序索引
CREATE INDEX IF NOT EXISTS idx_account_groups_display_order 
    ON account_groups(display_order);

-- 添加软删除索引
CREATE INDEX IF NOT EXISTS idx_account_groups_deleted_at 
    ON account_groups(deleted_at);

-- 添加表注释
COMMENT ON TABLE account_groups IS '邮箱账号分组表';
COMMENT ON COLUMN account_groups.id IS '分组ID';
COMMENT ON COLUMN account_groups.name IS '分组名称，唯一';
COMMENT ON COLUMN account_groups.description IS '分组描述';
COMMENT ON COLUMN account_groups.display_order IS '显示顺序，数值越小越靠前';
COMMENT ON COLUMN account_groups.created_at IS '创建时间';
COMMENT ON COLUMN account_groups.updated_at IS '更新时间';
COMMENT ON COLUMN account_groups.deleted_at IS '软删除时间';

-- =====================================================
-- Part 2: 为 email_accounts 表添加 group_id 字段
-- =====================================================

-- 添加 group_id 字段
ALTER TABLE email_accounts 
    ADD COLUMN IF NOT EXISTS group_id BIGINT;

-- 添加外键约束（删除分组时将 group_id 设为 NULL）
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM information_schema.table_constraints 
        WHERE constraint_name = 'fk_email_accounts_group_id'
    ) THEN
        ALTER TABLE email_accounts 
            ADD CONSTRAINT fk_email_accounts_group_id 
            FOREIGN KEY (group_id) 
            REFERENCES account_groups(id) 
            ON DELETE SET NULL;
    END IF;
END $$;

-- 添加索引以优化按分组查询
CREATE INDEX IF NOT EXISTS idx_email_accounts_group_id 
    ON email_accounts(group_id);

-- 添加字段注释
COMMENT ON COLUMN email_accounts.group_id IS '所属分组ID，NULL 表示未分组';
