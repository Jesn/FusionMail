-- 回滚 baseline：清空 public schema。
--
-- ���是破坏性操作，仅用于测试环境重建；生产回滚请使用备份恢复。
DROP SCHEMA IF EXISTS public CASCADE;
CREATE SCHEMA public;
