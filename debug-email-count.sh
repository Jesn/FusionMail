#!/bin/bash

# 调试邮件数量解析问题的脚本
# 使用方法: ./debug-email-count.sh

set -e

API_BASE="http://localhost:8080/api/v1"
PASSWORD="admin123"

echo "🔍 调试邮件数量解析问题"
echo "======================"

# 1. 登录获取token
echo "1️⃣ 登录..."
LOGIN_RESPONSE=$(curl -s -X POST "$API_BASE/auth/login" \
    -H "Content-Type: application/json" \
    -d "{\"password\": \"$PASSWORD\"}")

TOKEN=$(echo "$LOGIN_RESPONSE" | jq -r '.data.token')
echo "Token获取成功: ${TOKEN:0:20}..."

# 2. 获取QQ账户UID
echo ""
echo "2️⃣ 获取QQ账户..."
ACCOUNTS=$(curl -s "$API_BASE/accounts" \
    -H "Authorization: Bearer $TOKEN")

QQ_UID=$(echo "$ACCOUNTS" | jq -r '.data[] | select(.provider == "qq") | .uid')
echo "QQ账户UID: $QQ_UID"

# 3. 查询邮件（详细分析）
echo ""
echo "3️⃣ 查询邮件详情..."
EMAILS_RESPONSE=$(curl -s "$API_BASE/emails?account_uid=$QQ_UID&page_size=5" \
    -H "Authorization: Bearer $TOKEN")

echo "原始响应:"
echo "$EMAILS_RESPONSE" | jq '.'

echo ""
echo "4️⃣ 分析响应结构..."

# 检查响应是否成功
SUCCESS=$(echo "$EMAILS_RESPONSE" | jq -r '.success // false')
echo "请求成功: $SUCCESS"

# 检查data字段
if echo "$EMAILS_RESPONSE" | jq -e '.data' > /dev/null 2>&1; then
    echo "✅ 包含 .data 字段"
    
    # 检查data的结构
    DATA_TYPE=$(echo "$EMAILS_RESPONSE" | jq -r '.data | type')
    echo "data类型: $DATA_TYPE"
    
    if [ "$DATA_TYPE" = "object" ]; then
        echo "data是对象，检查子字段..."
        echo "data字段: $(echo "$EMAILS_RESPONSE" | jq -r '.data | keys')"
        
        # 检查total字段
        if echo "$EMAILS_RESPONSE" | jq -e '.data.total' > /dev/null 2>&1; then
            TOTAL=$(echo "$EMAILS_RESPONSE" | jq -r '.data.total')
            echo "✅ 找到total字段: $TOTAL"
        else
            echo "❌ 没有找到total字段"
        fi
        
        # 检查emails字段
        if echo "$EMAILS_RESPONSE" | jq -e '.data.emails' > /dev/null 2>&1; then
            EMAILS_ARRAY=$(echo "$EMAILS_RESPONSE" | jq -r '.data.emails')
            EMAILS_COUNT=$(echo "$EMAILS_RESPONSE" | jq -r '.data.emails | length')
            echo "✅ 找到emails数组，长度: $EMAILS_COUNT"
            
            if [ "$EMAILS_COUNT" -gt 0 ]; then
                echo ""
                echo "📧 邮件列表:"
                echo "$EMAILS_RESPONSE" | jq -r '.data.emails[] | "- ID: \(.id), Subject: \(.subject), From: \(.from_address)"'
            fi
        else
            echo "❌ 没有找到emails字段"
        fi
    else
        echo "data不是对象: $DATA_TYPE"
    fi
else
    echo "❌ 没有找到data字段"
fi

# 5. 测试不同的解析方法
echo ""
echo "5️⃣ 测试不同的解析方法..."

# 方法1：直接从data.total获取
TOTAL_1=$(echo "$EMAILS_RESPONSE" | jq -r '.data.total // 0')
echo "方法1 (.data.total): $TOTAL_1"

# 方法2：从total字段获取（如果在顶层）
TOTAL_2=$(echo "$EMAILS_RESPONSE" | jq -r '.total // 0')
echo "方法2 (.total): $TOTAL_2"

# 方法3：计算emails数组长度
TOTAL_3=$(echo "$EMAILS_RESPONSE" | jq -r '.data.emails | length // 0')
echo "方法3 (.data.emails | length): $TOTAL_3"

# 6. 检查原始脚本的解析逻辑
echo ""
echo "6️⃣ 原始脚本解析逻辑测试..."
EMAIL_COUNT=$(echo "$EMAILS_RESPONSE" | jq -r '.total // 0')
echo "原始脚本解析结果: $EMAIL_COUNT"

echo ""
echo "🎯 问题分析:"
echo "==========="
if [ "$TOTAL_1" != "0" ]; then
    echo "✅ 正确的total值在 .data.total: $TOTAL_1"
    echo "❌ 原始脚本使用了错误的路径 .total: $EMAIL_COUNT"
    echo ""
    echo "💡 修复方案: 将脚本中的 '.total' 改为 '.data.total'"
else
    echo "⚠️  需要进一步分析响应结构"
fi

echo ""
echo "🔚 调试完成"