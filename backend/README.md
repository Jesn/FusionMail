# FusionMail 后端

FusionMail 邮件聚合系统的 Go 后端服务。

## 快速开始

### 前置要求

- Go 1.21+
- PostgreSQL 15+
- Redis 7+

### 安装依赖

```bash
go mod download
```

### 配置环境变量

复制环境变量示例文件并修改配置：

```bash
cp .env.example .env
# 编辑 .env 文件，修改数据库连接等配置
```

### 启动开发环境

首先启动 PostgreSQL 和 Redis：

```bash
# 在项目根目录执行
./scripts/dev-start.sh
```

### 数据库初始化

#### 方式一：使用迁移工具（推荐）

```bash
# 执行数据库迁移
go run cmd/migrate/main.go -action=up

# 检查数据库状态
go run cmd/migrate/main.go -action=status
```

#### 方式二：启动服务器时自动迁移

```bash
# 服务器启动时会自动执行数据库迁移
go run cmd/server/main.go
```

### 运行服务器

```bash
go run cmd/server/main.go
```

服务器将在 `http://localhost:8080` 启动。

## 项目结构

```
backend/
├── cmd/                    # 命令行工具
│   ├── server/            # 主服务器
│   └── migrate/           # 数据库迁移工具
├── internal/              # 内部包
│   ├── model/            # 数据模型
│   ├── repository/       # 数据访问层
│   ├── service/          # 业务逻辑层
│   ├── handler/          # HTTP 处理层
│   ├── middleware/       # 中间件
│   ├── adapter/          # 邮箱协议适配层
│   └── dto/              # 数据传输对象
├── pkg/                   # 公共包
│   ├── database/         # 数据库工具
│   ├── crypto/           # 加密工具
│   ├── logger/           # 日志工具
│   ├── storage/          # 存储工具
│   ├── queue/            # 队列工具
│   └── event/            # 事件总线
├── config/                # 配置
├── migrations/            # 数据库迁移脚本
└── scripts/               # 脚本文件
```

## 开发命令

### 代码格式化

```bash
# 格式化代码
gofmt -w .

# 整理导入
goimports -w .
```

### 代码检查

```bash
# 静态分析
go vet ./...

# 运行 linter（需要安装 golangci-lint）
golangci-lint run
```

### 运行测试

```bash
# 运行所有测试
go test ./...

# 运行测试并显示覆盖率
go test -cover ./...

# 生成覆盖率报告
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out
```

### 构建

```bash
# 构建服务器
go build -o bin/server cmd/server/main.go

# 构建迁移工具
go build -o bin/migrate cmd/migrate/main.go

# 构建所有
make build
```

## 数据库操作

### 连接数据库

```bash
# 使用 Docker 容器连接
docker exec -it fusionmail-postgres psql -U fusionmail -d fusionmail

# 或使用本地 psql
psql -h localhost -p 5432 -U fusionmail -d fusionmail
```

### 常用 SQL 命令

```sql
-- 查看所有表
\dt

-- 查看表结构
\d emails

-- 查看索引
\di

-- 查看数据库大小
SELECT pg_size_pretty(pg_database_size('fusionmail'));

-- 查看表数据量
SELECT COUNT(*) FROM emails;
```

## Redis 操作

```bash
# 连接 Redis
docker exec -it fusionmail-redis redis-cli -a fusionmail_redis_password

# 查看所有键
KEYS *

# 查看键值
GET key_name

# 清空数据库
FLUSHDB
```

## 环境变量

详见 `.env.example` 文件。主要配置项：

- `DB_HOST`, `DB_PORT`, `DB_USER`, `DB_PASSWORD`, `DB_NAME` - 数据库配置
- `SERVER_HOST`, `SERVER_PORT` - 服务器配置
- `REDIS_HOST`, `REDIS_PORT`, `REDIS_PASSWORD` - Redis 配置
- `JWT_SECRET` - JWT 密钥
- `ENCRYPTION_KEY` - 数据加密密钥
- `STORAGE_TYPE`, `STORAGE_PATH` - 存储配置

## API 文档

API 文档将在后续版本中提供。

## 故障排查

### 数据库连接失败

1. 检查 PostgreSQL 是否运行：
   ```bash
   docker-compose -f docker-compose.dev.yml ps
   ```

2. 检查数据库配置是否正确（.env 文件）

3. 测试数据库连接：
   ```bash
   go run cmd/migrate/main.go -action=status
   ```

### 编译错误

1. 确保 Go 版本 >= 1.21：
   ```bash
   go version
   ```

2. 清理并重新下载依赖：
   ```bash
   go clean -modcache
   go mod download
   ```

### 运行时错误

查看日志输出，检查错误信息。常见问题：

- 端口被占用：修改 `SERVER_PORT` 环境变量
- 数据库表不存在：运行数据库迁移
- Redis 连接失败：检查 Redis 配置

## 贡献指南

1. Fork 项目
2. 创建特性分支
3. 提交更改
4. 推送到分支
5. 创建 Pull Request

## 许可证

MIT License
