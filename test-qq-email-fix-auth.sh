#!/bin/bash

# 测试QQ邮箱同步修复的脚本（带认证）
# 使用方法: ./test-qq-email-fix-auth.sh

set -e

API_BASE="http://localhost:8080/api/v1"
PASSWORD="admin123"  # 默认密码

echo "🧪 测试QQ邮箱同步修复（带认证）"
echo "=========================="

# 1. 检查服务状态
echo "1️⃣ 检查服务状态..."
if ! curl -s "$API_BASE/health" > /dev/null; then
    echo "❌ 服务未运行，请先启动后端服务"
    exit 1
fi
echo "✅ 服务正常运行"

# 2. 登录获取token
echo ""
echo "2️⃣ 登录获取认证token..."
LOGIN_RESPONSE=$(curl -s -X POST "$API_BASE/auth/login" \
    -H "Content-Type: application/json" \
    -d "{\"password\": \"$PASSWORD\"}")

echo "登录响应: $LOGIN_RESPONSE"

LOGIN_SUCCESS=$(echo "$LOGIN_RESPONSE" | jq -r '.success // false')
if [ "$LOGIN_SUCCESS" != "true" ]; then
    echo "❌ 登录失败"
    LOGIN_ERROR=$(echo "$LOGIN_RESPONSE" | jq -r '.error // "unknown"')
    echo "错误: $LOGIN_ERROR"
    exit 1
fi

TOKEN=$(echo "$LOGIN_RESPONSE" | jq -r '.data.token')
echo "✅ 登录成功，获得token: ${TOKEN:0:20}..."

# 3. 查找QQ邮箱账户
echo ""
echo "3️⃣ 查找QQ邮箱账户..."
ACCOUNTS=$(curl -s "$API_BASE/accounts" \
    -H "Authorization: Bearer $TOKEN")

echo "账户列表响应: $ACCOUNTS"

# 检查响应是否成功
ACCOUNTS_SUCCESS=$(echo "$ACCOUNTS" | jq -r '.success // false')
if [ "$ACCOUNTS_SUCCESS" != "true" ]; then
    echo "❌ 获取账户列表失败"
    ACCOUNTS_ERROR=$(echo "$ACCOUNTS" | jq -r '.error // "unknown"')
    echo "错误: $ACCOUNTS_ERROR"
    exit 1
fi

QQ_UID=$(echo "$ACCOUNTS" | jq -r '.data[] | select(.provider == "qq") | .uid // empty')

if [ -z "$QQ_UID" ]; then
    echo "❌ 没有找到QQ邮箱账户"
    echo "现有账户:"
    echo "$ACCOUNTS" | jq -r '.data[] | "- \(.email) (\(.provider))"'
    exit 1
fi

echo "✅ 找到QQ邮箱账户: $QQ_UID"

# 4. 获取账户详情
echo ""
echo "4️⃣ 获取账户详情..."
ACCOUNT_DETAIL=$(curl -s "$API_BASE/accounts/$QQ_UID" \
    -H "Authorization: Bearer $TOKEN")

echo "账户详情: $ACCOUNT_DETAIL"

STATUS=$(echo "$ACCOUNT_DETAIL" | jq -r '.data.status')
SYNC_ENABLED=$(echo "$ACCOUNT_DETAIL" | jq -r '.data.sync_enabled')
LAST_SYNC_AT=$(echo "$ACCOUNT_DETAIL" | jq -r '.data.last_sync_at')
EMAIL=$(echo "$ACCOUNT_DETAIL" | jq -r '.data.email')

echo "- 邮箱: $EMAIL"
echo "- 状态: $STATUS"
echo "- 同步启用: $SYNC_ENABLED"
echo "- 最后同步: $LAST_SYNC_AT"

# 5. 确保账户状态正确
if [ "$STATUS" != "active" ]; then
    echo ""
    echo "5️⃣ 启用QQ账户..."
    ENABLE_RESULT=$(curl -s -X POST "$API_BASE/accounts/$QQ_UID/enable" \
        -H "Authorization: Bearer $TOKEN")
    echo "启用结果: $ENABLE_RESULT"
    
    # 等待状态更新
    sleep 2
    UPDATED_ACCOUNT=$(curl -s "$API_BASE/accounts/$QQ_UID" \
        -H "Authorization: Bearer $TOKEN")
    NEW_STATUS=$(echo "$UPDATED_ACCOUNT" | jq -r '.data.status')
    
    if [ "$NEW_STATUS" = "active" ]; then
        echo "✅ 账户已启用"
        STATUS="active"
    else
        echo "❌ 账户启用失败: $NEW_STATUS"
        exit 1
    fi
else
    echo "✅ 账户状态正常"
fi

# 6. 测试连接
echo ""
echo "6️⃣ 测试连接..."
CONNECTION_TEST=$(curl -s -X POST "$API_BASE/accounts/$QQ_UID/test" \
    -H "Authorization: Bearer $TOKEN")
echo "连接测试: $CONNECTION_TEST"

CONNECTION_SUCCESS=$(echo "$CONNECTION_TEST" | jq -r '.success // false')
if [ "$CONNECTION_SUCCESS" != "true" ]; then
    echo "❌ 连接测试失败"
    CONNECTION_ERROR=$(echo "$CONNECTION_TEST" | jq -r '.error // "unknown"')
    echo "错误: $CONNECTION_ERROR"
    
    echo ""
    echo "💡 连接失败可能的原因:"
    echo "1. QQ邮箱未开启IMAP服务"
    echo "2. 使用的不是授权码"
    echo "3. 网络连接问题"
    echo "4. 邮箱服务器限制"
    
    # 继续测试，但标记连接问题
    CONNECTION_ISSUE=true
else
    echo "✅ 连接测试成功"
    CONNECTION_ISSUE=false
fi

# 7. 手动触发同步
echo ""
echo "7️⃣ 触发手动同步..."
SYNC_TRIGGER=$(curl -s -X POST "$API_BASE/sync/accounts/$QQ_UID" \
    -H "Authorization: Bearer $TOKEN")
echo "同步触发: $SYNC_TRIGGER"

SYNC_SUCCESS=$(echo "$SYNC_TRIGGER" | jq -r '.success // false')
if [ "$SYNC_SUCCESS" != "true" ]; then
    echo "❌ 同步触发失败"
    SYNC_ERROR=$(echo "$SYNC_TRIGGER" | jq -r '.error // "unknown"')
    echo "错误: $SYNC_ERROR"
    
    if [ "$CONNECTION_ISSUE" = "true" ]; then
        echo "这可能是由于连接问题导致的"
    fi
else
    echo "✅ 同步已触发"
fi

# 8. 监控同步进度
echo ""
echo "8️⃣ 监控同步进度..."
for i in {1..12}; do
    echo "检查进度... ($i/12)"
    sleep 5
    
    ACCOUNT_STATUS=$(curl -s "$API_BASE/accounts/$QQ_UID" \
        -H "Authorization: Bearer $TOKEN")
    SYNC_STATUS=$(echo "$ACCOUNT_STATUS" | jq -r '.data.last_sync_status // "unknown"')
    SYNC_ERROR=$(echo "$ACCOUNT_STATUS" | jq -r '.data.last_sync_error // ""')
    
    echo "同步状态: $SYNC_STATUS"
    
    if [ "$SYNC_STATUS" = "success" ]; then
        echo "✅ 同步成功完成"
        break
    elif [ "$SYNC_STATUS" = "failed" ]; then
        echo "❌ 同步失败: $SYNC_ERROR"
        break
    fi
    
    if [ "$i" = "12" ]; then
        echo "⏰ 同步超时，可能仍在进行中"
    fi
done

# 9. 检查邮件数量
echo ""
echo "9️⃣ 检查邮件数量..."
EMAILS_RESPONSE=$(curl -s "$API_BASE/emails?account_uid=$QQ_UID&page_size=5" \
    -H "Authorization: Bearer $TOKEN")

echo "邮件查询响应: $EMAILS_RESPONSE"

EMAIL_COUNT=$(echo "$EMAILS_RESPONSE" | jq -r '.data.total // 0')
echo "QQ邮箱邮件总数: $EMAIL_COUNT"

if [ "$EMAIL_COUNT" -gt 0 ]; then
    echo "✅ 成功同步 $EMAIL_COUNT 封邮件"
    
    echo ""
    echo "📧 最新邮件列表:"
    echo "$EMAILS_RESPONSE" | jq -r '.data.emails[]? | "- \(.subject) (from: \(.from_address), sent: \(.sent_at))"'
else
    echo "❌ 没有同步到邮件"
fi

# 10. 检查同步日志
echo ""
echo "🔟 检查同步日志..."
SYNC_LOGS=$(curl -s "$API_BASE/sync/logs?limit=3" \
    -H "Authorization: Bearer $TOKEN")
echo "最新同步日志:"
echo "$SYNC_LOGS" | jq -r '.data.emails[]? | "- \(.account_uid): \(.status) (\(.emails_new) new, \(.emails_updated) updated, error: \(.error_message // "none"))"'

# 11. 总结
echo ""
echo "🎯 测试总结"
echo "==========="

if [ "$EMAIL_COUNT" -gt 0 ]; then
    echo "✅ QQ邮箱同步修复成功！"
    echo "- 账户状态: $STATUS"
    echo "- 同步状态: $SYNC_STATUS"
    echo "- 邮件数量: $EMAIL_COUNT"
    echo ""
    echo "💡 现在你应该能够接收到新邮件了"
elif [ "$CONNECTION_ISSUE" = "true" ]; then
    echo "❌ QQ邮箱连接有问题"
    echo "- 账户状态: $STATUS"
    echo "- 连接测试: 失败"
    echo ""
    echo "💡 请检查:"
    echo "1. QQ邮箱IMAP设置是否正确"
    echo "2. 是否使用授权码而不是登录密码"
    echo "3. 网络连接是否正常"
elif [ "$SYNC_STATUS" = "failed" ]; then
    echo "❌ QQ邮箱同步失败"
    echo "- 账户状态: $STATUS"
    echo "- 同步状态: $SYNC_STATUS"
    echo "- 同步错误: $SYNC_ERROR"
    echo ""
    echo "💡 建议查看后端日志获取详细错误信息"
else
    echo "⚠️  QQ邮箱同步状态不明确"
    echo "- 账户状态: $STATUS"
    echo "- 同步状态: $SYNC_STATUS"
    echo "- 邮件数量: $EMAIL_COUNT"
    echo ""
    echo "💡 可能需要更多时间或检查配置"
fi

echo ""
echo "🔚 测试完成"