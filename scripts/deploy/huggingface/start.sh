#!/bin/sh
# FusionMail HuggingFace 启动脚本

set -e

echo "=========================================="
echo "  FusionMail Starting on HuggingFace"
echo "=========================================="

# 检查必要的环境变量
check_env() {
    local var_name=$1
    local var_value=$(eval echo \$$var_name)
    if [ -z "$var_value" ]; then
        echo "ERROR: 环境变量 $var_name 未设置"
        return 1
    fi
    echo "✓ $var_name 已配置"
}

echo ""
echo "检查环境变量..."
check_env "DB_HOST" || exit 1
check_env "DB_PASSWORD" || exit 1
check_env "JWT_SECRET" || exit 1
check_env "ENCRYPTION_KEY" || exit 1

# 设置默认值
export DB_PORT=${DB_PORT:-5432}
export DB_USER=${DB_USER:-postgres}
export DB_NAME=${DB_NAME:-fusionmail}
export DB_SSLMODE=${DB_SSLMODE:-require}

export REDIS_HOST=${REDIS_HOST:-}
export REDIS_PORT=${REDIS_PORT:-6379}
export REDIS_PASSWORD=${REDIS_PASSWORD:-}

export SERVER_HOST=0.0.0.0
export SERVER_PORT=3333

export STORAGE_TYPE=local
export STORAGE_LOCAL_PATH=/data/attachments

export CORS_ALLOWED_ORIGINS=${CORS_ALLOWED_ORIGINS:-*}
export COOKIE_SECURE=${COOKIE_SECURE:-true}

export RATE_LIMIT_ENABLED=${RATE_LIMIT_ENABLED:-true}
export SWAGGER_ENABLED=${SWAGGER_ENABLED:-false}

echo ""
echo "数据库配置:"
echo "  Host: $DB_HOST"
echo "  Port: $DB_PORT"
echo "  Database: $DB_NAME"
echo "  SSL Mode: $DB_SSLMODE"

if [ -n "$REDIS_HOST" ]; then
    echo ""
    echo "Redis 配置:"
    echo "  Host: $REDIS_HOST"
    echo "  Port: $REDIS_PORT"
fi

echo ""
echo "启动后端服务..."
./server &
BACKEND_PID=$!

# 等待后端启动（首次启动需要初始化数据库，可能需要较长时间）
echo "等待后端服务就绪（最长等待 120 秒）..."
sleep 5

for i in $(seq 1 120); do
    if wget -q --spider http://127.0.0.1:3333/api/v1/health 2>/dev/null; then
        echo "✓ 后端服务已就绪 (耗时 ${i} 秒)"
        break
    fi
    if [ $i -eq 120 ]; then
        echo "ERROR: 后端服务启动超时"
        exit 1
    fi
    sleep 1
done

echo ""
echo "启动 Nginx..."
nginx -g "daemon off;" &
NGINX_PID=$!

echo ""
echo "=========================================="
echo "  FusionMail 启动完成!"
echo "  访问: http://localhost:7860"
echo "=========================================="

# 等待任一进程退出
wait -n $BACKEND_PID $NGINX_PID

# 如果任一进程退出，终止另一个
echo "服务异常退出，正在清理..."
kill $BACKEND_PID $NGINX_PID 2>/dev/null || true
exit 1
