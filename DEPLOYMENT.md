# FusionMail 部署指南

## 快速部署

### 使用 Docker Compose（推荐）

1. **克隆项目**

```bash
git clone <repository-url>
cd fusionmail
```

2. **配置环境变量**

```bash
cp .env.example .env
# 编辑 .env 文件，修改密码和密钥
nano .env
```

3. **启动服务**

```bash
docker-compose up -d
```

4. **访问应用**

打开浏览器访问：`http://localhost:8080`

### 停止服务

```bash
docker-compose down
```

### 查看日志

```bash
# 查看所有服务日志
docker-compose logs -f

# 查看应用日志
docker-compose logs -f app

# 查看数据库日志
docker-compose logs -f postgres
```

## 手动构建

### 构建 Docker 镜像

```bash
docker build -t fusionmail:latest .
```

### 运行容器

```bash
docker run -d \
  --name fusionmail \
  -p 8080:8080 \
  -e DB_HOST=your-db-host \
  -e DB_PASSWORD=your-db-password \
  -e REDIS_HOST=your-redis-host \
  -e REDIS_PASSWORD=your-redis-password \
  fusionmail:latest
```

## 生产环境部署

### 1. 准备工作

- 服务器：Linux 服务器（推荐 Ubuntu 20.04+）
- Docker：安装 Docker 和 Docker Compose
- 域名：配置域名解析到服务器 IP
- SSL 证书：使用 Let's Encrypt 或其他 SSL 证书

### 2. 安全配置

**修改默认密码**：

```bash
# 生成强密码
openssl rand -base64 32

# 修改 .env 文件中的密码
DB_PASSWORD=<生成的强密码>
REDIS_PASSWORD=<生成的强密码>
JWT_SECRET=<生成的强密码>
ENCRYPTION_KEY=<生成的 32 字节密钥>
```

**配置防火墙**：

```bash
# 只开放必要的端口
ufw allow 22/tcp    # SSH
ufw allow 80/tcp    # HTTP
ufw allow 443/tcp   # HTTPS
ufw enable
```

### 3. 使用 Nginx 反向代理（可选）

创建 Nginx 配置文件 `/etc/nginx/sites-available/fusionmail`：

```nginx
server {
    listen 80;
    server_name your-domain.com;

    # 重定向到 HTTPS
    return 301 https://$server_name$request_uri;
}

server {
    listen 443 ssl http2;
    server_name your-domain.com;

    # SSL 证书配置
    ssl_certificate /etc/letsencrypt/live/your-domain.com/fullchain.pem;
    ssl_certificate_key /etc/letsencrypt/live/your-domain.com/privkey.pem;

    # SSL 安全配置
    ssl_protocols TLSv1.2 TLSv1.3;
    ssl_ciphers HIGH:!aNULL:!MD5;
    ssl_prefer_server_ciphers on;

    # 代理到 FusionMail
    location / {
        proxy_pass http://localhost:8080;
        proxy_http_version 1.1;
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection 'upgrade';
        proxy_set_header Host $host;
        proxy_cache_bypass $http_upgrade;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
    }

    # WebSocket 支持
    location /ws {
        proxy_pass http://localhost:8080;
        proxy_http_version 1.1;
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection "Upgrade";
        proxy_set_header Host $host;
    }

    # 客户端最大上传大小
    client_max_body_size 50M;
}
```

启用配置：

```bash
ln -s /etc/nginx/sites-available/fusionmail /etc/nginx/sites-enabled/
nginx -t
systemctl reload nginx
```

### 4. 配置 SSL 证书

使用 Certbot 获取免费 SSL 证书：

```bash
# 安装 Certbot
apt-get update
apt-get install certbot python3-certbot-nginx

# 获取证书
certbot --nginx -d your-domain.com

# 自动续期
certbot renew --dry-run
```

### 5. 启动服务

```bash
cd /opt/fusionmail
docker-compose up -d
```

### 6. 配置自动启动

创建 systemd 服务文件 `/etc/systemd/system/fusionmail.service`：

```ini
[Unit]
Description=FusionMail Service
Requires=docker.service
After=docker.service

[Service]
Type=oneshot
RemainAfterExit=yes
WorkingDirectory=/opt/fusionmail
ExecStart=/usr/local/bin/docker-compose up -d
ExecStop=/usr/local/bin/docker-compose down
TimeoutStartSec=0

[Install]
WantedBy=multi-user.target
```

启用服务：

```bash
systemctl daemon-reload
systemctl enable fusionmail
systemctl start fusionmail
```

## 数据备份

### 备份数据库

```bash
# 备份 PostgreSQL
docker exec fusionmail-postgres pg_dump -U fusionmail fusionmail > backup_$(date +%Y%m%d).sql

# 恢复数据库
docker exec -i fusionmail-postgres psql -U fusionmail fusionmail < backup_20240101.sql
```

### 备份附件

```bash
# 备份附件数据卷
docker run --rm \
  -v fusionmail_attachments_data:/data \
  -v $(pwd):/backup \
  alpine tar czf /backup/attachments_$(date +%Y%m%d).tar.gz /data
```

### 自动备份脚本

创建 `/opt/fusionmail/backup.sh`：

```bash
#!/bin/bash
BACKUP_DIR="/opt/fusionmail/backups"
DATE=$(date +%Y%m%d_%H%M%S)

mkdir -p $BACKUP_DIR

# 备份数据库
docker exec fusionmail-postgres pg_dump -U fusionmail fusionmail > $BACKUP_DIR/db_$DATE.sql

# 备份附件
docker run --rm \
  -v fusionmail_attachments_data:/data \
  -v $BACKUP_DIR:/backup \
  alpine tar czf /backup/attachments_$DATE.tar.gz /data

# 删除 7 天前的备份
find $BACKUP_DIR -name "*.sql" -mtime +7 -delete
find $BACKUP_DIR -name "*.tar.gz" -mtime +7 -delete

echo "Backup completed: $DATE"
```

添加到 crontab：

```bash
# 每天凌晨 2 点备份
0 2 * * * /opt/fusionmail/backup.sh >> /var/log/fusionmail-backup.log 2>&1
```

## 更新升级

### 更新到新版本

```bash
# 拉取最新代码
git pull

# 重新构建镜像
docker-compose build

# 重启服务
docker-compose down
docker-compose up -d
```

### 回滚到旧版本

```bash
# 查看镜像历史
docker images fusionmail

# 使用旧版本镜像
docker-compose down
docker tag fusionmail:old fusionmail:latest
docker-compose up -d
```

## 监控和维护

### 查看资源使用

```bash
# 查看容器资源使用
docker stats

# 查看磁盘使用
docker system df
```

### 清理无用数据

```bash
# 清理无用的镜像和容器
docker system prune -a

# 清理无用的数据卷
docker volume prune
```

### 查看应用日志

```bash
# 实时查看日志
docker-compose logs -f app

# 查看最近 100 行日志
docker-compose logs --tail=100 app
```

## 故障排查

### 应用无法启动

1. 检查日志：`docker-compose logs app`
2. 检查数据库连接：`docker-compose logs postgres`
3. 检查环境变量配置

### 数据库连接失败

1. 检查数据库是否运行：`docker-compose ps postgres`
2. 检查数据库密码是否正确
3. 检查网络连接：`docker network inspect fusionmail_fusionmail-network`

### 前端无法访问

1. 检查应用是否运行：`docker-compose ps app`
2. 检查端口映射：`docker-compose port app 8080`
3. 检查防火墙规则

### 性能问题

1. 检查资源使用：`docker stats`
2. 增加数据库连接池大小
3. 优化数据库索引
4. 增加 Redis 缓存

## 安全建议

1. **定期更新**：及时更新 Docker 镜像和系统补丁
2. **强密码**：使用强密码和密钥
3. **最小权限**：容器以非 root 用户运行
4. **网络隔离**：使用 Docker 网络隔离服务
5. **日志审计**：定期检查应用日志
6. **备份策略**：定期备份数据库和附件
7. **SSL/TLS**：使用 HTTPS 加密传输
8. **防火墙**：只开放必要的端口

## 技术支持

- 项目主页：[GitHub Repository]
- 问题反馈：[GitHub Issues]
- 文档：[Documentation]
