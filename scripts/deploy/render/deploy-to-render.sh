#!/bin/bash
# ============================================
# FusionMail 部署到 Render
# ============================================

set -e

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

print_info() { echo -e "${BLUE}[INFO]${NC} $1"; }
print_success() { echo -e "${GREEN}[SUCCESS]${NC} $1"; }
print_error() { echo -e "${RED}[ERROR]${NC} $1"; }

echo ""
echo "=========================================="
echo "  🚀 FusionMail Render 部署"
echo "=========================================="
echo ""

# 检查 API Key
if [ -z "$RENDER_API_KEY" ]; then
    print_error "请设置 RENDER_API_KEY"
    echo "  export RENDER_API_KEY=rnd_xxxxxxxx"
    exit 1
fi

print_success "API Key 已设置"

# 获取 Owner ID
print_info "获取账户信息..."
OWNER_INFO=$(curl -s -H "Authorization: Bearer ${RENDER_API_KEY}" \
    "https://api.render.com/v1/owners" 2>&1)

OWNER_ID=$(echo "$OWNER_INFO" | python3 -c "import sys,json; owners=json.load(sys.stdin); print(owners[0]['owner']['id'] if owners else '')" 2>/dev/null)

if [ -z "$OWNER_ID" ]; then
    print_error "无法获取 Owner ID"
    echo "$OWNER_INFO"
    exit 1
fi

print_success "Owner ID: $OWNER_ID"

# 检查是否已存在服务
print_info "检查现有服务..."
SERVICES=$(curl -s -H "Authorization: Bearer ${RENDER_API_KEY}" \
    "https://api.render.com/v1/services?ownerId=${OWNER_ID}&name=fusionmail" 2>&1)

SERVICE_ID=$(echo "$SERVICES" | python3 -c "
import sys,json
data = json.load(sys.stdin)
for svc in data:
    if svc.get('service',{}).get('name') == 'fusionmail':
        print(svc['service']['id'])
        break
" 2>/dev/null)

if [ -n "$SERVICE_ID" ]; then
    print_info "服务已存在: $SERVICE_ID"
    print_info "触发重新部署..."
    
    DEPLOY_RESULT=$(curl -s -X POST \
        -H "Authorization: Bearer ${RENDER_API_KEY}" \
        -H "Content-Type: application/json" \
        "https://api.render.com/v1/services/${SERVICE_ID}/deploys" \
        -d '{"clearCache": "do_not_clear"}')
    
    echo "$DEPLOY_RESULT" | python3 -c "import sys,json; d=json.load(sys.stdin); print('部署 ID:', d.get('id','未知'))"
    print_success "已触发重新部署"
else
    print_info "创建新服务..."
    echo ""
    print_error "Render API 不支持直接创建 Docker 服务"
    echo ""
    echo "请手动创建："
    echo "1. 访问 https://dashboard.render.com"
    echo "2. New → Web Service → Build and deploy from a Git repository"
    echo "3. 连接你的 GitHub/GitLab 仓库"
    echo "4. 选择 Docker 运行时"
    echo "5. 设置 Dockerfile 路径: render/Dockerfile"
    echo "6. 添加环境变量（见下方）"
    echo ""
    echo "环境变量配置："
    echo "  DB_HOST=aws-1-ap-northeast-1.pooler.supabase.com"
    echo "  DB_PORT=5432"
    echo "  DB_USER=postgres.oeufkcyahfhtpemzwsdt"
    echo "  DB_PASSWORD=<你的密码>"
    echo "  DB_NAME=postgres"
    echo "  DB_SSLMODE=require"
    echo "  REDIS_HOST=splendid-sunfish-36032.upstash.io"
    echo "  REDIS_PORT=6379"
    echo "  REDIS_PASSWORD=<你的密码>"
    echo "  REDIS_TLS=true"
    echo "  JWT_SECRET=<生成的密钥>"
    echo "  ENCRYPTION_KEY=<32字节密钥>"
    echo "  ADMIN_PASSWORD=<管理员密码>"
fi

echo ""
print_success "完成"
