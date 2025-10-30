#!/bin/bash

# FusionMail 项目停止脚本
# 功能：优雅停止所有服务
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
    echo -e "    🛑 $PROJECT_NAME 项目停止脚本"
    echo -e "=========================================="
    echo -e "${NC}"
}

# 停止前端服务
stop_frontend() {
    print_step "停止前端服务..."
    
    if [ -f "logs/frontend.pid" ]; then
        local frontend_pid=$(cat logs/frontend.pid)
        if kill -0 $frontend_pid 2>/dev/null; then
            print_info "停止前端服务 (PID: $frontend_pid)..."
            kill -TERM $frontend_pid 2>/dev/null || true
            sleep 2
            
            # 如果进程仍然存在，强制终止
            if kill -0 $frontend_pid 2>/dev/null; then
                kill -KILL $frontend_pid 2>/dev/null || true
                print_warning "已强制终止前端服务"
            else
                print_success "前端服务已停止"
            fi
        else
            print_warning "前端服务进程不存在"
        fi
        rm -f logs/frontend.pid
    else
        print_warning "未找到前端服务 PID 文件"
    fi
}

# 停止后端服务
stop_backend() {
    print_step "停止后端服务..."
    
    if [ -f "logs/backend.pid" ]; then
        local backend_pid=$(cat logs/backend.pid)
        if kill -0 $backend_pid 2>/dev/null; then
            print_info "停止后端服务 (PID: $backend_pid)..."
            kill -TERM $backend_pid 2>/dev/null || true
            sleep 2
            
            # 如果进程仍然存在，强制终止
            if kill -0 $backend_pid 2>/dev/null; then
                kill -KILL $backend_pid 2>/dev/null || true
                print_warning "已强制终止后端服务"
            else
                print_success "后端服务已停止"
            fi
        else
            print_warning "后端服务进程不存在"
        fi
        rm -f logs/backend.pid
    else
        print_warning "未找到后端服务 PID 文件"
    fi
}

# 停止基础设施服务
stop_infrastructure() {
    print_step "停止基础设施服务..."
    
    if [ -f "docker-compose.dev.yml" ]; then
        print_info "停止 PostgreSQL 和 Redis..."
        docker-compose -f docker-compose.dev.yml down
        
        if [ $? -eq 0 ]; then
            print_success "基础设施服务已停止"
        else
            print_error "停止基础设施服务时出现错误"
        fi
    else
        print_warning "docker-compose.dev.yml 文件不存在"
    fi
}

# 清理端口占用
cleanup_ports() {
    print_step "清理端口占用..."
    
    local ports=(3000 8080)
    local port_names=("前端服务" "后端API")
    
    for i in "${!ports[@]}"; do
        local port="${ports[$i]}"
        local service_name="${port_names[$i]}"
        
        local pids=$(lsof -ti :$port 2>/dev/null || true)
        
        if [ -n "$pids" ]; then
            print_info "清理端口 $port ($service_name) 上的进程..."
            for pid in $pids; do
                kill -TERM $pid 2>/dev/null || true
                sleep 1
                if kill -0 $pid 2>/dev/null; then
                    kill -KILL $pid 2>/dev/null || true
                fi
            done
            print_success "端口 $port 已清理"
        fi
    done
}

# 显示停止完成信息
show_completion_info() {
    echo ""
    print_success "=========================================="
    print_success "🛑 $PROJECT_NAME 项目已停止"
    print_success "=========================================="
    echo ""
    
    print_info "已停止的服务："
    echo "  ❌ 前端服务 (端口 3000)"
    echo "  ❌ 后端服务 (端口 8080)"
    echo "  ❌ PostgreSQL (端口 5432)"
    echo "  ❌ Redis (端口 6379)"
    echo ""
    
    print_info "日志文件已保留："
    echo "  📄 前端日志: logs/frontend.log"
    echo "  📄 后端日志: logs/backend.log"
    echo ""
    
    print_info "重新启动项目："
    echo "  🚀 运行: ./start.sh"
    echo ""
}

# 主函数
main() {
    # 打印横幅
    print_banner
    
    # 停止前端服务
    stop_frontend
    
    # 停止后端服务
    stop_backend
    
    # 停止基础设施服务
    stop_infrastructure
    
    # 清理端口占用
    cleanup_ports
    
    # 显示完成信息
    show_completion_info
}

# 错误处理
trap 'print_error "停止脚本执行过程中发生错误"; exit 1' ERR

# 执行主函数
main "$@"