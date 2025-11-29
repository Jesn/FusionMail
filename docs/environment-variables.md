# 环境变量配置说明

## 📋 完整环境变量列表

### 🔒 安全配置（必填）

#### JWT_SECRET
- **说明**：JWT 令牌签名密钥
- **默认值**：`dev-secret-key-for-testing-only`
- **生产环境**：⚠️ **必须修改**
- **要求**：至少 32 字符
- **生成命令**：
  ```bash
  openssl rand -base64 32
  ```
- **示例**：
  ```bash
  JWT_SECRET=your-jwt-secret-at-least-32-chars-long
  ```

#### ENCRYPTION_KEY
- **说明**：数据加密密钥（用于加密敏感数据）
- **默认值**：`fusionmail-default-key-32-bytes`
- **生产环境**：⚠️ **必须修改**
- **要求**：必须是 32 字节
- **⚠️ 警告**：设置后不可更改，否则已加密数据无法解密
- **生成命令**：
  ```bash
  openssl rand -base64 32 | cut -c1-32
  ```
- **示例**：
  ```bash
  ENCRYPTION_KEY=your-32-byte-encryption-key-here
  ```

#### ADMIN_PASSWORD
- **说明**：管理员初始密码
- **默认值**：无（系统自动生成随机密码）
- **生产环境**：建议设置
- **要求**：至少 8 字符
- **示例**：
  ```bash
  ADMIN_PASSWORD=YourStrongPassword123!
  ```
- **说明**：
  - 如果不设置，系统会自动生成随机密码并显示在日志中
  - 首次登录后建议立即修改

---

### 🗄️ 数据库配置

#### DB_HOST
- **说明**：PostgreSQL 数据库主机地址
- **默认值**：`localhost`
- **示例**：`192.168.2.200`

#### DB_PORT
- **说明**：PostgreSQL 数据库端口
- **默认值**：`5432`

#### DB_USER
- **说明**：数据库用户名
- **默认值**：`fusionmail`

#### DB_PASSWORD
- **说明**：数据库密码
- **默认值**：`fusionmail_password`
- **生产环境**：⚠️ **必须修改**
- **生成命令**：
  ```bash
  openssl rand -base64 24
  ```

#### DB_NAME
- **说明**：数据库名称
- **默认值**：`fusionmail`

#### DB_SSLMODE
- **说明**：SSL 连接模式
- **默认值**：`disable`
- **可选值**：`disable`, `require`, `verify-ca`, `verify-full`

---

### 🔴 Redis 配置

#### REDIS_HOST
- **说明**：Redis 服务器地址
- **默认值**：`localhost`

#### REDIS_PORT
- **说明**：Redis 端口
- **默认值**：`6379`

#### REDIS_PASSWORD
- **说明**：Redis 密码
- **默认值**：`fusionmail_redis_password`
- **说明**：如果 Redis 无密码，设置为空字符串

---

### 🌐 服务器配置

#### SERVER_HOST
- **说明**：服务器监听地址
- **默认值**：`0.0.0.0`
- **说明**：`0.0.0.0` 表示监听所有网络接口

#### SERVER_PORT
- **说明**：服务器监听端口
- **默认值**：`3333`

---

### 📦 存储配置

#### STORAGE_TYPE
- **说明**：存储类型
- **默认值**：`local`
- **可选值**：`local`, `s3`, `oss`

#### STORAGE_LOCAL_PATH
- **说明**：本地存储路径
- **默认值**：`./data/attachments`
- **Docker 环境**：`/data/attachments`

#### STORAGE_BASE_URL
- **说明**：存储基础 URL（可选）
- **默认值**：空

---

### 🚦 速率限制配置

#### RATE_LIMIT_ENABLED
- **说明**：是否启用速率限制
- **默认值**：`true`
- **可选值**：`true`, `false`

#### RATE_LIMIT_SITE_REQUESTS_PER_MINUTE
- **说明**：站点默认限速（每分钟请求数）
- **默认值**：`100`

#### RATE_LIMIT_PUBLIC_REQUESTS_PER_MINUTE
- **说明**：公共 API 限速（每分钟请求数）
- **默认值**：`200`

---

### 📚 Swagger 配置

#### SWAGGER_ENABLED
- **说明**：是否启用 Swagger API 文档
- **默认值**：`false`
- **可选值**：`true`, `false`
- **生产环境**：建议设置为 `false`

---

### 🔐 CORS 配置

#### CORS_ALLOWED_ORIGINS
- **说明**：允许的跨域来源（逗号分隔）
- **默认值**：`http://localhost:3333`
- **示例**：
  ```bash
  CORS_ALLOWED_ORIGINS=http://192.168.2.200:3333,https://mail.example.com
  ```

---

### 🍪 Cookie 安全配置

#### COOKIE_SECURE
- **说明**：Cookie Secure 属性配置
- **默认值**：未设置（自动检测）
- **可选值**：
  - 不设置：自动检测（TLS 连接或 `X-Forwarded-Proto: https` 时启用）
  - `true`：强制启用（适用于 Nginx 反向代理 + HTTPS）
  - `false`：强制禁用（适用于纯 HTTP 环境）
- **使用场景**：

  **场景 1：纯 HTTP 环境**
  ```bash
  # 不设置或设置为 false
  COOKIE_SECURE=false
  ```

  **场景 2：直接 HTTPS**
  ```bash
  # 不设置，自动检测
  # 或强制启用
  COOKIE_SECURE=true
  ```

  **场景 3：Nginx 反向代理 + HTTPS**
  ```bash
  # 强制启用（因为后端收到的是 HTTP 请求）
  COOKIE_SECURE=true
  ```
  
  Nginx 配置需要设置：
  ```nginx
  proxy_set_header X-Forwarded-Proto $scheme;
  ```

#### SAVE_PASSWORD_FILE
- **说明**：是否保存管理员密码到文件
- **默认值**：开发环境 `true`，生产环境 `false`
- **可选值**：`true`, `false`
- **说明**：
  - `true`：密码保存到 `/app/passwd` 文件
  - `false`：密码仅显示在日志中

---

### ⏱️ JWT 配置

#### JWT_EXPIRY_HOURS
- **说明**：JWT 令牌过期时间（小时）
- **默认值**：`24`

---

## 📝 配置示例

### 开发环境

```bash
# 数据库
DB_HOST=localhost
DB_PORT=5432
DB_USER=fusionmail
DB_PASSWORD=fusionmail_dev_password
DB_NAME=fusionmail

# Redis
REDIS_HOST=localhost
REDIS_PORT=6379
REDIS_PASSWORD=fusionmail_redis_password

# 安全（开发环境可以使用默认值）
JWT_SECRET=dev-secret-key-for-testing-only
ENCRYPTION_KEY=fusionmail-default-key-32-bytes

# Swagger（开发环境启用）
SWAGGER_ENABLED=true

# Cookie（开发环境自动检测）
# COOKIE_SECURE 不设置
```

### 生产环境（纯 HTTP）

```bash
# 数据库
DB_HOST=192.168.2.200
DB_PORT=5432
DB_USER=postgres
DB_PASSWORD=your-strong-db-password

# Redis
REDIS_HOST=192.168.2.200
REDIS_PORT=6379
REDIS_PASSWORD=

# 安全（必须修改）
JWT_SECRET=$(openssl rand -base64 32)
ENCRYPTION_KEY=$(openssl rand -base64 32 | cut -c1-32)
ADMIN_PASSWORD=YourStrongPassword123!

# Swagger（生产环境关闭）
SWAGGER_ENABLED=false

# Cookie（HTTP 环境）
COOKIE_SECURE=false

# CORS
CORS_ALLOWED_ORIGINS=http://192.168.2.200:3333
```

### 生产环境（Nginx + HTTPS）

```bash
# 数据库
DB_HOST=192.168.2.200
DB_PORT=5432
DB_USER=postgres
DB_PASSWORD=your-strong-db-password

# Redis
REDIS_HOST=192.168.2.200
REDIS_PORT=6379
REDIS_PASSWORD=

# 安全（必须修改）
JWT_SECRET=$(openssl rand -base64 32)
ENCRYPTION_KEY=$(openssl rand -base64 32 | cut -c1-32)
ADMIN_PASSWORD=YourStrongPassword123!

# Swagger（生产环境关闭）
SWAGGER_ENABLED=false

# Cookie（Nginx 反向代理 + HTTPS）
COOKIE_SECURE=true

# CORS
CORS_ALLOWED_ORIGINS=https://mail.example.com
```

---

## 🔍 配置验证

### 检查配置是否生效

```bash
# 查看容器日志
docker-compose logs fusionmail | grep "Configuration loaded"

# 检查 JWT Secret
docker-compose logs fusionmail | grep "JWT"

# 检查数据库连接
docker-compose logs fusionmail | grep "Database"

# 检查 Redis 连接
docker-compose logs fusionmail | grep "Redis"
```

### 常见配置错误

#### 1. JWT_SECRET 使用默认值
**错误**：生产环境使用 `dev-secret-key-for-testing-only`

**风险**：JWT 令牌可被伪造

**解决**：
```bash
JWT_SECRET=$(openssl rand -base64 32)
```

#### 2. ENCRYPTION_KEY 使用默认值
**错误**：生产环境使用 `fusionmail-default-key-32-bytes`

**风险**：敏感数据可被解密

**解决**：
```bash
ENCRYPTION_KEY=$(openssl rand -base64 32 | cut -c1-32)
```

#### 3. COOKIE_SECURE 配置错误
**错误**：Nginx HTTPS 反向代理场景下未设置 `COOKIE_SECURE=true`

**现象**：Cookie 在 HTTP 传输，可能被窃取

**解决**：
```bash
COOKIE_SECURE=true
```

并确保 Nginx 配置：
```nginx
proxy_set_header X-Forwarded-Proto $scheme;
```

---

## 🆘 故障排查

### 问题 1：无法连接数据库

**检查步骤**：
1. 确认数据库地址和端口正确
2. 确认数据库密码正确
3. 测试数据库连接：
   ```bash
   psql -h $DB_HOST -U $DB_USER -d $DB_NAME
   ```

### 问题 2：无法连接 Redis

**检查步骤**：
1. 确认 Redis 地址和端口正确
2. 测试 Redis 连接：
   ```bash
   redis-cli -h $REDIS_HOST -p $REDIS_PORT ping
   # 如果有密码
   redis-cli -h $REDIS_HOST -p $REDIS_PORT -a $REDIS_PASSWORD ping
   ```

### 问题 3：登录后 Cookie 无效

**可能原因**：`COOKIE_SECURE` 配置不正确

**解决方案**：
- HTTP 环境：`COOKIE_SECURE=false` 或不设置
- HTTPS 环境：`COOKIE_SECURE=true`

---

## 📚 相关文档

- [生产环境部署检查清单](./production-deployment-checklist.md)
- [HTTP/HTTPS 部署指南](./http-https-deployment-guide.md)
- [环境变量模板](../.env.prod.example)
