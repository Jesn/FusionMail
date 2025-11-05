#!/bin/bash

# 测试短期邮箱自动禁用功能
# 使用测试账号：cohuuexdw097@outlook.com

ACCOUNT_UID="7cc0fc08-ed59-4f1c-b659-82e6016d6d0c"
API_BASE="http://localhost:3333/api/v1"

echo "=== 测试短期邮箱自动禁用功能 ==="
echo ""

# 1. 获取账号初始状态
echo "1. 获取账号初始状态..."
curl -s "$API_BASE/accounts/$ACCOUNT_UID" | jq '{email, status, auth_type, consecutive_auth_failures, disable_reason}'
echo ""

# 2. 触发第一次同步（预期失败）
echo "2. 触发第一次同步（预期认证失败）..."
curl -X POST "$API_BASE/accounts/$ACCOUNT_UID/sync" 2>/dev/null
sleep 3
echo "检查失败计数..."
curl -s "$API_BASE/accounts/$ACCOUNT_UID" | jq '{consecutive_auth_failures, status}'
echo ""

# 3. 触发第二次同步（预期失败）
echo "3. 触发第二次同步（预期认证失败）..."
curl -X POST "$API_BASE/accounts/$ACCOUNT_UID/sync" 2>/dev/null
sleep 3
echo "检查失败计数..."
curl -s "$API_BASE/accounts/$ACCOUNT_UID" | jq '{consecutive_auth_failures, status}'
echo ""

# 4. 触发第三次同步（预期失败并自动禁用）
echo "4. 触发第三次同步（预期认证失败并自动禁用）..."
curl -X POST "$API_BASE/accounts/$ACCOUNT_UID/sync" 2>/dev/null
sleep 3
echo "检查账号状态..."
curl -s "$API_BASE/accounts/$ACCOUNT_UID" | jq '{consecutive_auth_failures, status, disable_reason, auto_disabled_at}'
echo ""

# 5. 验证账号已被禁用
echo "5. 验证账号状态..."
ACCOUNT_STATUS=$(curl -s "$API_BASE/accounts/$ACCOUNT_UID" | jq -r '.status')
DISABLE_REASON=$(curl -s "$API_BASE/accounts/$ACCOUNT_UID" | jq -r '.disable_reason')
FAILURE_COUNT=$(curl -s "$API_BASE/accounts/$ACCOUNT_UID" | jq -r '.consecutive_auth_failures')

if [ "$ACCOUNT_STATUS" = "disabled" ] && [ "$DISABLE_REASON" = "auto_disabled_auth_failure" ] && [ "$FAILURE_COUNT" = "3" ]; then
    echo "✅ 测试通过：账号已自动禁用"
    echo "   - 状态: $ACCOUNT_STATUS"
    echo "   - 原因: $DISABLE_REASON"
    echo "   - 失败次数: $FAILURE_COUNT"
else
    echo "❌ 测试失败：账号状态异常"
    echo "   - 状态: $ACCOUNT_STATUS (期望: disabled)"
    echo "   - 原因: $DISABLE_REASON (期望: auto_disabled_auth_failure)"
    echo "   - 失败次数: $FAILURE_COUNT (期望: 3)"
    exit 1
fi
echo ""

# 6. 测试手动重新启用
echo "6. 测试手动重新启用..."
curl -X POST "$API_BASE/accounts/$ACCOUNT_UID/enable" 2>/dev/null
sleep 2

ACCOUNT_STATUS=$(curl -s "$API_BASE/accounts/$ACCOUNT_UID" | jq -r '.status')
FAILURE_COUNT=$(curl -s "$API_BASE/accounts/$ACCOUNT_UID" | jq -r '.consecutive_auth_failures')
DISABLE_REASON=$(curl -s "$API_BASE/accounts/$ACCOUNT_UID" | jq -r '.disable_reason')

if [ "$ACCOUNT_STATUS" = "active" ] && [ "$FAILURE_COUNT" = "0" ] && [ "$DISABLE_REASON" = "null" ]; then
    echo "✅ 测试通过：账号已重新启用，计数已重置"
    echo "   - 状态: $ACCOUNT_STATUS"
    echo "   - 失败次数: $FAILURE_COUNT"
    echo "   - 禁用原因: (已清除)"
else
    echo "❌ 测试失败：重新启用异常"
    echo "   - 状态: $ACCOUNT_STATUS (期望: active)"
    echo "   - 失败次数: $FAILURE_COUNT (期望: 0)"
    echo "   - 禁用原因: $DISABLE_REASON (期望: null)"
    exit 1
fi
echo ""

echo "=== 所有测试通过 ==="
