# Docker 部署教程

本教程将指导你使用 Docker 在自有服务器或本地环境部署 FusionMail。

## 适用场景

- ✅ 自有 VPS/云服务器
- ✅ 本地开发和测试
- ✅ 完全控制数据和配置
- ✅ 无任何网络限制
- ✅ 支持所有邮箱服务商

## 前置准备

### 1. 服务器要求

- **操作系统**: Linux（推荐 Ubuntu 22.04）、macOS、Windows
- **内存**: ≥1GB（推荐 2GB）
- **存储**: ≥5GB
- **Docker**: 20.10+
- **Docker Compose**: 2.0+

### 2. 安装 Docker

**Ubuntu/Debian:**
```bash
# 安装 Docker
curl -fsSL https://get.docker.com | sh

# 启动 Docker
sudo systemctl start docker
sudo systemctl enable docker

# 添加当前用户到 docker 组（可选）
sudo usermod -aG docker $USER
```

**macOS:**
```bash
# 使用 Homebrew
brew install --cask docker

# 或下载 Docker Desktop
# https://www.docker.com/products/docker-desktop
```

**验证安装:**
```bash
docker --version
docker compose version
```

### 3. 准备数据库

**选项 A：使用外部数据库（推荐生产环境）**

参考 [Supabase 配置](../DEPLOYMENT.md#数据库配置)

**选项 B：使用 Docker 本地数据库（开发环境）**

见下方"完整本地部署"部分

---

## 快速部署（使用外部数据库）

### 步骤 1：克隆项目

```bash
git clone https://github.com/your-repo/fusionmail.git
cd fusionmail
```

### 步骤 2：配置环境变量

```bash
# 复制配置模板
cp .env.example .env

# 编辑配置
nano .env
```

**必需配置：**
```env
# 数据库（使用外部数据库）
DB_HOST=aws-0-xxx.pooler.supabase.com
DB_PORT=5432
DB_USER=postgres.your-project-ref
DB_PASSWORD=your-password
DB_NAME=postgres
DB_SSLMODE=require

# 安全配置
JWT_SECRET=your-32-char-jwt-secret
ENCRYPTION_KEY=your-32-byte-encryption-key

# 管理员密码（可选）
ADMIN_PASSWORD=your-admin-password
```

**可选配置：**
```env
# Redis（推荐）
REDIS_HOST=xxx.upstash.io
REDIS_PORT=6379
REDIS_PASSWORD=your-redis-password
REDIS_TLS=true

# 应用端口
APP_PORT=3333

# CORS 配置
CORS_ORIGINS=https://your-domain.com
```

### 步骤 3：启动服务

```bash
# 使用生产配置启动
docker compose -f scripts/deploy/docker/docker-compose.prod.yml up -d

# 查看日志
docker compose -f scripts/deploy/docker/docker-compose.prod.yml logs -f
```

### 步骤 4：验证部署

```bash
# 健康检查
curl http://localhost:3333/api/v1/health

# 查看容器状态
docker ps
```

访问 `http://localhost:3333` 或 `http://your-server-ip:3333`

---

## 完整本地部署（包含数据库）

适合开发环境或不想使用外部数据库的场景。

### 步骤 1：创建完整配置文件

创建 `docker-compose.local.yml`：

```yaml
services:
  # PostgreSQL 数据库
  postgres:
    image: postgres:15-alpine
    container_name: fusionmail-postgres
    restart: unless-stopped
    environment:
      POSTGRES_USER: fusionmail
      POSTGRES_PASSWORD: fusionmail_password
      POSTGRES_DB: fusionmail
    volumes:
      - postgres_data:/var/lib/postgresql/data
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U fusionmail"]
      interval: 10s
      timeout: 5s
      retries: 5

  # Redis 缓存
  redis:
    image: redis:7-alpine
    container_name: fusionmail-redis
    restart: unless-stopped
    command: redis-server --appendonly yes
    volumes:
      - redis_data:/data
    healthcheck:
      test: ["CMD", "redis-cli", "ping"]
      interval: 10s
      timeout: 5s
      retries: 5

  # FusionMail 应用
  fusionmail:
    build:
      context: .
      dockerfile: Dockerfile
    container_name: fusionmail
    restart: unless-stopped
    ports:
      - "3333:3333"
    environment:
      DB_HOST: postgres
      DB_PORT: 5432
      DB_USER: fusionmail
      DB_PASSWORD: fusionmail_password
      DB_NAME: fusionmail
      DB_SSLMODE: disable
      
      REDIS_HOST: redis
      REDIS_PORT: 6379
      
      JWT_SECRET: your-jwt-secret-at-least-32-characters
      ENCRYPTION_KEY: your-32-byte-encryption-key1
      ADMIN_PASSWORD: admin123
      
      SERVER_HOST: 0.0.0.0
      SERVER_PORT: 3333
    volumes:
      - fusionmail_data:/data
    depends_on:
      postgres:
        condition: service_healthy
      redis:
        condition: service_healthy

volumes:
  postgres_data:
  redis_data:
  fusionmail_data:
```

### 步骤 2：启动所有服务

```bash
docker compose -f docker-compose.local.yml up -d
```

### 步骤 3：查看状态

```bash
# 查看所有容器
docker compose -f docker-compose.local.yml ps

# 查看日志
docker compose -f docker-compose.local.yml logs -f fusionmail
```

---

## 配置 Nginx 反向代理

### 安装 Nginx

```bash
# Ubuntu/Debian
sudo apt update
sudo apt install nginx
```

### 配置站点

创建 `/etc/nginx/sites-available/fusionmail`：

```nginx
server {
    listen 80;
    server_name your-domain.com;

    location / {
        proxy_pass http://127.0.0.1:3333;
        proxy_http_version 1.1;
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection 'upgrade';
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
        proxy_cache_bypass $http_upgrade;
        
        # SSE 支持
        proxy_buffering off;
        proxy_read_timeout 86400s;
    }
}
```

### 启用站点

```bash
sudo ln -s /etc/nginx/sites-available/fusionmail /etc/nginx/sites-enabled/
sudo nginx -t
sudo systemctl reload nginx
```

---

## 配置 SSL 证书

### 使用 Let's Encrypt

```bash
# 安装 Certbot
sudo apt install certbot python3-certbot-nginx

# 获取证书
sudo certbot --nginx -d your-domain.com

# 自动续期测试
sudo certbot renew --dry-run
```

### 手动配置 SSL

更新 Nginx 配置：

```nginx
server {
    listen 443 ssl http2;
    server_name your-domain.com;

    ssl_certificate /path/to/fullchain.pem;
    ssl_certificate_key /path/to/privkey.pem;
    ssl_protocols TLSv1.2 TLSv1.3;
    ssl_ciphers ECDHE-ECDSA-AES128-GCM-SHA256:ECDHE-RSA-AES128-GCM-SHA256;
    ssl_prefer_server_ciphers off;

    location / {
        proxy_pass http://127.0.0.1:3333;
        # ... 其他配置同上
    }
}

server {
    listen 80;
    server_name your-domain.com;
    return 301 https://$server_name$request_uri;
}
```

---

## 常用命令

### 容器管理

```bash
# 启动
docker compose -f scripts/deploy/docker/docker-compose.prod.yml up -d

# 停止
docker compose -f scripts/deploy/docker/docker-compose.prod.yml down

# 重启
docker compose -f scripts/deploy/docker/docker-compose.prod.yml restart

# 查看日志
docker compose -f scripts/deploy/docker/docker-compose.prod.yml logs -f

# 进入容器
docker exec -it fusionmail sh
```

### 数据管理

```bash
# 备份数据卷
docker run --rm -v fusionmail_data:/data -v $(pwd):/backup alpine \
  tar czf /backup/fusionmail-backup.tar.gz /data

# 恢复数据卷
docker run --rm -v fusionmail_data:/data -v $(pwd):/backup alpine \
  tar xzf /backup/fusionmail-backup.tar.gz -C /
```

### 更新部署

```bash
# 拉取最新代码
git pull

# 重新构建并启动
docker compose -f scripts/deploy/docker/docker-compose.prod.yml up -d --build
```

---

## 故障排查

### 容器无法启动

```bash
# 查看详细日志
docker compose -f scripts/deploy/docker/docker-compose.prod.yml logs fusionmail

# 检查配置
docker compose -f scripts/deploy/docker/docker-compose.prod.yml config
```

### 数据库连接失败

1. 检查 `DB_HOST` 是否可访问
2. 确认用户名格式（Supabase: `postgres.{project-ref}`）
3. 检查 SSL 模式设置

```bash
# 测试数据库连接
docker exec -it fusionmail sh -c "nc -zv $DB_HOST $DB_PORT"
```

### 端口冲突

```bash
# 检查端口占用
sudo lsof -i :3333

# 修改端口
# 编辑 .env 文件，设置 APP_PORT=8080
```

### 内存不足

```bash
# 查看容器资源使用
docker stats fusionmail

# 限制内存使用（在 docker-compose.yml 中添加）
# deploy:
#   resources:
#     limits:
#       memory: 1G
```

---

## 生产环境建议

### 1. 安全配置

- 使用强密码
- 启用 HTTPS
- 配置防火墙
- 定期更新

### 2. 监控

```bash
# 使用 Docker 健康检查
docker inspect --format='{{.State.Health.Status}}' fusionmail

# 配置外部监控（如 UptimeRobot）
```

### 3. 备份策略

```bash
# 创建定时备份脚本
cat > /etc/cron.daily/fusionmail-backup << 'EOF'
#!/bin/bash
docker run --rm -v fusionmail_data:/data -v /backups:/backup alpine \
  tar czf /backup/fusionmail-$(date +%Y%m%d).tar.gz /data
# 保留最近 7 天的备份
find /backups -name "fusionmail-*.tar.gz" -mtime +7 -delete
EOF
chmod +x /etc/cron.daily/fusionmail-backup
```

### 4. 日志管理

```bash
# 配置日志轮转
cat > /etc/logrotate.d/fusionmail << 'EOF'
/var/lib/docker/containers/*/*.log {
    rotate 7
    daily
    compress
    missingok
    delaycompress
    copytruncate
}
EOF
```

---

## 下一步

1. 访问 `http://your-server:3333` 或你的域名
2. 使用 `admin` 和设置的密码登录
3. 添加邮箱账户开始使用

如有问题，查看日志或提交 Issue。
