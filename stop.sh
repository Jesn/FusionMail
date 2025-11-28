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
CYAN='\033[0;36m'
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

# 打印使用帮助
print_usage() {
    echo ""
    echo -e "${CYAN}用法：${NC}"
    echo "  $0 [选项]"
    echo ""
    echo -e "${CYAN}选项：${NC}"
    echo "  -h, --help          显示此帮助信息"
    echo "  -a, --all           停止所有服务（包括 Docker）"
    echo "  -b, --backend       仅停止后端服务"
    echo "  -f, --frontend      仅停止前端服务"
    echo "  -d, --docker        仅停止 Docker 容器"
    echo "  -c, --clean         停止并清理数据卷"
    echo ""
    echo -e "${CYAN}示例：${NC}"
    echo "  $0                  # 停止前后端服务（保留 Docker）"
    echo "  $0 -a               # 停止所有服务（包括 Docker）"
    echo "  $0 -b               # 仅停止后端"
    echo "  $0 -f               # 仅停止前端"
    echo "  $0 -c               # 停止并清理所有数据"
    echo ""
}

# 解析命令行参数
STOP_ALL=false
STOP_BACKEND=false
STOP_FRONTEND=false
STOP_DOCKER=false
CLEAN_DATA=false

while [[ $# -gt 0 ]]; do
    case $1 in
        -h|--help)
            print_usage
            exit 0
            ;;
        -a|--all)
            STOP_ALL=true
            shift
            ;;
        -b|--backend)
            STOP_BACKEND=true
            shift
            ;;
        -f|--frontend)
            STOP_FRONTEND=true
            shift
            ;;
        -d|--docker)
            STOP_DOCKER=true
            shift
            ;;
        -c|--clean)
            CLEAN_DATA=true
            shift
            ;;
        *)
            print_error "未知选项: $1"
            print_usage
            exit 1
            ;;
    esac
done

# 如果没有指定选项，默认停止前后端（不停止 Docker）
if [ "$STOP_ALL" = false ] && [ "$STOP_BACKEND" = false ] && [ "$STOP_FRONTEND" = false ] && [ "$STOP_DOCKER" = false ] && [ "$CLEAN_DATA" = false ]; then
    STOP_BACKEND=true
    STOP_FRONTEND=true
fi

# 如果指定了 -a，停止所有服务
if [ "$STOP_ALL" = true ]; then
    STOP_BACKEND=true
    STOP_FRONTEND=true
    STOP_DOCKER=true
fi

echo ""
echo -e "${CYAN}=========================================="
echo -e "    🛑 FusionMail 项目停止脚本"
echo -e "=========================================="
echo -e "${NC}"

# 停止后端服务
if [ "$STOP_BACKEND" = true ]; then
    print_info "停止后端服务..."
    
    # 从 PID 文件读取并停止
    if [ -f "logs/backend.pid" ]; then
        backend_pid=$(cat logs/backend.pid)
        if kill -0 $backend_pid 2>/dev/null; then
            kill -TERM $backend_pid 2>/dev/null || true
            sleep 2
            
            # 如果进程仍在运行，强制终止
            if kill -0 $backend_pid 2>/dev/null; then
                kill -KILL $backend_pid 2>/dev/null || true
            fi
            
            print_success "后端服务已停止 (PID: $backend_pid)"
        else
            print_warning "后端服务未运行"
        fi
        rm -f logs/backend.pid
    else
        # 尝试通过进程名终止
        pkill -f "fusionmail" 2>/dev/null && print_success "后端服务已停止" || print_warning "后端服务未运行"
    fi
fi

# 停止前端服务
if [ "$STOP_FRONTEND" = true ]; then
    print_info "停止前端服务..."
    
    # 从 PID 文件读取并停止
    if [ -f "logs/frontend.pid" ]; then
        frontend_pid=$(cat logs/frontend.pid)
        if kill -0 $frontend_pid 2>/dev/null; then
            kill -TERM $frontend_pid 2>/dev/null || true
            sleep 2
            
            # 如果进程仍在运行，强制终止
            if kill -0 $frontend_pid 2>/dev/null; then
                kill -KILL $frontend_pid 2>/dev/null || true
            fi
            
            print_success "前端服务已停止 (PID: $frontend_pid)"
        else
            print_warning "前端服务未运行"
        fi
        rm -f logs/frontend.pid
    else
        # 尝试通过端口终止
        lsof -ti :4444 | xargs kill -TERM 2>/dev/null && print_success "前端服务已停止" || print_warning "前端服务未运行"
    fi
fi

# 停止 Docker 容器
if [ "$STOP_DOCKER" = true ] || [ "$CLEAN_DATA" = true ]; then
    print_info "停止 Docker 容器..."
    
    if [ "$CLEAN_DATA" = true ]; then
        print_warning "⚠️  警告：将删除所有数据库数据和缓存！"
        docker-compose -f docker-compose.dev.yml down -v
        print_success "Docker 容器已停止并清理数据卷"
    else
        docker-compose -f docker-compose.dev.yml down
        print_success "Docker 容器已停止"
    fi
fi

echo ""
print_success "=========================================="
print_success "🎉 服务停止完成！"
print_success "=========================================="
echo ""

# 显示停止的服务
print_info "已停止的服务："
if [ "$STOP_BACKEND" = true ]; then
    echo "  ✅ 后端服务"
fi
if [ "$STOP_FRONTEND" = true ]; then
    echo "  ✅ 前端服务"
fi
if [ "$STOP_DOCKER" = true ] || [ "$CLEAN_DATA" = true ]; then
    echo "  ✅ Docker 容器 (PostgreSQL + Redis)"
fi
if [ "$CLEAN_DATA" = true ]; then
    echo "  ✅ 数据卷已清理"
fi
echo ""

print_info "重新启动项目："
echo "  ./start.sh"
echo ""
