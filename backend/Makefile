.PHONY: help build run migrate test clean fmt vet

# 默认目标
help:
	@echo "FusionMail 后端 Makefile"
	@echo ""
	@echo "可用命令:"
	@echo "  make build      - 构建所有二进制文件"
	@echo "  make run        - 运行服务器"
	@echo "  make migrate    - 执行数据库迁移"
	@echo "  make status     - 检查数据库状态"
	@echo "  make test       - 运行测试"
	@echo "  make fmt        - 格式化代码"
	@echo "  make vet        - 代码静态分析"
	@echo "  make clean      - 清理构建文件"
	@echo "  make deps       - 下载依赖"

# 构建
build:
	@echo "构建服务器..."
	@go build -o bin/server cmd/server/main.go
	@echo "构建迁移工具..."
	@go build -o bin/migrate cmd/migrate/main.go
	@echo "构建完成!"

# 运行服务器
run:
	@echo "启动服务器..."
	@go run cmd/server/main.go

# 执行数据库迁移
migrate:
	@echo "执行数据库迁移..."
	@go run cmd/migrate/main.go -action=up

# 检查数据库状态
status:
	@echo "检查数据库状态..."
	@go run cmd/migrate/main.go -action=status

# 运行测试
test:
	@echo "运行测试..."
	@go test -v ./...

# 运行测试并生成覆盖率报告
test-coverage:
	@echo "运行测试并生成覆盖率报告..."
	@go test -coverprofile=coverage.out ./...
	@go tool cover -html=coverage.out -o coverage.html
	@echo "覆盖率报告已生成: coverage.html"

# 格式化代码
fmt:
	@echo "格式化代码..."
	@gofmt -w .
	@echo "代码格式化完成!"

# 代码静态分析
vet:
	@echo "执行代码静态分析..."
	@go vet ./...
	@echo "静态分析完成!"

# 清理构建文件
clean:
	@echo "清理构建文件..."
	@rm -rf bin/
	@rm -f coverage.out coverage.html
	@echo "清理完成!"

# 下载依赖
deps:
	@echo "下载依赖..."
	@go mod download
	@go mod tidy
	@echo "依赖下载完成!"

# 开发环境启动
dev-up:
	@echo "启动开发环境..."
	@cd .. && ./scripts/dev-start.sh

# 开发环境停止
dev-down:
	@echo "停止开发环境..."
	@cd .. && ./scripts/dev-stop.sh
