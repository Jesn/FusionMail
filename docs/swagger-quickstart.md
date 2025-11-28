# Swagger API 文档快速开始

## 5 分钟快速上手

### 步骤 1：启用 Swagger

编辑 `backend/.env` 文件，添加或修改：

```bash
SWAGGER_ENABLED=true
```

### 步骤 2：启动服务

```bash
cd backend
go run cmd/server/main.go
```

### 步骤 3：访问文档

打开浏览器访问：

```
http://localhost:3333/swagger/index.html
```

### 步骤 4：测试接口

1. **登录获取 Token**
   - 找到 `POST /api/v1/auth/login` 接口
   - 点击 "Try it out"
   - 输入用户名和密码
   - 点击 "Execute"
   - 复制返回的 `token`

2. **配置认证**
   - 点击页面右上角的 "Authorize" 按钮
   - 在 "BearerAuth" 输入框输入：`Bearer {刚才复制的token}`
   - 点击 "Authorize" 确认

3. **调用其他接口**
   - 现在可以测试任何需要认证的接口了
   - 例如：`GET /api/v1/emails` 获取邮件列表

## Docker 环境

### 步骤 1：修改 Docker Compose

编辑 `docker-compose.dev.yml`：

```yaml
services:
  backend:
    environment:
      - SWAGGER_ENABLED=true
```

### 步骤 2：启动容器

```bash
docker-compose -f docker-compose.dev.yml up -d
```

### 步骤 3：访问文档

```
http://localhost:3333/swagger/index.html
```

## 常用命令

```bash
# 生成 Swagger 文档
cd backend
make swagger

# 构建项目
make build

# 运行项目
make run

# 完整构建（生成文档 + 构建）
make all
```

## 接口测试示例

### 1. 用户登录

```bash
curl -X POST "http://localhost:3333/api/v1/auth/login" \
  -H "Content-Type: application/json" \
  -d '{
    "username": "admin",
    "password": "your_password"
  }'
```

### 2. 获取邮件列表

```bash
curl -X GET "http://localhost:3333/api/v1/emails?page=1&page_size=20" \
  -H "Authorization: Bearer {your_token}"
```

### 3. 创建规则

```bash
curl -X POST "http://localhost:3333/api/v1/rules" \
  -H "Authorization: Bearer {your_token}" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "测试规则",
    "enabled": true,
    "conditions": {...},
    "actions": {...}
  }'
```

## 注意事项

### ⚠️ 生产环境

**生产环境请务必关闭 Swagger 文档**：

```bash
SWAGGER_ENABLED=false
```

### ✅ 开发环境

开发环境可以放心启用，方便调试和测试。

## 下一步

- 查看 [完整使用指南](./swagger-guide.md)
- 查看 [集成总结](./swagger-integration-summary.md)
- 访问 [Swagger 官方文档](https://swagger.io/docs/)

## 问题反馈

如果遇到问题，请检查：

1. 环境变量是否正确设置
2. 服务是否正常启动
3. 端口 3333 是否被占用
4. 查看服务日志获取详细错误信息

---

**提示**：首次使用建议先阅读 [swagger-guide.md](./swagger-guide.md) 了解更多功能和安全建议。
