#!/bin/bash

# 一键重启服务并启用 Swagger

set -e

echo "🔄 重启服务并启用 Swagger..."
echo ""

# 进入 backend 目录
cd "$(dirname "$0")/.."

# 1. 停止旧服务
echo "1️⃣ 停止旧服务..."
pkill -f "cmd/server/main.go" 2>/dev/null || true
pkill -f "bin/server" 2>/dev/null || true
sleep 2
echo "✅ 旧服务已停止"
echo ""

# 2. 检查并设置环境变量
echo "2️⃣ 检查环境变量..."
if grep -q "^SWAGGER_ENABLED=true" .env 2>/dev/null; then
    echo "✅ SWAGGER_ENABLED=true 已设置"
else
    if grep -q "^SWAGGER_ENABLED=" .env 2>/dev/null; then
        # 替换现有配置
        sed -i '' 's/^SWAGGER_ENABLED=.*/SWAGGER_ENABLED=true/' .env
        echo "✅ 已更新 SWAGGER_ENABLED=true"
    else
        # 添加新配置
        echo "" >> .env
        echo "# Swagger API 文档配置" >> .env
        echo "SWAGGER_ENABLED=true" >> .env
        echo "✅ 已添加 SWAGGER_ENABLED=true"
    fi
fi
echo ""

# 3. 重新生成 Swagger 文档
echo "3️⃣ 重新生成 Swagger 文档..."
if command -v swag &> /dev/null; then
    swag init -g cmd/server/main.go -o docs --parseDependency --parseInternal
    echo "✅ Swagger 文档生成成功"
elif [ -f ~/go/bin/swag ]; then
    ~/go/bin/swag init -g cmd/server/main.go -o docs --parseDependency --parseInternal
    echo "✅ Swagger 文档生成成功"
else
    echo "⚠️  swag 工具未找到，跳过文档生成"
    echo "如需生成文档，请运行: go install github.com/swaggo/swag/cmd/swag@latest"
fi
echo ""

# 4. 构建项目
echo "4️⃣ 构建项目..."
go build -o bin/server ./cmd/server
echo "✅ 项目构建成功"
echo ""

# 5. 启动服务
echo "5️⃣ 启动服务..."
nohup ./bin/server > /tmp/fusionmail-server.log 2>&1 &
SERVER_PID=$!
echo "✅ 服务已启动 (PID: $SERVER_PID)"
echo ""

# 6. 等待服务启动
echo "6️⃣ 等待服务启动..."
for i in {1..10}; do
    if curl -s http://localhost:3333/api/v1/health > /dev/null 2>&1; then
        echo "✅ 服务已就绪"
        break
    fi
    if [ $i -eq 10 ]; then
        echo "❌ 服务启动超时"
        echo "请查看日志: tail -f /tmp/fusionmail-server.log"
        exit 1
    fi
    sleep 1
    echo -n "."
done
echo ""
echo ""

# 7. 测试 Swagger 访问
echo "7️⃣ 测试 Swagger 访问..."
response=$(curl -s -o /dev/null -w "%{http_code}" http://localhost:3333/swagger/index.html)

if [ "$response" = "200" ]; then
    echo "✅ Swagger 文档可访问！"
    echo ""
    echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
    echo "✅ 服务启动成功！"
    echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
    echo ""
    echo "📖 Swagger 文档: http://localhost:3333/swagger/index.html"
    echo "🏥 健康检查: http://localhost:3333/api/v1/health"
    echo "📝 服务日志: tail -f /tmp/fusionmail-server.log"
    echo "🛑 停止服务: kill $SERVER_PID"
    echo ""
else
    echo "❌ Swagger 文档无法访问 (HTTP $response)"
    echo ""
    echo "请检查服务日志:"
    echo "  tail -f /tmp/fusionmail-server.log"
    echo ""
    echo "或运行排查脚本:"
    echo "  ./scripts/test-swagger-route.sh"
    echo ""
    exit 1
fi
