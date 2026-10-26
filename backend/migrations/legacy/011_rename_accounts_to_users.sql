-- 重命名 accounts 表为 users（系统用户表）
-- 目的：使表名与模型名称匹配，提高代码可读性

-- 1. 删除冗余的 users 表（如果存在）
DROP TABLE IF EXISTS users CASCADE;

-- 2. 重命名 accounts 表为 users
ALTER TABLE accounts RENAME TO users;

-- 3. 重命名相关的序列
ALTER SEQUENCE IF EXISTS accounts_id_seq RENAME TO users_id_seq;
ALTER SEQUENCE IF EXISTS accounts_id_seq1 RENAME TO users_id_seq1;

-- 4. 重命名索引
ALTER INDEX IF EXISTS idx_accounts_username RENAME TO idx_users_username;
ALTER INDEX IF EXISTS idx_accounts_email RENAME TO idx_users_email;
ALTER INDEX IF EXISTS idx_accounts_is_active RENAME TO idx_users_is_active;
ALTER INDEX IF EXISTS idx_accounts_role RENAME TO idx_users_role;

-- 5. 重命名触发器
DROP TRIGGER IF EXISTS update_accounts_updated_at ON users;
CREATE TRIGGER update_users_updated_at
    BEFORE UPDATE ON users
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

-- 6. 更新约束名称（如果有）
DO $$
DECLARE
    constraint_record RECORD;
BEGIN
    FOR constraint_record IN 
        SELECT conname 
        FROM pg_constraint 
        WHERE conrelid = 'users'::regclass 
        AND conname LIKE '%accounts%'
    LOOP
        EXECUTE format('ALTER TABLE users RENAME CONSTRAINT %I TO %I',
            constraint_record.conname,
            replace(constraint_record.conname, 'accounts', 'users'));
    END LOOP;
END $$;

-- 验证迁移结果
DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM information_schema.tables WHERE table_name = 'users') THEN
        RAISE NOTICE '✅ 表 users 创建成功';
    ELSE
        RAISE EXCEPTION '❌ 表 users 不存在';
    END IF;
    
    IF NOT EXISTS (SELECT 1 FROM information_schema.tables WHERE table_name = 'accounts') THEN
        RAISE NOTICE '✅ 表 accounts 已删除';
    ELSE
        RAISE NOTICE '⚠️  表 accounts 仍然存在';
    END IF;
END $$;
