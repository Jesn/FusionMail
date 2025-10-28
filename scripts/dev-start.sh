#!/bin/bash

# FusionMail 开发环境启动脚本
# 用途：一键启动 PostgreSQL 和 Redis 基础设施服务

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

# 检查 Docker 是否安装
check_docker() {
    if ! command -v docker &> /dev/null; then
        print_error "Docker 未安装，请先安装 Docker"
        print_info "访问 https://docs.docker.com/get-docker/ 获取安装指南"
        exit 1
    fi
    
    if ! command -v docker-compose &> /dev/null; then
        print_error "Docker Compose 未安装，请先安装 Docker Compose"
        print_info "访问 https://docs.docker.com/compose/install/ 获取安装指南"
        exit 1
    fi
}

# 检查 Docker 服务是否运行
check_docker_running() {
    if ! docker info &> /dev/null; then
        print_error "Docker 服务未运行，请先启动 Docker"
        exit 1
    fi
}

# 检查端口是否被占用
check_ports() {
    local ports=("5432" "6379")
    local port_names=("PostgreSQL" "Redis")
    local has_conflict=false
    
    for i in "${!ports[@]}"; do
        local port="${ports[$i]}"
        local name="${port_names[$i]}"
        
        if lsof -Pi :$port -sTCP:LISTEN -t >/dev/null 2>&1; then
            print_warning "端口 $port ($name) 已被占用"
            has_conflict=true
        fi
    done
    
    if [ "$has_conflict" = true ]; then
        print_warning "部分端口已被占用，可能导致服务启动失败"
        read -p "是否继续启动？(y/n) " -n 1 -r
        echo
        if [[ ! $REPLY =~ ^[Yy]$ ]]; then
            print_info "已取消启动"
            exit 0
        fi
    fi
}

# 启动服务
start_services() {
    print_info "正在启动 FusionMail 开发环境基础设施..."
    
    # 启动服务
    docker-compose -f docker-compose.dev.yml up -d
    
    if [ $? -eq 0 ]; then
        print_success "服务启动命令已执行"
    else
        print_error "服务启动失败"
        exit 1
    fi
}

# 等待服务就绪
wait_for_services() {
    print_info "等待服务启动..."
    
    local max_attempts=30
    local attempt=0
    
    # 等待 PostgreSQL
    print_info "等待 PostgreSQL 就绪..."
    while [ $attempt -lt $max_attempts ]; do
        if docker exec fusionmail-postgres pg_isready -U fusionmail &> /dev/null; then
            print_success "PostgreSQL 已就绪"
            break
        fi
        attempt=$((attempt + 1))
        sleep 1
    done
    
    if [ $attempt -eq $max_attempts ]; then
        print_error "PostgreSQL 启动超时"
        exit 1
    fi
    
    # 等待 Redis
    print_info "等待 Redis 就绪..."
    attempt=0
    while [ $attempt -lt $max_attempts ]; do
        if docker exec fusionmail-redis redis-cli -a fusionmail_redis_password ping &> /dev/null; then
            print_success "Redis 已就绪"
            break
        fi
        attempt=$((attempt + 1))
        sleep 1
    done
    
    if [ $attempt -eq $max_attempts ]; then
        print_error "Redis 启动超时"
        exit 1
    fi
}

# 显示服务信息
show_service_info() {
    echo ""
    print_success "=========================================="
    print_success "FusionMail 开发环境已启动"
    print_success "=========================================="
    echo ""
    
    print_info "PostgreSQL 连接信息："
    echo "  Host:     localhost"
    echo "  Port:     5432"
    echo "  Database: fusionmail"
    echo "  Username: fusionmail"
    echo "  Password: fusionmail_dev_password"
    echo "  URL:      postgresql://fusionmail:fusionmail_dev_password@localhost:5432/fusionmail"
    echo ""
    
    print_info "Redis 连接信息："
    echo "  Host:     localhost"
    echo "  Port:     6379"
    echo "  Password: fusionmail_redis_password"
    echo "  URL:      redis://:fusionmail_redis_password@localhost:6379/0"
    echo ""
    
    print_info "常用命令："
    echo "  查看日志:   docker-compose -f docker-compose.dev.yml logs -f"
    echo "  停止服务:   ./scripts/dev-stop.sh"
    echo "  重启服务:   docker-compose -f docker-compose.dev.yml restart"
    echo "  查看状态:   docker-compose -f docker-compose.dev.yml ps"
    echo ""
    
    print_info "数据库管理："
    echo "  进入 PostgreSQL: docker exec -it fusionmail-postgres psql -U fusionmail -d fusionmail"
    echo "  进入 Redis:      docker exec -it fusionmail-redis redis-cli -a fusionmail_redis_password"
    echo ""
}

# 主函数
main() {
    echo ""
    print_info "=========================================="
    print_info "FusionMail 开发环境启动脚本"
    print_info "=========================================="
    echo ""
    
    # 检查环境
    check_docker
    check_docker_running
    check_ports
    
    # 启动服务
    start_services
    wait_for_services
    
    # 显示信息
    show_service_info
    
    print_success "开发环境启动完成！"
}

# 执行主函数
main
