# Kiro Provider 使用指南

## 概述

Kiro 是一个提供 Claude API 访问的第三方服务商。通过 Go AI Proxy，您可以使用 Kiro 的 OAuth 认证来访问 Claude 模型。

## 配置方式

### 1. OAuth 凭据配置

Kiro 使用 OAuth 2.0 进行认证。您需要准备以下凭据：

```json
{
  "client_id": "your-kiro-client-id",
  "client_secret": "your-kiro-client-secret",
  "token_url": "https://api.kiro.com/oauth/token",
  "refresh_token": "your-refresh-token",
  "scope": "api.access"
}
```

### 2. 启动方式

有三种方式配置 Kiro 凭据：

#### 方式一：使用凭据文件
```bash
# 保存凭据到文件
echo '{
  "client_id": "your-client-id",
  "client_secret": "your-secret",
  "token_url": "https://api.kiro.com/oauth/token",
  "refresh_token": "your-refresh-token"
}' > kiro-creds.json

# 启动服务器
./aiproxy \
  --model-provider kiro-api \
  --kiro-oauth-creds-file kiro-creds.json \
  --api-key your-api-key
```

#### 方式二：使用 Base64 编码的凭据
```bash
# 将凭据编码为 Base64
CREDS=$(echo '{...}' | base64)

# 启动服务器
./aiproxy \
  --model-provider kiro-api \
  --kiro-oauth-creds-base64 "$CREDS" \
  --api-key your-api-key
```

#### 方式三：使用环境变量
```bash
export KIRO_OAUTH_CREDS_FILE=/path/to/kiro-creds.json
# 或
export KIRO_OAUTH_CREDS_BASE64=<base64-encoded-creds>

./aiproxy --model-provider kiro-api
```

## 测试模式

如果您没有真实的 Kiro 凭据，可以使用模拟模式进行测试：

```bash
# 启用模拟模式
export KIRO_MOCK_MODE=true

# 启动服务器
./aiproxy --model-provider kiro-api --api-key test-key
```

## API 使用示例

### 1. 列出可用模型
```bash
curl http://localhost:3000/v1/models \
  -H "Authorization: Bearer your-api-key" \
  -H "X-Model-Provider: kiro-api"
```

响应：
```json
{
  "object": "list",
  "data": [
    {"id": "claude-3-opus-20240229", "object": "model", "owned_by": "kiro"},
    {"id": "claude-3-sonnet-20240229", "object": "model", "owned_by": "kiro"},
    {"id": "claude-3-haiku-20240307", "object": "model", "owned_by": "kiro"}
  ]
}
```

### 2. 发送聊天请求
```bash
curl -X POST http://localhost:3000/v1/chat/completions \
  -H "Authorization: Bearer your-api-key" \
  -H "X-Model-Provider: kiro-api" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "claude-3-sonnet-20240229",
    "messages": [
      {"role": "user", "content": "Hello, Claude!"}
    ],
    "max_tokens": 100
  }'
```

### 3. 流式响应
```bash
curl -X POST http://localhost:3000/v1/chat/completions \
  -H "Authorization: Bearer your-api-key" \
  -H "X-Model-Provider: kiro-api" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "claude-3-sonnet-20240229",
    "messages": [
      {"role": "user", "content": "Count from 1 to 5"}
    ],
    "stream": true
  }'
```

### 4. 使用 Python 客户端
```python
import openai

# 配置客户端
openai.api_base = "http://localhost:3000/v1"
openai.api_key = "your-api-key"

# 发送请求
response = openai.ChatCompletion.create(
    model="claude-3-sonnet-20240229",
    messages=[
        {"role": "user", "content": "Hello!"}
    ],
    headers={
        "X-Model-Provider": "kiro-api"
    }
)

print(response.choices[0].message.content)
```

## 特性支持

Kiro Provider 支持以下特性：

- ✅ 所有 Claude 3 模型（Opus、Sonnet、Haiku）
- ✅ 流式和非流式响应
- ✅ 多轮对话
- ✅ System prompts
- ✅ 自动令牌刷新
- ✅ 错误重试机制
- ✅ 健康检查

## 故障排除

### 1. 认证失败
- 检查凭据是否正确
- 确认 refresh_token 是否有效
- 查看服务器日志获取详细错误信息

### 2. 请求超时
- 检查网络连接
- 增加超时时间：`--request-timeout 60`

### 3. 模型不可用
- 确认使用的模型名称正确
- 检查 Kiro 服务状态

## 性能优化

1. **启用缓存**：减少重复请求
   ```bash
   ./aiproxy --model-provider kiro-api --cache-enabled
   ```

2. **使用连接池**：提高并发性能
   ```bash
   ./aiproxy --model-provider kiro-api --max-connections 100
   ```

3. **负载均衡**：多个 Kiro 账号轮询
   ```bash
   # 在 provider_pools.json 中配置多个 Kiro 实例
   ```

## 监控

查看 Kiro Provider 的运行状态：

```bash
# Prometheus 指标
curl http://localhost:3000/metrics | grep kiro

# 健康检查
curl http://localhost:3000/health

# 查看活跃连接
curl http://localhost:3000/loadbalancer/instances | jq '.[] | select(.provider == "kiro-api")'
```

## 注意事项

1. Kiro 的 API 速率限制可能与官方 Claude API 不同
2. 某些高级功能可能需要特定的 Kiro 订阅计划
3. 建议在生产环境中启用错误重试和健康检查
4. 定期检查令牌过期时间，确保自动刷新正常工作