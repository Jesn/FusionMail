.PHONY: swagger build run test clean

# 生成 Swagger 文档
swagger:
	@echo "生成 Swagger API 文档..."
	@swag init -g cmd/server/main.go -o docs --parseDependency --parseInternal
	@echo "✅ Swagger 文档生成成功"

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
