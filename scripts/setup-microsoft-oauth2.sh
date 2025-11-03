#!/bin/bash

# Microsoft Graph OAuth2 快速配置脚本
# 用于验证和测试Microsoft OAuth2配置

set -e

echo "🔧 Microsoft Graph OAuth2 配置验证工具"
echo "========================================"

# 检查环境变量
echo "📋 检查环境变量配置..."

if [ -f "backend/.env" ]; then
    source backend/.env
    echo "✅ 找到环境变量文件"
else
    echo "❌ 未找到 backend/.env 文件"
    exit 1
fi

# 验证必需的环境变量
required_vars=("MICROSOFT_CLIENT_ID" "MICROSOFT_CLIENT_SECRET" "MICROSOFT_REDIRECT_URI")
missing_vars=()

for var in "${required_vars[@]}"; do
    if [ -z "${!var}" ] || [ "${!var}" = "your_microsoft_client_id" ] || [ "${!var}" = "your_microsoft_client_secret" ]; then
        missing_vars+=("$var")
    fi
done

if [ ${#missing_vars[@]} -ne 0 ]; then
    echo "❌ 以下环境变量未正确配置："
    for var in "${missing_vars[@]}"; do
        echo "   - $var"
    done
    echo ""
    echo "请按照以下步骤配置："
    echo "1. 访问 https://portal.azure.com/"
    echo "2. 创建应用注册"
    echo "3. 配置API权限：Mail.ReadWrite, User.Read, offline_access"
    echo "4. 创建客户端密钥"
    echo "5. 更新 backend/.env 文件"
    echo ""
    echo "详细配置指南：docs/azure-oauth2-setup.md"
    exit 1
fi

echo "✅ 环境变量配置完整"

# 显示配置信息
echo ""
echo "📊 当前配置信息："
echo "   Client ID: ${MICROSOFT_CLIENT_ID:0:8}...${MICROSOFT_CLIENT_ID: -4}"
echo "   Client Secret: ${MICROSOFT_CLIENT_SECRET:0:8}...${MICROSOFT_CLIENT_SECRET: -4}"
echo "   Redirect URI: $MICROSOFT_REDIRECT_URI"

# 验证重定向URI格式
if [[ ! "$MICROSOFT_REDIRECT_URI" =~ ^https?://.*auth/microsoft/callback$ ]]; then
    echo "⚠️  重定向URI格式可能不正确"
    echo "   期望格式: http(s)://domain/api/v1/auth/microsoft/callback"
    echo "   当前配置: $MICROSOFT_REDIRECT_URI"
fi

# 检查服务器端口配置
if [ -n "$SERVER_PORT" ]; then
    expected_port_in_uri=$(echo "$MICROSOFT_REDIRECT_URI" | grep -o ':[0-9]*' | tr -d ':')
    if [ -n "$expected_port_in_uri" ] && [ "$expected_port_in_uri" != "$SERVER_PORT" ]; then
        echo "⚠️  重定向URI中的端口($expected_port_in_uri)与服务器端口($SERVER_PORT)不匹配"
    fi
fi

echo ""
echo "🧪 测试OAuth2端点..."

# 检查后端是否编译成功
if [ -f "backend/fusionmail" ]; then
    echo "✅ 后端可执行文件存在"
else
    echo "📦 编译后端..."
    cd backend
    if go build -o fusionmail ./cmd/server; then
        echo "✅ 后端编译成功"
    else
        echo "❌ 后端编译失败"
        exit 1
    fi
    cd ..
fi

echo ""
echo "🚀 配置验证完成！"
echo ""
echo "下一步操作："
echo "1. 启动后端服务: cd backend && ./fusionmail"
echo "2. 启动前端服务: cd frontend && npm run dev"
echo "3. 访问 http://localhost:5173 测试OAuth2流程"
echo ""
echo "📚 更多信息："
echo "   - Azure配置指南: docs/azure-oauth2-setup.md"
echo "   - API文档: http://localhost:$SERVER_PORT/swagger/index.html"
echo ""
echo "🔗 测试链接："
echo "   - 授权URL: http://localhost:$SERVER_PORT/api/v1/auth/microsoft/authorize"
echo "   - 健康检查: http://localhost:$SERVER_PORT/api/v1/health"