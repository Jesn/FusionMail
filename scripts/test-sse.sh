#!/bin/bash

# SSE Cookie 测试脚本
# 用于测试 SSE 连接和 Cookie 鉴权

set -e

echo "========================================="
echo "SSE Cookie 鉴权测试"
echo "========================================="

# 检查前端是否运行
echo ""
echo "1. 检查前端服务..."
if ! curl -s http://localhost:4444 > /dev/null; then
    echo "❌ 前端服务未运行，请先启动前端服务："
    echo "   cd frontend && npm run dev"
    exit 1
fi
echo "✅ 前端服务正常"

# 检查后端是否运行
echo ""
echo "2. 检查后端服务..."
if ! curl -s http://localhost:3333/api/v1/health > /dev/null; then
    echo "❌ 后端服务未运行，请先启动后端服务："
    echo "   cd backend && go run cmd/server/main.go"
    exit 1
fi
echo "✅ 后端服务正常"

# 检查 Playwright 是否安装
echo ""
echo "3. 检查 Playwright..."
if ! command -v npx &> /dev/null; then
    echo "❌ npx 未安装，请先安装 Node.js"
    exit 1
fi

if ! npx playwright --version &> /dev/null; then
    echo "⚠️  Playwright 未安装，正在安装..."
    npm install -D @playwright/test
    npx playwright install chromium
fi
echo "✅ Playwright 已安装"

# 运行测试
echo ""
echo "4. 运行 SSE Cookie 测试..."
echo "========================================="
npx playwright test tests/e2e/sse-cookie-test.spec.ts --headed

# 显示测试报告
echo ""
echo "========================================="
echo "测试完成！"
echo "========================================="
echo ""
echo "查看详细报告："
echo "  npx playwright show-report"

