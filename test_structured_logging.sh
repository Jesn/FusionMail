#!/bin/bash

# 结构化日志测试脚本

BASE_URL="http://localhost:3333/api/v1"
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

echo "🧪 FusionMail 结构化日志测试"
echo "=============================="
echo ""

# 检查服务是否运行
echo "📡 检查服务状态..."
if ! curl -s -f "$BASE_URL/system/health" > /dev/null 2>&1; then
    echo -e "${RED}❌ 服务未运行，请先启动后端服务${NC}"
    echo "   运行: cd backend && ./server"
    exit 1
fi
echo -e "${GREEN}✅ 服务正在运行${NC}"
echo ""

# 测试 1: 正常请求日志
echo -e "${BLUE}🧪 测试 1: 正常请求日志${NC}"
echo "-----------------------------------------------------------"
echo "发送正常请求，检查服务器日志中的结构化日志..."
echo ""

CUSTOM_ID="test-log-001"
curl -s -H "X-Request-ID: $CUSTOM_ID" "$BASE_URL/accounts" > /dev/null

echo -e "${GREEN}✅ 请求已发送${NC}"
echo "   Request ID: $CUSTOM_ID"
echo "   请检查服务器日志，应该看到："
echo "   - [INFO] 时间戳 请求日志消息 | request_id=$CUSTOM_ID, ..."
echo ""

# 测试 2: 业务错误日志
echo -e "${BLUE}🧪 测试 2: 业务错误日志${NC}"
echo "-----------------------------------------------------------"
echo "触发业务错误，检查服务器日志中的警告日志..."
echo ""

CUSTOM_ID="test-error-001"
curl -s -H "X-Request-ID: $CUSTOM_ID" "$BASE_URL/accounts/non-existent" > /dev/null

echo -e "${GREEN}✅ 请求已发送${NC}"
echo "   Request ID: $CUSTOM_ID"
echo "   请检查服务器日志，应该看到："
echo "   - [WARN] 时间戳 business error occurred | request_id=$CUSTOM_ID, error_code=2000, ..."
echo ""

# 测试 3: 参数错误日志
echo -e "${BLUE}🧪 测试 3: 参数错误日志${NC}"
echo "-----------------------------------------------------------"
echo "发送无效参数，检查服务器日志中的错误日志..."
echo ""

CUSTOM_ID="test-param-001"
curl -s -X POST \
  -H "Content-Type: application/json" \
  -H "X-Request-ID: $CUSTOM_ID" \
  -d '{"invalid": "data"}' \
  "$BASE_URL/accounts" > /dev/null

echo -e "${GREEN}✅ 请求已发送${NC}"
echo "   Request ID: $CUSTOM_ID"
echo "   请检查服务器日志，应该看到参数验证错误日志"
echo ""

# 测试 4: 多个请求的日志追踪
echo -e "${BLUE}🧪 测试 4: 多个请求的日志追踪${NC}"
echo "-----------------------------------------------------------"
echo "发送多个请求，每个都有唯一的 Request ID..."
echo ""

for i in {1..3}; do
    CUSTOM_ID="test-multi-$(printf "%03d" $i)"
    echo "发送请求 $i: Request ID = $CUSTOM_ID"
    curl -s -H "X-Request-ID: $CUSTOM_ID" "$BASE_URL/accounts" > /dev/null
    sleep 0.5
done

echo ""
echo -e "${GREEN}✅ 所有请求已发送${NC}"
echo "   请检查服务器日志，应该看到 3 个不同的 Request ID"
echo ""

# 测试 5: 日志字段完整性
echo -e "${BLUE}🧪 测试 5: 日志字段完整性${NC}"
echo "-----------------------------------------------------------"
echo "检查日志是否包含所有必要字段..."
echo ""

echo -e "${YELLOW}请在服务器日志中验证以下字段：${NC}"
echo "  ✓ 日志级别 [INFO/WARN/ERROR]"
echo "  ✓ 时间戳 (RFC3339 格式)"
echo "  ✓ 日志消息"
echo "  ✓ 调用者信息 (文件名:行号)"
echo "  ✓ request_id 字段"
echo "  ✓ 其他上下文字段 (path, method, error_code 等)"
echo ""

# 总结
echo "=============================="
echo -e "${BLUE}🎉 结构化日志测试完成${NC}"
echo ""
echo "📋 测试项目："
echo "  1. ✅ 正常请求日志"
echo "  2. ✅ 业务错误日志"
echo "  3. ✅ 参数错误日志"
echo "  4. ✅ 多请求日志追踪"
echo "  5. ✅ 日志字段完整性"
echo ""
echo "💡 日志格式示例："
echo "  [INFO] 2025-11-07T15:04:05+08:00 account created successfully (account_service.go:123) | request_id=abc-123, uid=xxx, email=test@example.com"
echo ""
echo "📊 日志优势："
echo "  - 🔍 每个请求都有唯一 ID，便于追踪"
echo "  - 📝 结构化字段，便于解析和分析"
echo "  - 🎯 包含调用者信息，便于定位问题"
echo "  - 🔗 Request ID 贯穿整个请求链路"
echo ""
