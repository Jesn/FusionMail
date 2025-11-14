#!/bin/bash

# 手动测试 SSE 连接和 Cookie 鉴权

set -e

echo "========================================="
echo "SSE 手动测试脚本"
echo "========================================="

# 1. 登录获取 Cookie
echo ""
echo "1. 登录获取 Cookie..."
LOGIN_RESPONSE=$(curl -s -c cookies.txt -X POST http://localhost:3333/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{
    "email": "admin@example.com",
    "password": "admin123"
  }')

echo "登录响应: $LOGIN_RESPONSE"

# 检查是否登录成功
if echo "$LOGIN_RESPONSE" | grep -q '"success":true'; then
    echo "✅ 登录成功"
else
    echo "❌ 登录失败"
    exit 1
fi

# 显示 Cookie
echo ""
echo "2. 查看 Cookie..."
if [ -f cookies.txt ]; then
    echo "Cookie 文件内容:"
    cat cookies.txt
    echo ""
else
    echo "❌ Cookie 文件不存在"
    exit 1
fi

# 3. 测试 SSE 连接（使用 Cookie）
echo ""
echo "3. 测试 SSE 连接（使用 Cookie）..."
echo "正在连接 SSE 端点..."
echo "按 Ctrl+C 停止"
echo ""

# 使用 curl 连接 SSE（带 Cookie）
curl -N -b cookies.txt http://localhost:3333/api/v1/events

# 清理
rm -f cookies.txt

