# ============================================
# FusionMail Render Dockerfile
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
COPY --from=frontend-builder /frontend/dist ./static
COPY --from=backend-builder /backend/config ./config

COPY scripts/deploy/render/nginx.conf /etc/nginx/nginx.conf
COPY scripts/deploy/render/start.sh ./start.sh

RUN chmod +x ./start.sh && \
    mkdir -p /data/attachments /var/log/nginx /var/lib/nginx/tmp /run/nginx && \
    chown -R fusionmail:fusionmail /app /data /var/log/nginx /var/lib/nginx /run/nginx /etc/nginx

# Render 使用 PORT 环境变量，默认 10000
EXPOSE 10000

USER fusionmail

HEALTHCHECK --interval=30s --timeout=3s --start-period=30s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:10000/api/v1/health || exit 1

CMD ["./start.sh"]
