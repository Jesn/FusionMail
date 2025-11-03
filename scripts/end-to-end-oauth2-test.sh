#!/bin/bash

# 端到端OAuth2测试脚本
# 解决state和token交换问题

set -e

echo "🚀 端到端OAuth2测试"
echo "=================="

# 加载环境变量
if [ -f "backend/.env" ]; then
    source backend/.env
fi

echo "📋 当前配置："
echo "   Client ID: $MICROSOFT_CLIENT_ID"
echo "   Redirect URI: $MICROSOFT_REDIRECT_URI"
echo "   Server Port: $SERVER_PORT"

echo ""
echo "🔧 步骤1: 检查Redis连接..."
if redis-cli ping > /dev/null 2>&1; then
    echo "✅ Redis连接正常"
else
    echo "❌ Redis连接失败，请启动Redis服务"
    exit 1
fi

echo ""
echo "🔧 步骤2: 清理Redis中的旧state..."
redis-cli --scan --pattern "oauth2:state:*" | xargs -r redis-cli del
echo "✅ 旧state已清理"

echo ""
echo "🔧 步骤3: 生成新的授权URL..."
response=$(curl -s "http://localhost:$SERVER_PORT/api/v1/auth/microsoft/authorize?email=test@hotmail.com")

if echo "$response" | grep -q "auth_url"; then
    auth_url=$(echo "$response" | python3 -c "
import sys, json
data = json.load(sys.stdin)
print(data['data']['auth_url'])
")
    state=$(echo "$response" | python3 -c "
import sys, json
data = json.load(sys.stdin)
print(data['data']['state'])
")
    
    echo "✅ 授权URL生成成功"
    echo "🔑 State: $state"
    
    # 验证state是否存储在Redis中
    if redis-cli exists "oauth2:state:$state" | grep -q "1"; then
        echo "✅ State已存储在Redis中"
    else
        echo "❌ State未存储在Redis中"
        exit 1
    fi
    
    echo ""
    echo "🔗 请使用以下URL进行授权（5分钟内有效）："
    echo "$auth_url"
    
    echo ""
    echo "⚠️  重要提示："
    echo "1. 必须在5分钟内完成授权"
    echo "2. 授权码只能使用一次"
    echo "3. 完成授权后立即检查回调结果"
    
    echo ""
    echo "📊 实时监控命令（在另一个终端运行）："
    echo "tail -f logs/backend.log | grep -i 'callback\\|state\\|token\\|microsoft'"
    
    echo ""
    echo "🔍 手动验证步骤："
    echo "1. 复制上述URL到浏览器"
    echo "2. 完成Microsoft登录和授权"
    echo "3. 观察回调页面结果"
    echo "4. 检查后端日志中的详细错误信息"
    
    echo ""
    echo "📋 预期的成功流程："
    echo "✅ Microsoft OAuth2 callback received"
    echo "✅ OAuth2 state validated successfully"
    echo "✅ OAuth2 token exchange successful"
    echo "✅ Microsoft user info retrieved successfully"
    echo "✅ OAuth2 account created successfully"
    
else
    echo "❌ 无法生成授权URL"
    echo "响应: $response"
    exit 1
fi

echo ""
echo "🔧 故障排除："
echo ""
echo "如果仍然出现 'invalid_grant' 错误："
echo "1. 检查Azure Portal中的重定向URI配置"
echo "2. 确保客户端密钥有效且未过期"
echo "3. 验证API权限已正确配置"
echo "4. 尝试重新生成客户端密钥"
echo ""
echo "如果出现 'invalid state' 错误："
echo "1. 确保在5分钟内完成授权"
echo "2. 不要重复使用同一个授权URL"
echo "3. 检查Redis服务是否正常运行"