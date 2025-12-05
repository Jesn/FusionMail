#!/bin/bash
# ============================================
# FusionMail 一键部署到 HuggingFace Spaces
# ============================================
# 使用方法:
#   1. 设置环境变量: export HF_TOKEN=hf_xxxxxxxx
#   2. 运行脚本: ./scripts/deploy-to-hf.sh
# ============================================

set -e

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

print_info() { echo -e "${BLUE}[INFO]${NC} $1"; }
print_success() { echo -e "${GREEN}[SUCCESS]${NC} $1"; }
print_warning() { echo -e "${YELLOW}[WARNING]${NC} $1"; }
print_error() { echo -e "${RED}[ERROR]${NC} $1"; }

# 配置
HF_USERNAME="jesn"
HF_SPACE="FusionMail"
HF_REPO="https://${HF_USERNAME}:${HF_TOKEN}@huggingface.co/spaces/${HF_USERNAME}/${HF_SPACE}"
TEMP_DIR="/tmp/hf-fusionmail-deploy"

echo ""
echo "=========================================="
echo "  🚀 FusionMail HuggingFace 部署脚本"
echo "=========================================="
echo ""

# 检查 HF_TOKEN
if [ -z "$HF_TOKEN" ]; then
    print_error "请先设置 HF_TOKEN 环境变量"
    echo ""
    echo "使用方法:"
    echo "  export HF_TOKEN=hf_xxxxxxxxxxxxxxxx"
    echo "  ./scripts/deploy-to-hf.sh"
    echo ""
    echo "获取 Token: https://huggingface.co/settings/tokens"
    exit 1
fi

print_success "HF_TOKEN 已设置"

# 清理临时目录
print_info "清理临时目录..."
rm -rf "$TEMP_DIR"

# 克隆 HF Space
print_info "克隆 HuggingFace Space..."
git clone "$HF_REPO" "$TEMP_DIR" 2>&1 | grep -v "Cloning into" || true

if [ ! -d "$TEMP_DIR/.git" ]; then
    print_error "克隆失败，请检查 Token 是否正确"
    exit 1
fi

print_success "克隆成功"

# 复制部署文件
print_info "复制部署文件..."
cp scripts/deploy/huggingface/Dockerfile "$TEMP_DIR/"
cp scripts/deploy/huggingface/README.md "$TEMP_DIR/"
cp scripts/deploy/huggingface/nginx.conf "$TEMP_DIR/"
cp scripts/deploy/huggingface/start.sh "$TEMP_DIR/"

# 复制源代码（排除不需要的文件）
print_info "复制后端代码..."
rsync -av --exclude='bin' --exclude='fusionmail' --exclude='.env' --exclude='passwd' \
    backend/ "$TEMP_DIR/backend/"

print_info "复制前端代码..."
rsync -av --exclude='node_modules' --exclude='dist' --exclude='.env' \
    frontend/ "$TEMP_DIR/frontend/"

# 创建 .gitignore
cat > "$TEMP_DIR/.gitignore" << 'EOF'
node_modules/
frontend/node_modules/
frontend/dist/
backend/bin/
backend/fusionmail
.env
.env.local
backend/.env
backend/passwd
logs/
data/
.DS_Store
EOF

# 提交并推送
print_info "提交更改..."
cd "$TEMP_DIR"
git add -A

# 检查是否有更改
if git diff --cached --quiet; then
    print_warning "没有检测到更改，跳过推送"
else
    git commit -m "Deploy FusionMail $(date '+%Y-%m-%d %H:%M:%S')"
    
    print_info "推送到 HuggingFace..."
    git push origin main
    
    print_success "推送成功！"
fi

cd - > /dev/null

# 清理
print_info "清理临时文件..."
rm -rf "$TEMP_DIR"

echo ""
echo "=========================================="
print_success "🎉 部署完成！"
echo "=========================================="
echo ""
echo "访问地址: https://huggingface.co/spaces/${HF_USERNAME}/${HF_SPACE}"
echo "直接访问: https://jesn-fusionmail.hf.space"
echo ""

# 检查 HF Space 的 Secrets 配置状态
print_info "检查 Secrets 配置状态..."
echo ""

# 通过 HF API 检查 Space 信息（需要 Token）
SPACE_INFO=$(curl -s -H "Authorization: Bearer ${HF_TOKEN}" \
    "https://huggingface.co/api/spaces/${HF_USERNAME}/${HF_SPACE}" 2>/dev/null)

if echo "$SPACE_INFO" | grep -q '"runtime"'; then
    print_success "Space 状态正常"
    
    # 检查是否有 secrets（API 不直接暴露 secrets 值，只能提示用户确认）
    echo ""
    print_warning "请确认以下 Secrets 已在 Space Settings 中配置："
    echo "  必需:"
    echo "    ✓ DB_HOST"
    echo "    ✓ DB_PASSWORD" 
    echo "    ✓ JWT_SECRET"
    echo "    ✓ ENCRYPTION_KEY"
    echo "  可选:"
    echo "    - DB_PORT (默认 5432)"
    echo "    - DB_USER (默认 postgres)"
    echo "    - DB_NAME (默认 postgres)"
    echo "    - DB_SSLMODE (默认 require)"
    echo "    - REDIS_HOST"
    echo "    - REDIS_PORT"
    echo "    - REDIS_PASSWORD"
    echo "    - REDIS_TLS (默认 false)"
    echo "    - ADMIN_PASSWORD"
    echo ""
    echo "配置地址: https://huggingface.co/spaces/${HF_USERNAME}/${HF_SPACE}/settings"
else
    print_warning "无法获取 Space 状态，请手动检查配置"
fi

echo ""
print_info "等待 HuggingFace 构建完成后即可访问应用"
echo ""
