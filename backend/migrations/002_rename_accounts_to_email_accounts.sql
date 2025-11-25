-- 重命名accounts表为email_accounts
ALTER TABLE accounts RENAME TO email_accounts;

-- 更新相关索引名称
ALTER INDEX accounts_pkey RENAME TO email_accounts_pkey;
ALTER INDEX accounts_uid_key RENAME TO email_accounts_uid_key;
ALTER INDEX accounts_email_provider_key RENAME TO email_accounts_email_provider_key;

-- 更新外键约束（如果存在）
-- 注意：需要先删除并重新创建外键约束，因为表名更改了

-- 检查是否存在外键约束
DO $$
BEGIN
    -- 检查并更新外键约束
    IF EXISTS (SELECT 1 FROM information_schema.table_constraints
              WHERE constraint_name = 'emails_account_id_fkey'
              AND table_name = 'emails') THEN
        ALTER TABLE emails DROP CONSTRAINT emails_account_id_fkey;
        ALTER TABLE emails ADD CONSTRAINT emails_account_id_fkey
            FOREIGN KEY (account_id) REFERENCES email_accounts(id) ON DELETE CASCADE;
    END IF;

    IF EXISTS (SELECT 1 FROM information_schema.table_constraints
              WHERE constraint_name = 'email_attachments_account_id_fkey'
              AND table_name = 'email_attachments') THEN
        ALTER TABLE email_attachments DROP CONSTRAINT email_attachments_account_id_fkey;
        ALTER TABLE email_attachments ADD CONSTRAINT email_attachments_account_id_fkey
            FOREIGN KEY (account_id) REFERENCES email_accounts(id) ON DELETE CASCADE;
    END IF;

    IF EXISTS (SELECT 1 FROM information_schema.table_constraints
              WHERE constraint_name = 'email_labels_account_id_fkey'
              AND table_name = 'email_labels') THEN
        ALTER TABLE email_labels DROP CONSTRAINT email_labels_account_id_fkey;
        ALTER TABLE email_labels ADD CONSTRAINT email_labels_account_id_fkey
            FOREIGN KEY (account_id) REFERENCES email_accounts(id) ON DELETE CASCADE;
    END IF;

    IF EXISTS (SELECT 1 FROM information_schema.table_constraints
              WHERE constraint_name = 'email_label_relations_account_id_fkey'
              AND table_name = 'email_label_relations') THEN
        ALTER TABLE email_label_relations DROP CONSTRAINT email_label_relations_account_id_fkey;
        ALTER TABLE email_label_relations ADD CONSTRAINT email_label_relations_account_id_fkey
            FOREIGN KEY (account_id) REFERENCES email_accounts(id) ON DELETE CASCADE;
    END IF;

    IF EXISTS (SELECT 1 FROM information_schema.table_constraints
              WHERE constraint_name = 'email_rules_account_id_fkey'
              AND table_name = 'email_rules') THEN
        ALTER TABLE email_rules DROP CONSTRAINT email_rules_account_id_fkey;
        ALTER TABLE email_rules ADD CONSTRAINT email_rules_account_id_fkey
            FOREIGN KEY (account_id) REFERENCES email_accounts(id) ON DELETE CASCADE;
    END IF;

    IF EXISTS (SELECT 1 FROM information_schema.table_constraints
              WHERE constraint_name = 'sync_logs_account_id_fkey'
              AND table_name = 'sync_logs') THEN
        ALTER TABLE sync_logs DROP CONSTRAINT sync_logs_account_id_fkey;
        ALTER TABLE sync_logs ADD CONSTRAINT sync_logs_account_id_fkey
            FOREIGN KEY (account_id) REFERENCES email_accounts(id) ON DELETE CASCADE;
    END IF;
END $$;