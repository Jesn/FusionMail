#!/bin/bash

# ============================================
# 使用 Docker 创建开发数据库
# ============================================
# 适用于没有安装 psql 的环境
# ============================================

set -e

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

# 数据库配置
DB_HOST="192.168.2.200"
DB_PORT="5432"
DB_USER="postgres"
DB_PASSWORD="8QMZn3yfrbkVG7"
DB_NAME="fusionmail-dev"

print_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

print_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

print_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

echo ""
echo -e "${BLUE}=========================================="
echo -e "  使用 Docker 创建开发数据库"
echo -e "==========================================${NC}"
echo ""

# 检查 Docker 是否可用
if ! command -v docker &> /dev/null; then
    print_error "Docker 未安装或未运行"
    print_info "请安装 Docker 或使用 ./scripts/setup-dev-db.sh"
    exit 1
fi

# 创建数据库
print_info "创建数据库 $DB_NAME..."

docker run --rm \
    -e PGPASSWORD=$DB_PASSWORD \
    postgres:15-alpine \
    psql -h $DB_HOST -p $DB_PORT -U $DB_USER -c "CREATE DATABASE \"$DB_NAME\" OWNER $DB_USER;"

if [ $? -eq 0 ]; then
    print_success "数据库创建成功"
else
    print_error "数据库创建失败（可能已存在）"
fi

# 创建扩展
print_info "创建数据库扩展..."

docker run --rm \
    -e PGPASSWORD=$DB_PASSWORD \
    postgres:15-alpine \
    psql -h $DB_HOST -p $DB_PORT -U $DB_USER -d "$DB_NAME" \
    -c "CREATE EXTENSION IF NOT EXISTS \"uuid-ossp\"; CREATE EXTENSION IF NOT EXISTS \"pg_trgm\";"

if [ $? -eq 0 ]; then
    print_success "数据库扩展创建成功"
fi

echo ""
print_success "数据库设置完成！"
echo ""
echo -e "${BLUE}下一步：${NC}"
echo "  启动项目: ./start.sh"
echo ""
