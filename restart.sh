#!/bin/bash

# FusionMail 项目重启脚本
# 功能：优雅重启所有服务
# 作者：FusionMail Team
# 版本：1.0.0

set -e

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
PURPLE='\033[0;35m'
CYAN='\033[0;36m'
NC='\033[0m' # No Color

# 项目配置
PROJECT_NAME="FusionMail"

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

print_step() {
    echo -e "${PURPLE}[STEP]${NC} $1"
}

# 打印横幅
print_banner() {
    echo ""
    echo -e "${CYAN}=========================================="
    echo -e "    🔄 $PROJECT_NAME 项目重启脚本"
    echo -e "=========================================="
    echo -e "${NC}"
}

# 主函数
main() {
    # 打印横幅
    print_banner
    
    print_step "开始重启 $PROJECT_NAME 项目..."
    echo ""
    
    # 检查脚本是否存在
    if [ ! -f "stop.sh" ]; then
        print_error "stop.sh 脚本不存在"
        exit 1
    fi
    
    if [ ! -f "start.sh" ]; then
        print_error "start.sh 脚本不存在"
        exit 1
    fi
    
    # 停止服务
    print_info "第一步：停止现有服务..."
    ./stop.sh
    
    echo ""
    print_info "等待 3 秒后启动服务..."
    sleep 3
    echo ""
    
    # 启动服务
    print_info "第二步：启动所有服务..."
    ./start.sh
    
    echo ""
    print_success "🎉 $PROJECT_NAME 项目重启完成！"
}

# 错误处理
trap 'print_error "重启脚本执行过程中发生错误"; exit 1' ERR

# 执行主函数
main "$@"