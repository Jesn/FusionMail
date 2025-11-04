#!/bin/bash

# FusionMail E2E 测试运行脚本

set -e

echo "=========================================="
echo "FusionMail 自动化测试"
echo "=========================================="
echo ""

# 检查 Node.js
if ! command -v node &> /dev/null; then
    echo "❌ Node.js 未安装"
    exit 1
fi

# 检查 npm
if ! command -v npm &> /dev/null; then
    echo "❌ npm 未安装"
    exit 1
fi

# 进入测试目录
cd "$(dirname "$0")"

# 安装依赖
if [ ! -d "node_modules" ]; then
    echo "📦 安装测试依赖..."
    npm install
    echo ""
fi

# 安装 Playwright 浏览器
if [ ! -d "node_modules/@playwright/test" ]; then
    echo "🌐 安装 Playwright 浏览器..."
    npx playwright install chromium
    echo ""
fi

# 检查后端服务
echo "🔍 检查后端服务..."
if curl -s http://localhost:3333/api/v1/health > /dev/null; then
    echo "✅ 后端服务运行中"
else
    echo "❌ 后端服务未运行"
    echo "请先启动后端服务: cd backend && go run cmd/server/main.go"
    exit 1
fi
echo ""

# 运行测试
echo "🧪 开始运行测试..."
echo ""

# 按顺序运行测试
echo "1️⃣ 运行环境检查测试..."
npx playwright test tests/health.spec.ts --reporter=list

echo ""
echo "2️⃣ 运行认证测试..."
npx playwright test tests/auth.spec.ts --reporter=list

echo ""
echo "3️⃣ 运行 API 测试..."
npx playwright test tests/api.spec.ts --reporter=list

echo ""
echo "4️⃣ 运行邮件管理测试..."
npx playwright test tests/email.spec.ts --reporter=list

echo ""
echo "5️⃣ 运行速率限制测试..."
npx playwright test tests/ratelimit.spec.ts --reporter=list

echo ""
echo "6️⃣ 运行附件存储测试..."
npx playwright test tests/storage.spec.ts --reporter=list

echo ""
echo "=========================================="
echo "✅ 测试完成！"
echo "=========================================="
echo ""
echo "📊 查看详细报告: npx playwright show-report"
echo "📋 查看测试清单: cat ../../.kiro/specs/fusionmail/test-checklist.md"
echo ""
