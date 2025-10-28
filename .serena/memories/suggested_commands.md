# FusionMail 常用命令

## 开发环境管理

### 启动开发环境
```bash
# 启动基础设施（PostgreSQL + Redis）
./scripts/dev-start.sh

# 停止基础设施
./scripts/dev-stop.sh

# 查看服务状态
docker-compose -f docker-compose.dev.yml ps

# 查看日志
docker-compose -f docker-compose.dev.yml logs -f
```

## 后端开发

### Go 项目管理
```bash
cd backend

# 下载依赖
go mod download

# 运行后端服务
go run cmd/server/main.go

# 构建
go build -o bin/server cmd/server/main.go

# 运行测试
go test ./...

# 代码格式化
gofmt -w .
goimports -w .

# 代码检查
go vet ./...
golangci-lint run
```

### 数据库操作
```bash
# 进入 PostgreSQL
docker exec -it fusionmail-postgres psql -U fusionmail -d fusionmail

# 运行迁移
cd backend
go run cmd/migrate/main.go up

# 回滚迁移
go run cmd/migrate/main.go down
```

## 前端开发

### Node.js 项目管理
```bash
cd frontend

# 安装依赖
npm install

# 启动开发服务器
npm run dev

# 构建生产版本
npm run build

# 预览生产构建
npm run preview

# 代码检查
npm run lint

# 类型检查
npm run type-check
```

## Redis 操作
```bash
# 进入 Redis CLI
docker exec -it fusionmail-redis redis-cli -a fusionmail_redis_password

# 查看所有键
KEYS *

# 清空数据库
FLUSHDB
```

## Git 操作
```bash
# 查看状态
git status

# 添加文件
git add .

# 提交（遵循 Conventional Commits）
git commit -m "feat(core): 添加数据库初始化功能"

# 推送
git push origin main
```

## 系统工具（macOS）
```bash
# 查看文件
ls -la

# 查找文件
find . -name "*.go"

# 搜索内容
grep -r "pattern" .

# 查看进程
ps aux | grep fusionmail

# 查看端口占用
lsof -i :8080
```
