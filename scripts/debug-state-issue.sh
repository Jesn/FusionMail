#!/bin/bash

# 调试OAuth2 state参数截断问题

echo "🔍 调试OAuth2 State参数问题"
echo "============================"

# 加载环境变量
if [ -f "backend/.env" ]; then
    source backend/.env
fi

echo "📊 生成新的授权URL并分析state参数..."

# 生成授权URL
response=$(curl -s "http://localhost:$SERVER_PORT/api/v1/auth/microsoft/authorize?email=test@hotmail.com")

if echo "$response" | grep -q "auth_url"; then
    auth_url=$(echo "$response" | python3 -c "
import sys, json, urllib.parse
data = json.load(sys.stdin)
auth_url = data['data']['auth_url']
state = data['data']['state']

print('生成的State:', state)
print('State长度:', len(state))
print()

# 解析URL参数
parsed = urllib.parse.urlparse(auth_url)
params = urllib.parse.parse_qs(parsed.query)

if 'state' in params:
    url_state = params['state'][0]
    print('URL中的State:', url_state)
    print('URL State长度:', len(url_state))
    print('State匹配:', state == url_state)
else:
    print('❌ URL中没有找到state参数')

print()
print('完整授权URL:')
print(auth_url)
")

    echo ""
    echo "🧪 测试state参数解析..."
    
    # 提取state参数进行测试
    state=$(echo "$response" | python3 -c "
import sys, json
data = json.load(sys.stdin)
print(data['data']['state'])
")
    
    echo "提取的State: $state"
    
    # 测试URL编码
    encoded_state=$(python3 -c "
import urllib.parse
state = '$state'
encoded = urllib.parse.quote(state, safe='')
print('URL编码后:', encoded)
print('URL解码后:', urllib.parse.unquote(encoded))
")
    
    echo "$encoded_state"
    
    echo ""
    echo "🔧 手动测试回调URL..."
    echo "请复制以下URL到浏览器测试（使用完整的state参数）："
    echo "http://localhost:$SERVER_PORT/api/v1/auth/microsoft/callback?code=TEST_CODE&state=$state"
    
else
    echo "❌ 无法生成授权URL"
    echo "响应: $response"
fi

echo ""
echo "📋 可能的问题和解决方案："
echo ""
echo "1. URL长度限制："
echo "   - 某些浏览器对URL长度有限制"
echo "   - 解决：缩短state参数长度"
echo ""
echo "2. URL编码问题："
echo "   - state参数包含特殊字符"
echo "   - 解决：确保正确的URL编码"
echo ""
echo "3. 中间件截断："
echo "   - 代理或中间件截断了URL参数"
echo "   - 解决：检查网络配置"
echo ""
echo "4. 浏览器问题："
echo "   - 浏览器自动截断了URL"
echo "   - 解决：尝试不同的浏览器"