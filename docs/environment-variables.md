# FusionMail 环境变量配置说明

## 概述

FusionMail 支持通过环境变量进行灵活配置。本文档详细说明所有可用的环境变量及其用途。

## 配置文件位置

### 开发环境
- `backend/.env` - 后端环境变量配置

### Docker 部署
- `.env` - Docker Compose 环境变量配置（项目根目录）
- `.env.example` - 配置模板文件

## 环境变量列表

### 数据库配置

| 变量名 | 默认值 | 说明 | 必需 |
|--------|--------|------|------|
| `DB_HOST` | localhost | 数据库主机地址 | 是 |
| `DB_PORT` | 5432 | 数据库端口 | 是 |
| `DB_USER` | fusionmail | 数据库用户名 | 是 |
| `DB_PASSWORD` | - | 数据库密码 | 是 |
| `DB_NAME` | fusionmail | 数据库名称 | 是 |
| `DB_SSLMODE` | disable | SSL 模式 | 否 |

**注意**：
- 开发环境使用 `localhost`
- Docker 部署使用服务名 `postgres`

### Redis 配置

| 变量名 | 默认值 | 说明 | 必需 |
|--------|--------|------|------|
| `REDIS_HOST` | localhost | Redis 主机地址 | 是 |
| `REDIS_PORT` | 6379 | Redis 端口 | 是 |
| `REDIS_PASSWORD` | - | Redis 密码 | 是 |
| `REDIS_DB` | 0 | Redis 数据库编号 | 否 |

**注意**：
- 开发环境使用 `localhost`
- Docker 部署使用服务名 `redis`

### 服务器配置

| 变量名 | 默认值 | 说明 | 必需 |
|--------|--------|------|------|
| `SERVER_HOST` | 0.0.0.0 | 服务器监听地址 | 否 |
| `SERVER_PORT` | 3333 | 服务器监听端口 | 否 |
| `CORS_ALLOWED_ORIGINS` | - | CORS 允许的源（逗号分隔） | 否 |

### 安全配置

#### JWT 配置

| 变量名 | 默认值 | 说明 | 必需 |
|--------|--------|------|------|
| `JWT_SECRET` | - | JWT 签名密钥（至少 32 字符） | 是 |
| `JWT_EXPIRY_HOURS` | 24 | JWT 过期时间（小时） | 否 |

**生成方法**：
```bash
openssl rand -base64 32
```

#### 加密配置

| 变量名 | 默认值 | 说明 | 必需 |
|--------|--------|------|------|
| `ENCRYPTION_KEY` | - | AES-256 加密密钥（32 字节） | 是 |

**生成方法**：
```bash
openssl rand -base64 32 | head -c 32
```

#### 管理员密码配置 ⭐ 新增

| 变量名 | 默认值 | 说明 | 必需 |
|--------|--------|------|------|
| `ADMIN_PASSWORD` | - | 管理员初始密码（至少 8 字符） | 否 |
| `SAVE_PASSWORD_FILE` | - | 是否保存密码到文件 | 否 |

**说明**：

1. **ADMIN_PASSWORD**
   - 如果设置：使用指定的密码作为管理员初始密码
   - 如果不设置：自动生成 16 字符的随机密码
   - 生产环境建议设置强密码
   - 密码要求：至少 8 个字符

2. **SAVE_PASSWORD_FILE**
   - `true`：强制保存密码到 `passwd` 文件
   - `false`：不保存密码文件
   - 不设置：根据 `GIN_MODE` 自动决定
     - 开发环境（`GIN_MODE=debug`）：自动保存
     - 生产环境（`GIN_MODE=release`）：不保存

**使用示例**：

```bash
# 开发环境 - 使用自动生成的密码
# 不设置 ADMIN_PASSWORD，密码会保存到 backend/passwd

# 生产环境 - 使用预设密码
ADMIN_PASSWORD=MySecureP@ssw0rd123
SAVE_PASSWORD_FILE=false
GIN_MODE=release
```

### 存储配置

| 变量名 | 默认值 | 说明 | 必需 |
|--------|--------|------|------|
| `STORAGE_TYPE` | local | 存储类型（local/s3/oss） | 否 |
| `STORAGE_LOCAL_PATH` | ./data/attachments | 本地存储路径 | 否 |
| `STORAGE_BASE_URL` | - | 存储基础 URL | 否 |

#### S3 配置（可选）

| 变量名 | 默认值 | 说明 | 必需 |
|--------|--------|------|------|
| `AWS_REGION` | - | AWS 区域 | 否 |
| `AWS_ACCESS_KEY_ID` | - | AWS 访问密钥 ID | 否 |
| `AWS_SECRET_ACCESS_KEY` | - | AWS 访问密钥 | 否 |
| `S3_BUCKET` | - | S3 存储桶名称 | 否 |

### 速率限制配置

| 变量名 | 默认值 | 说明 | 必需 |
|--------|--------|------|------|
| `RATE_LIMIT_ENABLED` | true | 是否启用速率限制 | 否 |
| `RATE_LIMIT_SITE_REQUESTS_PER_MINUTE` | 100 | 站点级别限制（次/分钟） | 否 |
| `RATE_LIMIT_PUBLIC_REQUESTS_PER_MINUTE` | 200 | 公共 API 限制（次/分钟） | 否 |

### Swagger 配置

| 变量名 | 默认值 | 说明 | 必需 |
|--------|--------|------|------|
| `SWAGGER_ENABLED` | false | 是否启用 Swagger 文档 | 否 |

**注意**：生产环境建议关闭 Swagger 文档。

### 运行模式配置

| 变量名 | 默认值 | 说明 | 必需 |
|--------|--------|------|------|
| `GIN_MODE` | release | Gin 运行模式（debug/release） | 否 |
| `STATIC_PATH` | - | 静态文件路径 | 否 |

**GIN_MODE 说明**：
- `debug`：开发模式，输出详细日志
- `release`：生产模式，优化性能

## 配置示例

### 开发环境配置

```bash
# backend/.env

# 数据库配置
DB_HOST=localhost
DB_PORT=5432
DB_USER=fusionmail
DB_PASSWORD=fusionmail_dev_password
DB_NAME=fusionmail

# Redis 配置
REDIS_HOST=localhost
REDIS_PORT=6379
REDIS_PASSWORD=fusionmail_redis_password

# 安全配置
JWT_SECRET=dev-secret-key-for-testing-only
ENCRYPTION_KEY=fusionmail-default-key-32-bytes

# 管理员密码（可选，不设置则自动生成）
# ADMIN_PASSWORD=dev-password-123

# Swagger 文档
SWAGGER_ENABLED=true

# 运行模式
GIN_MODE=debug
```

### 生产环境配置

```bash
# .env (Docker Compose)

# 数据库配置
DB_PASSWORD=your-strong-db-password-here

# Redis 配置
REDIS_PASSWORD=your-strong-redis-password-here

# 安全配置（必须修改）
JWT_SECRET=your-generated-jwt-secret-32-chars-min
ENCRYPTION_KEY=your-generated-encryption-key-32

# 管理员密码（强烈建议设置）
ADMIN_PASSWORD=YourSecureP@ssw0rd123!
SAVE_PASSWORD_FILE=false

# 应用配置
APP_PORT=3333
CORS_ORIGINS=https://yourdomain.com

# 速率限制
RATE_LIMIT_ENABLED=true
RATE_LIMIT_SITE=100
RATE_LIMIT_PUBLIC=200

# Swagger 文档（生产环境关闭）
SWAGGER_ENABLED=false
```

## 安全最佳实践

### 1. 密钥生成

**JWT Secret**：
```bash
openssl rand -base64 32
```

**Encryption Key**：
```bash
openssl rand -base64 32 | head -c 32
```

**管理员密码**：
- 至少 16 个字符
- 包含大小写字母、数字和特殊字符
- 不使用常见密码或字典词汇

### 2. 生产环境检查清单

- [ ] 修改所有默认密码
- [ ] 使用强 JWT Secret（至少 32 字符）
- [ ] 使用强 Encryption Key（32 字节）
- [ ] 设置强管理员密码（`ADMIN_PASSWORD`）
- [ ] 禁用密码文件保存（`SAVE_PASSWORD_FILE=false`）
- [ ] 关闭 Swagger 文档（`SWAGGER_ENABLED=false`）
- [ ] 设置正确的 CORS 源
- [ ] 启用速率限制
- [ ] 使用 `GIN_MODE=release`

### 3. 敏感信息保护

**不要提交到 Git**：
- `.env` 文件
- `backend/.env` 文件
- `backend/passwd` 文件

**已在 .gitignore 中配置**：
```gitignore
.env
.env.local
.env.*.local
backend/.env
backend/passwd
```

### 4. 密码轮换

建议定期更换：
- 数据库密码：每 90 天
- Redis 密码：每 90 天
- JWT Secret：每 180 天
- 管理员密码：每 90 天

## 故障排查

### 问题：环境变量未生效

**检查步骤**：

1. 确认文件位置正确
   ```bash
   # 开发环境
   ls -la backend/.env
   
   # Docker 部署
   ls -la .env
   ```

2. 检查文件格式
   - 使用 `KEY=VALUE` 格式
   - 不要有多余的空格
   - 不要使用引号（除非值包含空格）

3. 重启服务
   ```bash
   # 开发环境
   ./stop.sh && ./start.sh
   
   # Docker 部署
   docker-compose down && docker-compose up -d
   ```

### 问题：管理员密码不正确

**检查步骤**：

1. 确认是否设置了 `ADMIN_PASSWORD`
   ```bash
   # Docker 部署
   docker-compose exec app env | grep ADMIN_PASSWORD
   ```

2. 查看密码文件
   ```bash
   # 开发环境
   cat backend/passwd
   
   # Docker 部署
   docker-compose exec app cat /app/passwd
   ```

3. 检查日志
   ```bash
   # Docker 部署
   docker-compose logs app | grep -i password
   ```

## 相关文档

- [登录密码指南](login-password-guide.md)
- [Docker 部署指南](../DOCKER_DEPLOYMENT_TEST.md)
- [安全配置指南](../README.md#安全配置)
