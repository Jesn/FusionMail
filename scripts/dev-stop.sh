#!/bin/bash

# FusionMail 开发环境停止脚本
# 用途：停止 PostgreSQL 和 Redis 基础设施服务

set -e

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# 打印带颜色的消息
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

# 停止服务
stop_services() {
    print_info "正在停止 FusionMail 开发环境基础设施..."
    
    docker-compose -f docker-compose.dev.yml down
    
    if [ $? -eq 0 ]; then
        print_success "服务已停止"
    else
        print_error "服务停止失败"
        exit 1
    fi
}

# 询问是否删除数据卷
ask_remove_volumes() {
    echo ""
    print_warning "是否删除数据卷？（这将删除所有数据库数据和 Redis 缓存）"
    read -p "删除数据卷？(y/n) " -n 1 -r
    echo
    
    if [[ $REPLY =~ ^[Yy]$ ]]; then
        print_info "正在删除数据卷..."
        docker-compose -f docker-compose.dev.yml down -v
        print_success "数据卷已删除"
    else
        print_info "保留数据卷"
    fi
}

# 主函数
main() {
    echo ""
    print_info "=========================================="
    print_info "FusionMail 开发环境停止脚本"
    print_info "=========================================="
    echo ""
    
    # 停止服务
    stop_services
    
    # 询问是否删除数据卷
    ask_remove_volumes
    
    echo ""
    print_success "开发环境已停止"
    echo ""
}

# 执行主函数
main
