# ============================================
# FusionMail Fly.io Dockerfile
# ============================================

# 阶段 1: 构建前端
FROM node:20-alpine AS frontend-builder

WORKDIR /frontend
COPY frontend/package*.json ./
RUN npm ci
COPY frontend/ ./

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

# 编译迁移工具
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -ldflags="-w -s" \
    -o migrate \
    cmd/migrate/main.go

# 阶段 3: 最终运行镜像
FROM alpine:latest

RUN apk --no-cache add \
    ca-certificates \
    tzdata \
    nginx \
    && cp /usr/share/zoneinfo/Asia/Shanghai /etc/localtime \
    && echo "Asia/Shanghai" > /etc/timezone

RUN addgroup -g 1000 fusionmail && \
    adduser -D -u 1000 -G fusionmail fusionmail

WORKDIR /app

COPY --from=backend-builder /backend/server .
COPY --from=backend-builder /backend/migrate .
COPY --from=frontend-builder /frontend/dist ./static
COPY --from=backend-builder /backend/config ./config

COPY scripts/deploy/fly/nginx.conf /etc/nginx/nginx.conf
COPY scripts/deploy/fly/start.sh ./start.sh

RUN chmod +x ./start.sh && \
    mkdir -p /data/attachments /var/log/nginx /var/lib/nginx/tmp /run/nginx && \
    chown -R fusionmail:fusionmail /app /data /var/log/nginx /var/lib/nginx /run/nginx /etc/nginx

# Fly.io 使用 8080 端口
EXPOSE 8080

USER fusionmail

CMD ["./start.sh"]
