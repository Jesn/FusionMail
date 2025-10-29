#!/bin/bash

# FusionMail 开发环境启动脚本

set -e

echo "🚀 启动 FusionMail 开发环境..."
echo ""

# 颜色定义
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m' # No Color

# 检查 Docker
if ! command -v docker &> /dev/null; then
    echo -e "${RED}❌ Docker 未安装，请先安装 Docker${NC}"
    exit 1
fi

# 检查 Docker Compose
if ! command -v docker-compose &> /dev/null; then
    echo -e "${RED}❌ Docker Compose 未安装，请先安装 Docker Compose${NC}"
    exit 1
fi

# 1. 启动基础设施（PostgreSQL + Redis）
echo -e "${YELLOW}📦 步骤 1/3: 启动基础设施（PostgreSQL + Redis）${NC}"
./scripts/dev-start.sh
echo -e "${GREEN}✓ 基础设施启动成功${NC}"
echo ""

# 等待数据库就绪
echo -e "${YELLOW}⏳ 等待数据库就绪...${NC}"
sleep 5
echo -e "${GREEN}✓ 数据库就绪${NC}"
echo ""

# 2. 启动后端服务
echo -e "${YELLOW}🔧 步骤 2/3: 启动后端服务${NC}"
echo "后端服务将在新终端窗口中启动..."
echo "如果没有自动打开，请手动运行："
echo "  cd backend && go run cmd/server/main.go"
echo ""

# 检查操作系统并在新终端中启动后端
if [[ "$OSTYPE" == "darwin"* ]]; then
    # macOS
    osascript -e 'tell app "Terminal" to do script "cd '"$(pwd)"'/backend && go run cmd/server/main.go"'
elif [[ "$OSTYPE" == "linux-gnu"* ]]; then
    # Linux
    if command -v gnome-terminal &> /dev/null; then
        gnome-terminal -- bash -c "cd $(pwd)/backend && go run cmd/server/main.go; exec bash"
    else
        echo -e "${YELLOW}⚠️  请在新终端中手动运行：cd backend && go run cmd/server/main.go${NC}"
    fi
else
    echo -e "${YELLOW}⚠️  请在新终端中手动运行：cd backend && go run cmd/server/main.go${NC}"
fi

sleep 3
echo ""

# 3. 启动前端服务
echo -e "${YELLOW}🎨 步骤 3/3: 启动前端服务${NC}"
echo "前端服务将在新终端窗口中启动..."
echo "如果没有自动打开，请手动运行："
echo "  cd frontend && npm run dev"
echo ""

# 检查 node_modules
if [ ! -d "frontend/node_modules" ]; then
    echo -e "${YELLOW}📦 首次运行，正在安装前端依赖...${NC}"
    cd frontend && npm install && cd ..
    echo -e "${GREEN}✓ 前端依赖安装完成${NC}"
fi

# 在新终端中启动前端
if [[ "$OSTYPE" == "darwin"* ]]; then
    # macOS
    osascript -e 'tell app "Terminal" to do script "cd '"$(pwd)"'/frontend && npm run dev"'
elif [[ "$OSTYPE" == "linux-gnu"* ]]; then
    # Linux
    if command -v gnome-terminal &> /dev/null; then
        gnome-terminal -- bash -c "cd $(pwd)/frontend && npm run dev; exec bash"
    else
        echo -e "${YELLOW}⚠️  请在新终端中手动运行：cd frontend && npm run dev${NC}"
    fi
else
    echo -e "${YELLOW}⚠️  请在新终端中手动运行：cd frontend && npm run dev${NC}"
fi

sleep 3
echo ""

# 完成
echo -e "${GREEN}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo -e "${GREEN}✨ FusionMail 开发环境启动完成！${NC}"
echo -e "${GREEN}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo ""
echo "📍 访问地址："
echo "  前端：http://localhost:3000"
echo "  后端：http://localhost:8080"
echo "  API：http://localhost:8080/api/v1"
echo ""
echo "📚 快速开始："
echo "  1. 打开浏览器访问 http://localhost:3000"
echo "  2. 添加邮箱账户（账户管理页面）"
echo "  3. 同步邮件"
echo "  4. 查看邮件列表"
echo ""
echo "🔧 停止服务："
echo "  ./scripts/dev-stop.sh"
echo ""
