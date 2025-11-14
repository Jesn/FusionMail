#!/bin/bash

# FusionMail 项目完整启动脚本
# 功能：检查端口占用、终止冲突进程、启动完整项目
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
REQUIRED_PORTS=(4444 3333)
PORT_NAMES=("前端服务" "后端API")
DOCKER_CONTAINERS=("fusionmail-postgres" "fusionmail-redis")
CONTAINER_NAMES=("PostgreSQL" "Redis")
BACKEND_DIR="backend"
FRONTEND_DIR="frontend"

# 默认账号密码配置
DEFAULT_ADMIN_EMAIL="admin@fusionmail.local"
DEFAULT_ADMIN_PASSWORD="FusionMail2024!"
DB_USER="fusionmail"
DB_PASSWORD="fusionmail_dev_password"
REDIS_PASSWORD="fusionmail_redis_password"

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

print_highlight() {
    echo -e "${CYAN}[HIGHLIGHT]${NC} $1"
}

# 打印横幅
print_banner() {
    echo ""
    echo -e "${CYAN}=========================================="
    echo -e "    🚀 $PROJECT_NAME 项目启动脚本"
    echo -e "=========================================="
    echo -e "${NC}"
}

# 检查系统依赖
check_dependencies() {
    print_step "检查系统依赖..."
    
    local missing_deps=()
    
    # 检查 Docker
    if ! command -v docker &> /dev/null; then
        missing_deps+=("docker")
    fi
    
    # 检查 Docker Compose
    if ! command -v docker-compose &> /dev/null; then
        missing_deps+=("docker-compose")
    fi
    
    # 检查 Node.js
    if ! command -v node &> /dev/null; then
        missing_deps+=("node")
    fi
    
    # 检查 Go
    if ! command -v go &> /dev/null; then
        missing_deps+=("go")
    fi
    
    # 检查 lsof
    if ! command -v lsof &> /dev/null; then
        missing_deps+=("lsof")
    fi
    
    if [ ${#missing_deps[@]} -ne 0 ]; then
        print_error "缺少以下依赖："
        for dep in "${missing_deps[@]}"; do
            echo "  - $dep"
        done
        print_info "请安装缺少的依赖后重新运行脚本"
        exit 1
    fi
    
    print_success "系统依赖检查通过"
}

# 检查 Docker 服务状态
check_docker_service() {
    print_step "检查 Docker 服务状态..."
    
    if ! docker info &> /dev/null; then
        print_error "Docker 服务未运行"
        print_info "请启动 Docker 服务后重新运行脚本"
        exit 1
    fi
    
    print_success "Docker 服务运行正常"
}

# 检查端口占用并终止冲突进程
check_and_kill_ports() {
    print_step "检查端口占用情况..."
    
    local killed_processes=()
    
    for i in "${!REQUIRED_PORTS[@]}"; do
        local port="${REQUIRED_PORTS[$i]}"
        local service_name="${PORT_NAMES[$i]}"
        
        # 查找占用端口的进程
        local pids=$(lsof -ti :$port 2>/dev/null || true)
        
        if [ -n "$pids" ]; then
            print_warning "端口 $port ($service_name) 被以下进程占用，正在终止..."

            # 显示进程信息
            for pid in $pids; do
                local process_info=$(ps -p $pid -o pid,ppid,comm,args --no-headers 2>/dev/null || echo "$pid unknown unknown")
                echo "  PID: $process_info"
            done

            # 直接终止进程
            for pid in $pids; do
                if kill -TERM $pid 2>/dev/null; then
                    print_info "已发送 TERM 信号给进程 $pid"
                    sleep 2

                    # 如果进程仍然存在，强制终止
                    if kill -0 $pid 2>/dev/null; then
                        if kill -KILL $pid 2>/dev/null; then
                            print_warning "已强制终止进程 $pid"
                        fi
                    fi

                    killed_processes+=("$pid ($service_name)")
                else
                    print_error "无法终止进程 $pid"
                fi
            done
        else
            print_success "端口 $port ($service_name) 可用"
        fi
    done
    
    if [ ${#killed_processes[@]} -gt 0 ]; then
        print_info "已终止以下进程："
        for process in "${killed_processes[@]}"; do
            echo "  - $process"
        done
        sleep 1
    fi
}

# 检查 Docker 容器状态
check_docker_containers() {
    print_step "检查 Docker 容器状态..."
    
    local containers_to_start=()
    
    for i in "${!DOCKER_CONTAINERS[@]}"; do
        local container_name="${DOCKER_CONTAINERS[$i]}"
        local service_name="${CONTAINER_NAMES[$i]}"
        
        # 检查容器是否存在且正在运行
        if docker ps --format "table {{.Names}}" | grep -q "^${container_name}$"; then
            print_success "容器 $container_name ($service_name) 正在运行"
        elif docker ps -a --format "table {{.Names}}" | grep -q "^${container_name}$"; then
            print_warning "容器 $container_name ($service_name) 存在但未运行"
            containers_to_start+=("$container_name")
        else
            print_info "容器 $container_name ($service_name) 不存在，需要创建"
            containers_to_start+=("$container_name")
        fi
    done
    
    # 如果有容器需要启动，返回标志
    if [ ${#containers_to_start[@]} -gt 0 ]; then
        print_info "需要启动以下容器："
        for container in "${containers_to_start[@]}"; do
            echo "  - $container"
        done
        return 1  # 需要启动基础设施
    else
        print_success "所有 Docker 容器都在运行"
        return 0  # 跳过基础设施启动
    fi
}

# 启动基础设施服务 (PostgreSQL + Redis)
start_infrastructure() {
    print_step "启动基础设施服务 (PostgreSQL + Redis)..."
    
    # 检查 docker-compose.dev.yml 是否存在
    if [ ! -f "docker-compose.dev.yml" ]; then
        print_error "docker-compose.dev.yml 文件不存在"
        exit 1
    fi
    
    # 启动基础设施
    print_info "启动 PostgreSQL 和 Redis 容器..."
    docker-compose -f docker-compose.dev.yml up -d
    
    if [ $? -ne 0 ]; then
        print_error "基础设施启动失败"
        exit 1
    fi
    
    # 等待服务就绪
    print_info "等待服务就绪..."
    local max_attempts=30
    local attempt=0
    
    # 等待 PostgreSQL
    print_info "等待 PostgreSQL 启动..."
    while [ $attempt -lt $max_attempts ]; do
        if docker exec fusionmail-postgres pg_isready -U fusionmail &> /dev/null; then
            print_success "PostgreSQL 已就绪"
            break
        fi
        attempt=$((attempt + 1))
        sleep 1
        echo -n "."
    done
    echo ""
    
    if [ $attempt -eq $max_attempts ]; then
        print_error "PostgreSQL 启动超时"
        exit 1
    fi
    
    # 等待 Redis
    print_info "等待 Redis 启动..."
    attempt=0
    while [ $attempt -lt $max_attempts ]; do
        if docker exec fusionmail-redis redis-cli -a fusionmail_redis_password ping &> /dev/null 2>&1; then
            print_success "Redis 已就绪"
            break
        fi
        attempt=$((attempt + 1))
        sleep 1
        echo -n "."
    done
    echo ""
    
    if [ $attempt -eq $max_attempts ]; then
        print_error "Redis 启动超时"
        exit 1
    fi
    
    print_success "基础设施服务启动完成"
}

# 启动后端服务
start_backend() {
    print_step "启动后端服务..."
    
    # 检查后端目录
    if [ ! -d "$BACKEND_DIR" ]; then
        print_error "后端目录 $BACKEND_DIR 不存在"
        exit 1
    fi
    
    cd "$BACKEND_DIR"
    
    # 检查环境变量文件
    if [ ! -f ".env" ]; then
        if [ -f ".env.example" ]; then
            print_info "复制环境变量配置文件..."
            cp .env.example .env
            print_success "已创建 .env 文件"
        else
            print_error ".env.example 文件不存在"
            cd ..
            exit 1
        fi
    fi
    
    # 检查 Go 模块
    if [ ! -f "go.mod" ]; then
        print_error "go.mod 文件不存在"
        cd ..
        exit 1
    fi
    
    # 下载依赖
    print_info "下载 Go 依赖..."
    go mod download
    
    # 构建项目
    print_info "构建后端项目..."
    go build -o fusionmail ./cmd/server
    
    if [ $? -ne 0 ]; then
        print_error "后端构建失败"
        cd ..
        exit 1
    fi
    
    # 启动后端服务
    print_info "启动后端服务 (端口 3333)..."
    nohup ./fusionmail > ../logs/backend.log 2>&1 &
    local backend_pid=$!
    
    # 保存 PID
    echo $backend_pid > ../logs/backend.pid
    
    cd ..
    
    # 等待后端启动
    print_info "等待后端服务启动..."
    local attempt=0
    local max_attempts=20
    
    while [ $attempt -lt $max_attempts ]; do
        if curl -s http://localhost:3333/api/v1/health &> /dev/null; then
            print_success "后端服务已启动 (PID: $backend_pid)"
            break
        fi
        attempt=$((attempt + 1))
        sleep 1
        echo -n "."
    done
    echo ""
    
    if [ $attempt -eq $max_attempts ]; then
        print_error "后端服务启动超时"
        exit 1
    fi
}

# 启动前端服务
start_frontend() {
    print_step "启动前端服务..."
    
    # 检查前端目录
    if [ ! -d "$FRONTEND_DIR" ]; then
        print_error "前端目录 $FRONTEND_DIR 不存在"
        exit 1
    fi
    
    cd "$FRONTEND_DIR"
    
    # 检查 package.json
    if [ ! -f "package.json" ]; then
        print_error "package.json 文件不存在"
        cd ..
        exit 1
    fi
    
    # 检查 node_modules
    if [ ! -d "node_modules" ]; then
        print_info "安装前端依赖..."
        npm install
        
        if [ $? -ne 0 ]; then
            print_error "前端依赖安装失败"
            cd ..
            exit 1
        fi
    fi
    
    # 启动前端开发服务器
    print_info "启动前端开发服务器 (端口 4444)..."
    nohup npm run dev > ../logs/frontend.log 2>&1 &
    local frontend_pid=$!
    
    # 保存 PID
    echo $frontend_pid > ../logs/frontend.pid
    
    cd ..
    
    # 等待前端启动
    print_info "等待前端服务启动..."
    local attempt=0
    local max_attempts=30
    
    while [ $attempt -lt $max_attempts ]; do
        if curl -s http://localhost:4444 &> /dev/null; then
            print_success "前端服务已启动 (PID: $frontend_pid)"
            break
        fi
        attempt=$((attempt + 1))
        sleep 1
        echo -n "."
    done
    echo ""
    
    if [ $attempt -eq $max_attempts ]; then
        print_error "前端服务启动超时"
        exit 1
    fi
}

# 创建日志目录
create_log_directory() {
    if [ ! -d "logs" ]; then
        mkdir -p logs
        print_info "已创建日志目录"
    fi
}

# 显示启动完成信息
show_completion_info() {
    echo ""
    print_success "=========================================="
    print_success "🎉 $PROJECT_NAME 项目启动完成！"
    print_success "=========================================="
    echo ""
    
    print_highlight "📱 前端访问地址："
    echo "  🌐 Web 界面:    http://localhost:4444"
    echo "  📱 移动端:      http://localhost:4444 (响应式设计)"
    echo ""
    
    print_highlight "🔧 后端 API 地址："
    echo "  🚀 API 服务:    http://localhost:3333"
    echo "  📚 API 文档:    http://localhost:3333/docs (如果已配置)"
    echo "  ❤️  健康检查:    http://localhost:3333/api/v1/health"
    echo ""
    
    print_highlight "🗄️  数据库连接信息："
    echo "  🐘 PostgreSQL:  postgresql://$DB_USER:$DB_PASSWORD@localhost:5432/fusionmail"
    echo "  🔴 Redis:       redis://:$REDIS_PASSWORD@localhost:6379/0"
    echo ""
    
    print_highlight "👤 默认管理员账号："
    echo "  📧 邮箱:        $DEFAULT_ADMIN_EMAIL"
    echo "  🔑 密码:        $DEFAULT_ADMIN_PASSWORD"
    echo ""
    
    print_highlight "📋 服务状态："
    echo "  ✅ 前端服务:    运行中 (PID: $(cat logs/frontend.pid 2>/dev/null || echo 'N/A'))"
    echo "  ✅ 后端服务:    运行中 (PID: $(cat logs/backend.pid 2>/dev/null || echo 'N/A'))"
    echo "  ✅ PostgreSQL: 运行中 (Docker)"
    echo "  ✅ Redis:      运行中 (Docker)"
    echo ""
    
    print_highlight "📝 日志文件："
    echo "  📄 前端日志:    logs/frontend.log"
    echo "  📄 后端日志:    logs/backend.log"
    echo "  📄 Docker 日志: docker-compose -f docker-compose.dev.yml logs -f"
    echo ""
    
    print_highlight "🛠️  常用命令："
    echo "  🔍 查看前端日志: tail -f logs/frontend.log"
    echo "  🔍 查看后端日志: tail -f logs/backend.log"
    echo "  🔍 查看所有日志: tail -f logs/*.log"
    echo "  🛑 停止项目:    ./stop.sh (需要创建)"
    echo "  🔄 重启项目:    ./restart.sh (需要创建)"
    echo ""
    
    print_highlight "🚀 快速开始："
    echo "  1. 打开浏览器访问: http://localhost:4444"
    echo "  2. 使用默认账号登录: $DEFAULT_ADMIN_EMAIL"
    echo "  3. 添加您的邮箱账户开始使用"
    echo ""
    
    print_info "💡 提示："
    echo "  - 首次使用请先添加邮箱账户"
    echo "  - 支持 Gmail、Outlook、QQ、163 等主流邮箱"
    echo "  - 可以在设置页面修改同步频率和其他配置"
    echo "  - 如遇问题请查看日志文件或联系技术支持"
    echo ""
    
    print_success "🎊 享受使用 $PROJECT_NAME！"
}

# 主函数
main() {
    # 打印横幅
    print_banner
    
    # 创建日志目录
    create_log_directory
    
    # 检查系统依赖
    check_dependencies
    
    # 检查 Docker 服务
    check_docker_service
    
    # 检查端口并终止冲突进程
    check_and_kill_ports
    
    # 检查 Docker 容器状态
    if check_docker_containers; then
        print_info "Docker 容器已运行，跳过基础设施启动"
    else
        # 启动基础设施服务
        start_infrastructure
    fi
    
    # 启动后端服务
    start_backend
    
    # 启动前端服务
    start_frontend
    
    # 显示完成信息
    show_completion_info
}

# 错误处理
trap 'print_error "脚本执行过程中发生错误，请检查日志"; exit 1' ERR

# 执行主函数
main "$@"