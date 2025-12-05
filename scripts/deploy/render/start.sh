#!/bin/sh
# FusionMail Render 启动脚本

set -e

echo "=========================================="
echo "  FusionMail Starting on Render"
echo "=========================================="

# 检查环境变量
check_env() {
    local var_name=$1
    local var_value=$(eval echo \$$var_name)
    if [ -z "$var_value" ]; then
        echo "ERROR: $var_name 未设置"
        return 1
    fi
    echo "✓ $var_name"
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
export DB_NAME=${DB_NAME:-postgres}
export DB_SSLMODE=${DB_SSLMODE:-require}

export REDIS_HOST=${REDIS_HOST:-}
export REDIS_PORT=${REDIS_PORT:-6379}
export REDIS_PASSWORD=${REDIS_PASSWORD:-}
export REDIS_TLS=${REDIS_TLS:-false}

export SERVER_HOST=0.0.0.0
export SERVER_PORT=3333

export STORAGE_TYPE=local
export STORAGE_LOCAL_PATH=/data/attachments

export CORS_ALLOWED_ORIGINS=${CORS_ALLOWED_ORIGINS:-*}
export COOKIE_SECURE=${COOKIE_SECURE:-true}

export RATE_LIMIT_ENABLED=${RATE_LIMIT_ENABLED:-true}
export SWAGGER_ENABLED=${SWAGGER_ENABLED:-false}

echo ""
echo "数据库: $DB_HOST:$DB_PORT/$DB_NAME"
[ -n "$REDIS_HOST" ] && echo "Redis: $REDIS_HOST:$REDIS_PORT (TLS=$REDIS_TLS)"

echo ""
echo "启动后端..."
./server &
BACKEND_PID=$!

echo "等待后端就绪（最长 180 秒）..."
sleep 5

for i in $(seq 1 180); do
    if wget -q --spider http://127.0.0.1:3333/api/v1/health 2>/dev/null; then
        echo "✓ 后端就绪 (${i}s)"
        break
    fi
    if [ $i -eq 180 ]; then
        echo "ERROR: 后端启动超时"
        exit 1
    fi
    sleep 1
done

echo ""
echo "启动 Nginx (端口 10000)..."
nginx -g "daemon off;" &
NGINX_PID=$!

echo ""
echo "=========================================="
echo "  FusionMail 启动完成!"
echo "=========================================="

wait -n $BACKEND_PID $NGINX_PID
echo "服务退出，清理中..."
kill $BACKEND_PID $NGINX_PID 2>/dev/null || true
exit 1
