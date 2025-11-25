#!/bin/bash

# FusionMail 管理员密码哈希生成器
# 使用方法: ./generate_admin_hash.sh [密码]

echo "=== FusionMail 管理员密码哈希生成器 ==="
echo ""

# 检查是否安装了Go
if ! command -v go > /dev/null 2>&1; then
    echo "错误: 未安装 Go，请先安装 Go"
    exit 1
fi

# 获取密码参数或提示输入
if [ $# -eq 1 ]; then
    password="$1"
    echo "使用提供的密码生成哈希..."
else
    read -sp "请输入管理员密码: " password
    echo ""
fi

# 验证密码不为空
if [ -z "$password" ]; then
    echo "错误: 密码不能为空"
    exit 1
fi

# 创建临时Go文件来生成哈希
cat > /tmp/generate_hash.go << 'EOF'
package main

import (
    "fmt"
    "os"
    "golang.org/x/crypto/bcrypt"
)

func main() {
    if len(os.Args) != 2 {
        fmt.Println("使用方法: go run generate_hash.go <密码>")
        os.Exit(1)
    }

    password := os.Args[1]
    hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
    if err != nil {
        fmt.Printf("生成哈希失败: %v\n", err)
        os.Exit(1)
    }

    fmt.Printf("%s", string(hash))
}
EOF

echo "正在生成哈希..."
cd /tmp

# 初始化Go模块
go mod init temp_hash_generator > /dev/null 2>&1
go get golang.org/x/crypto/bcrypt > /dev/null 2>&1

# 生成哈希
hash=$(go run generate_hash.go "$password")

# 检查是否成功生成哈希
if [ $? -ne 0 ] || [ -z "$hash" ]; then
    echo "错误: 生成哈希失败"
    exit 1
fi

# 清理临时文件
rm -f /tmp/generate_hash.go /tmp/go.mod /tmp/go.sum

# 显示结果
echo ""
echo "=== 生成的哈希值 ==="
echo "ADMIN_PASSWORD_HASH=$hash"
echo ""
echo "=== 重要说明 ==="
echo "注意: 每次生成的哈希都不同，这是bcrypt的正常安全特性！"
echo "即使是相同的密码，每次使用bcrypt生成的哈希都会不同，"
echo "但都可以用于验证同一个密码。"
echo ""
echo "=== 使用方法 ==="
echo "1. 复制上面的 ADMIN_PASSWORD_HASH 值"
echo "2. 将其添加到您的 .env 文件中"
echo "3. 或者设置为环境变量:"
echo "   export ADMIN_PASSWORD_HASH=\"$hash\""
echo ""
echo "=== 测试验证 ==="
echo "您可以使用以下命令测试生成的哈希:"
echo "ADMIN_PASSWORD_HASH=\"$hash\" go run test_bcrypt.go"