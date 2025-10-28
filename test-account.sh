#!/bin/bash

# FusionMail 账户测试脚本
# 用于添加邮箱账户并测试同步功能

API_BASE="http://localhost:8080/api/v1"

echo "=== FusionMail 账户测试脚本 ==="
echo ""

# 颜色定义
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# 1. 检查服务器状态
echo "1. 检查服务器状态..."
HEALTH=$(curl -s "${API_BASE}/health")
if [ $? -eq 0 ]; then
    echo -e "${GREEN}✓ 服务器运行正常${NC}"
    echo "   响应: $HEALTH"
else
    echo -e "${RED}✗ 服务器未运行，请先启动服务器${NC}"
    exit 1
fi
echo ""

# 2. 添加测试账户
echo "2. 添加测试账户..."
echo -e "${YELLOW}请输入邮箱信息：${NC}"

read -p "邮箱地址 (例如: your@qq.com): " EMAIL
read -p "邮箱提供商 (icloud/qq/163/gmail/outlook): " PROVIDER
read -sp "密码或授权码: " PASSWORD
echo ""
read -p "协议 (imap/pop3，默认 imap): " PROTOCOL
PROTOCOL=${PROTOCOL:-imap}

echo ""
echo "正在添加账户..."

RESPONSE=$(curl -s -X POST "${API_BASE}/accounts" \
  -H "Content-Type: application/json" \
  -d "{
    \"email\": \"${EMAIL}\",
    \"provider\": \"${PROVIDER}\",
    \"protocol\": \"${PROTOCOL}\",
    \"auth_type\": \"password\",
    \"password\": \"${PASSWORD}\",
    \"sync_enabled\": true,
    \"sync_interval\": 5
  }")

# 检查是否成功
if echo "$RESPONSE" | grep -q "Account created successfully"; then
    echo -e "${GREEN}✓ 账户添加成功${NC}"
    ACCOUNT_UID=$(echo "$RESPONSE" | grep -o '"uid":"[^"]*"' | cut -d'"' -f4)
    echo "   账户 UID: $ACCOUNT_UID"
else
    echo -e "${RED}✗ 账户添加失败${NC}"
    echo "   错误: $RESPONSE"
    exit 1
fi
echo ""

# 3. 测试连接
echo "3. 测试账户连接..."
TEST_RESPONSE=$(curl -s -X POST "${API_BASE}/accounts/${ACCOUNT_UID}/test")

if echo "$TEST_RESPONSE" | grep -q '"success":true'; then
    echo -e "${GREEN}✓ 连接测试成功${NC}"
else
    echo -e "${RED}✗ 连接测试失败${NC}"
    echo "   错误: $TEST_RESPONSE"
    echo ""
    echo -e "${YELLOW}提示：${NC}"
    echo "   - 请检查邮箱地址和密码是否正确"
    echo "   - QQ/163 邮箱需要使用授权码而非登录密码"
    echo "   - Gmail 需要开启 IMAP 并使用应用专用密码"
    exit 1
fi
echo ""

# 4. 手动触发同步
echo "4. 手动触发邮件同步..."
SYNC_RESPONSE=$(curl -s -X POST "${API_BASE}/sync/accounts/${ACCOUNT_UID}")

if echo "$SYNC_RESPONSE" | grep -q "Sync started"; then
    echo -e "${GREEN}✓ 同步已启动${NC}"
    echo "   同步将在后台进行，请稍候..."
else
    echo -e "${RED}✗ 同步启动失败${NC}"
    echo "   错误: $SYNC_RESPONSE"
fi
echo ""

# 5. 查看账户列表
echo "5. 查看所有账户..."
ACCOUNTS=$(curl -s "${API_BASE}/accounts")
echo "$ACCOUNTS" | python3 -m json.tool 2>/dev/null || echo "$ACCOUNTS"
echo ""

# 6. 查看同步状态
echo "6. 查看同步状态..."
sleep 3  # 等待同步开始
STATUS=$(curl -s "${API_BASE}/sync/status")
echo "$STATUS" | python3 -m json.tool 2>/dev/null || echo "$STATUS"
echo ""

echo "=== 测试完成 ==="
echo ""
echo -e "${GREEN}后续操作：${NC}"
echo "1. 查看账户详情: curl ${API_BASE}/accounts/${ACCOUNT_UID}"
echo "2. 手动同步: curl -X POST ${API_BASE}/sync/accounts/${ACCOUNT_UID}"
echo "3. 同步所有账户: curl -X POST ${API_BASE}/sync/all"
echo "4. 查看同步状态: curl ${API_BASE}/sync/status"
echo ""
echo -e "${YELLOW}注意：${NC}"
echo "- 首次同步会拉取最近 30 天的邮件"
echo "- 同步间隔默认为 5 分钟"
echo "- 可以通过 API 修改同步间隔"
