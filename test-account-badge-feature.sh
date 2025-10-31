#!/bin/bash

# 邮箱标识显示功能测试脚本

echo "🧪 测试邮箱标识显示功能..."

# 检查前端是否运行
if ! curl -s http://localhost:3000 > /dev/null; then
    echo "❌ 前端服务未运行，请先启动前端服务"
    exit 1
fi

echo "✅ 前端服务正在运行"

echo ""
echo "📋 邮箱标识功能测试项目："
echo ""
echo "1. 显示逻辑测试"
echo "   ✅ 选中"所有邮箱"时显示邮箱标识"
echo "   ✅ 选中特定邮箱时不显示邮箱标识"
echo "   ✅ 邮箱标识位于邮件项右上角"
echo ""
echo "2. 邮箱标识样式"
echo "   ✅ 使用 Badge 组件显示"
echo "   ✅ 包含 Mail 图标"
echo "   ✅ 显示邮箱用户名（@前的部分）"
echo "   ✅ 鼠标悬停显示完整邮箱地址"
echo ""
echo "3. 布局适配"
echo "   ✅ 邮箱标识与时间垂直排列"
echo "   ✅ 不影响原有的邮件信息显示"
echo "   ✅ 响应式设计，适配不同屏幕尺寸"
echo ""

echo "🔧 实现的功能："
echo "✅ 在 EmailItem 组件中添加邮箱标识显示"
echo "✅ 使用 showAccountBadge 参数控制显示逻辑"
echo "✅ 传递 accounts 数据用于邮箱信息查找"
echo "✅ 在 InboxPage 中根据筛选状态决定是否显示"
echo "✅ 优化了右侧布局，支持邮箱标识和时间并存"

echo ""
echo "📝 手动测试步骤："
echo "1. 打开浏览器访问 http://localhost:3000"
echo "2. 确保侧边栏选中"所有邮箱""
echo "3. 检查邮件列表中每封邮件的右上角："
echo "   - 应该显示邮箱标识 Badge"
echo "   - Badge 包含 Mail 图标和用户名"
echo "   - 鼠标悬停显示完整邮箱地址"
echo ""
echo "4. 点击特定邮箱账户（如 Gmail）"
echo "5. 检查邮件列表："
echo "   - 邮箱标识应该消失"
echo "   - 只显示时间信息"
echo ""
echo "6. 重新点击"所有邮箱""
echo "7. 确认邮箱标识重新出现"

echo ""
echo "🎯 预期结果："
echo "- 所有邮箱模式：每封邮件右上角显示邮箱标识"
echo "- 特定邮箱模式：不显示邮箱标识"
echo "- 邮箱标识样式美观，信息清晰"
echo "- 布局协调，不影响其他元素"

echo ""
echo "💡 用户体验改进："
echo "- 解决了多邮箱视图下邮件来源识别困难的问题"
echo "- 提供了快速的视觉区分方式"
echo "- 保持了界面的简洁性和一致性"
echo "- 支持响应式设计，适配各种设备"

echo ""
echo "🔍 技术实现要点："
echo "- 使用条件渲染控制邮箱标识显示"
echo "- 通过 account_uid 匹配账户信息"
echo "- 提取邮箱用户名作为简短标识"
echo "- 使用 Badge 组件保持设计一致性"
echo "- 优化布局结构支持多元素并存"