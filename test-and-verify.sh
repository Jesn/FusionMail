#!/bin/bash

# FusionMail 完整测试和验证脚本
# 包含 API 测试和数据库验证

API_BASE="http://localhost:8080/api/v1"

# 测试账号信息（从环境变量或配置文件读取）
EMAIL="${TEST_EMAIL:-}"
PASSWORD="${TEST_PASSWORD:-}"
PROVIDER="${TEST_PROVIDER:-qq}"
PROTOCOL="${TEST_PROTOCOL:-imap}"

# 从配置文件读取（如果环境变量未设置）
if [ -z "$EMAIL" ] && [ -f ".test-config" ]; then
    source .test-config
fi

# 检查必需参数
if [ -z "$EMAIL" ] || [ -z "$PASSWORD" ]; then
    echo -e "${RED}错误: 缺少测试账号信息${NC}"
    echo ""
    echo "请设置环境变量或创建 .test-config 文件"
    echo "详见: docs/testing-guide.md"
    exit 1
fi

# 颜色定义
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
CYAN='\033[0;36m'
NC='\033[0m'

echo "=========================================="
echo "  FusionMail 完整测试和验证"
echo "=========================================="
echo ""

# 检查 Docker 容器状态
echo -e "${CYAN}检查 Docker 容器状态...${NC}"
if docker ps | grep -q fusionmail-postgres; then
    echo -e "${GREEN}✓ PostgreSQL 容器运行中${NC}"
else
    echo -e "${RED}✗ PostgreSQL 容器未运行${NC}"
    echo "   启动命令: docker-compose up -d"
    exit 1
fi

if docker ps | grep -q fusionmail-redis; then
    echo -e "${GREEN}✓ Redis 容器运行中${NC}"
else
    echo -e "${YELLOW}⚠ Redis 容器未运行（可选）${NC}"
fi
echo ""

# 1. 检查服务器
echo -e "${BLUE}[1/10] 检查服务器状态...${NC}"
HEALTH=$(curl -s "${API_BASE}/health")
if [ $? -eq 0 ]; then
    echo -e "${GREEN}✓ 服务器运行正常${NC}"
else
    echo -e "${RED}✗ 服务器未运行${NC}"
    echo "   启动命令: cd backend && ./bin/server"
    exit 1
fi
echo ""

# 2. 清理旧数据（可选）
echo -e "${BLUE}[2/10] 清理旧测试数据...${NC}"
read -p "是否清理旧的测试账户？(y/N): " CLEAN
if [ "$CLEAN" = "y" ] || [ "$CLEAN" = "Y" ]; then
    # 获取所有账户
    ACCOUNTS=$(curl -s "${API_BASE}/accounts")
    
    # 提取所有 UID 并删除
    UIDS=$(echo "$ACCOUNTS" | grep -o '"uid":"[^"]*"' | cut -d'"' -f4)
    
    for uid in $UIDS; do
        echo "   删除账户: $uid"
        curl -s -X DELETE "${API_BASE}/accounts/${uid}" > /dev/null
    done
    
    echo -e "${GREEN}✓ 清理完成${NC}"
else
    echo "   跳过清理"
fi
echo ""

# 3. 添加账户
echo -e "${BLUE}[3/10] 添加 QQ 邮箱账户...${NC}"
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

if echo "$RESPONSE" | grep -q "Account created successfully"; then
    ACCOUNT_UID=$(echo "$RESPONSE" | grep -o '"uid":"[^"]*"' | cut -d'"' -f4)
    echo -e "${GREEN}✓ 账户添加成功${NC}"
    echo "   UID: $ACCOUNT_UID"
else
    # 尝试获取现有账户
    ACCOUNTS=$(curl -s "${API_BASE}/accounts")
    ACCOUNT_UID=$(echo "$ACCOUNTS" | grep -o '"uid":"[^"]*"' | head -1 | cut -d'"' -f4)
    
    if [ -n "$ACCOUNT_UID" ]; then
        echo -e "${YELLOW}⚠ 使用现有账户${NC}"
        echo "   UID: $ACCOUNT_UID"
    else
        echo -e "${RED}✗ 无法添加或获取账户${NC}"
        exit 1
    fi
fi
echo ""

# 4. 测试连接
echo -e "${BLUE}[4/10] 测试 IMAP 连接...${NC}"
TEST_RESPONSE=$(curl -s -X POST "${API_BASE}/accounts/${ACCOUNT_UID}/test")

if echo "$TEST_RESPONSE" | grep -q '"success":true'; then
    echo -e "${GREEN}✓ IMAP 连接成功${NC}"
else
    echo -e "${RED}✗ IMAP 连接失败${NC}"
    echo "   $TEST_RESPONSE"
    exit 1
fi
echo ""

# 5. 查看数据库初始状态
echo -e "${BLUE}[5/10] 查看数据库初始状态...${NC}"
INITIAL_COUNT=$(docker exec fusionmail-postgres psql -U fusionmail -d fusionmail -t -c "SELECT COUNT(*) FROM emails;" 2>/dev/null | tr -d ' ')
echo "   当前邮件数: ${INITIAL_COUNT:-0}"
echo ""

# 6. 触发同步
echo -e "${BLUE}[6/10] 触发邮件同步...${NC}"
SYNC_RESPONSE=$(curl -s -X POST "${API_BASE}/sync/accounts/${ACCOUNT_UID}")

if echo "$SYNC_RESPONSE" | grep -q "Sync started"; then
    echo -e "${GREEN}✓ 同步已启动${NC}"
else
    echo -e "${RED}✗ 同步启动失败${NC}"
    echo "   $SYNC_RESPONSE"
fi
echo ""

# 7. 等待同步
echo -e "${BLUE}[7/10] 等待同步完成...${NC}"
echo -n "   进度: "
for i in {1..15}; do
    sleep 1
    echo -n "█"
done
echo ""
echo -e "${GREEN}✓ 等待完成${NC}"
echo ""

# 8. 查看同步结果
echo -e "${BLUE}[8/10] 查看同步结果...${NC}"
ACCOUNT_DETAIL=$(curl -s "${API_BASE}/accounts/${ACCOUNT_UID}")

LAST_SYNC_STATUS=$(echo "$ACCOUNT_DETAIL" | grep -o '"last_sync_status":"[^"]*"' | cut -d'"' -f4)
LAST_SYNC_ERROR=$(echo "$ACCOUNT_DETAIL" | grep -o '"last_sync_error":"[^"]*"' | cut -d'"' -f4)
TOTAL_EMAILS=$(echo "$ACCOUNT_DETAIL" | grep -o '"total_emails":[0-9]*' | cut -d':' -f2)

echo "   同步状态: $LAST_SYNC_STATUS"
if [ -n "$LAST_SYNC_ERROR" ] && [ "$LAST_SYNC_ERROR" != "" ]; then
    echo "   错误信息: $LAST_SYNC_ERROR"
fi
echo "   邮件总数: ${TOTAL_EMAILS:-0}"
echo ""

# 9. 验证数据库
echo -e "${BLUE}[9/10] 验证数据库数据...${NC}"

# 邮件总数
FINAL_COUNT=$(docker exec fusionmail-postgres psql -U fusionmail -d fusionmail -t -c "SELECT COUNT(*) FROM emails;" 2>/dev/null | tr -d ' ')
echo "   数据库邮件数: ${FINAL_COUNT:-0}"

# 新增邮件数
NEW_EMAILS=$((FINAL_COUNT - INITIAL_COUNT))
echo "   新增邮件数: $NEW_EMAILS"

# 查看最新的 5 封邮件
echo ""
echo "   最新的 5 封邮件:"
docker exec fusionmail-postgres psql -U fusionmail -d fusionmail -c \
  "SELECT id, subject, from_address, TO_CHAR(sent_at, 'YYYY-MM-DD HH24:MI') as sent_at 
   FROM emails 
   ORDER BY sent_at DESC 
   LIMIT 5;" 2>/dev/null

echo ""

# 10. 测试总结
echo -e "${BLUE}[10/10] 测试总结${NC}"
echo ""
echo "=========================================="
echo "  测试结果"
echo "=========================================="
echo ""

if [ "$LAST_SYNC_STATUS" = "success" ] && [ "$FINAL_COUNT" -gt "$INITIAL_COUNT" ]; then
    echo -e "${GREEN}✓✓✓ 测试完全成功！✓✓✓${NC}"
    echo ""
    echo "验证项目:"
    echo "  ✓ 服务器运行正常"
    echo "  ✓ 账户添加成功"
    echo "  ✓ IMAP 连接成功"
    echo "  ✓ 邮件同步成功"
    echo "  ✓ 数据库写入成功"
    echo ""
    echo "同步统计:"
    echo "  - 同步前邮件数: $INITIAL_COUNT"
    echo "  - 同步后邮件数: $FINAL_COUNT"
    echo "  - 新增邮件数: $NEW_EMAILS"
    
elif [ "$LAST_SYNC_STATUS" = "success" ] && [ "$FINAL_COUNT" -eq "$INITIAL_COUNT" ]; then
    echo -e "${YELLOW}⚠ 同步成功但没有新邮件${NC}"
    echo ""
    echo "可能原因:"
    echo "  - 邮箱中没有最近 30 天的新邮件"
    echo "  - 邮件已经在之前的同步中拉取过"
    
else
    echo -e "${RED}✗ 测试失败${NC}"
    echo ""
    echo "失败信息:"
    echo "  - 同步状态: $LAST_SYNC_STATUS"
    if [ -n "$LAST_SYNC_ERROR" ]; then
        echo "  - 错误信息: $LAST_SYNC_ERROR"
    fi
fi

echo ""
echo "=========================================="
echo "  数据库查询命令"
echo "=========================================="
echo ""
echo "1. 查看所有邮件:"
echo "   docker exec -it fusionmail-postgres psql -U fusionmail -d fusionmail -c 'SELECT * FROM emails;'"
echo ""
echo "2. 查看账户信息:"
echo "   docker exec -it fusionmail-postgres psql -U fusionmail -d fusionmail -c 'SELECT * FROM accounts;'"
echo ""
echo "3. 查看同步日志:"
echo "   docker exec -it fusionmail-postgres psql -U fusionmail -d fusionmail -c 'SELECT * FROM sync_logs ORDER BY started_at DESC LIMIT 10;'"
echo ""
echo "4. 按发件人统计:"
echo "   docker exec -it fusionmail-postgres psql -U fusionmail -d fusionmail -c 'SELECT from_address, COUNT(*) FROM emails GROUP BY from_address ORDER BY COUNT(*) DESC LIMIT 10;'"
echo ""

echo "=========================================="
echo "  API 测试命令"
echo "=========================================="
echo ""
echo "账户 UID: $ACCOUNT_UID"
echo ""
echo "1. 查看账户详情:"
echo "   curl ${API_BASE}/accounts/${ACCOUNT_UID} | python3 -m json.tool"
echo ""
echo "2. 再次同步:"
echo "   curl -X POST ${API_BASE}/sync/accounts/${ACCOUNT_UID}"
echo ""
echo "3. 删除账户:"
echo "   curl -X DELETE ${API_BASE}/accounts/${ACCOUNT_UID}"
echo ""
