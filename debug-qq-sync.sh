#!/bin/bash

# 调试QQ邮箱同步问题的脚本
# 使用方法: ./debug-qq-sync.sh

set -e

API_BASE="http://localhost:8080/api/v1"

echo "🔍 调试QQ邮箱同步问题"
echo "======================"

# 检查服务是否运行
echo "📡 检查服务状态..."
if ! curl -s "$API_BASE/health" > /dev/null; then
    echo "❌ 服务未运行，请先启动后端服务"
    exit 1
fi
echo "✅ 服务正常运行"

# 获取账户列表
echo ""
echo "📋 获取账户列表..."
ACCOUNTS_RESPONSE=$(curl -s "$API_BASE/accounts" -H "Content-Type: application/json")
echo "账户列表响应: $ACCOUNTS_RESPONSE"

# 查找QQ邮箱账户
QQ_ACCOUNT_UID=$(echo "$ACCOUNTS_RESPONSE" | jq -r '.data[] | select(.provider == "qq") | .uid // empty')

if [ -z "$QQ_ACCOUNT_UID" ]; then
    echo "❌ 没有找到QQ邮箱账户"
    exit 1
fi

echo "✅ 找到QQ邮箱账户: $QQ_ACCOUNT_UID"

# 获取QQ账户详情
echo ""
echo "🔍 获取QQ账户详情..."
QQ_ACCOUNT_DETAIL=$(curl -s "$API_BASE/accounts/$QQ_ACCOUNT_UID" -H "Content-Type: application/json")
echo "QQ账户详情: $QQ_ACCOUNT_DETAIL"

# 提取关键信息
EMAIL=$(echo "$QQ_ACCOUNT_DETAIL" | jq -r '.data.email // "unknown"')
STATUS=$(echo "$QQ_ACCOUNT_DETAIL" | jq -r '.data.status // "unknown"')
SYNC_ENABLED=$(echo "$QQ_ACCOUNT_DETAIL" | jq -r '.data.sync_enabled // false')
LAST_SYNC_AT=$(echo "$QQ_ACCOUNT_DETAIL" | jq -r '.data.last_sync_at // "never"')
LAST_SYNC_STATUS=$(echo "$QQ_ACCOUNT_DETAIL" | jq -r '.data.last_sync_status // "unknown"')
LAST_SYNC_ERROR=$(echo "$QQ_ACCOUNT_DETAIL" | jq -r '.data.last_sync_error // "none"')

echo ""
echo "📊 QQ账户状态信息:"
echo "- 邮箱地址: $EMAIL"
echo "- 账户状态: $STATUS"
echo "- 同步启用: $SYNC_ENABLED"
echo "- 最后同步: $LAST_SYNC_AT"
echo "- 同步状态: $LAST_SYNC_STATUS"
echo "- 同步错误: $LAST_SYNC_ERROR"

# 测试连接
echo ""
echo "🔗 测试QQ邮箱连接..."
CONNECTION_TEST=$(curl -s -X POST "$API_BASE/accounts/$QQ_ACCOUNT_UID/test" -H "Content-Type: application/json")
echo "连接测试结果: $CONNECTION_TEST"

# 手动触发同步
echo ""
echo "🔄 手动触发QQ邮箱同步..."
SYNC_RESPONSE=$(curl -s -X POST "$API_BASE/sync/accounts/$QQ_ACCOUNT_UID" -H "Content-Type: application/json")
echo "同步触发结果: $SYNC_RESPONSE"

# 等待同步完成
echo ""
echo "⏳ 等待同步完成（10秒）..."
sleep 10

# 再次检查账户状态
echo ""
echo "🔍 检查同步后的账户状态..."
UPDATED_ACCOUNT=$(curl -s "$API_BASE/accounts/$QQ_ACCOUNT_UID" -H "Content-Type: application/json")
NEW_LAST_SYNC_AT=$(echo "$UPDATED_ACCOUNT" | jq -r '.data.last_sync_at // "never"')
NEW_LAST_SYNC_STATUS=$(echo "$UPDATED_ACCOUNT" | jq -r '.data.last_sync_status // "unknown"')
NEW_LAST_SYNC_ERROR=$(echo "$UPDATED_ACCOUNT" | jq -r '.data.last_sync_error // "none"')

echo "- 最后同步: $NEW_LAST_SYNC_AT"
echo "- 同步状态: $NEW_LAST_SYNC_STATUS"
echo "- 同步错误: $NEW_LAST_SYNC_ERROR"

# 检查邮件数量
echo ""
echo "📧 检查QQ邮箱的邮件数量..."
EMAILS_RESPONSE=$(curl -s "$API_BASE/emails?account_uid=$QQ_ACCOUNT_UID" -H "Content-Type: application/json")
EMAIL_COUNT=$(echo "$EMAILS_RESPONSE" | jq -r '.total // 0')
echo "QQ邮箱邮件总数: $EMAIL_COUNT"

if [ "$EMAIL_COUNT" -gt 0 ]; then
    echo "✅ 找到 $EMAIL_COUNT 封邮件"
    
    # 显示最新的几封邮件
    echo ""
    echo "📬 最新邮件列表:"
    echo "$EMAILS_RESPONSE" | jq -r '.data[0:3][] | "- \(.subject) (from: \(.from_address), sent: \(.sent_at))"'
else
    echo "❌ 没有找到邮件"
fi

# 检查同步日志
echo ""
echo "📝 检查同步日志..."
SYNC_LOGS=$(curl -s "$API_BASE/sync/logs?account_uid=$QQ_ACCOUNT_UID&limit=5" -H "Content-Type: application/json")
echo "同步日志: $SYNC_LOGS"

echo ""
echo "🎯 诊断总结:"
echo "============"
if [ "$STATUS" != "active" ]; then
    echo "❌ 账户状态不是 active: $STATUS"
fi

if [ "$SYNC_ENABLED" != "true" ]; then
    echo "❌ 同步未启用"
fi

if [ "$NEW_LAST_SYNC_STATUS" = "failed" ]; then
    echo "❌ 最近同步失败: $NEW_LAST_SYNC_ERROR"
fi

if [ "$EMAIL_COUNT" = "0" ]; then
    echo "❌ 没有同步到邮件，可能的原因:"
    echo "  1. 账户认证问题"
    echo "  2. IMAP配置错误"
    echo "  3. 网络连接问题"
    echo "  4. 邮箱服务器限制"
fi

echo ""
echo "💡 建议检查:"
echo "1. 确认QQ邮箱已开启IMAP服务"
echo "2. 确认使用的是授权码而不是登录密码"
echo "3. 检查网络连接和防火墙设置"
echo "4. 查看后端服务日志获取详细错误信息"