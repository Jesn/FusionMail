#!/bin/bash

# FusionMail QQ 邮箱自动化测试脚本
# 使用测试账号自动添加账户并测试同步功能

API_BASE="http://localhost:8080/api/v1"

# 测试账号信息（从环境变量或配置文件读取）
# 方式1: 从环境变量读取
EMAIL="${TEST_EMAIL:-}"
PASSWORD="${TEST_PASSWORD:-}"
PROVIDER="${TEST_PROVIDER:-qq}"
PROTOCOL="${TEST_PROTOCOL:-imap}"

# 方式2: 从配置文件读取（如果环境变量未设置）
if [ -z "$EMAIL" ] && [ -f ".test-config" ]; then
    source .test-config
fi

# 检查必需参数
if [ -z "$EMAIL" ] || [ -z "$PASSWORD" ]; then
    echo -e "${RED}错误: 缺少测试账号信息${NC}"
    echo ""
    echo "请通过以下方式之一提供测试账号："
    echo ""
    echo "方式1: 使用环境变量"
    echo "  export TEST_EMAIL='your@qq.com'"
    echo "  export TEST_PASSWORD='your_authorization_code'"
    echo "  ./test-qq-account.sh"
    echo ""
    echo "方式2: 创建配置文件 .test-config"
    echo "  echo 'EMAIL=\"your@qq.com\"' > .test-config"
    echo "  echo 'PASSWORD=\"your_authorization_code\"' >> .test-config"
    echo "  ./test-qq-account.sh"
    echo ""
    exit 1
fi

# 颜色定义
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

echo "=========================================="
echo "  FusionMail QQ 邮箱自动化测试"
echo "=========================================="
echo ""

# 1. 检查服务器状态
echo -e "${BLUE}[1/8] 检查服务器状态...${NC}"
HEALTH=$(curl -s "${API_BASE}/health")
if [ $? -eq 0 ]; then
    echo -e "${GREEN}✓ 服务器运行正常${NC}"
    echo "   响应: $HEALTH"
else
    echo -e "${RED}✗ 服务器未运行，请先启动服务器${NC}"
    echo "   启动命令: cd backend && ./bin/server"
    exit 1
fi
echo ""

# 2. 添加测试账户
echo -e "${BLUE}[2/8] 添加 QQ 邮箱账户...${NC}"
echo "   邮箱: $EMAIL"
echo "   提供商: $PROVIDER"
echo "   协议: $PROTOCOL"

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
    
    # 保存 UID 到文件，方便后续使用
    echo "$ACCOUNT_UID" > .test_account_uid
else
    echo -e "${RED}✗ 账户添加失败${NC}"
    echo "   错误响应: $RESPONSE"
    
    # 检查是否是账户已存在
    if echo "$RESPONSE" | grep -q "duplicate"; then
        echo -e "${YELLOW}   提示: 账户可能已存在，尝试获取现有账户...${NC}"
        
        # 获取账户列表
        ACCOUNTS=$(curl -s "${API_BASE}/accounts")
        ACCOUNT_UID=$(echo "$ACCOUNTS" | grep -o '"uid":"[^"]*"' | head -1 | cut -d'"' -f4)
        
        if [ -n "$ACCOUNT_UID" ]; then
            echo -e "${GREEN}✓ 找到现有账户${NC}"
            echo "   账户 UID: $ACCOUNT_UID"
            echo "$ACCOUNT_UID" > .test_account_uid
        else
            echo -e "${RED}✗ 无法获取账户信息${NC}"
            exit 1
        fi
    else
        exit 1
    fi
fi
echo ""

# 3. 查看账户详情
echo -e "${BLUE}[3/8] 查看账户详情...${NC}"
ACCOUNT_DETAIL=$(curl -s "${API_BASE}/accounts/${ACCOUNT_UID}")
echo "$ACCOUNT_DETAIL" | python3 -m json.tool 2>/dev/null || echo "$ACCOUNT_DETAIL"
echo ""

# 4. 测试连接
echo -e "${BLUE}[4/8] 测试账户连接...${NC}"
TEST_RESPONSE=$(curl -s -X POST "${API_BASE}/accounts/${ACCOUNT_UID}/test")

if echo "$TEST_RESPONSE" | grep -q '"success":true'; then
    echo -e "${GREEN}✓ 连接测试成功${NC}"
    echo "   可以正常连接到 QQ 邮箱 IMAP 服务器"
else
    echo -e "${RED}✗ 连接测试失败${NC}"
    echo "   错误: $TEST_RESPONSE"
    echo ""
    echo -e "${YELLOW}可能的原因：${NC}"
    echo "   1. 授权码已过期或失效"
    echo "   2. QQ 邮箱 IMAP 服务未开启"
    echo "   3. 网络连接问题"
    exit 1
fi
echo ""

# 5. 手动触发同步
echo -e "${BLUE}[5/8] 手动触发邮件同步...${NC}"
SYNC_RESPONSE=$(curl -s -X POST "${API_BASE}/sync/accounts/${ACCOUNT_UID}")

if echo "$SYNC_RESPONSE" | grep -q "Sync started"; then
    echo -e "${GREEN}✓ 同步已启动${NC}"
    echo "   同步将在后台进行..."
else
    echo -e "${RED}✗ 同步启动失败${NC}"
    echo "   错误: $SYNC_RESPONSE"
fi
echo ""

# 6. 等待同步完成
echo -e "${BLUE}[6/8] 等待同步完成...${NC}"
echo -n "   等待中"
for i in {1..10}; do
    sleep 1
    echo -n "."
done
echo ""
echo -e "${GREEN}✓ 等待完成${NC}"
echo ""

# 7. 查看同步状态
echo -e "${BLUE}[7/8] 查看同步状态...${NC}"
STATUS=$(curl -s "${API_BASE}/sync/status")
echo "$STATUS" | python3 -m json.tool 2>/dev/null || echo "$STATUS"
echo ""

# 8. 再次查看账户详情（包含同步结果）
echo -e "${BLUE}[8/8] 查看账户同步结果...${NC}"
ACCOUNT_DETAIL=$(curl -s "${API_BASE}/accounts/${ACCOUNT_UID}")
echo "$ACCOUNT_DETAIL" | python3 -m json.tool 2>/dev/null || echo "$ACCOUNT_DETAIL"
echo ""

# 提取关键信息
TOTAL_EMAILS=$(echo "$ACCOUNT_DETAIL" | grep -o '"total_emails":[0-9]*' | cut -d':' -f2)
UNREAD_COUNT=$(echo "$ACCOUNT_DETAIL" | grep -o '"unread_count":[0-9]*' | cut -d':' -f2)
LAST_SYNC_STATUS=$(echo "$ACCOUNT_DETAIL" | grep -o '"last_sync_status":"[^"]*"' | cut -d'"' -f4)

echo "=========================================="
echo "  测试结果汇总"
echo "=========================================="
echo ""
echo -e "${GREEN}账户信息：${NC}"
echo "  邮箱地址: $EMAIL"
echo "  账户 UID: $ACCOUNT_UID"
echo ""
echo -e "${GREEN}同步结果：${NC}"
echo "  同步状态: $LAST_SYNC_STATUS"
echo "  邮件总数: ${TOTAL_EMAILS:-0}"
echo "  未读数量: ${UNREAD_COUNT:-0}"
echo ""

if [ "$LAST_SYNC_STATUS" = "success" ]; then
    echo -e "${GREEN}✓ 测试完全成功！${NC}"
else
    echo -e "${YELLOW}⚠ 同步状态异常，请查看服务器日志${NC}"
fi

echo ""
echo "=========================================="
echo "  后续操作"
echo "=========================================="
echo ""
echo "1. 查看所有账户:"
echo "   curl ${API_BASE}/accounts"
echo ""
echo "2. 再次手动同步:"
echo "   curl -X POST ${API_BASE}/sync/accounts/${ACCOUNT_UID}"
echo ""
echo "3. 同步所有账户:"
echo "   curl -X POST ${API_BASE}/sync/all"
echo ""
echo "4. 查看同步状态:"
echo "   curl ${API_BASE}/sync/status"
echo ""
echo "5. 删除测试账户:"
echo "   curl -X DELETE ${API_BASE}/accounts/${ACCOUNT_UID}"
echo ""
echo "6. 查看数据库中的邮件:"
echo "   docker exec -it fusionmail-postgres psql -U fusionmail -d fusionmail -c 'SELECT COUNT(*) FROM emails;'"
echo ""
echo "7. 查看最新邮件:"
echo "   docker exec -it fusionmail-postgres psql -U fusionmail -d fusionmail -c 'SELECT id, subject, from_address, sent_at FROM emails ORDER BY sent_at DESC LIMIT 10;'"
echo ""

# 保存账户 UID 供后续使用
echo -e "${YELLOW}提示: 账户 UID 已保存到 .test_account_uid 文件${NC}"
echo ""
