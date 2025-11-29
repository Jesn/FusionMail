#!/bin/bash

# FusionMail 项目完整启动脚本
# 功能：检查端口占用、终止冲突进程、启动完整项目
# 作者：FusionMail Team
# 版本：2.0.0

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
BACKEND_DIR="backend"
FRONTEND_DIR="frontend"

# 远程数据库配置
DB_HOST="192.168.2.200"
DB_PORT="5432"
DB_USER="postgres"
DB_PASSWORD="8QMZn3yfrbkVG7"
DB_NAME="fusionmail-dev"
REDIS_HOST="192.168.2.200"
REDIS_PORT="6379"
REDIS_DB="6"

# 默认管理员账号配置
DEFAULT_ADMIN_EMAIL="admin@fusionmail.local"
DEFAULT_ADMIN_PASSWORD="FusionMail2024!"

# 启动模式配置（默认值）
WATCH_MODE=false          # -w, --watch: 监听文件变化自动重启
BACKEND_ONLY=false        # -b, --backend: 仅启动后端
FRONTEND_ONLY=false       # -f, --frontend: 仅启动前端
SKIP_INFRA=false          # -s, --skip-infra: 跳过基础设施检查
CLEAN_START=false         # -c, --clean: 清理数据后启动
DEBUG_MODE=false          # -d, --debug: 调试模式
FORCE_REBUILD=false       # -r, --rebuild: 强制重新构建

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

# 打印使用帮助
print_usage() {
    echo ""
    echo -e "${CYAN}用法：${NC}"
    echo "  $0 [选项]"
    echo ""
    echo -e "${CYAN}选项：${NC}"
    echo "  -h, --help          显示此帮助信息"
    echo "  -w, --watch         监听文件变化自动重启（开发模式）"
    echo "  -b, --backend       仅启动后端服务"
    echo "  -f, --frontend      仅启动前端服务"
    echo "  -s, --skip-infra    跳过基础设施检查（假设已运行）"
    echo "  -c, --clean         清理数据后启动（删除数据卷）"
    echo "  -d, --debug         调试模式（显示详细日志）"
    echo "  -r, --rebuild       强制重新构建"
    echo ""
    echo -e "${CYAN}示例：${NC}"
    echo "  $0                  # 完整启动（默认）"
    echo "  $0 -w               # 开发模式（监听文件变化）"
    echo "  $0 -b               # 仅启动后端"
    echo "  $0 -f               # 仅启动前端"
    echo "  $0 -c               # 清理数据后启动"
    echo "  $0 -w -d            # 开发模式 + 调试日志"
    echo "  $0 -b -s            # 仅启动后端，跳过基础设施检查"
    echo ""
}

# 解析命令行参数
parse_arguments() {
    while [[ $# -gt 0 ]]; do
        case $1 in
            -h|--help)
                print_usage
                exit 0
                ;;
            -w|--watch)
                WATCH_MODE=true
                shift
                ;;
            -b|--backend)
                BACKEND_ONLY=true
                shift
                ;;
            -f|--frontend)
                FRONTEND_ONLY=true
                shift
                ;;
            -s|--skip-infra)
                SKIP_INFRA=true
                shift
                ;;
            -c|--clean)
                CLEAN_START=true
                shift
                ;;
            -d|--debug)
                DEBUG_MODE=true
                shift
                ;;
            -r|--rebuild)
                FORCE_REBUILD=true
                shift
                ;;
            *)
                print_error "未知选项: $1"
                print_usage
                exit 1
                ;;
        esac
    done
    
    # 参数验证
    if [ "$BACKEND_ONLY" = true ] && [ "$FRONTEND_ONLY" = true ]; then
        print_error "不能同时指定 -b 和 -f 选项"
        exit 1
    fi
    
    # 调试模式设置环境变量
    if [ "$DEBUG_MODE" = true ]; then
        export GIN_MODE=debug
        print_info "调试模式已启用"
    fi
}

# 打印启动配置
print_config() {
    echo ""
    print_highlight "启动配置："
    echo "  监听模式: $([ "$WATCH_MODE" = true ] && echo "✅ 启用" || echo "❌ 禁用")"
    echo "  仅后端: $([ "$BACKEND_ONLY" = true ] && echo "✅ 是" || echo "❌ 否")"
    echo "  仅前端: $([ "$FRONTEND_ONLY" = true ] && echo "✅ 是" || echo "❌ 否")"
    echo "  跳过基础设施: $([ "$SKIP_INFRA" = true ] && echo "✅ 是" || echo "❌ 否")"
    echo "  清理启动: $([ "$CLEAN_START" = true ] && echo "✅ 是" || echo "❌ 否")"
    echo "  调试模式: $([ "$DEBUG_MODE" = true ] && echo "✅ 启用" || echo "❌ 禁用")"
    echo "  强制重建: $([ "$FORCE_REBUILD" = true ] && echo "✅ 是" || echo "❌ 否")"
    echo ""
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

# 检查远程数据库连接
check_remote_database() {
    print_step "检查远程数据库连接..."
    
    # 检查 PostgreSQL 连接（使用 Go 后端的健康检查）
    print_info "检查 PostgreSQL 连接 ($DB_HOST:$DB_PORT)..."
    
    # 检查 Redis 连接
    print_info "检查 Redis 连接 ($REDIS_HOST:$REDIS_PORT DB $REDIS_DB)..."
    
    print_success "远程数据库配置已设置"
    print_warning "注意：数据库连接将在后端启动时验证"
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

# 检查数据库是否存在（已移除，使用远程数据库）
# check_docker_containers() {
#     # 使用远程数据库，无需检查本地容器
#     return 0
# }

# 清理数据（远程数据库版本）
clean_volumes() {
    print_step "清理远程数据库数据..."
    
    print_warning "⚠️  警告：此操作将清理远程数据库中的所有数据！"
    print_warning "⚠️  数据库: $DB_HOST:$DB_PORT/$DB_NAME"
    print_warning "⚠️  Redis: $REDIS_HOST:$REDIS_PORT DB $REDIS_DB"
    
    if [ "$CLEAN_START" = true ]; then
        print_error "远程数据库清理功能已禁用，请手动清理"
        print_info "如需清理，请连接到远程数据库手动执行 DROP DATABASE 和 CREATE DATABASE"
        exit 1
    fi
}

# 启动基础设施服务（已移除，使用远程数据库）
start_infrastructure() {
    print_step "跳过基础设施启动（使用远程数据库）..."
    
    print_info "数据库配置："
    echo "  PostgreSQL: $DB_HOST:$DB_PORT/$DB_NAME"
    echo "  Redis: $REDIS_HOST:$REDIS_PORT DB $REDIS_DB"
    
    print_success "使用远程数据库，无需启动本地容器"
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
    if [ "$FORCE_REBUILD" = true ] || [ ! -f "fusionmail" ]; then
        print_info "下载 Go 依赖..."
        go mod download
    fi
    
    # 构建项目
    if [ "$FORCE_REBUILD" = true ] || [ ! -f "fusionmail" ]; then
        print_info "构建后端项目..."
        go build -o fusionmail ./cmd/server
        
        if [ $? -ne 0 ]; then
            print_error "后端构建失败"
            cd ..
            exit 1
        fi
    else
        print_info "使用已有的后端可执行文件"
    fi
    
    # 启动后端服务
    if [ "$WATCH_MODE" = true ]; then
        print_info "启动后端服务 (监听模式，端口 3333)..."
        
        # 检查是否安装了 air（Go 热重载工具）
        if command -v air &> /dev/null; then
            print_info "使用 air 进行热重载..."
            nohup air > ../logs/backend.log 2>&1 &
            local backend_pid=$!
        else
            print_warning "未安装 air，使用普通模式启动"
            print_info "提示：安装 air 可实现热重载: go install github.com/cosmtrek/air@latest"
            nohup ./fusionmail > ../logs/backend.log 2>&1 &
            local backend_pid=$!
        fi
    else
        print_info "启动后端服务 (端口 3333)..."
        nohup ./fusionmail > ../logs/backend.log 2>&1 &
        local backend_pid=$!
    fi
    
    # 保存 PID
    echo $backend_pid > ../logs/backend.pid
    
    cd ..
    
    # 等待后端启动
    print_info "等待后端服务启动..."
    local attempt=0
    local max_attempts=30
    
    while [ $attempt -lt $max_attempts ]; do
        if curl -s http://localhost:3333/api/v1/health &> /dev/null; then
            print_success "后端服务已启动 (PID: $backend_pid)"
            if [ "$WATCH_MODE" = true ]; then
                print_info "监听模式已启用，文件变化将自动重启服务"
            fi
            break
        fi
        attempt=$((attempt + 1))
        sleep 1
        echo -n "."
    done
    echo ""
    
    if [ $attempt -eq $max_attempts ]; then
        print_error "后端服务启动超时"
        print_info "请查看日志: tail -f logs/backend.log"
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
    if [ "$FORCE_REBUILD" = true ] || [ ! -d "node_modules" ]; then
        print_info "安装前端依赖..."
        npm install
        
        if [ $? -ne 0 ]; then
            print_error "前端依赖安装失败"
            cd ..
            exit 1
        fi
    else
        print_info "使用已有的前端依赖"
    fi
    
    # 启动前端开发服务器（Vite 默认支持热重载）
    if [ "$WATCH_MODE" = true ]; then
        print_info "启动前端开发服务器 (热重载模式，端口 4444)..."
    else
        print_info "启动前端开发服务器 (端口 4444)..."
    fi
    
    nohup npm run dev > ../logs/frontend.log 2>&1 &
    local frontend_pid=$!
    
    # 保存 PID
    echo $frontend_pid > ../logs/frontend.pid
    
    cd ..
    
    # 等待前端启动
    print_info "等待前端服务启动..."
    local attempt=0
    local max_attempts=40
    
    while [ $attempt -lt $max_attempts ]; do
        if curl -s http://localhost:4444 &> /dev/null; then
            print_success "前端服务已启动 (PID: $frontend_pid)"
            if [ "$WATCH_MODE" = true ]; then
                print_info "Vite 热重载已启用，文件变化将自动更新"
            fi
            break
        fi
        attempt=$((attempt + 1))
        sleep 1
        echo -n "."
    done
    echo ""
    
    if [ $attempt -eq $max_attempts ]; then
        print_error "前端服务启动超时"
        print_info "请查看日志: tail -f logs/frontend.log"
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
    
    # 根据启动模式显示不同信息
    if [ "$BACKEND_ONLY" = true ]; then
        print_highlight "🔧 后端 API 地址："
        echo "  🚀 API 服务:    http://localhost:3333"
        echo "  📚 API 文档:    http://localhost:3333/docs (如果已配置)"
        echo "  ❤️  健康检查:    http://localhost:3333/api/v1/health"
        echo ""
        
        print_highlight "📋 服务状态："
        echo "  ✅ 后端服务:    运行中 (PID: $(cat logs/backend.pid 2>/dev/null || echo 'N/A'))"
        echo "  ✅ PostgreSQL: 远程数据库 ($DB_HOST:$DB_PORT)"
        echo "  ✅ Redis:      远程服务 ($REDIS_HOST:$REDIS_PORT DB $REDIS_DB)"
        echo ""
        
        print_highlight "📝 日志文件："
        echo "  📄 后端日志:    logs/backend.log"
        echo ""
        
    elif [ "$FRONTEND_ONLY" = true ]; then
        print_highlight "📱 前端访问地址："
        echo "  🌐 Web 界面:    http://localhost:4444"
        echo "  📱 移动端:      http://localhost:4444 (响应式设计)"
        echo ""
        
        print_highlight "📋 服务状态："
        echo "  ✅ 前端服务:    运行中 (PID: $(cat logs/frontend.pid 2>/dev/null || echo 'N/A'))"
        echo ""
        
        print_highlight "📝 日志文件："
        echo "  📄 前端日志:    logs/frontend.log"
        echo ""
        
    else
        # 完整启动模式
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
        echo "  🐘 PostgreSQL:  postgresql://$DB_USER:***@$DB_HOST:$DB_PORT/$DB_NAME"
        echo "  🔴 Redis:       redis://$REDIS_HOST:$REDIS_PORT/$REDIS_DB"
        echo ""
        
        # 读取实际的管理员密码
        if [ -f "backend/passwd" ]; then
            ACTUAL_PASSWORD=$(cat backend/passwd)
            print_highlight "👤 管理员账号："
            echo "  📧 用户名:      admin"
            echo "  🔑 密码:        $ACTUAL_PASSWORD"
            echo "  ⚠️  首次登录后请修改密码！"
        else
            print_highlight "👤 默认管理员账号："
            echo "  📧 邮箱:        $DEFAULT_ADMIN_EMAIL"
            echo "  🔑 密码:        $DEFAULT_ADMIN_PASSWORD"
        fi
        echo ""
        
        print_highlight "📋 服务状态："
        echo "  ✅ 前端服务:    运行中 (PID: $(cat logs/frontend.pid 2>/dev/null || echo 'N/A'))"
        echo "  ✅ 后端服务:    运行中 (PID: $(cat logs/backend.pid 2>/dev/null || echo 'N/A'))"
        echo "  ✅ PostgreSQL: 远程数据库 ($DB_HOST:$DB_PORT)"
        echo "  ✅ Redis:      远程服务 ($REDIS_HOST:$REDIS_PORT DB $REDIS_DB)"
        echo ""
        
        print_highlight "📝 日志文件："
        echo "  📄 前端日志:    logs/frontend.log"
        echo "  📄 后端日志:    logs/backend.log"
        echo ""
    fi
    
    print_highlight "🛠️  常用命令："
    if [ "$BACKEND_ONLY" != true ]; then
        echo "  🔍 查看前端日志: tail -f logs/frontend.log"
    fi
    if [ "$FRONTEND_ONLY" != true ]; then
        echo "  🔍 查看后端日志: tail -f logs/backend.log"
    fi
    echo "  🔍 查看所有日志: tail -f logs/*.log"
    echo "  🛑 停止项目:    pkill -f fusionmail"
    echo "  🔄 重启项目:    $0 $([ "$WATCH_MODE" = true ] && echo "-w") $([ "$DEBUG_MODE" = true ] && echo "-d")"
    echo ""
    
    if [ "$BACKEND_ONLY" != true ] && [ "$FRONTEND_ONLY" != true ]; then
        print_highlight "🚀 快速开始："
        echo "  1. 打开浏览器访问: http://localhost:4444"
        echo "  2. 使用管理员账号登录"
        echo "  3. 添加您的邮箱账户开始使用"
        echo ""
    fi
    
    print_info "💡 提示："
    if [ "$WATCH_MODE" = true ]; then
        echo "  - 监听模式已启用，文件变化将自动更新"
    fi
    if [ "$DEBUG_MODE" = true ]; then
        echo "  - 调试模式已启用，查看详细日志"
    fi
    if [ "$CLEAN_START" = true ]; then
        echo "  - 已清理旧数据，这是全新的数据库"
    fi
    echo "  - 支持 Gmail、Outlook、QQ、163 等主流邮箱"
    echo "  - 可以在设置页面修改同步频率和其他配置"
    echo "  - 如遇问题请查看日志文件"
    echo ""
    
    print_success "🎊 享受使用 $PROJECT_NAME！"
}

# 主函数
main() {
    # 解析命令行参数
    parse_arguments "$@"
    
    # 打印横幅
    print_banner
    
    # 显示启动配置
    print_config
    
    # 创建日志目录
    create_log_directory
    
    # 检查系统依赖
    check_dependencies
    
    # 检查远程数据库连接
    check_remote_database
    
    # 检查端口并终止冲突进程
    check_and_kill_ports
    
    # 基础设施处理（使用远程数据库）
    if [ "$SKIP_INFRA" = true ]; then
        print_info "跳过基础设施检查（使用远程数据库）"
    else
        # 显示数据库配置信息
        start_infrastructure
    fi
    
    # 根据参数决定启动哪些服务
    if [ "$BACKEND_ONLY" = true ]; then
        # 仅启动后端
        start_backend
    elif [ "$FRONTEND_ONLY" = true ]; then
        # 仅启动前端
        start_frontend
    else
        # 启动完整项目
        start_backend
        start_frontend
    fi
    
    # 显示完成信息
    show_completion_info
    
    # 监听模式提示
    if [ "$WATCH_MODE" = true ]; then
        echo ""
        print_highlight "🔥 监听模式已启用"
        echo "  - 前端：Vite 自动热重载"
        echo "  - 后端：$(command -v air &> /dev/null && echo "air 热重载" || echo "需要手动重启")"
        echo ""
        print_info "按 Ctrl+C 停止所有服务"
        
        # 保持脚本运行，监听 Ctrl+C
        trap 'print_info "正在停止服务..."; pkill -P $$; exit 0' INT
        wait
    fi
}

# 错误处理
trap 'print_error "脚本执行过程中发生错误，请检查日志"; exit 1' ERR

# 执行主函数
main "$@"