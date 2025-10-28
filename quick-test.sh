#!/bin/bash

# FusionMail 快速测试脚本（一键运行）
# 自动添加 QQ 邮箱账户并测试同步

API_BASE="http://localhost:8080/api/v1"

# 从环境变量或配置文件读取
EMAIL="${TEST_EMAIL:-}"
PASSWORD="${TEST_PASSWORD:-}"

if [ -z "$EMAIL" ] && [ -f ".test-config" ]; then
    source .test-config
fi

if [ -z "$EMAIL" ] || [ -z "$PASSWORD" ]; then
    echo "错误: 缺少测试账号信息"
    echo "请设置 TEST_EMAIL 和 TEST_PASSWORD 环境变量"
    echo "或创建 .test-config 文件"
    exit 1
fi

# 颜色
GREEN='\033[0;32m'
RED='\033[0;31m'
BLUE='\033[0;34m'
NC='\033[0m'

echo "FusionMail 快速测试"
echo "===================="
echo ""

# 检查服务器
echo -n "检查服务器... "
if curl -s "${API_BASE}/health" > /dev/null 2>&1; then
    echo -e "${GREEN}✓${NC}"
else
    echo -e "${RED}✗ 服务器未运行${NC}"
    exit 1
fi

# 添加账户
echo -n "添加账户... "
RESPONSE=$(curl -s -X POST "${API_BASE}/accounts" \
  -H "Content-Type: application/json" \
  -d "{
    \"email\": \"${EMAIL}\",
    \"provider\": \"qq\",
    \"protocol\": \"imap\",
    \"auth_type\": \"password\",
    \"password\": \"${PASSWORD}\",
    \"sync_enabled\": true,
    \"sync_interval\": 5
  }")

if echo "$RESPONSE" | grep -q "uid"; then
    ACCOUNT_UID=$(echo "$RESPONSE" | grep -o '"uid":"[^"]*"' | cut -d'"' -f4)
    echo -e "${GREEN}✓${NC} ($ACCOUNT_UID)"
else
    # 获取现有账户
    ACCOUNTS=$(curl -s "${API_BASE}/accounts")
    ACCOUNT_UID=$(echo "$ACCOUNTS" | grep -o '"uid":"[^"]*"' | head -1 | cut -d'"' -f4)
    echo -e "${GREEN}✓${NC} (已存在: $ACCOUNT_UID)"
fi

# 测试连接
echo -n "测试连接... "
if curl -s -X POST "${API_BASE}/accounts/${ACCOUNT_UID}/test" | grep -q '"success":true'; then
    echo -e "${GREEN}✓${NC}"
else
    echo -e "${RED}✗${NC}"
    exit 1
fi

# 触发同步
echo -n "触发同步... "
if curl -s -X POST "${API_BASE}/sync/accounts/${ACCOUNT_UID}" | grep -q "Sync started"; then
    echo -e "${GREEN}✓${NC}"
else
    echo -e "${RED}✗${NC}"
fi

# 等待
echo -n "等待同步... "
sleep 10
echo -e "${GREEN}✓${NC}"

# 查看结果
echo ""
echo "同步结果:"
echo "--------"
curl -s "${API_BASE}/accounts/${ACCOUNT_UID}" | python3 -c "
import sys, json
data = json.load(sys.stdin)
if 'data' in data:
    acc = data['data']
    print(f\"  邮箱: {acc.get('email', 'N/A')}\")
    print(f\"  状态: {acc.get('last_sync_status', 'N/A')}\")
    print(f\"  邮件数: {acc.get('total_emails', 0)}\")
    print(f\"  未读数: {acc.get('unread_count', 0)}\")
" 2>/dev/null || echo "  无法解析结果"

echo ""
echo "数据库邮件数:"
docker exec fusionmail-postgres psql -U fusionmail -d fusionmail -t -c "SELECT COUNT(*) FROM emails;" 2>/dev/null | tr -d ' ' || echo "  无法查询"

echo ""
echo -e "${BLUE}测试完成！${NC}"
echo ""
echo "账户 UID: $ACCOUNT_UID"
echo ""
echo "查看详情: curl ${API_BASE}/accounts/${ACCOUNT_UID}"
echo "再次同步: curl -X POST ${API_BASE}/sync/accounts/${ACCOUNT_UID}"
