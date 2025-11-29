#!/bin/bash

# ============================================
# FusionMail 数据库连接测试脚本
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
REDIS_HOST="192.168.2.200"
REDIS_PORT="6379"

print_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

print_success() {
    echo -e "${GREEN}[✓]${NC} $1"
}

print_error() {
    echo -e "${RED}[✗]${NC} $1"
}

echo ""
echo -e "${BLUE}=========================================="
echo -e "  FusionMail 数据库连接测试"
echo -e "==========================================${NC}"
echo ""

# 测试网络连接
print_info "测试网络连接..."
if ping -c 1 -W 2 $DB_HOST &> /dev/null; then
    print_success "服务器 $DB_HOST 可访问"
else
    print_error "无法访问服务器 $DB_HOST"
    exit 1
fi

# 测试 PostgreSQL 端口
print_info "测试 PostgreSQL 端口..."
if nc -z -w 2 $DB_HOST $DB_PORT 2>/dev/null; then
    print_success "PostgreSQL 端口 $DB_PORT 可访问"
else
    print_error "无法访问 PostgreSQL 端口 $DB_PORT"
    exit 1
fi

# 测试 Redis 端口
print_info "测试 Redis 端口..."
if nc -z -w 2 $REDIS_HOST $REDIS_PORT 2>/dev/null; then
    print_success "Redis 端口 $REDIS_PORT 可访问"
else
    print_error "无法访问 Redis 端口 $REDIS_PORT"
    exit 1
fi

echo ""
print_success "所有连接测试通过！"
echo ""
echo -e "${BLUE}连接信息：${NC}"
echo "  PostgreSQL: $DB_HOST:$DB_PORT"
echo "  Redis: $REDIS_HOST:$REDIS_PORT"
echo ""
echo -e "${BLUE}下一步：${NC}"
echo "  1. 创建数据库: ./scripts/setup-dev-db.sh"
echo "  2. 启动项目: ./start.sh"
echo ""
