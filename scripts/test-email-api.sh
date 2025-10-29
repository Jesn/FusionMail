#!/bin/bash

# FusionMail 邮件 API 测试脚本
# 用于测试邮件管理相关的 API 接口

set -e

# 配置
API_BASE_URL="${API_BASE_URL:-http://localhost:8080/api/v1}"
ACCOUNT_UID="${ACCOUNT_UID:-}"

# 颜色输出
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# 打印函数
print_success() {
    echo -e "${GREEN}✓ $1${NC}"
}

print_error() {
    echo -e "${RED}✗ $1${NC}"
}

print_info() {
    echo -e "${YELLOW}ℹ $1${NC}"
}

print_section() {
    echo ""
    echo "========================================="
    echo "$1"
    echo "========================================="
}

# 检查服务器是否运行
check_server() {
    print_section "检查服务器状态"
    
    if curl -s "${API_BASE_URL}/health" > /dev/null; then
        print_success "服务器运行正常"
    else
        print_error "服务器未运行，请先启动服务器"
        exit 1
    fi
}

# 测试获取邮件列表
test_get_email_list() {
    print_section "测试：获取邮件列表"
    
    local url="${API_BASE_URL}/emails?page=1&page_size=10"
    if [ -n "$ACCOUNT_UID" ]; then
        url="${url}&account_uid=${ACCOUNT_UID}"
    fi
    
    response=$(curl -s "$url")
    
    if echo "$response" | jq -e '.emails' > /dev/null 2>&1; then
        total=$(echo "$response" | jq -r '.total')
        count=$(echo "$response" | jq -r '.emails | length')
        print_success "获取邮件列表成功：共 $total 封邮件，当前页 $count 封"
        echo "$response" | jq '.'
    else
        print_error "获取邮件列表失败"
        echo "$response"
        return 1
    fi
}

# 测试获取邮件详情
test_get_email_detail() {
    print_section "测试：获取邮件详情"
    
    # 先获取第一封邮件的 ID
    local url="${API_BASE_URL}/emails?page=1&page_size=1"
    if [ -n "$ACCOUNT_UID" ]; then
        url="${url}&account_uid=${ACCOUNT_UID}"
    fi
    
    email_id=$(curl -s "$url" | jq -r '.emails[0].id')
    
    if [ "$email_id" = "null" ] || [ -z "$email_id" ]; then
        print_info "没有邮件可供测试"
        return 0
    fi
    
    response=$(curl -s "${API_BASE_URL}/emails/${email_id}")
    
    if echo "$response" | jq -e '.id' > /dev/null 2>&1; then
        subject=$(echo "$response" | jq -r '.subject')
        print_success "获取邮件详情成功：ID=$email_id, 主题=$subject"
        echo "$response" | jq '.'
    else
        print_error "获取邮件详情失败"
        echo "$response"
        return 1
    fi
}

# 测试搜索邮件
test_search_emails() {
    print_section "测试：搜索邮件"
    
    local query="${1:-test}"
    local url="${API_BASE_URL}/emails/search?q=${query}&page=1&page_size=10"
    if [ -n "$ACCOUNT_UID" ]; then
        url="${url}&account_uid=${ACCOUNT_UID}"
    fi
    
    response=$(curl -s "$url")
    
    if echo "$response" | jq -e '.emails' > /dev/null 2>&1; then
        total=$(echo "$response" | jq -r '.total')
        print_success "搜索邮件成功：找到 $total 封匹配的邮件"
        echo "$response" | jq '.'
    else
        print_error "搜索邮件失败"
        echo "$response"
        return 1
    fi
}

# 测试获取未读邮件数
test_get_unread_count() {
    print_section "测试：获取未读邮件数"
    
    local url="${API_BASE_URL}/emails/unread-count"
    if [ -n "$ACCOUNT_UID" ]; then
        url="${url}?account_uid=${ACCOUNT_UID}"
    fi
    
    response=$(curl -s "$url")
    
    if echo "$response" | jq -e '.unread_count' > /dev/null 2>&1; then
        count=$(echo "$response" | jq -r '.unread_count')
        print_success "获取未读邮件数成功：$count 封未读"
        echo "$response" | jq '.'
    else
        print_error "获取未读邮件数失败"
        echo "$response"
        return 1
    fi
}

# 测试获取账户统计
test_get_account_stats() {
    print_section "测试：获取账户统计"
    
    if [ -z "$ACCOUNT_UID" ]; then
        print_info "未指定 ACCOUNT_UID，跳过此测试"
        return 0
    fi
    
    response=$(curl -s "${API_BASE_URL}/emails/stats/${ACCOUNT_UID}")
    
    if echo "$response" | jq -e '.total_count' > /dev/null 2>&1; then
        total=$(echo "$response" | jq -r '.total_count')
        unread=$(echo "$response" | jq -r '.unread_count')
        starred=$(echo "$response" | jq -r '.starred_count')
        archived=$(echo "$response" | jq -r '.archived_count')
        print_success "获取账户统计成功：总计=$total, 未读=$unread, 星标=$starred, 归档=$archived"
        echo "$response" | jq '.'
    else
        print_error "获取账户统计失败"
        echo "$response"
        return 1
    fi
}

# 测试标记邮件为已读
test_mark_as_read() {
    print_section "测试：标记邮件为已读"
    
    # 先获取一封未读邮件的 ID
    local url="${API_BASE_URL}/emails?is_read=false&page=1&page_size=1"
    if [ -n "$ACCOUNT_UID" ]; then
        url="${url}&account_uid=${ACCOUNT_UID}"
    fi
    
    email_id=$(curl -s "$url" | jq -r '.emails[0].id')
    
    if [ "$email_id" = "null" ] || [ -z "$email_id" ]; then
        print_info "没有未读邮件可供测试"
        return 0
    fi
    
    response=$(curl -s -X POST "${API_BASE_URL}/emails/mark-read" \
        -H "Content-Type: application/json" \
        -d "{\"ids\": [$email_id]}")
    
    if echo "$response" | jq -e '.message' > /dev/null 2>&1; then
        print_success "标记邮件为已读成功：ID=$email_id"
        echo "$response" | jq '.'
    else
        print_error "标记邮件为已读失败"
        echo "$response"
        return 1
    fi
}

# 测试切换星标状态
test_toggle_star() {
    print_section "测试：切换星标状态"
    
    # 先获取第一封邮件的 ID
    local url="${API_BASE_URL}/emails?page=1&page_size=1"
    if [ -n "$ACCOUNT_UID" ]; then
        url="${url}&account_uid=${ACCOUNT_UID}"
    fi
    
    email_id=$(curl -s "$url" | jq -r '.emails[0].id')
    
    if [ "$email_id" = "null" ] || [ -z "$email_id" ]; then
        print_info "没有邮件可供测试"
        return 0
    fi
    
    response=$(curl -s -X POST "${API_BASE_URL}/emails/${email_id}/toggle-star")
    
    if echo "$response" | jq -e '.message' > /dev/null 2>&1; then
        print_success "切换星标状态成功：ID=$email_id"
        echo "$response" | jq '.'
    else
        print_error "切换星标状态失败"
        echo "$response"
        return 1
    fi
}

# 主函数
main() {
    echo "FusionMail 邮件 API 测试"
    echo "========================"
    echo "API Base URL: $API_BASE_URL"
    if [ -n "$ACCOUNT_UID" ]; then
        echo "Account UID: $ACCOUNT_UID"
    else
        echo "Account UID: (未指定，将测试所有账户)"
    fi
    
    # 检查依赖
    if ! command -v jq &> /dev/null; then
        print_error "需要安装 jq 工具：brew install jq"
        exit 1
    fi
    
    # 运行测试
    check_server
    test_get_email_list
    test_get_email_detail
    test_search_emails "test"
    test_get_unread_count
    test_get_account_stats
    test_mark_as_read
    test_toggle_star
    
    print_section "测试完成"
    print_success "所有测试通过！"
}

# 运行主函数
main "$@"
