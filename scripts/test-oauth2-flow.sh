#!/bin/bash

# Microsoft OAuth2 流程测试脚本
# 用于测试完整的OAuth2授权流程

set -e

echo "🧪 Microsoft OAuth2 流程测试"
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
echo "🔗 生成测试授权URL..."

# 生成授权URL
auth_response=$(curl -s "http://localhost:$SERVER_PORT/api/v1/auth/microsoft/authorize?email=test@hotmail.com")

if echo "$auth_response" | grep -q "auth_url"; then
    auth_url=$(echo "$auth_response" | jq -r '.data.auth_url' 2>/dev/null || echo "$auth_response" | grep -o '"auth_url":"[^"]*"' | cut -d'"' -f4)
    state=$(echo "$auth_response" | jq -r '.data.state' 2>/dev/null || echo "$auth_response" | grep -o '"state":"[^"]*"' | cut -d'"' -f4)
    
    echo "✅ 授权URL生成成功"
    echo "🔗 URL: $auth_url"
    echo "🔑 State: $state"
    
    echo ""
    echo "📱 手动测试步骤："
    echo "1. 复制以下URL到浏览器："
    echo "   $auth_url"
    echo ""
    echo "2. 完成Microsoft登录和授权"
    echo ""
    echo "3. 观察回调结果"
    
    # 创建简化的测试URL
    simple_url="https://login.microsoftonline.com/common/oauth2/v2.0/authorize"
    simple_url+="?client_id=$MICROSOFT_CLIENT_ID"
    simple_url+="&response_type=code"
    simple_url+="&redirect_uri=$(echo $MICROSOFT_REDIRECT_URI | sed 's/:/%3A/g' | sed 's/\//%2F/g')"
    simple_url+="&scope=https%3A%2F%2Fgraph.microsoft.com%2FMail.ReadWrite%20https%3A%2F%2Fgraph.microsoft.com%2FUser.Read%20offline_access"
    simple_url+="&state=manual_test"
    
    echo ""
    echo "🔧 简化测试URL（用于手动测试）："
    echo "$simple_url"
    
else
    echo "❌ 授权URL生成失败"
    echo "响应: $auth_response"
    exit 1
fi

echo ""
echo "📊 常见错误代码说明："
echo ""
echo "server_error:"
echo "  - 原因: Microsoft服务器内部错误"
echo "  - 解决: 检查Azure应用配置，重试授权"
echo ""
echo "invalid_request:"
echo "  - 原因: 请求参数错误"
echo "  - 解决: 检查client_id、redirect_uri等参数"
echo ""
echo "unauthorized_client:"
echo "  - 原因: 客户端未授权"
echo "  - 解决: 检查客户端密钥和权限配置"
echo ""
echo "access_denied:"
echo "  - 原因: 用户拒绝授权"
echo "  - 解决: 用户需要同意授权"
echo ""
echo "unsupported_response_type:"
echo "  - 原因: 不支持的响应类型"
echo "  - 解决: 检查Azure应用配置"

echo ""
echo "🔍 实时日志监控："
echo "在另一个终端运行以下命令监控日志："
echo "tail -f logs/backend.log | grep -i 'oauth\\|microsoft\\|error\\|callback'"

echo ""
echo "📋 Azure Portal 检查清单："
echo "1. 访问 https://portal.azure.com/"
echo "2. 导航到 Azure Active Directory > 应用注册"
echo "3. 找到应用: FusionMail (ID: $MICROSOFT_CLIENT_ID)"
echo "4. 检查以下配置："
echo ""
echo "   身份验证:"
echo "   - 重定向URI: $MICROSOFT_REDIRECT_URI ✓"
echo "   - 访问令牌: 已启用 ✓"
echo "   - ID令牌: 已启用 ✓"
echo ""
echo "   API权限:"
echo "   - Mail.ReadWrite (委托) ✓"
echo "   - User.Read (委托) ✓"
echo "   - offline_access (委托) ✓"
echo "   - 管理员同意状态: 已授予 ✓"
echo ""
echo "   证书和密钥:"
echo "   - 客户端密钥: 有效且未过期 ✓"

echo ""
echo "🚀 测试完成！"
echo "请使用上述URL进行手动测试，并观察任何错误信息。"