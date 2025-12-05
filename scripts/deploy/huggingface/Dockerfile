# ============================================
# FusionMail HuggingFace Spaces Dockerfile
# ============================================
# 适配 HuggingFace Spaces 的特殊要求：
# - 端口必须是 7860
# - 必须使用非 root 用户
# - 使用外部数据库（PostgreSQL + Redis）
# ============================================

# 阶段 1: 构建前端
FROM node:20-alpine AS frontend-builder

WORKDIR /frontend
COPY frontend/package*.json ./
RUN npm ci
COPY frontend/ ./

# 构建时设置 API 路径为相对路径（同域部署）
ENV VITE_API_BASE_URL=/api/v1
RUN npm run build

# 阶段 2: 构建后端
FROM golang:1.24-alpine AS backend-builder

WORKDIR /backend
RUN apk add --no-cache git
ENV GOPROXY=https://goproxy.cn,direct

COPY backend/go.mod backend/go.sum ./
RUN go mod download
COPY backend/ ./

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -ldflags="-w -s" \
    -o server \
    cmd/server/main.go

# 阶段 3: 最终运行镜像
FROM alpine:latest

# 安装运行时依赖和 nginx
RUN apk --no-cache add \
    ca-certificates \
    tzdata \
    nginx \
    && cp /usr/share/zoneinfo/Asia/Shanghai /etc/localtime \
    && echo "Asia/Shanghai" > /etc/timezone

# 创建用户（HuggingFace 要求非 root）
RUN addgroup -g 1000 fusionmail && \
    adduser -D -u 1000 -G fusionmail fusionmail

WORKDIR /app

# 复制构建产物
COPY --from=backend-builder /backend/server .
COPY --from=frontend-builder /frontend/dist ./static
COPY --from=backend-builder /backend/config ./config

# 复制 nginx 配置（文件在 Space 根目录）
COPY nginx.conf /etc/nginx/nginx.conf

# 复制启动脚本（文件在 Space 根目录）
COPY start.sh ./start.sh
RUN chmod +x ./start.sh

# 创建必要目录并设置权限
RUN mkdir -p /data/attachments /var/log/nginx /var/lib/nginx/tmp /run/nginx && \
    chown -R fusionmail:fusionmail /app /data /var/log/nginx /var/lib/nginx /run/nginx /etc/nginx

# HuggingFace 要求的端口
EXPOSE 7860

# 切换到非 root 用户
USER fusionmail

# 健康检查
HEALTHCHECK --interval=30s --timeout=3s --start-period=10s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:7860/api/v1/health || exit 1

# 启动
CMD ["./start.sh"]
