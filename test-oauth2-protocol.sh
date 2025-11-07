#!/bin/bash

# OAuth2 协议重构测试脚本

set -e

API_BASE="http://localhost:3333/api/v1"

echo "=========================================="
echo "OAuth2 协议重构测试"
echo "=========================================="
echo ""

# 测试 1: 获取支持的提供商列表
echo "测试 1: 获取支持的提供商列表"
echo "GET $API_BASE/system/providers"
echo ""

PROVIDERS_RESPONSE=$(curl -s "$API_BASE/system/providers")
echo "响应:"
echo "$PROVIDERS_RESPONSE" | jq '.'
echo ""

# 检查 Gmail 是否支持 oauth2 协议
echo "检查 Gmail 提供商配置:"
echo "$PROVIDERS_RESPONSE" | jq '.data[] | select(.name == "gmail")'
echo ""

# 检查 Outlook 是否支持 oauth2 协议
echo "检查 Outlook 提供商配置:"
echo "$PROVIDERS_RESPONSE" | jq '.data[] | select(.name == "outlook")'
echo ""

# 验证 Gmail 的推荐协议是否为 oauth2
GMAIL_RECOMMENDED=$(echo "$PROVIDERS_RESPONSE" | jq -r '.data[] | select(.name == "gmail") | .recommended_protocol')
if [ "$GMAIL_RECOMMENDED" = "oauth2" ]; then
    echo "✅ Gmail 推荐协议正确: oauth2"
else
    echo "❌ Gmail 推荐协议错误: $GMAIL_RECOMMENDED (期望: oauth2)"
fi
echo ""

# 验证 Outlook 的推荐协议是否为 oauth2
OUTLOOK_RECOMMENDED=$(echo "$PROVIDERS_RESPONSE" | jq -r '.data[] | select(.name == "outlook") | .recommended_protocol')
if [ "$OUTLOOK_RECOMMENDED" = "oauth2" ]; then
    echo "✅ Outlook 推荐协议正确: oauth2"
else
    echo "❌ Outlook 推荐协议错误: $OUTLOOK_RECOMMENDED (期望: oauth2)"
fi
echo ""

# 验证 Gmail 支持的协议列表
GMAIL_PROTOCOLS=$(echo "$PROVIDERS_RESPONSE" | jq -r '.data[] | select(.name == "gmail") | .supported_protocols | join(", ")')
echo "Gmail 支持的协议: $GMAIL_PROTOCOLS"
if echo "$GMAIL_PROTOCOLS" | grep -q "oauth2"; then
    echo "✅ Gmail 支持 oauth2 协议"
else
    echo "❌ Gmail 不支持 oauth2 协议"
fi
echo ""

# 验证 Outlook 支持的协议列表
OUTLOOK_PROTOCOLS=$(echo "$PROVIDERS_RESPONSE" | jq -r '.data[] | select(.name == "outlook") | .supported_protocols | join(", ")')
echo "Outlook 支持的协议: $OUTLOOK_PROTOCOLS"
if echo "$OUTLOOK_PROTOCOLS" | grep -q "oauth2"; then
    echo "✅ Outlook 支持 oauth2 协议"
else
    echo "❌ Outlook 不支持 oauth2 协议"
fi
echo ""

# 验证 QQ 邮箱的推荐协议是否为 imap
QQ_RECOMMENDED=$(echo "$PROVIDERS_RESPONSE" | jq -r '.data[] | select(.name == "qq") | .recommended_protocol')
if [ "$QQ_RECOMMENDED" = "imap" ]; then
    echo "✅ QQ 邮箱推荐协议正确: imap"
else
    echo "❌ QQ 邮箱推荐协议错误: $QQ_RECOMMENDED (期望: imap)"
fi
echo ""

echo "=========================================="
echo "测试完成"
echo "=========================================="
