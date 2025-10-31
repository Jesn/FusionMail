#!/bin/bash

# 调试API响应格式的脚本
# 使用方法: ./debug-api-response.sh

set -e

API_BASE="http://localhost:8080/api/v1"

echo "🔍 调试API响应格式"
echo "=================="

# 1. 检查健康状态
echo "1️⃣ 检查健康状态..."
HEALTH_RESPONSE=$(curl -s "$API_BASE/health")
echo "健康状态响应:"
echo "$HEALTH_RESPONSE"
echo ""

# 2. 检查账户列表API
echo "2️⃣ 检查账户列表API..."
echo "请求URL: $API_BASE/accounts"
ACCOUNTS_RESPONSE=$(curl -s "$API_BASE/accounts")
echo "账户列表原始响应:"
echo "$ACCOUNTS_RESPONSE"
echo ""

# 3. 检查响应是否为有效JSON
echo "3️⃣ 检查JSON有效性..."
if echo "$ACCOUNTS_RESPONSE" | jq . > /dev/null 2>&1; then
    echo "✅ 响应是有效的JSON"
    
    # 检查响应结构
    echo ""
    echo "4️⃣ 分析响应结构..."
    echo "响应类型: $(echo "$ACCOUNTS_RESPONSE" | jq -r 'type')"
    
    if echo "$ACCOUNTS_RESPONSE" | jq -e '.data' > /dev/null 2>&1; then
        echo "✅ 包含 .data 字段"
        echo "data类型: $(echo "$ACCOUNTS_RESPONSE" | jq -r '.data | type')"
        
        if echo "$ACCOUNTS_RESPONSE" | jq -e '.data | length' > /dev/null 2>&1; then
            DATA_LENGTH=$(echo "$ACCOUNTS_RESPONSE" | jq -r '.data | length')
            echo "data长度: $DATA_LENGTH"
            
            if [ "$DATA_LENGTH" -gt 0 ]; then
                echo "✅ 找到 $DATA_LENGTH 个账户"
                echo ""
                echo "5️⃣ 账户详情:"
                echo "$ACCOUNTS_RESPONSE" | jq -r '.data[] | "- UID: \(.uid // "null"), Email: \(.email // "null"), Provider: \(.provider // "null")"'
                
                # 查找QQ账户
                echo ""
                echo "6️⃣ 查找QQ账户..."
                QQ_ACCOUNTS=$(echo "$ACCOUNTS_RESPONSE" | jq '.data[] | select(.provider == "qq")')
                if [ -n "$QQ_ACCOUNTS" ] && [ "$QQ_ACCOUNTS" != "null" ]; then
                    echo "✅ 找到QQ账户:"
                    echo "$QQ_ACCOUNTS" | jq '.'
                else
                    echo "❌ 没有找到QQ账户"
                    echo ""
                    echo "所有账户的provider字段:"
                    echo "$ACCOUNTS_RESPONSE" | jq -r '.data[] | .provider'
                fi
            else
                echo "❌ 没有账户数据"
            fi
        else
            echo "❌ data字段不是数组"
            echo "data内容: $(echo "$ACCOUNTS_RESPONSE" | jq '.data')"
        fi
    else
        echo "❌ 响应中没有 .data 字段"
        echo "响应字段: $(echo "$ACCOUNTS_RESPONSE" | jq 'keys')"
    fi
else
    echo "❌ 响应不是有效的JSON"
    echo "响应内容:"
    echo "$ACCOUNTS_RESPONSE"
    
    # 检查是否是HTML错误页面
    if echo "$ACCOUNTS_RESPONSE" | grep -q "<html>"; then
        echo ""
        echo "⚠️  响应似乎是HTML页面，可能是错误页面或路由问题"
    fi
    
    # 检查是否是认证错误
    if echo "$ACCOUNTS_RESPONSE" | grep -qi "unauthorized\|forbidden\|login"; then
        echo ""
        echo "⚠️  可能是认证问题"
    fi
fi

echo ""
echo "🔚 调试完成"