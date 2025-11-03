#!/bin/bash

# Git 提交前代码清理脚本
# 用于清理调试信息、敏感数据和不必要的日志

set -e

echo "🔍 开始检查代码中的敏感信息和调试代码..."

# 检查是否有未提交的更改
if ! git diff --quiet; then
    echo "⚠️  检测到未提交的更改，将进行清理..."
else
    echo "✅ 没有未提交的更改"
    exit 0
fi

# 1. 检查是否包含真实邮箱地址
echo "🔍 检查真实邮箱地址..."
if git diff --cached | grep -qE "[a-zA-Z0-9._%+-]+@(qq|163|gmail|outlook|hotmail)\.com" 2>/dev/null; then
    echo "❌ 发现可能的真实邮箱地址！"
    git diff --cached | grep -E "[a-zA-Z0-9._%+-]+@(qq|163|gmail|outlook|hotmail)\.com" || true
    echo "请将真实邮箱替换为占位符，如 your@example.com"
    exit 1
fi

# 2. 检查是否包含密码或授权码
echo "🔍 检查密码和授权码..."
if git diff --cached | grep -qiE "(password|token|secret|key).*[:=].*['\"][^'\"]{8,}" 2>/dev/null; then
    echo "❌ 发现可能的密码或授权码！"
    git diff --cached | grep -iE "(password|token|secret|key).*[:=].*['\"][^'\"]{8,}" || true
    echo "请将真实密码替换为占位符"
    exit 1
fi

echo "✅ 敏感信息检查通过"

# 3. 清理后端调试日志
echo "🧹 清理后端调试日志..."

# 清理 sync_service.go 中的调试日志
if [ -f "backend/internal/service/sync_service.go" ]; then
    echo "  - 清理 sync_service.go 中的调试日志"
    
    # 移除包含敏感信息的日志
    sed -i.bak 's/log\.Printf("Incremental sync for account %s since %s", account\.UID, since\.Format(time\.RFC3339))/\/\/ Incremental sync started/' backend/internal/service/sync_service.go
    sed -i.bak 's/log\.Printf("Initial sync for account %s since %s", account\.UID, since\.Format(time\.RFC3339))/\/\/ Initial sync started/' backend/internal/service/sync_service.go
    sed -i.bak 's/log\.Printf("Sync completed for account %s: %d new emails", accountUID, syncLog\.EmailsNew)/\/\/ Sync completed successfully/' backend/internal/service/sync_service.go
    sed -i.bak 's/log\.Printf("Sync failed for account %s: %v", accountUID, err)/\/\/ Sync failed/' backend/internal/service/sync_service.go
    sed -i.bak 's/log\.Printf("Failed to process email %s: %v", email\.ProviderID, err)/\/\/ Failed to process email/' backend/internal/service/sync_service.go
    sed -i.bak 's/log\.Printf("Failed to sync account %s: %v", accountUID, err)/\/\/ Failed to sync account/' backend/internal/service/sync_service.go
    sed -i.bak 's/log\.Printf("Auto-fixing incorrect host: %s -> mail\.linux\.do", credentials\.Host)/\/\/ Auto-fixing incorrect host configuration/' backend/internal/service/sync_service.go
    
    # 移除备份文件
    rm -f backend/internal/service/sync_service.go.bak
fi

echo "✅ 后端调试日志清理完成"
echo ""
echo "📋 提交前检查清单："
echo "  ✅ 敏感信息已检查"
echo "  ✅ 调试日志已清理"
echo ""
echo "💡 建议的提交命令："
echo "  git add ."
echo "  git commit -m 'feat: 实现 Gmail OAuth2 认证和邮件同步功能'"
echo ""