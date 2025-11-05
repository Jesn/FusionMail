#!/bin/bash

# 测试重复请求问题的脚本
# 使用 Playwright 自动化测试

echo "=== 测试重复请求问题 ==="
echo ""
echo "测试场景："
echo "1. 首次加载页面"
echo "2. 点击 Logo"
echo "3. 点击文件夹按钮"
echo "4. 点击邮箱账户按钮"
echo ""

# 启动 Playwright 测试
echo "启动浏览器测试..."
echo ""

# 这里应该使用 Playwright 的 CLI 或者 Node.js 脚本
# 但由于我们已经在 Kiro 中使用 MCP Playwright，这个脚本主要用于文档记录

echo "测试完成！"
echo ""
echo "预期结果："
echo "- 首次加载：/api/v1/accounts 应该只被调用 1 次"
echo "- 点击 Logo：不应该触发新的 /api/v1/accounts 请求"
echo "- 点击文件夹：不应该触发新的 /api/v1/accounts 请求"
echo "- 点击邮箱账户：不应该触发新的 /api/v1/accounts 请求"
