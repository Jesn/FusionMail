#!/bin/bash

# FusionMail 质量检查脚本
# 用途：一键执行后端和前端的本地质量门禁

set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
RUN_BACKEND=true
RUN_FRONTEND=true

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

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

show_help() {
    cat <<EOF
用法: ./scripts/check-quality.sh [选项]

选项:
  -b, --backend-only    仅运行后端检查
  -f, --frontend-only   仅运行前端检查
  -h, --help            显示帮助信息

默认检查:
  后端: go test ./...
  前端: npm run lint, npm test, npm run build
EOF
}

parse_args() {
    while [ $# -gt 0 ]; do
        case "$1" in
            -b|--backend-only)
                RUN_BACKEND=true
                RUN_FRONTEND=false
                ;;
            -f|--frontend-only)
                RUN_BACKEND=false
                RUN_FRONTEND=true
                ;;
            -h|--help)
                show_help
                exit 0
                ;;
            *)
                print_error "未知选项: $1"
                show_help
                exit 1
                ;;
        esac
        shift
    done
}

check_command() {
    local command_name="$1"
    local install_hint="$2"

    if ! command -v "$command_name" > /dev/null 2>&1; then
        print_error "未找到命令: $command_name"
        print_info "$install_hint"
        exit 1
    fi
}

run_step() {
    local title="$1"
    shift

    print_info "$title"
    "$@"
    print_success "$title 通过"
}

run_backend_checks() {
    check_command "go" "请先安装 Go 1.21+"

    cd "$ROOT_DIR/backend"
    run_step "后端测试: go test ./..." go test ./...
}

run_frontend_checks() {
    check_command "npm" "请先安装 Node.js 和 npm"

    cd "$ROOT_DIR/frontend"
    if [ ! -d "node_modules" ]; then
        print_error "frontend/node_modules 不存在"
        print_info "请先运行: cd frontend && npm install"
        exit 1
    fi

    run_step "前端 lint: npm run lint" npm run lint
    run_step "前端测试: npm test" npm test
    run_step "前端构建: npm run build" npm run build
}

main() {
    parse_args "$@"

    echo ""
    print_info "=========================================="
    print_info "FusionMail 质量检查"
    print_info "=========================================="
    echo ""

    if [ "$RUN_BACKEND" = true ]; then
        run_backend_checks
    else
        print_warning "跳过后端检查"
    fi

    if [ "$RUN_FRONTEND" = true ]; then
        run_frontend_checks
    else
        print_warning "跳过前端检查"
    fi

    echo ""
    print_success "所有质量检查通过"
}

main "$@"
