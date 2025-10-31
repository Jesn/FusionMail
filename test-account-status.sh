#!/bin/bash

# 测试账户状态功能的脚本
# 使用方法: ./test-account-status.sh

set -e

API_BASE="http://localhost:8080/api/v1"
ACCOUNT_UID=""

echo "🧪 测试账户状态功能"
echo "===================="

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

# 提取第一个账户的 UID（假设存在账户）
ACCOUNT_UID=$(echo "$ACCOUNTS_RESPONSE" | jq -r '.data[0].uid // empty')

if [ -z "$ACCOUNT_UID" ]; then
    echo "❌ 没有找到账户，请先添加一个账户"
    exit 1
fi

echo "✅ 找到账户: $ACCOUNT_UID"

# 获取账户详情
echo ""
echo "🔍 获取账户详情..."
ACCOUNT_DETAIL=$(curl -s "$API_BASE/accounts/$ACCOUNT_UID" -H "Content-Type: application/json")
echo "账户详情: $ACCOUNT_DETAIL"

CURRENT_STATUS=$(echo "$ACCOUNT_DETAIL" | jq -r '.data.status // "unknown"')
echo "当前状态: $CURRENT_STATUS"

# 测试禁用账户
echo ""
echo "🔒 测试禁用账户..."
DISABLE_RESPONSE=$(curl -s -X POST "$API_BASE/accounts/$ACCOUNT_UID/disable" -H "Content-Type: application/json")
echo "禁用响应: $DISABLE_RESPONSE"

# 验证状态已更改
echo ""
echo "🔍 验证账户已禁用..."
sleep 1
UPDATED_ACCOUNT=$(curl -s "$API_BASE/accounts/$ACCOUNT_UID" -H "Content-Type: application/json")
NEW_STATUS=$(echo "$UPDATED_ACCOUNT" | jq -r '.data.status // "unknown"')
echo "新状态: $NEW_STATUS"

if [ "$NEW_STATUS" = "disabled" ]; then
    echo "✅ 账户已成功禁用"
else
    echo "❌ 账户禁用失败，状态仍为: $NEW_STATUS"
fi

# 测试启用账户
echo ""
echo "🔓 测试启用账户..."
ENABLE_RESPONSE=$(curl -s -X POST "$API_BASE/accounts/$ACCOUNT_UID/enable" -H "Content-Type: application/json")
echo "启用响应: $ENABLE_RESPONSE"

# 验证状态已恢复
echo ""
echo "🔍 验证账户已启用..."
sleep 1
FINAL_ACCOUNT=$(curl -s "$API_BASE/accounts/$ACCOUNT_UID" -H "Content-Type: application/json")
FINAL_STATUS=$(echo "$FINAL_ACCOUNT" | jq -r '.data.status // "unknown"')
echo "最终状态: $FINAL_STATUS"

if [ "$FINAL_STATUS" = "active" ]; then
    echo "✅ 账户已成功启用"
else
    echo "❌ 账户启用失败，状态为: $FINAL_STATUS"
fi

echo ""
echo "🎉 测试完成！"
echo "===================="
echo "总结:"
echo "- 初始状态: $CURRENT_STATUS"
echo "- 禁用后状态: $NEW_STATUS"
echo "- 启用后状态: $FINAL_STATUS"