#!/usr/bin/env python3
"""
Kiro Provider 测试脚本
用于测试 Go AI Proxy 的 Kiro 集成
"""

import requests
import json
import time
import os

# 配置
BASE_URL = "http://localhost:3000"
API_KEY = "test-key-123"
PROVIDER = "kiro-api"

def test_health():
    """测试健康检查"""
    print("1. 测试健康检查...")
    try:
        response = requests.get(f"{BASE_URL}/health")
        if response.status_code == 200:
            print("✓ 服务器运行正常")
            print(f"  响应: {response.json()}")
        else:
            print("✗ 健康检查失败")
            return False
    except Exception as e:
        print(f"✗ 无法连接到服务器: {e}")
        print("\n请确保服务器正在运行:")
        print("./aiproxy --model-provider kiro-api --api-key test-key-123")
        return False
    return True

def test_list_models():
    """测试模型列表"""
    print("\n2. 获取 Kiro 支持的模型...")
    headers = {
        "Authorization": f"Bearer {API_KEY}",
        "X-Model-Provider": PROVIDER
    }
    
    try:
        response = requests.get(f"{BASE_URL}/v1/models", headers=headers)
        if response.status_code == 200:
            models = response.json()
            print("✓ 可用模型:")
            for model in models.get("data", []):
                print(f"  - {model['id']} (by {model.get('owned_by', 'unknown')})")
        else:
            print(f"✗ 获取模型失败: {response.status_code}")
            print(f"  响应: {response.text}")
    except Exception as e:
        print(f"✗ 请求失败: {e}")

def test_chat_completion():
    """测试聊天完成"""
    print("\n3. 测试非流式聊天...")
    headers = {
        "Authorization": f"Bearer {API_KEY}",
        "X-Model-Provider": PROVIDER,
        "Content-Type": "application/json"
    }
    
    data = {
        "model": "claude-3-sonnet-20240229",
        "messages": [
            {"role": "user", "content": "你好！请用一句话介绍你自己。"}
        ],
        "max_tokens": 100
    }
    
    try:
        response = requests.post(f"{BASE_URL}/v1/chat/completions", 
                               headers=headers, 
                               json=data)
        if response.status_code == 200:
            result = response.json()
            content = result["choices"][0]["message"]["content"]
            print("✓ AI 回复:")
            print(f"  {content}")
            
            # 显示 token 使用情况
            usage = result.get("usage", {})
            print(f"\n  Token 使用: {usage.get('total_tokens', 0)} " +
                  f"(输入: {usage.get('prompt_tokens', 0)}, " +
                  f"输出: {usage.get('completion_tokens', 0)})")
        else:
            print(f"✗ 聊天请求失败: {response.status_code}")
            print(f"  响应: {response.text}")
    except Exception as e:
        print(f"✗ 请求失败: {e}")

def test_stream_chat():
    """测试流式聊天"""
    print("\n4. 测试流式聊天...")
    headers = {
        "Authorization": f"Bearer {API_KEY}",
        "X-Model-Provider": PROVIDER,
        "Content-Type": "application/json"
    }
    
    data = {
        "model": "claude-3-sonnet-20240229",
        "messages": [
            {"role": "user", "content": "请从1数到5，每个数字之间停顿一下。"}
        ],
        "max_tokens": 100,
        "stream": True
    }
    
    try:
        response = requests.post(f"{BASE_URL}/v1/chat/completions", 
                               headers=headers, 
                               json=data,
                               stream=True)
        
        if response.status_code == 200:
            print("✓ 流式响应:")
            print("  ", end="", flush=True)
            
            for line in response.iter_lines():
                if line:
                    line = line.decode('utf-8')
                    if line.startswith("data: "):
                        data = line[6:]
                        if data == "[DONE]":
                            print()
                            break
                        try:
                            chunk = json.loads(data)
                            content = chunk["choices"][0]["delta"].get("content", "")
                            print(content, end="", flush=True)
                        except:
                            pass
        else:
            print(f"✗ 流式请求失败: {response.status_code}")
    except Exception as e:
        print(f"✗ 请求失败: {e}")

def test_with_system_prompt():
    """测试带系统提示的请求"""
    print("\n5. 测试系统提示...")
    headers = {
        "Authorization": f"Bearer {API_KEY}",
        "X-Model-Provider": PROVIDER,
        "Content-Type": "application/json"
    }
    
    data = {
        "model": "claude-3-sonnet-20240229",
        "messages": [
            {"role": "system", "content": "你是一个专业的数学老师。"},
            {"role": "user", "content": "什么是勾股定理？"}
        ],
        "max_tokens": 150
    }
    
    try:
        response = requests.post(f"{BASE_URL}/v1/chat/completions", 
                               headers=headers, 
                               json=data)
        if response.status_code == 200:
            result = response.json()
            content = result["choices"][0]["message"]["content"]
            print("✓ AI 回复（作为数学老师）:")
            print(f"  {content}")
        else:
            print(f"✗ 请求失败: {response.status_code}")
    except Exception as e:
        print(f"✗ 请求失败: {e}")

def main():
    """主测试函数"""
    print("=== Kiro Provider 测试 ===")
    print(f"服务器: {BASE_URL}")
    print(f"API Key: {API_KEY}")
    print(f"Provider: {PROVIDER}")
    
    # 检查是否使用模拟模式
    if os.getenv("KIRO_MOCK_MODE") == "true":
        print("\n注意: 使用模拟模式 (KIRO_MOCK_MODE=true)")
    
    print("\n开始测试...\n")
    
    # 运行测试
    if not test_health():
        return
    
    test_list_models()
    test_chat_completion()
    test_stream_chat()
    test_with_system_prompt()
    
    print("\n✓ 所有测试完成！")
    
    # 显示如何使用真实凭据
    print("\n如何使用真实 Kiro 凭据:")
    print("1. 创建凭据文件 kiro-creds.json")
    print("2. 启动服务器:")
    print("   ./aiproxy --model-provider kiro-api --kiro-oauth-creds-file kiro-creds.json")
    print("3. 运行此测试脚本")

if __name__ == "__main__":
    main()