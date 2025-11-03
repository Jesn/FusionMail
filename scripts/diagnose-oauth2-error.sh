#!/bin/bash

# Microsoft OAuth2 错误诊断脚本
# 用于排查 server_error 问题

set -e

echo "🔍 Microsoft OAuth2 错误诊断工具"
echo "=================================="

# 加载环境变量
if [ -f "backend/.env" ]; then
    source backend/.env
    echo "✅ 环境变量已加载"
else
    echo "❌ 未找到 backend/.env 文件"
    exit 1
fi

echo ""
echo "📊 当前配置信息："
echo "   Client ID: $MICROSOFT_CLIENT_ID"
echo "   Client Secret: ${MICROSOFT_CLIENT_SECRET:0:10}...${MICROSOFT_CLIENT_SECRET: -4}"
echo "   Redirect URI: $MICROSOFT_REDIRECT_URI"

echo ""
echo "🌐 网络连接测试..."

# 测试Microsoft登录端点
echo "📡 测试Microsoft登录端点..."
if curl -s -I "https://login.microsoftonline.com/common/oauth2/v2.0/authorize" | head -1 | grep -q "200\|302"; then
    echo "✅ Microsoft登录端点可访问"
else
    echo "❌ Microsoft登录端点不可访问"
fi

# 测试Graph API端点
echo "📡 测试Microsoft Graph API端点..."
if curl -s -I "https://graph.microsoft.com/v1.0/" | head -1 | grep -q "200\|401"; then
    echo "✅ Microsoft Graph API端点可访问"
else
    echo "❌ Microsoft Graph API端点不可访问"
fi

echo ""
echo "🔧 本地服务测试..."

# 检查后端服务是否运行
if curl -s "http://localhost:$SERVER_PORT/api/v1/health" > /dev/null; then
    echo "✅ 后端服务运行正常"
    
    # 测试授权URL生成
    echo "📡 测试授权URL生成..."
    auth_response=$(curl -s "http://localhost:$SERVER_PORT/api/v1/auth/microsoft/authorize?email=test@hotmail.com")
    
    if echo "$auth_response" | grep -q "auth_url"; then
        echo "✅ 授权URL生成成功"
        
        # 提取授权URL
        auth_url=$(echo "$auth_response" | grep -o '"auth_url":"[^"]*"' | cut -d'"' -f4)
        echo "🔗 授权URL: $auth_url"
        
        # 验证URL格式
        if echo "$auth_url" | grep -q "login.microsoftonline.com"; then
            echo "✅ 授权URL格式正确"
        else
            echo "❌ 授权URL格式异常"
        fi
        
    else
        echo "❌ 授权URL生成失败"
        echo "响应: $auth_response"
    fi
    
else
    echo "❌ 后端服务未运行或不可访问"
    echo "请先启动后端服务: cd backend && ./fusionmail"
fi

echo ""
echo "🔍 Azure配置验证建议..."

echo "请在Azure Portal中验证以下配置："
echo ""
echo "1. 应用注册基本信息："
echo "   - 应用程序(客户端) ID: $MICROSOFT_CLIENT_ID"
echo "   - 支持的账户类型: 任何组织目录中的账户和个人 Microsoft 账户"
echo ""
echo "2. 身份验证配置："
echo "   - 平台: Web"
echo "   - 重定向URI: $MICROSOFT_REDIRECT_URI"
echo "   - 访问令牌: 已启用"
echo "   - ID令牌: 已启用"
echo ""
echo "3. API权限配置："
echo "   - Microsoft Graph - Mail.ReadWrite (委托)"
echo "   - Microsoft Graph - User.Read (委托)"
echo "   - Microsoft Graph - offline_access (委托)"
echo "   - 状态: 已授予管理员同意"
echo ""
echo "4. 证书和密钥："
echo "   - 客户端密钥: ${MICROSOFT_CLIENT_SECRET:0:10}...${MICROSOFT_CLIENT_SECRET: -4}"
echo "   - 状态: 有效且未过期"

echo ""
echo "🧪 手动测试步骤："
echo "1. 复制以下URL到浏览器："
echo "   https://login.microsoftonline.com/common/oauth2/v2.0/authorize?client_id=$MICROSOFT_CLIENT_ID&response_type=code&redirect_uri=$(echo $MICROSOFT_REDIRECT_URI | sed 's/:/%3A/g' | sed 's/\//%2F/g')&scope=https%3A%2F%2Fgraph.microsoft.com%2FMail.ReadWrite%20https%3A%2F%2Fgraph.microsoft.com%2FUser.Read%20offline_access&state=test_state"
echo ""
echo "2. 观察是否出现以下情况："
echo "   - Microsoft登录页面正常显示 ✅"
echo "   - 提示应用未验证 ⚠️"
echo "   - 提示权限错误 ❌"
echo "   - 提示重定向URI错误 ❌"

echo ""
echo "🔧 常见解决方案："
echo "1. 如果提示'应用未验证'："
echo "   - 这是正常的，点击'高级' -> '转到应用'"
echo ""
echo "2. 如果提示'重定向URI不匹配'："
echo "   - 检查Azure Portal中的重定向URI配置"
echo "   - 确保与环境变量完全一致"
echo ""
echo "3. 如果提示'权限错误'："
echo "   - 检查API权限配置"
echo "   - 重新授予管理员同意"
echo ""
echo "4. 如果仍然出现server_error："
echo "   - 尝试重新生成客户端密钥"
echo "   - 检查Microsoft服务状态"
echo "   - 联系Azure支持"

echo ""
echo "📋 下一步操作："
echo "1. 按照上述建议检查Azure配置"
echo "2. 尝试手动测试URL"
echo "3. 如果问题仍然存在，查看详细日志："
echo "   tail -f logs/backend.log | grep -i 'oauth\\|microsoft\\|error'"

echo ""
echo "📞 获取帮助："
echo "   - 排查指南: docs/oauth2-troubleshooting.md"
echo "   - Azure配置: docs/azure-oauth2-setup.md"
echo "   - 配置清单: docs/azure-oauth2-checklist.md"