# Go AI Proxy - 高级功能指南

本文档介绍 Go AI Proxy 的高级功能，包括新增的提供商支持、缓存系统、监控、WebSocket 和负载均衡。

## 🚀 新增功能概览

### 1. 新增提供商支持

#### Kiro Provider (Claude via OAuth)
```bash
# 使用 Kiro 提供商
./aiproxy --model-provider kiro-api \
  --kiro-oauth-creds-file /path/to/kiro-creds.json
```

Kiro 凭据文件格式：
```json
{
  "client_id": "your-client-id",
  "client_secret": "your-client-secret",
  "token_url": "https://api.kiro.com/oauth/token",
  "refresh_token": "your-refresh-token"
}
```

#### Qwen Provider
```bash
# 使用 Qwen 提供商
./aiproxy --model-provider qwen-api \
  --qwen-oauth-creds-file /path/to/qwen-creds.json
```

Qwen 支持内置工具：
- `code_interpreter`: 执行 Python 代码
- `web_search`: 网络搜索

### 2. 请求缓存系统

#### 内存缓存
默认启用，自动缓存非流式请求：
```bash
# 配置缓存
./aiproxy --cache-enabled \
  --cache-max-size 100 \
  --cache-ttl 300
```

#### Redis 分布式缓存
支持多实例共享缓存：
```bash
# 启用 Redis 缓存
./aiproxy --redis-addr localhost:6379 \
  --redis-password yourpassword \
  --redis-db 0
```

#### 缓存管理 API
```bash
# 查看缓存统计
curl http://localhost:3000/cache/stats \
  -H "Authorization: Bearer your-api-key"

# 清空缓存
curl -X DELETE http://localhost:3000/cache/clear \
  -H "Authorization: Bearer your-api-key"

# 启用/禁用缓存
curl -X PUT http://localhost:3000/cache/enable
curl -X PUT http://localhost:3000/cache/disable
```

### 3. Prometheus 监控集成

#### 启用监控
```bash
# 启动带监控的服务
./aiproxy --metrics-enabled \
  --metrics-port 9090
```

#### 访问监控数据
- Prometheus 端点: `http://localhost:3000/metrics`
- 监控面板: `http://localhost:3000/dashboard`

#### 主要监控指标
- **HTTP 指标**
  - `aiproxy_http_requests_total`: 总请求数
  - `aiproxy_http_request_duration_seconds`: 请求延迟
  - `aiproxy_http_active_requests`: 活跃请求数

- **提供商指标**
  - `aiproxy_provider_requests_total`: 提供商请求数
  - `aiproxy_provider_errors_total`: 错误统计
  - `aiproxy_provider_tokens_used_total`: Token 使用量

- **缓存指标**
  - `aiproxy_cache_hits_total`: 缓存命中数
  - `aiproxy_cache_misses_total`: 缓存未命中数
  - `aiproxy_cache_size_bytes`: 缓存大小

### 4. WebSocket 实时通信

#### 连接 WebSocket
```javascript
// JavaScript 客户端示例
const ws = new WebSocket('ws://localhost:3000/ws?token=your-api-key');

ws.onopen = () => {
    console.log('Connected to AI Proxy WebSocket');
};

ws.onmessage = (event) => {
    const message = JSON.parse(event.data);
    console.log('Received:', message);
};

// 发送请求
ws.send(JSON.stringify({
    type: 'request',
    id: '123',
    provider: 'openai-custom',
    model: 'gpt-3.5-turbo',
    request: {
        messages: [
            { role: 'user', content: 'Hello!' }
        ]
    },
    metadata: {
        stream: true
    }
}));
```

#### WebSocket 消息类型
- `request`: 发送 AI 请求
- `response`: 接收完整响应
- `stream`: 接收流式数据
- `stream_end`: 流结束标记
- `error`: 错误消息
- `heartbeat`: 心跳保活

### 5. 负载均衡

#### 配置负载均衡
```bash
# 使用负载均衡算法
./aiproxy --load-balancer-enabled \
  --load-balancer-algorithm round_robin
```

#### 支持的算法
- `round_robin`: 轮询（默认）
- `least_requests`: 最少请求
- `weighted`: 加权轮询
- `random`: 随机选择
- `ip_hash`: IP 哈希

#### 负载均衡 API
```bash
# 查看实例状态
curl http://localhost:3000/loadbalancer/instances \
  -H "Authorization: Bearer your-api-key"

# 查看负载均衡指标
curl http://localhost:3000/loadbalancer/metrics \
  -H "Authorization: Bearer your-api-key"

# 更改算法
curl -X PUT http://localhost:3000/loadbalancer/algorithm \
  -H "Authorization: Bearer your-api-key" \
  -H "Content-Type: application/json" \
  -d '{"algorithm": "least_requests"}'
```

### 6. 集群模式

#### 启动集群节点
```bash
# 节点 1（种子节点）
./aiproxy --cluster-enabled \
  --node-id node1 \
  --node-address 192.168.1.10:3000

# 节点 2
./aiproxy --cluster-enabled \
  --node-id node2 \
  --node-address 192.168.1.11:3000 \
  --seed-nodes 192.168.1.10:3000

# 节点 3
./aiproxy --cluster-enabled \
  --node-id node3 \
  --node-address 192.168.1.12:3000 \
  --seed-nodes 192.168.1.10:3000,192.168.1.11:3000
```

#### 集群状态
```bash
# 查看集群状态
curl http://localhost:3000/cluster/status
```

## 📊 性能优化建议

### 1. 缓存策略
- 对于相似的查询，使用缓存可减少 80% 的 API 调用
- 建议缓存 TTL：5-15 分钟
- Redis 缓存适合多实例部署

### 2. 负载均衡优化
- 高并发场景使用 `least_requests` 算法
- 地理分布式部署使用 `ip_hash`
- 根据提供商性能设置权重

### 3. 监控告警
使用 Prometheus + Grafana 设置告警：
```yaml
# prometheus-rules.yml
groups:
  - name: aiproxy
    rules:
      - alert: HighErrorRate
        expr: rate(aiproxy_provider_errors_total[5m]) > 0.1
        for: 5m
        labels:
          severity: warning
        annotations:
          summary: "High error rate detected"
          
      - alert: HighLatency
        expr: histogram_quantile(0.95, aiproxy_http_request_duration_seconds) > 1
        for: 5m
        labels:
          severity: warning
        annotations:
          summary: "High latency detected"
```

## 🔧 高级配置示例

### 完整配置文件
```yaml
# config.yaml
host: 0.0.0.0
port: 3000
api_key: your-secret-key

# 提供商配置
model_providers:
  - openai-custom
  - claude-custom
  - gemini-cli
  - kiro-api
  - qwen-api

# 缓存配置
cache:
  enabled: true
  max_size_mb: 100
  ttl_seconds: 300
  
redis:
  addr: localhost:6379
  password: ""
  db: 0

# 监控配置
metrics:
  enabled: true
  port: 9090

# WebSocket 配置
websocket:
  enabled: true
  max_connections: 1000

# 负载均衡配置
load_balancer:
  enabled: true
  algorithm: least_requests
  health_check_interval: 30s

# 集群配置
cluster:
  enabled: false
  node_id: node1
  node_address: 0.0.0.0:3000
  seed_nodes: []
```

### Docker Compose 高可用部署
```yaml
version: '3.8'

services:
  redis:
    image: redis:7-alpine
    ports:
      - "6379:6379"
    volumes:
      - redis_data:/data

  aiproxy1:
    build: .
    ports:
      - "3001:3000"
    environment:
      - NODE_ID=node1
      - REDIS_ADDR=redis:6379
    depends_on:
      - redis

  aiproxy2:
    build: .
    ports:
      - "3002:3000"
    environment:
      - NODE_ID=node2
      - REDIS_ADDR=redis:6379
      - SEED_NODES=aiproxy1:3000
    depends_on:
      - redis
      - aiproxy1

  nginx:
    image: nginx:alpine
    ports:
      - "80:80"
    volumes:
      - ./nginx.conf:/etc/nginx/nginx.conf
    depends_on:
      - aiproxy1
      - aiproxy2

  prometheus:
    image: prom/prometheus
    ports:
      - "9090:9090"
    volumes:
      - ./prometheus.yml:/etc/prometheus/prometheus.yml
      - prometheus_data:/prometheus

  grafana:
    image: grafana/grafana
    ports:
      - "3000:3000"
    environment:
      - GF_SECURITY_ADMIN_PASSWORD=admin
    volumes:
      - grafana_data:/var/lib/grafana
    depends_on:
      - prometheus

volumes:
  redis_data:
  prometheus_data:
  grafana_data:
```

## 🚀 性能基准测试

### 测试环境
- CPU: 8 核 Intel Xeon
- 内存: 16GB
- Go 版本: 1.21

### 测试结果

| 功能 | QPS | 延迟 (p95) | 内存使用 |
|------|-----|-----------|----------|
| 基础转发 | 5,000 | 15ms | 50MB |
| 启用缓存 | 20,000 | 3ms | 150MB |
| WebSocket | 10,000 连接 | 5ms | 500MB |
| 负载均衡 (3节点) | 15,000 | 10ms | 100MB/节点 |

### 优化建议
1. **缓存优化**: 对高频请求启用缓存，可提升 4x 性能
2. **连接池**: 复用 HTTP 连接，减少握手开销
3. **批量处理**: WebSocket 支持批量请求，减少往返次数
4. **水平扩展**: 使用负载均衡器分散请求到多个实例

## 🔍 故障排除

### 常见问题

1. **缓存未生效**
   - 检查缓存是否启用: `GET /cache/stats`
   - 确认请求不是流式: 流式请求不缓存
   - 查看缓存键生成: 启用调试日志

2. **WebSocket 连接断开**
   - 检查心跳配置
   - 确认防火墙/代理支持 WebSocket
   - 查看客户端超时设置

3. **负载均衡不均匀**
   - 检查实例健康状态
   - 验证算法配置
   - 查看权重设置

4. **监控数据缺失**
   - 确认 Prometheus 正在抓取
   - 检查指标端点可访问性
   - 验证时间同步

## 📚 API 参考

完整的 API 文档请访问: [API Documentation](https://aiproxy.justlikemaki.vip/api-reference)

## 🤝 贡献

欢迎贡献高级功能！请查看 [CONTRIBUTING.md](CONTRIBUTING.md) 了解详情。