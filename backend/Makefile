.PHONY: swagger swagger-check build run test clean

# 生成 Swagger 文档
swagger:
	@echo "生成 Swagger API 文档..."
	@swag init -g cmd/server/main.go -o docs --parseDependency --parseInternal
	@echo "✅ Swagger 文档生成成功"

# 检查 Swagger 文档是否最新（CI 用）
swagger-check:
	@echo "检查 Swagger 文档是否最新..."
	@cp docs/docs.go /tmp/docs.go.bak
	@cp docs/swagger.json /tmp/swagger.json.bak
	@cp docs/swagger.yaml /tmp/swagger.yaml.bak
	@swag init -g cmd/server/main.go -o docs --parseDependency --parseInternal 2>/dev/null
	@if diff -q /tmp/docs.go.bak docs/docs.go >/dev/null && \
	    diff -q /tmp/swagger.json.bak docs/swagger.json >/dev/null && \
	    diff -q /tmp/swagger.yaml.bak docs/swagger.yaml >/dev/null; then \
	    echo "✅ Swagger 文档已是最新"; \
	    rm -f /tmp/docs.go.bak /tmp/swagger.json.bak /tmp/swagger.yaml.bak; \
	else \
	    echo "❌ Swagger 文档过期，请运行 'make swagger' 更新后提交"; \
	    diff /tmp/docs.go.bak docs/docs.go | head -20; \
	    rm -f /tmp/docs.go.bak /tmp/swagger.json.bak /tmp/swagger.yaml.bak; \
	    exit 1; \
	fi

# 构建项目
build:
	@echo "构建后端项目..."
	@go build -o bin/server ./cmd/server
	@echo "✅ 构建成功"

# 运行项目
run:
	@echo "启动后端服务..."
	@go run cmd/server/main.go

# 运行测试
test:
	@echo "运行测试..."
	@go test -v ./...

# 清理构建产物
clean:
	@echo "清理构建产物..."
	@rm -rf bin/
	@echo "✅ 清理完成"

# 格式化代码
fmt:
	@echo "格式化代码..."
	@gofmt -w .
	@go mod tidy
	@echo "✅ 格式化完成"

# 安装依赖
deps:
	@echo "安装依赖..."
	@go mod download
	@go install github.com/swaggo/swag/cmd/swag@latest
	@echo "✅ 依赖安装完成"

# 完整构建流程（生成文档 + 构建）
all: swagger build
	@echo "✅ 完整构建完成"
