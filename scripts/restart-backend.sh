#!/bin/bash

# 重启后端服务脚本

echo "🔄 重启 FusionMail 后端服务..."
echo ""

# 查找并停止旧的后端进程
echo "1️⃣ 停止旧的后端进程..."
OLD_PID=$(ps aux | grep "cmd/server/main.go" | grep -v grep | awk '{print $2}')

if [ -n "$OLD_PID" ]; then
    echo "   找到进程 PID: $OLD_PID"
    kill $OLD_PID
    sleep 2
    echo "   ✅ 旧进程已停止"
else
    echo "   ℹ️  没有找到运行中的后端进程"
fi

# 检查编译后的二进制文件
echo ""
echo "2️⃣ 检查编译后的二进制文件..."
if [ -f "backend/bin/server" ]; then
    echo "   ✅ 找到编译后的二进制文件"
else
    echo "   ⚠️  未找到编译后的二进制文件，正在编译..."
    cd backend && go build -o bin/server ./cmd/server
    cd ..
fi

# 启动新的后端服务
echo ""
echo "3️⃣ 启动新的后端服务..."
cd backend
nohup ./bin/server > ../logs/backend.log 2>&1 &
NEW_PID=$!
cd ..

sleep 2

# 验证服务是否启动成功
echo ""
echo "4️⃣ 验证服务状态..."
if curl -s http://localhost:8080/api/v1/health > /dev/null; then
    echo "   ✅ 后端服务启动成功！"
    echo "   PID: $NEW_PID"
    echo "   URL: http://localhost:8080"
    echo ""
    echo "📋 查看日志: tail -f logs/backend.log"
else
    echo "   ❌ 后端服务启动失败"
    echo "   请查看日志: cat logs/backend.log"
    exit 1
fi

echo ""
echo "=========================================="
echo "✅ 后端服务重启完成！"
echo "=========================================="
