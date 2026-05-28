# ============================================
# 阶段 1: 构建前端
# ============================================
FROM node:20-alpine AS frontend-builder

WORKDIR /frontend

# 复制前端依赖文件
COPY frontend/package*.json ./

# 安装依赖（包括 devDependencies，因为构建需要）
RUN npm ci

# 复制前端源码
COPY frontend/ ./

# 构建前端生产版本
RUN npm run build

# ============================================
# 阶段 2: 构建后端
# ============================================
FROM golang:1.24-alpine AS backend-builder

WORKDIR /backend

# 安装构建依赖
RUN apk add --no-cache git

# 禁用固定国内 Go 代理，避免海外构建器解析 goproxy.cn 失败
# ENV GOPROXY=https://goproxy.cn,direct

# 复制 Go 模块文件
COPY backend/go.mod backend/go.sum ./

# 下载依赖
RUN go mod download

# 复制后端源码
COPY backend/ ./

# 编译 Go 二进制文件
# CGO_ENABLED=0: 静态编译，不依赖 C 库
# -ldflags="-w -s": 去除调试信息，减小体积
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -ldflags="-w -s" \
    -o server \
    cmd/server/main.go

# 编译迁移工具
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -ldflags="-w -s" \
    -o migrate \
    cmd/migrate/main.go

# ============================================
# 阶段 3: 最终运行镜像
# ============================================
FROM alpine:latest

# 安装运行时依赖
RUN apk --no-cache add \
    ca-certificates \
    tzdata \
    && cp /usr/share/zoneinfo/Asia/Shanghai /etc/localtime \
    && echo "Asia/Shanghai" > /etc/timezone

# 创建非 root 用户
RUN addgroup -g 1000 fusionmail && \
    adduser -D -u 1000 -G fusionmail fusionmail

WORKDIR /app

# 从构建阶段复制文件
COPY --from=backend-builder /backend/server .
COPY --from=backend-builder /backend/migrate .
COPY --from=frontend-builder /frontend/dist ./static
COPY --from=backend-builder /backend/config ./config

# 创建数据目录
RUN mkdir -p /data/attachments && \
    chown -R fusionmail:fusionmail /app /data

# 设置 Docker 环境默认值（覆盖代码中的 ./data/attachments）
ENV STORAGE_LOCAL_PATH=/data/attachments

# 切换到非 root 用户
USER fusionmail

# 暴露端口（与应用监听端口一致）
EXPOSE 3333

# 健康检查（使用应用实际端口）
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:3333/api/v1/health || exit 1

# 启动应用
CMD ["./server"]
