# Swagger API 文档使用指南

## 概述

FusionMail 已集成 Swagger API 文档，提供交互式的 API 接口文档和在线测试功能。

## 配置说明

### 1. 启用 Swagger 文档

Swagger 文档默认是**关闭**的，需要通过环境变量启用。

#### 方式一：环境变量配置

在 `backend/.env` 文件中添加：

```bash
# 启用 Swagger API 文档
SWAGGER_ENABLED=true
```

#### 方式二：Docker Compose 配置

在 `docker-compose.yml` 或 `docker-compose.dev.yml` 中添加环境变量：

```yaml
services:
  backend:
    environment:
      - SWAGGER_ENABLED=true
```

### 2. 重新生成 Swagger 文档

当修改了 API 接口或注释后，需要重新生成 Swagger 文档：

```bash
cd backend
~/go/bin/swag init -g cmd/server/main.go -o docs --parseDependency --parseInternal
```

或者使用 Makefile（如果有）：

```bash
make swagger
```

### 3. 启动服务

```bash
# 开发环境
cd backend
go run cmd/server/main.go

# 或使用 Docker
docker-compose up -d
```

## 访问 Swagger UI

启用后，可以通过以下地址访问 Swagger 文档：

```
http://localhost:3333/swagger/index.html
```

## 功能特性

### 1. 接口文档浏览

- **分组展示**：按功能模块分组（认证、账户管理、邮件管理等）
- **详细说明**：每个接口包含请求参数、响应格式、状态码说明
- **数据模型**：查看请求和响应的数据结构

### 2. 在线测试

- **认证支持**：支持 JWT Token 和 API Key 两种认证方式
- **参数填写**：直接在页面填写请求参数
- **实时调用**：点击 "Try it out" 按钮即可发送真实请求
- **响应查看**：查看实际的响应数据和状态码

### 3. 认证配置

#### JWT Token 认证

1. 点击页面右上角的 "Authorize" 按钮
2. 在 "BearerAuth" 输入框中输入：`Bearer {your_token}`
3. 点击 "Authorize" 确认
4. 现在可以调用需要认证的接口了

#### API Key 认证

1. 点击页面右上角的 "Authorize" 按钮
2. 在 "ApiKeyAuth" 输入框中输入你的 API Key
3. 点击 "Authorize" 确认
4. 现在可以调用公共接口了

## 接口分组说明

### 认证 (Authentication)
- 用户登录
- Token 刷新
- 修改密码
- 获取当前用户信息

### 账户管理 (Accounts)
- 创建邮箱账户
- 查询账户列表
- 更新账户信息
- 删除账户
- 测试连接
- 手动同步

### 邮件管理 (Emails)
- 获取邮件列表
- 查询邮件详情
- 搜索邮件
- 标记已读/未读
- 星标邮件
- 归档/删除邮件

### 规则管理 (Rules)
- 创建规则
- 查询规则列表
- 更新规则
- 删除规则
- 启用/禁用规则
- 测试规则

### Webhook
- 创建 Webhook
- 查询 Webhook 列表
- 更新 Webhook
- 删除 Webhook
- 测试 Webhook
- 查看 Webhook 日志

### 垃圾邮件 (Spam)
- 标记垃圾邮件
- 查询垃圾邮件列表
- 垃圾邮件统计
- 贝叶斯分类器管理
- 垃圾邮件规则管理

### 系统管理 (System)
- 健康检查
- 系统统计
- 同步状态
- 同步日志

### 公共接口 (Public)
- 接收邮件（需要 API Key）
- 搜索邮件（需要 API Key）

## 安全建议

### 生产环境

**强烈建议在生产环境中关闭 Swagger 文档**，原因：

1. **安全风险**：暴露 API 接口结构可能被恶意利用
2. **性能影响**：Swagger UI 会占用一定的服务器资源
3. **信息泄露**：可能暴露内部业务逻辑和数据结构

生产环境配置：

```bash
# 生产环境关闭 Swagger
SWAGGER_ENABLED=false
```

### 开发/测试环境

开发和测试环境可以启用 Swagger 文档，方便：

- 前端开发人员查看 API 接口
- 测试人员进行接口测试
- 第三方集成开发

### 访问控制

如果必须在生产环境启用 Swagger，建议：

1. **IP 白名单**：通过 Nginx 限制只允许特定 IP 访问
2. **VPN 访问**：只允许通过 VPN 访问
3. **HTTP 基本认证**：添加额外的认证层

Nginx 配置示例：

```nginx
location /swagger/ {
    # 只允许内网 IP 访问
    allow 192.168.1.0/24;
    allow 10.0.0.0/8;
    deny all;
    
    proxy_pass http://backend:3333;
}
```

## 常见问题

### 1. 访问 /swagger/index.html 返回 404

**原因**：Swagger 文档未启用

**解决**：
- 检查环境变量 `SWAGGER_ENABLED=true` 是否设置
- 重启服务使配置生效

### 2. Swagger UI 显示但接口列表为空

**原因**：Swagger 文档未生成或生成失败

**解决**：
```bash
cd backend
~/go/bin/swag init -g cmd/server/main.go -o docs --parseDependency --parseInternal
```

### 3. 接口调用返回 401 Unauthorized

**原因**：未配置认证信息

**解决**：
- 点击 "Authorize" 按钮
- 输入有效的 JWT Token 或 API Key
- 确认后重新调用接口

### 4. 修改代码后 Swagger 文档未更新

**原因**：需要重新生成文档

**解决**：
```bash
cd backend
~/go/bin/swag init -g cmd/server/main.go -o docs --parseDependency --parseInternal
go run cmd/server/main.go
```

### 5. Swagger 生成报错

**原因**：Swagger 注释格式不正确

**解决**：
- 检查注释格式是否符合 Swagger 规范
- 确保类型定义存在且可访问
- 查看错误信息定位具体文件和行号

## Swagger 注释规范

### 基本格式

```go
// FunctionName 函数说明
// @Summary 简短描述
// @Description 详细描述
// @Tags 标签名称
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param name query string false "参数说明"
// @Success 200 {object} response.Response
// @Failure 400 {object} response.Response
// @Router /path [method]
func FunctionName(c *gin.Context) {
    // 实现
}
```

### 参数类型

- `query`：URL 查询参数
- `path`：URL 路径参数
- `body`：请求体参数
- `header`：请求头参数

### 数据类型

- 基本类型：`string`, `int`, `bool`, `float64`
- 对象：`{object} ModelName`
- 数组：`{array} ModelName`

## 相关资源

- [Swagger 官方文档](https://swagger.io/docs/)
- [Swaggo GitHub](https://github.com/swaggo/swag)
- [Gin Swagger](https://github.com/swaggo/gin-swagger)
- [OpenAPI 规范](https://spec.openapis.org/oas/latest.html)

## 更新日志

- **2024-11-28**：初始版本，集成 Swagger 支持
  - 添加配置开关（默认关闭）
  - 生成基础 API 文档
  - 支持 JWT 和 API Key 认证
