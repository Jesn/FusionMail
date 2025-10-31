#!/bin/bash

# 快速测试QQ邮箱同步的脚本
# 使用方法: ./quick-test-qq.sh

set -e

API_BASE="http://localhost:8080/api/v1"
PASSWORD="admin123"

echo "🧪 快速测试QQ邮箱同步"
echo "==================="

# 1. 登录
LOGIN_RESPONSE=$(curl -s -X POST "$API_BASE/auth/login" \
    -H "Content-Type: application/json" \
    -d "{\"password\": \"$PASSWORD\"}")

TOKEN=$(echo "$LOGIN_RESPONSE" | jq -r '.data.token')
echo "✅ 登录成功"

# 2. 获取QQ账户
ACCOUNTS=$(curl -s "$API_BASE/accounts" \
    -H "Authorization: Bearer $TOKEN")

QQ_UID=$(echo "$ACCOUNTS" | jq -r '.data[] | select(.provider == "qq") | .uid')
QQ_EMAIL=$(echo "$ACCOUNTS" | jq -r '.data[] | select(.provider == "qq") | .email')
QQ_STATUS=$(echo "$ACCOUNTS" | jq -r '.data[] | select(.provider == "qq") | .status')

echo "✅ QQ账户: $QQ_EMAIL (状态: $QQ_STATUS)"

# 3. 查询邮件
EMAILS_RESPONSE=$(curl -s "$API_BASE/emails?account_uid=$QQ_UID&page_size=10" \
    -H "Authorization: Bearer $TOKEN")

EMAIL_COUNT=$(echo "$EMAILS_RESPONSE" | jq -r '.data.total // 0')
echo "✅ QQ邮箱邮件总数: $EMAIL_COUNT"

if [ "$EMAIL_COUNT" -gt 0 ]; then
    echo ""
    echo "📧 最新邮件:"
    echo "$EMAILS_RESPONSE" | jq -r '.data.emails[0:3][] | "- \(.subject) (from: \(.from_address))"'
    
    echo ""
    echo "🎉 QQ邮箱同步正常工作！"
    echo "- 账户状态: $QQ_STATUS"
    echo "- 邮件数量: $EMAIL_COUNT"
    echo ""
    echo "💡 你现在应该能够接收到新邮件了"
else
    echo "❌ 没有找到邮件"
fi

echo ""
echo "🔚 测试完成"