#!/bin/bash

echo "=== Kiro Provider Test Script ==="
echo

# 设置颜色
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# 检查是否有 Kiro 凭据
if [ -z "$KIRO_OAUTH_CREDS_FILE" ] && [ -z "$KIRO_OAUTH_CREDS_BASE64" ]; then
    echo -e "${YELLOW}警告: 未设置 Kiro OAuth 凭据${NC}"
    echo "请设置以下环境变量之一:"
    echo "  - KIRO_OAUTH_CREDS_FILE: OAuth 凭据文件路径"
    echo "  - KIRO_OAUTH_CREDS_BASE64: Base64 编码的 OAuth 凭据"
    echo
    echo "示例:"
    echo "  export KIRO_OAUTH_CREDS_FILE=/path/to/kiro-credentials.json"
    echo
    echo "或者使用示例凭据文件:"
    echo "  cp configs/kiro-credentials-example.json configs/kiro-credentials.json"
    echo "  # 编辑 configs/kiro-credentials.json 填入真实凭据"
    echo "  export KIRO_OAUTH_CREDS_FILE=configs/kiro-credentials.json"
    echo
    
    # 如果没有设置，使用模拟模式
    echo -e "${YELLOW}使用模拟模式进行测试...${NC}"
    export KIRO_MOCK_MODE=true
fi

# 编译项目
echo "编译项目..."
cd /workspace/go-aiproxy
go mod download
go build -o aiproxy cmd/server/main.go

if [ $? -ne 0 ]; then
    echo -e "${RED}编译失败！${NC}"
    exit 1
fi

echo -e "${GREEN}编译成功！${NC}"
echo

# 启动服务器
echo "启动 AI Proxy 服务器（包含 Kiro 提供商）..."
./aiproxy \
    --host 0.0.0.0 \
    --port 3000 \
    --api-key test-key-123 \
    --model-provider kiro-api,openai-custom \
    --openai-api-key sk-test \
    --log-prompts console &

SERVER_PID=$!
echo "服务器 PID: $SERVER_PID"

# 等待服务器启动
echo "等待服务器启动..."
sleep 3

# 检查服务器是否运行
if ! ps -p $SERVER_PID > /dev/null; then
    echo -e "${RED}服务器启动失败！${NC}"
    exit 1
fi

echo -e "${GREEN}服务器已启动！${NC}"
echo

# 测试健康检查
echo "测试健康检查..."
curl -s http://localhost:3000/health | jq .
echo

# 测试列出模型
echo "测试列出 Kiro 模型..."
curl -s http://localhost:3000/v1/models \
    -H "Authorization: Bearer test-key-123" \
    -H "X-Model-Provider: kiro-api" | jq .
echo

# 测试聊天完成（非流式）
echo "测试 Kiro 聊天完成（非流式）..."
curl -s -X POST http://localhost:3000/v1/chat/completions \
    -H "Authorization: Bearer test-key-123" \
    -H "X-Model-Provider: kiro-api" \
    -H "Content-Type: application/json" \
    -d '{
        "model": "claude-3-sonnet-20240229",
        "messages": [
            {"role": "user", "content": "Hello! Please respond with a short greeting."}
        ],
        "max_tokens": 50
    }' | jq .
echo

# 测试聊天完成（流式）
echo "测试 Kiro 聊天完成（流式）..."
echo -e "${YELLOW}流式响应:${NC}"
curl -s -X POST http://localhost:3000/v1/chat/completions \
    -H "Authorization: Bearer test-key-123" \
    -H "X-Model-Provider: kiro-api" \
    -H "Content-Type: application/json" \
    -d '{
        "model": "claude-3-sonnet-20240229",
        "messages": [
            {"role": "user", "content": "Count from 1 to 5 slowly."}
        ],
        "max_tokens": 100,
        "stream": true
    }'
echo
echo

# 测试多模态（如果支持）
echo "测试 Kiro 多模态支持..."
curl -s -X POST http://localhost:3000/v1/chat/completions \
    -H "Authorization: Bearer test-key-123" \
    -H "X-Model-Provider: kiro-api" \
    -H "Content-Type: application/json" \
    -d '{
        "model": "claude-3-sonnet-20240229",
        "messages": [
            {
                "role": "user",
                "content": [
                    {"type": "text", "text": "What is in this image?"},
                    {"type": "image_url", "image_url": {"url": "data:image/png;base64,iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVR42mNkYPhfDwAChwGA60e6kgAAAABJRU5ErkJggg=="}}
                ]
            }
        ],
        "max_tokens": 50
    }' | jq .
echo

# 清理
echo -e "${GREEN}测试完成！${NC}"
echo "停止服务器..."
kill $SERVER_PID 2>/dev/null
wait $SERVER_PID 2>/dev/null

echo "清理完成。"