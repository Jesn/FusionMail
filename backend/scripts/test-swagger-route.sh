#!/bin/bash

# 测试 Swagger 路由是否正确配置

echo "🧪 测试 Swagger 路由配置..."
echo ""

# 检查环境变量
echo "1️⃣ 检查环境变量..."
if grep -q "SWAGGER_ENABLED=true" ../.env 2>/dev/null; then
    echo "✅ SWAGGER_ENABLED=true 已设置"
else
    echo "❌ SWAGGER_ENABLED 未设置为 true"
    echo "请在 backend/.env 中添加: SWAGGER_ENABLED=true"
    exit 1
fi
echo ""

# 检查服务是否运行
echo "2️⃣ 检查服务状态..."
if curl -s http://localhost:3333/api/v1/health > /dev/null 2>&1; then
    echo "✅ 后端服务正在运行"
else
    echo "❌ 后端服务未运行"
    echo "请先启动服务: go run cmd/server/main.go"
    exit 1
fi
echo ""

# 测试 Swagger 路由
echo "3️⃣ 测试 Swagger 路由..."
response=$(curl -s -o /dev/null -w "%{http_code}" http://localhost:3333/swagger/index.html)

if [ "$response" = "200" ]; then
    echo "✅ Swagger 文档可访问 (HTTP $response)"
    echo ""
    echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
    echo "✅ Swagger 路由配置正确！"
    echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
    echo ""
    echo "📖 访问地址: http://localhost:3333/swagger/index.html"
    echo ""
elif [ "$response" = "404" ]; then
    echo "❌ Swagger 文档返回 404"
    echo ""
    echo "可能的原因："
    echo "  1. SWAGGER_ENABLED 未设置为 true"
    echo "  2. 服务未重启（修改环境变量后需要重启）"
    echo "  3. Swagger 文档未生成"
    echo ""
    echo "解决方法："
    echo "  1. 确认 backend/.env 中 SWAGGER_ENABLED=true"
    echo "  2. 重新生成文档: make swagger"
    echo "  3. 重启服务: go run cmd/server/main.go"
    echo ""
    exit 1
elif [ "$response" = "301" ] || [ "$response" = "302" ]; then
    echo "⚠️  Swagger 路由被重定向 (HTTP $response)"
    echo ""
    echo "这可能是因为 NoRoute 处理器捕获了 Swagger 路由"
    echo "请检查 cmd/server/main.go 中的路由配置"
    echo ""
    exit 1
else
    echo "❌ 意外的响应码: HTTP $response"
    echo ""
    echo "请检查服务日志获取详细信息"
    echo ""
    exit 1
fi
