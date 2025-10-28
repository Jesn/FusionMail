#!/bin/bash

# Git 提交安全检查脚本
# 在提交前运行此脚本检查是否包含敏感信息

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

echo "=========================================="
echo "  Git 提交安全检查"
echo "=========================================="
echo ""

ISSUES_FOUND=0

# 1. 检查暂存的文件
echo "1. 检查暂存的文件..."
STAGED_FILES=$(git diff --cached --name-only)

if [ -z "$STAGED_FILES" ]; then
    echo -e "${YELLOW}⚠ 没有暂存的文件${NC}"
    exit 0
fi

echo "   暂存的文件:"
echo "$STAGED_FILES" | sed 's/^/     - /'
echo ""

# 2. 检查邮箱地址
echo "2. 检查邮箱地址..."
EMAIL_MATCHES=$(git diff --cached | grep -nE "[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}" | grep -v "your@" | grep -v "example.com" | grep -v "test@")

if [ -n "$EMAIL_MATCHES" ]; then
    echo -e "${RED}✗ 发现可能的真实邮箱地址:${NC}"
    echo "$EMAIL_MATCHES" | head -5
    ISSUES_FOUND=$((ISSUES_FOUND + 1))
else
    echo -e "${GREEN}✓ 未发现真实邮箱地址${NC}"
fi
echo ""

# 3. 检查密码字段
echo "3. 检查密码字段..."
PASSWORD_MATCHES=$(git diff --cached | grep -niE "(password|passwd|pwd).*=.*['\"][^'\"]{8,}" | grep -v "your_" | grep -v "example" | grep -v "placeholder")

if [ -n "$PASSWORD_MATCHES" ]; then
    echo -e "${RED}✗ 发现可能的密码:${NC}"
    echo "$PASSWORD_MATCHES" | head -5
    ISSUES_FOUND=$((ISSUES_FOUND + 1))
else
    echo -e "${GREEN}✓ 未发现硬编码密码${NC}"
fi
echo ""

# 4. 检查 API 密钥
echo "4. 检查 API 密钥..."
API_KEY_MATCHES=$(git diff --cached | grep -niE "(api[_-]?key|token|secret).*=.*['\"][a-zA-Z0-9]{20,}" | grep -v "your_" | grep -v "example")

if [ -n "$API_KEY_MATCHES" ]; then
    echo -e "${RED}✗ 发现可能的 API 密钥:${NC}"
    echo "$API_KEY_MATCHES" | head -5
    ISSUES_FOUND=$((ISSUES_FOUND + 1))
else
    echo -e "${GREEN}✓ 未发现 API 密钥${NC}"
fi
echo ""

# 5. 检查敏感文件
echo "5. 检查敏感文件..."
SENSITIVE_FILES=$(echo "$STAGED_FILES" | grep -E "(\.env$|\.test-config$|credentials|secrets|\.pem$|\.key$)")

if [ -n "$SENSITIVE_FILES" ]; then
    echo -e "${RED}✗ 发现敏感文件:${NC}"
    echo "$SENSITIVE_FILES" | sed 's/^/     - /'
    ISSUES_FOUND=$((ISSUES_FOUND + 1))
else
    echo -e "${GREEN}✓ 未发现敏感文件${NC}"
fi
echo ""

# 6. 检查 .gitignore
echo "6. 检查 .gitignore..."
if [ -f ".gitignore" ]; then
    if grep -q "\.test-config" .gitignore && grep -q "\.env" .gitignore; then
        echo -e "${GREEN}✓ .gitignore 配置正确${NC}"
    else
        echo -e "${YELLOW}⚠ .gitignore 可能缺少敏感文件配置${NC}"
        ISSUES_FOUND=$((ISSUES_FOUND + 1))
    fi
else
    echo -e "${RED}✗ 缺少 .gitignore 文件${NC}"
    ISSUES_FOUND=$((ISSUES_FOUND + 1))
fi
echo ""

# 7. 检查提交消息（如果已经写了）
echo "7. 检查最近的提交消息..."
LAST_COMMIT_MSG=$(git log -1 --pretty=%B 2>/dev/null)
if [ -n "$LAST_COMMIT_MSG" ]; then
    if echo "$LAST_COMMIT_MSG" | grep -qE "[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}"; then
        echo -e "${YELLOW}⚠ 最近的提交消息包含邮箱地址${NC}"
    else
        echo -e "${GREEN}✓ 提交消息安全${NC}"
    fi
else
    echo "   (尚未提交)"
fi
echo ""

# 总结
echo "=========================================="
echo "  检查结果"
echo "=========================================="
echo ""

if [ $ISSUES_FOUND -eq 0 ]; then
    echo -e "${GREEN}✓✓✓ 安全检查通过！${NC}"
    echo ""
    echo "可以安全提交："
    echo "  git commit -m 'your message'"
    exit 0
else
    echo -e "${RED}✗ 发现 $ISSUES_FOUND 个潜在问题${NC}"
    echo ""
    echo "建议："
    echo "  1. 检查上述标记的内容"
    echo "  2. 使用占位符替换真实信息"
    echo "  3. 更新 .gitignore"
    echo "  4. 重新运行此脚本"
    echo ""
    echo "如果确认无误，可以继续提交"
    exit 1
fi
