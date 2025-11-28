#!/bin/bash

# Swagger 集成验证脚本

set -e

echo "🔍 开始验证 Swagger 集成..."
echo ""

# 检查 swag 命令是否存在
echo "1️⃣ 检查 swag 工具..."
if command -v swag &> /dev/null; then
    echo "✅ swag 工具已安装"
    swag --version
else
    echo "❌ swag 工具未安装"
    echo "请运行: go install github.com/swaggo/swag/cmd/swag@latest"
    exit 1
fi
echo ""

# 检查 Swagger 文档文件
echo "2️⃣ 检查 Swagger 文档文件..."
if [ -f "docs/docs.go" ]; then
    echo "✅ docs/docs.go 存在"
else
    echo "❌ docs/docs.go 不存在"
    exit 1
fi

if [ -f "docs/swagger.json" ]; then
    echo "✅ docs/swagger.json 存在"
else
    echo "⚠️  docs/swagger.json 不存在，尝试生成..."
    swag init -g cmd/server/main.go -o docs --parseDependency --parseInternal
fi

if [ -f "docs/swagger.yaml" ]; then
    echo "✅ docs/swagger.yaml 存在"
else
    echo "⚠️  docs/swagger.yaml 不存在"
fi
echo ""

# 检查配置文件
echo "3️⃣ 检查配置文件..."
if grep -q "SwaggerConfig" config/config.go; then
    echo "✅ config/config.go 包含 SwaggerConfig"
else
    echo "❌ config/config.go 缺少 SwaggerConfig"
    exit 1
fi

if grep -q "SWAGGER_ENABLED" .env.example; then
    echo "✅ .env.example 包含 SWAGGER_ENABLED"
else
    echo "❌ .env.example 缺少 SWAGGER_ENABLED"
    exit 1
fi
echo ""

# 检查路由配置
echo "4️⃣ 检查路由配置..."
if grep -q "ginSwagger" internal/router/router.go; then
    echo "✅ router.go 包含 Swagger 路由"
else
    echo "❌ router.go 缺少 Swagger 路由"
    exit 1
fi
echo ""

# 尝试构建项目
echo "5️⃣ 验证项目构建..."
if go build -o /tmp/fusionmail-test ./cmd/server > /dev/null 2>&1; then
    echo "✅ 项目构建成功"
    rm -f /tmp/fusionmail-test
else
    echo "❌ 项目构建失败"
    echo "请运行: go build -o bin/server ./cmd/server"
    exit 1
fi
echo ""

# 检查文档文件
echo "6️⃣ 检查文档文件..."
docs=(
    "../docs/swagger-quickstart.md"
    "../docs/swagger-guide.md"
    "../docs/swagger-integration-summary.md"
)

for doc in "${docs[@]}"; do
    if [ -f "$doc" ]; then
        echo "✅ $(basename $doc) 存在"
    else
        echo "⚠️  $(basename $doc) 不存在"
    fi
done
echo ""

# 检查 Makefile
echo "7️⃣ 检查 Makefile..."
if [ -f "Makefile" ]; then
    echo "✅ Makefile 存在"
    if grep -q "swagger:" Makefile; then
        echo "✅ Makefile 包含 swagger 目标"
    else
        echo "⚠️  Makefile 缺少 swagger 目标"
    fi
else
    echo "⚠️  Makefile 不存在"
fi
echo ""

# 最终总结
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "✅ Swagger 集成验证完成！"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo ""
echo "📝 下一步："
echo "  1. 设置环境变量: SWAGGER_ENABLED=true"
echo "  2. 启动服务: go run cmd/server/main.go"
echo "  3. 访问文档: http://localhost:3333/swagger/index.html"
echo ""
echo "📚 相关文档："
echo "  - 快速开始: docs/swagger-quickstart.md"
echo "  - 使用指南: docs/swagger-guide.md"
echo "  - 集成总结: docs/swagger-integration-summary.md"
echo ""
