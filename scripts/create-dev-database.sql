-- ============================================
-- FusionMail 开发环境数据库创建脚本
-- ============================================
-- 使用方法：
--   psql -h 192.168.2.200 -p 5432 -U postgres -f scripts/create-dev-database.sql
--
-- 或者使用密码连接：
--   PGPASSWORD=8QMZn3yfrbkVG7 psql -h 192.168.2.200 -p 5432 -U postgres -f scripts/create-dev-database.sql
-- ============================================

-- 创建开发数据库
CREATE DATABASE "fusionmail-dev"
    WITH 
    OWNER = postgres
    ENCODING = 'UTF8'
    LC_COLLATE = 'en_US.utf8'
    LC_CTYPE = 'en_US.utf8'
    TABLESPACE = pg_default
    CONNECTION LIMIT = -1;

-- 连接到新创建的数据库
\c fusionmail-dev

-- 创建必要的扩展
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS "pg_trgm";

-- 授予权限
GRANT ALL PRIVILEGES ON DATABASE "fusionmail-dev" TO postgres;

-- 显示创建结果
\l fusionmail-dev
