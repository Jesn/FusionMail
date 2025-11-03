#!/bin/bash

# Microsoft OAuth2 回调测试脚本
# 用于测试完整的OAuth2授权流程，包括正确的state管理

set -e

echo "🧪 Microsoft OAuth2 回调测试"
echo "============================"

# 加载环境变量
if [ -f "backend/.env" ]; then
    source backend/.env
else
    echo "❌ 未找到 backend/.env 文件"
    exit 1
fi

echo "📋 测试配置："
echo "   Client ID: $MICROSOFT_CLIENT_ID"
echo "   Redirect URI: $MICROSOFT_REDIRECT_URI"
echo "   Server Port: $SERVER_PORT"

echo ""
echo "🔗 生成正确的授权URL（包含有效的state）..."

# 生成授权URL（这会在Redis中存储state）
auth_response=$(curl -s "http://localhost:$SERVER_PORT/api/v1/auth/microsoft/authorize?email=test@hotmail.com")

if echo "$auth_response" | grep -q "auth_url"; then
    auth_url=$(echo "$auth_response" | jq -r '.data.auth_url' 2>/dev/null || echo "$auth_response" | grep -o '"auth_url":"[^"]*"' | cut -d'"' -f4)
    state=$(echo "$auth_response" | jq -r '.data.state' 2>/dev/null || echo "$auth_response" | grep -o '"state":"[^"]*"' | cut -d'"' -f4)
    
    echo "✅ 授权URL生成成功"
    echo "🔑 有效的State: $state"
    echo ""
    echo "🔗 请使用以下URL进行测试："
    echo "$auth_url"
    echo ""
    echo "⚠️  重要提示："
    echo "1. 必须使用上面生成的URL，不要使用之前的手动测试URL"
    echo "2. State参数 '$state' 已存储在Redis中，有效期5分钟"
    echo "3. 完成授权后，回调将正常处理"
    
else
    echo "❌ 授权URL生成失败"
    echo "响应: $auth_response"
    exit 1
fi

echo ""
echo "📊 State管理说明："
echo "问题原因: 之前使用的 'manual_test' state没有存储在Redis中"
echo "解决方案: 使用API生成的URL，确保state正确存储和验证"

echo ""
echo "🔍 实时监控回调处理："
echo "在另一个终端运行以下命令："
echo "tail -f logs/backend.log | grep -i 'callback\\|state\\|microsoft'"