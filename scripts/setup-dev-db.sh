#!/bin/bash

# ============================================
# FusionMail 开发环境数据库快速设置脚本
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
DB_PASSWORD="${DB_PASSWORD:?请设置 DB_PASSWORD 环境变量}"
DB_NAME="fusionmail-dev"

REDIS_HOST="192.168.2.200"
REDIS_PORT="6379"
REDIS_DB="6"

print_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

print_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

print_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

print_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

echo ""
echo -e "${BLUE}=========================================="
echo -e "  FusionMail 开发环境数据库设置"
echo -e "==========================================${NC}"
echo ""

# 检查网络连接
print_info "检查数据库服务器连接..."
if ping -c 1 -W 2 $DB_HOST &> /dev/null; then
    print_success "服务器 $DB_HOST 可访问"
else
    print_error "无法访问服务器 $DB_HOST"
    print_info "请检查网络连接和服务器地址"
    exit 1
fi

# 检查 PostgreSQL 端口
print_info "检查 PostgreSQL 端口..."
if nc -z -w 2 $DB_HOST $DB_PORT 2>/dev/null; then
    print_success "PostgreSQL 端口 $DB_PORT 可访问"
else
    print_error "无法访问 PostgreSQL 端口 $DB_PORT"
    print_info "请检查 PostgreSQL 服务是否运行"
    exit 1
fi

# 检查 Redis 端口
print_info "检查 Redis 端口..."
if nc -z -w 2 $REDIS_HOST $REDIS_PORT 2>/dev/null; then
    print_success "Redis 端口 $REDIS_PORT 可访问"
else
    print_warning "无法访问 Redis 端口 $REDIS_PORT"
    print_info "Redis 可能未运行，但不影响数据库创建"
fi

# 检查是否安装了 psql
if ! command -v psql &> /dev/null; then
    print_error "未找到 psql 命令"
    print_info "请安装 PostgreSQL 客户端："
    echo "  macOS:   brew install postgresql"
    echo "  Ubuntu:  sudo apt-get install postgresql-client"
    echo "  CentOS:  sudo yum install postgresql"
    echo ""
    print_info "或者使用数据库管理工具手动创建数据库："
    echo "  数据库名: $DB_NAME"
    echo "  主机: $DB_HOST:$DB_PORT"
    echo "  用户: $DB_USER"
    exit 1
fi

# 检查数据库是否已存在
print_info "检查数据库是否已存在..."
if PGPASSWORD="$DB_PASSWORD" psql -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" -lqt | cut -d \| -f 1 | grep -qw "$DB_NAME"; then
    print_warning "数据库 $DB_NAME 已存在"
    
    read -p "是否删除并重新创建？(y/N): " confirm
    if [ "$confirm" = "y" ] || [ "$confirm" = "Y" ]; then
        print_info "删除现有数据库..."
        PGPASSWORD="$DB_PASSWORD" psql -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" -c "DROP DATABASE IF EXISTS \"$DB_NAME\";"
        print_success "数据库已删除"
    else
        print_info "保留现有数据库，跳过创建"
        exit 0
    fi
fi

# 创建数据库
print_info "创建数据库 $DB_NAME..."
PGPASSWORD="$DB_PASSWORD" psql -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" -c "CREATE DATABASE \"$DB_NAME\" OWNER $DB_USER;"

if [ $? -eq 0 ]; then
    print_success "数据库创建成功"
else
    print_error "数据库创建失败"
    exit 1
fi

# 创建扩展
print_info "创建数据库扩展..."
PGPASSWORD="$DB_PASSWORD" psql -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" -d "$DB_NAME" -c "CREATE EXTENSION IF NOT EXISTS \"uuid-ossp\";"
PGPASSWORD="$DB_PASSWORD" psql -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" -d "$DB_NAME" -c "CREATE EXTENSION IF NOT EXISTS \"pg_trgm\";"

if [ $? -eq 0 ]; then
    print_success "数据库扩展创建成功"
else
    print_warning "数据库扩展创建失败（可能已存在）"
fi

# 验证数据库
print_info "验证数据库..."
if PGPASSWORD="$DB_PASSWORD" psql -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" -d "$DB_NAME" -c "SELECT version();" &> /dev/null; then
    print_success "数据库连接验证成功"
else
    print_error "数据库连接验证失败"
    exit 1
fi

# 显示配置信息
echo ""
print_success "=========================================="
print_success "  数据库设置完成！"
print_success "=========================================="
echo ""
echo -e "${BLUE}数据库连接信息：${NC}"
echo "  主机: $DB_HOST"
echo "  端口: $DB_PORT"
echo "  用户: $DB_USER"
echo "  密码: ********"
echo "  数据库: $DB_NAME"
echo ""
echo -e "${BLUE}Redis 连接信息：${NC}"
echo "  主机: $REDIS_HOST"
echo "  端口: $REDIS_PORT"
echo "  数据库: $REDIS_DB"
echo ""
echo -e "${BLUE}连接字符串：${NC}"
echo "  PostgreSQL: postgresql://$DB_USER:***@$DB_HOST:$DB_PORT/$DB_NAME"
echo "  Redis: redis://$REDIS_HOST:$REDIS_PORT/$REDIS_DB"
echo ""
echo -e "${BLUE}下一步：${NC}"
echo "  1. 启动项目: ./start.sh"
echo "  2. 访问前端: http://localhost:4444"
echo "  3. 访问 API: http://localhost:3333"
echo ""
print_success "🎉 准备就绪，开始开发吧！"
echo ""
