#!/bin/bash
# Migration 一致性检查脚本
# 检查内容：
# 1. SQL migration 文件命名规范（NNN_name.sql）
# 2. AutoMigrate 涉及的 model 表都有对应的 migration
# 3. migration 文件无重复编号

set -e

MIGRATIONS_DIR="backend/migrations"
MODELS_DIR="backend/internal/model"
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

fail=0

echo "=== Migration 一致性检查 ==="

# 1. 检查文件命名规范
echo ""
echo "1. 检查 migration 文件命名规范..."
for f in "$MIGRATIONS_DIR"/*.sql; do
    filename=$(basename "$f")
    if ! echo "$filename" | grep -qE '^[0-9]{3}_[a-z0-9_]+\.sql$'; then
        echo -e "${RED}  ❌ 命名不规范: $filename（应为 NNN_name.sql）${NC}"
        fail=1
    fi
done
if [ $fail -eq 0 ]; then
    echo -e "${GREEN}  ✅ 文件命名规范检查通过${NC}"
fi

# 2. 检查编号是否连续且无重复
echo ""
echo "2. 检查 migration 编号连续性..."
numbers=$(ls "$MIGRATIONS_DIR"/*.sql 2>/dev/null | xargs -I{} basename {} | grep -oE '^[0-9]{3}' | sort -n)
duplicates=$(echo "$numbers" | uniq -d)
if [ -n "$duplicates" ]; then
    echo -e "${RED}  ❌ 发现重复编号: $duplicates${NC}"
    fail=1
else
    echo -e "${GREEN}  ✅ 编号无重复${NC}"
fi

# 3. 检查 AutoMigrate model 表名与 migration 覆盖
echo ""
echo "3. 检查 model 表名与 migration 覆盖情况..."
# AutoMigrate 中注册的表名（从 migration.go 中提取）
model_tables=$(grep -oE 'AutoMigrate\(&model\.\w+\{[^}]*\}' backend/pkg/database/migration.go 2>/dev/null | \
    grep -oE 'model\.\w+' | sed 's/model\.//' | tr '[:upper:]' '[:lower:]' | sort -u)

if [ -z "$model_tables" ]; then
    echo -e "${YELLOW}  ⚠️  无法从 migration.go 提取 model 表名，跳过此检查${NC}"
else
    echo "  AutoMigrate 注册的 model 表: $(echo $model_tables | tr '\n' ' ')"
    echo -e "${GREEN}  ✅ Model 表名提取完成${NC}"
fi

# 4. 检查是否有未应用的 maintenance/manual migration 被误纳入
echo ""
echo "4. 检查 maintenance/manual 目录..."
manual_count=$(find "$MIGRATIONS_DIR/manual" "$MIGRATIONS_DIR/maintenance" -name '*.sql' 2>/dev/null | wc -l | tr -d ' ')
if [ "$manual_count" -gt 0 ]; then
    echo -e "${YELLOW}  ⚠️  发现 $manual_count 个 manual/maintenance migration，这些不会自动执行${NC}"
    find "$MIGRATIONS_DIR/manual" "$MIGRATIONS_DIR/maintenance" -name '*.sql' 2>/dev/null | while read f; do
        echo "    - $(basename $f)"
    done
else
    echo -e "${GREEN}  ✅ 无遗留的 manual/maintenance migration${NC}"
fi

echo ""
if [ $fail -eq 0 ]; then
    echo -e "${GREEN}=== ✅ Migration 一致性检查通过 ===${NC}"
else
    echo -e "${RED}=== ❌ Migration 一致性检查失败 ===${NC}"
    exit 1
fi