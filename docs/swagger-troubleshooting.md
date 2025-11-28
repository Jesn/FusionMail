# Swagger 访问问题排查指南

## 问题：访问 /swagger/index.html 跳转到登录页

### 原因分析

这个问题通常由以下原因导致：

1. **Swagger 未启用** - 环境变量未设置
2. **服务未重启** - 修改配置后未重启服务
3. **路由优先级问题** - 静态文件服务覆盖了 Swagger 路由

### 解决步骤

#### 步骤 1：确认环境变量

编辑 `backend/.env` 文件，确保包含：

```bash
SWAGGER_ENABLED=true
```

#### 步骤 2：重新生成 Swagger 文档

```bash
cd backend
make swagger
# 或
swag init -g cmd/server/main.go -o docs --parseDependency --parseInternal
```

#### 步骤 3：重新构建项目

```bash
cd backend
go build -o bin/server ./cmd/server
```

#### 步骤 4：停止旧服务

```bash
# 查找正在运行的服务
ps aux | grep "cmd/server\|bin/server" | grep -v grep

# 停止服务（替换 PID）
kill <PID>

# 或使用 pkill
pkill -f "cmd/server"
pkill -f "bin/server"
```

#### 步骤 5：启动新服务

```bash
cd backend
./bin/server
# 或
go run cmd/server/main.go
```

#### 步骤 6：验证 Swagger 状态

查看启动日志，应该看到：

```
Swagger documentation enabled at: http://0.0.0.0:3333/swagger/index.html
```

如果看到：

```
Swagger documentation is disabled (set SWAGGER_ENABLED=true to enable)
```

说明环境变量未生效，返回步骤 1。

#### 步骤 7：测试访问

```bash
# 测试 Swagger UI
curl -I http://localhost:3333/swagger/index.html

# 应该返回 HTTP 200
# 如果返回 301/302，说明路由被重定向
# 如果返回 404，说明 Swagger 未启用或路由未注册
```

### 常见错误及解决方法

#### 错误 1：HTTP 301 Moved Permanently

**原因**：静态文件服务干扰了 Swagger 路由

**解决**：
1. 确认 Swagger 路由在 `internal/router/router.go` 中正确注册
2. 确认 `NoRoute` 处理器排除了 `/swagger/` 路径
3. 重启服务

#### 错误 2：HTTP 404 Not Found

**原因**：Swagger 未启用或文档未生成

**解决**：
1. 检查 `SWAGGER_ENABLED=true`
2. 重新生成文档：`make swagger`
3. 检查 `backend/docs/` 目录是否有文件
4. 重启服务

#### 错误 3：页面跳转到登录页

**原因**：`NoRoute` 处理器捕获了 Swagger 请求

**解决**：
1. 检查 `cmd/server/main.go` 中的 `NoRoute` 处理器
2. 确认排除了 `/swagger/` 路径
3. 重启服务

### 验证清单

- [ ] `backend/.env` 包含 `SWAGGER_ENABLED=true`
- [ ] `backend/docs/` 目录存在且包含文件
- [ ] 项目重新构建成功
- [ ] 旧服务已停止
- [ ] 新服务已启动
- [ ] 启动日志显示 "Swagger documentation enabled"
- [ ] `curl -I http://localhost:3333/swagger/index.html` 返回 200

### 调试命令

```bash
# 1. 检查环境变量
cat backend/.env | grep SWAGGER

# 2. 检查 Swagger 文档
ls -la backend/docs/

# 3. 检查服务进程
ps aux | grep server | grep -v grep

# 4. 测试 Swagger 路由
curl -I http://localhost:3333/swagger/index.html
curl -I http://localhost:3333/swagger/doc.json

# 5. 查看服务日志
# （如果使用 systemd）
journalctl -u fusionmail -f

# 6. 测试 API 健康检查
curl http://localhost:3333/api/v1/health
```

### 完整重启流程

```bash
#!/bin/bash

echo "🔄 完整重启 Swagger 服务..."

# 1. 停止服务
echo "1️⃣ 停止旧服务..."
pkill -f "cmd/server" || true
pkill -f "bin/server" || true
sleep 2

# 2. 检查环境变量
echo "2️⃣ 检查环境变量..."
if ! grep -q "SWAGGER_ENABLED=true" backend/.env; then
    echo "添加 SWAGGER_ENABLED=true 到 .env"
    echo "SWAGGER_ENABLED=true" >> backend/.env
fi

# 3. 重新生成文档
echo "3️⃣ 重新生成 Swagger 文档..."
cd backend
make swagger || swag init -g cmd/server/main.go -o docs --parseDependency --parseInternal

# 4. 构建项目
echo "4️⃣ 构建项目..."
go build -o bin/server ./cmd/server

# 5. 启动服务
echo "5️⃣ 启动服务..."
./bin/server &

# 6. 等待服务启动
echo "6️⃣ 等待服务启动..."
sleep 3

# 7. 测试访问
echo "7️⃣ 测试 Swagger 访问..."
response=$(curl -s -o /dev/null -w "%{http_code}" http://localhost:3333/swagger/index.html)

if [ "$response" = "200" ]; then
    echo "✅ Swagger 文档可访问！"
    echo "📖 访问地址: http://localhost:3333/swagger/index.html"
else
    echo "❌ Swagger 文档无法访问 (HTTP $response)"
    echo "请检查服务日志"
fi
```

### 联系支持

如果以上步骤都无法解决问题，请：

1. 收集服务启动日志
2. 收集 `curl -v http://localhost:3333/swagger/index.html` 的完整输出
3. 提供 `backend/.env` 文件内容（隐藏敏感信息）
4. 提供 Go 版本：`go version`
5. 提供 Gin 版本：`go list -m github.com/gin-gonic/gin`

---

**最后更新**：2024-11-28
