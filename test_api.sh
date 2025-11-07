#!/bin/bash

# API 响应格式测试脚本

BASE_URL="http://localhost:3333/api/v1"
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

echo "🧪 FusionMail API 响应格式测试"
echo "================================"
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

# 测试 1: 获取不存在的账户
echo "🧪 测试 1: 获取不存在的账户 (应返回 404 + 错误码 2000)"
echo "-----------------------------------------------------------"
RESPONSE=$(curl -s -w "\n%{http_code}" -X GET "$BASE_URL/accounts/non-existent-uid")
HTTP_CODE=$(echo "$RESPONSE" | tail -n1)
BODY=$(echo "$RESPONSE" | sed '$d')

echo "HTTP 状态码: $HTTP_CODE"
echo "响应内容:"
echo "$BODY" | jq . 2>/dev/null || echo "$BODY"

if [ "$HTTP_CODE" = "404" ]; then
    if echo "$BODY" | jq -e '.code == 2000' > /dev/null 2>&1; then
        echo -e "${GREEN}✅ 测试通过：返回正确的错误码 2000${NC}"
    else
        echo -e "${YELLOW}⚠️  警告：HTTP 状态码正确，但错误码不是 2000${NC}"
    fi
else
    echo -e "${RED}❌ 测试失败：HTTP 状态码应为 404，实际为 $HTTP_CODE${NC}"
fi
echo ""

# 测试 2: 无效的请求参数
echo "🧪 测试 2: 无效的请求参数 (应返回 400)"
echo "-----------------------------------------------------------"
RESPONSE=$(curl -s -w "\n%{http_code}" -X POST "$BASE_URL/accounts" \
  -H "Content-Type: application/json" \
  -d '{"email": "invalid"}')
HTTP_CODE=$(echo "$RESPONSE" | tail -n1)
BODY=$(echo "$RESPONSE" | sed '$d')

echo "HTTP 状态码: $HTTP_CODE"
echo "响应内容:"
echo "$BODY" | jq . 2>/dev/null || echo "$BODY"

if [ "$HTTP_CODE" = "400" ]; then
    echo -e "${GREEN}✅ 测试通过：返回 400 错误${NC}"
else
    echo -e "${RED}❌ 测试失败：HTTP 状态码应为 400，实际为 $HTTP_CODE${NC}"
fi
echo ""

# 测试 3: 获取账户列表
echo "🧪 测试 3: 获取账户列表 (应返回 200 + success: true)"
echo "-----------------------------------------------------------"
RESPONSE=$(curl -s -w "\n%{http_code}" -X GET "$BASE_URL/accounts")
HTTP_CODE=$(echo "$RESPONSE" | tail -n1)
BODY=$(echo "$RESPONSE" | sed '$d')

echo "HTTP 状态码: $HTTP_CODE"
echo "响应内容:"
echo "$BODY" | jq . 2>/dev/null || echo "$BODY"

if [ "$HTTP_CODE" = "200" ]; then
    if echo "$BODY" | jq -e '.success == true' > /dev/null 2>&1; then
        echo -e "${GREEN}✅ 测试通过：返回成功响应${NC}"
    else
        echo -e "${YELLOW}⚠️  警告：HTTP 状态码正确，但 success 不为 true${NC}"
    fi
else
    echo -e "${RED}❌ 测试失败：HTTP 状态码应为 200，实际为 $HTTP_CODE${NC}"
fi
echo ""

# 测试 4: 获取不存在的邮件
echo "🧪 测试 4: 获取不存在的邮件 (应返回 404 + 错误码 6000)"
echo "-----------------------------------------------------------"
RESPONSE=$(curl -s -w "\n%{http_code}" -X GET "$BASE_URL/emails/999999")
HTTP_CODE=$(echo "$RESPONSE" | tail -n1)
BODY=$(echo "$RESPONSE" | sed '$d')

echo "HTTP 状态码: $HTTP_CODE"
echo "响应内容:"
echo "$BODY" | jq . 2>/dev/null || echo "$BODY"

if [ "$HTTP_CODE" = "404" ]; then
    if echo "$BODY" | jq -e '.code == 6000' > /dev/null 2>&1; then
        echo -e "${GREEN}✅ 测试通过：返回正确的错误码 6000${NC}"
    else
        echo -e "${YELLOW}⚠️  警告：HTTP 状态码正确，但错误码不是 6000${NC}"
    fi
else
    echo -e "${RED}❌ 测试失败：HTTP 状态码应为 404，实际为 $HTTP_CODE${NC}"
fi
echo ""

# 总结
echo "================================"
echo "🎉 测试完成"
echo ""
echo "📋 验证要点："
echo "  1. 所有错误响应都包含 success: false"
echo "  2. 业务错误包含 code 字段"
echo "  3. HTTP 状态码与错误类型匹配"
echo "  4. 成功响应包含 success: true 和 data 字段"
echo ""
echo "💡 提示："
echo "  - 如果测试失败，请检查后端服务是否正常运行"
echo "  - 查看完整的测试文档: TEST_API_RESPONSES.md"
echo ""
