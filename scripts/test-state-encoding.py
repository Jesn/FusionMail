#!/usr/bin/env python3

import urllib.parse
import requests
import json

# 测试state参数编码问题

print("🔍 测试OAuth2 State参数编码")
print("=" * 40)

# 生成授权URL
try:
    response = requests.get("http://localhost:3333/api/v1/auth/microsoft/authorize?email=test@hotmail.com")
    data = response.json()
    
    if data.get('success'):
        auth_url = data['data']['auth_url']
        state = data['data']['state']
        
        print(f"生成的State: {state}")
        print(f"State长度: {len(state)}")
        print()
        
        # 解析URL
        parsed = urllib.parse.urlparse(auth_url)
        params = urllib.parse.parse_qs(parsed.query)
        
        if 'state' in params:
            url_state = params['state'][0]
            print(f"URL中的State: {url_state}")
            print(f"URL State长度: {len(url_state)}")
            print(f"State匹配: {state == url_state}")
        
        print()
        print("🔗 完整授权URL:")
        print(auth_url)
        
        print()
        print("🧪 模拟回调测试:")
        
        # 模拟不同的state参数情况
        test_cases = [
            ("完整state", state),
            ("截断state", state[:3]),  # 模拟截断情况
            ("URL编码state", urllib.parse.quote(state, safe='')),
        ]
        
        for name, test_state in test_cases:
            print(f"\n{name}: {test_state}")
            callback_url = f"http://localhost:3333/api/v1/auth/microsoft/callback?code=TEST_CODE&state={test_state}"
            print(f"回调URL: {callback_url}")
            
            try:
                callback_response = requests.get(callback_url)
                if "授权成功" in callback_response.text:
                    print("✅ 回调成功")
                elif "授权失败" in callback_response.text:
                    print("❌ 回调失败")
                else:
                    print("⚠️ 未知响应")
            except Exception as e:
                print(f"❌ 请求失败: {e}")
    
    else:
        print("❌ 无法生成授权URL")
        print(f"响应: {data}")

except Exception as e:
    print(f"❌ 请求失败: {e}")

print()
print("📋 分析结论:")
print("如果截断state测试失败，说明问题确实是state参数被截断")
print("如果完整state测试也失败，说明问题在其他地方")