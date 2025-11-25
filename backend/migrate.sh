#!/bin/bash

# 数据库迁移脚本

echo "开始执行数据库迁移..."

# 检查数据库连接
echo "检查数据库连接..."
docker exec fusionmail-postgres psql -U fusionmail -d fusionmail -c "SELECT 1;" > /dev/null 2>&1

if [ $? -ne 0 ]; then
    echo "数据库连接失败，请确保PostgreSQL容器正在运行"
    exit 1
fi

echo "数据库连接正常"

# 执行迁移脚本
echo "执行迁移001: 创建初始表（如果已存在则跳过）"
docker exec -i fusionmail-postgres psql -U fusionmail -d fusionmail < migrations/001_create_tables.sql

echo "执行迁移002: 重命名accounts表为email_accounts"
docker exec -i fusionmail-postgres psql -U fusionmail -d fusionmail < migrations/002_rename_accounts_to_email_accounts.sql

echo "执行迁移003: 创建新的用户accounts表"
docker exec -i fusionmail-postgres psql -U fusionmail -d fusionmail < migrations/003_create_accounts_table.sql

echo "迁移执行完成！"

# 验证迁移结果
echo "验证迁移结果："
docker exec fusionmail-postgres psql -U fusionmail -d fusionmail -c "
SELECT table_name, column_name, data_type
FROM information_schema.columns
WHERE table_name IN ('accounts', 'email_accounts', 'users')
ORDER BY table_name, ordinal_position;
"

echo "迁移脚本执行完成"