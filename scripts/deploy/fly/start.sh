#!/bin/sh
# FusionMail Fly.io 启动脚本

set -e

echo "=========================================="
echo "  FusionMail Starting on Fly.io"
echo "=========================================="

# 检查环境变量
for var in DB_HOST DB_PASSWORD JWT_SECRET ENCRYPTION_KEY; do
    val=$(eval echo \$$var)
    if [ -z "$val" ]; then
        echo "ERROR: $var 未设置"
        exit 1
    fi
    echo "✓ $var"
done

# 设置默认值
export DB_PORT=${DB_PORT:-5432}
export DB_USER=${DB_USER:-postgres}
export DB_NAME=${DB_NAME:-postgres}
export DB_SSLMODE=${DB_SSLMODE:-require}
export REDIS_TLS=${REDIS_TLS:-false}

echo ""
echo "DB: $DB_HOST:$DB_PORT/$DB_NAME"
[ -n "$REDIS_HOST" ] && echo "Redis: $REDIS_HOST:$REDIS_PORT"

echo ""
echo "启动后端..."
./server &
BACKEND_PID=$!

echo "等待后端就绪..."
sleep 5

for i in $(seq 1 180); do
    if wget -q --spider http://127.0.0.1:3333/api/v1/health 2>/dev/null; then
        echo "✓ 后端就绪 (${i}s)"
        break
    fi
    [ $i -eq 180 ] && echo "ERROR: 超时" && exit 1
    sleep 1
done

echo "启动 Nginx..."
nginx -g "daemon off;" &
NGINX_PID=$!

echo ""
echo "=========================================="
echo "  FusionMail 启动完成!"
echo "=========================================="

wait -n $BACKEND_PID $NGINX_PID
kill $BACKEND_PID $NGINX_PID 2>/dev/null || true
exit 1
