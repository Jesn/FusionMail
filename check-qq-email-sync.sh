#!/bin/bash

# 检查QQ邮箱同步问题的详细脚本
# 使用方法: ./check-qq-email-sync.sh

set -e

API_BASE="http://localhost:8080/api/v1"
PASSWORD="admin123"

echo "🔍 详细检查QQ邮箱同步问题"
echo "=========================="

# 0. 登录获取认证token
echo "0️⃣ 登录获取认证token..."
LOGIN_RESPONSE=$(curl -s -X POST "$API_BASE/auth/login" \
    -H "Content-Type: application/json" \
    -d "{\"password\": \"$PASSWORD\"}")

TOKEN=$(echo "$LOGIN_RESPONSE" | jq -r '.data.token')
if [ "$TOKEN" = "null" ] || [ -z "$TOKEN" ]; then
    echo "❌ 登录失败"
    echo "登录响应: $LOGIN_RESPONSE"
    exit 1
fi
echo "✅ 登录成功，获得token: ${TOKEN:0:20}..."

# 1. 检查服务状态
echo "1️⃣ 检查服务状态..."
HEALTH_RESPONSE=$(curl -s "$API_BASE/health")
echo "服务健康状态: $HEALTH_RESPONSE"

# 2. 获取所有账户
echo ""
echo "2️⃣ 获取所有账户..."
ACCOUNTS=$(curl -s "$API_BASE/accounts" \
    -H "Authorization: Bearer $TOKEN")
echo "所有账户: $ACCOUNTS"

# 3. 查找QQ邮箱账户
QQ_ACCOUNTS=$(echo "$ACCOUNTS" | jq -r '.data[] | select(.provider == "qq")')
if [ -z "$QQ_ACCOUNTS" ]; then
    echo "❌ 没有找到QQ邮箱账户"
    exit 1
fi

echo ""
echo "3️⃣ QQ邮箱账户详情:"
echo "$QQ_ACCOUNTS" | jq '.'

QQ_UID=$(echo "$QQ_ACCOUNTS" | jq -r '.uid')
QQ_EMAIL=$(echo "$QQ_ACCOUNTS" | jq -r '.email')
QQ_STATUS=$(echo "$QQ_ACCOUNTS" | jq -r '.status')
QQ_SYNC_ENABLED=$(echo "$QQ_ACCOUNTS" | jq -r '.sync_enabled')

echo ""
echo "📊 QQ账户关键信息:"
echo "- UID: $QQ_UID"
echo "- 邮箱: $QQ_EMAIL"
echo "- 状态: $QQ_STATUS"
echo "- 同步启用: $QQ_SYNC_ENABLED"

# 4. 检查账户状态是否正确
if [ "$QQ_STATUS" != "active" ]; then
    echo ""
    echo "❌ 问题发现: QQ账户状态不是 'active'"
    echo "当前状态: $QQ_STATUS"
    echo "需要启用账户才能同步邮件"
    
    echo ""
    echo "🔧 尝试启用QQ账户..."
    ENABLE_RESULT=$(curl -s -X POST "$API_BASE/accounts/$QQ_UID/enable" \
        -H "Authorization: Bearer $TOKEN")
    echo "启用结果: $ENABLE_RESULT"
    
    # 重新获取账户状态
    sleep 2
    UPDATED_ACCOUNT=$(curl -s "$API_BASE/accounts/$QQ_UID" \
        -H "Authorization: Bearer $TOKEN")
    NEW_STATUS=$(echo "$UPDATED_ACCOUNT" | jq -r '.data.status')
    echo "更新后状态: $NEW_STATUS"
    
    if [ "$NEW_STATUS" = "active" ]; then
        echo "✅ QQ账户已成功启用"
        QQ_STATUS="active"
    else
        echo "❌ QQ账户启用失败"
    fi
fi

# 5. 检查同步是否启用
if [ "$QQ_SYNC_ENABLED" != "true" ]; then
    echo ""
    echo "❌ 问题发现: QQ账户同步未启用"
    echo "需要启用同步功能"
fi

# 6. 测试连接
echo ""
echo "4️⃣ 测试QQ邮箱连接..."
CONNECTION_TEST=$(curl -s -X POST "$API_BASE/accounts/$QQ_UID/test" \
    -H "Authorization: Bearer $TOKEN")
echo "连接测试: $CONNECTION_TEST"

CONNECTION_SUCCESS=$(echo "$CONNECTION_TEST" | jq -r '.success // false')
if [ "$CONNECTION_SUCCESS" = "true" ]; then
    echo "✅ QQ邮箱连接测试成功"
else
    echo "❌ QQ邮箱连接测试失败"
    CONNECTION_ERROR=$(echo "$CONNECTION_TEST" | jq -r '.error // "unknown"')
    echo "错误信息: $CONNECTION_ERROR"
fi

# 7. 手动触发同步
echo ""
echo "5️⃣ 手动触发QQ邮箱同步..."
SYNC_TRIGGER=$(curl -s -X POST "$API_BASE/sync/accounts/$QQ_UID" \
    -H "Authorization: Bearer $TOKEN")
echo "同步触发: $SYNC_TRIGGER"

# 8. 等待并检查同步结果
echo ""
echo "6️⃣ 等待同步完成..."
for i in {1..6}; do
    echo "等待中... ($i/6)"
    sleep 5
    
    # 检查账户同步状态
    ACCOUNT_STATUS=$(curl -s "$API_BASE/accounts/$QQ_UID" \
        -H "Authorization: Bearer $TOKEN")
    LAST_SYNC_STATUS=$(echo "$ACCOUNT_STATUS" | jq -r '.data.last_sync_status // "unknown"')
    LAST_SYNC_ERROR=$(echo "$ACCOUNT_STATUS" | jq -r '.data.last_sync_error // ""')
    
    echo "同步状态: $LAST_SYNC_STATUS"
    if [ "$LAST_SYNC_ERROR" != "" ] && [ "$LAST_SYNC_ERROR" != "null" ]; then
        echo "同步错误: $LAST_SYNC_ERROR"
    fi
    
    if [ "$LAST_SYNC_STATUS" = "success" ] || [ "$LAST_SYNC_STATUS" = "failed" ]; then
        break
    fi
done

# 9. 检查邮件数量
echo ""
echo "7️⃣ 检查QQ邮箱邮件..."
EMAILS=$(curl -s "$API_BASE/emails?account_uid=$QQ_UID&page_size=10" \
    -H "Authorization: Bearer $TOKEN")
echo "邮件查询结果: $EMAILS"

EMAIL_TOTAL=$(echo "$EMAILS" | jq -r '.data.total // 0')
echo "QQ邮箱邮件总数: $EMAIL_TOTAL"

if [ "$EMAIL_TOTAL" -gt 0 ]; then
    echo "✅ 找到 $EMAIL_TOTAL 封邮件"
    
    echo ""
    echo "📧 最新邮件:"
    echo "$EMAILS" | jq -r '.data.emails[0:3][] | "- \(.subject) (from: \(.from_address), sent: \(.sent_at))"'
else
    echo "❌ 没有找到邮件"
fi

# 10. 检查同步日志
echo ""
echo "8️⃣ 检查同步日志..."
SYNC_LOGS=$(curl -s "$API_BASE/sync/logs?page_size=5" \
    -H "Authorization: Bearer $TOKEN")
echo "同步日志响应: $SYNC_LOGS"

# 解析同步日志
SYNC_LOG_COUNT=$(echo "$SYNC_LOGS" | jq -r '.data | length // 0')
echo "同步日志数量: $SYNC_LOG_COUNT"

if [ "$SYNC_LOG_COUNT" -gt 0 ]; then
    echo ""
    echo "📋 最新同步日志:"
    echo "$SYNC_LOGS" | jq -r '.data[0:3][] | "- \(.start_time): \(.status) (\(.emails_added) 新增/\(.emails_total) 总计) - \(.account_name // .account_uid)"'
else
    echo "❌ 没有找到同步日志"
fi

# 11. 总结诊断结果
echo ""
echo "🎯 诊断总结"
echo "==========="

ISSUES_FOUND=0

if [ "$QQ_STATUS" != "active" ]; then
    echo "❌ 账户状态问题: $QQ_STATUS (应该是 active)"
    ISSUES_FOUND=$((ISSUES_FOUND + 1))
fi

if [ "$QQ_SYNC_ENABLED" != "true" ]; then
    echo "❌ 同步未启用"
    ISSUES_FOUND=$((ISSUES_FOUND + 1))
fi

if [ "$CONNECTION_SUCCESS" != "true" ]; then
    echo "❌ 连接测试失败"
    ISSUES_FOUND=$((ISSUES_FOUND + 1))
fi

if [ "$LAST_SYNC_STATUS" = "failed" ]; then
    echo "❌ 最近同步失败: $LAST_SYNC_ERROR"
    ISSUES_FOUND=$((ISSUES_FOUND + 1))
fi

if [ "$EMAIL_TOTAL" = "0" ]; then
    echo "❌ 没有同步到邮件"
    ISSUES_FOUND=$((ISSUES_FOUND + 1))
fi

if [ "$ISSUES_FOUND" = "0" ]; then
    echo "✅ 没有发现明显问题"
else
    echo ""
    echo "💡 可能的解决方案:"
    echo "1. 确保QQ邮箱已开启IMAP服务"
    echo "2. 确认使用授权码而不是登录密码"
    echo "3. 检查网络连接"
    echo "4. 查看后端服务日志获取详细信息"
    echo "5. 尝试重新添加QQ邮箱账户"
fi

echo ""
echo "🔚 诊断完成"