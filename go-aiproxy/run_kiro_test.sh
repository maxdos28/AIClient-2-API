#!/bin/bash

echo "=== 启动 Kiro Provider 测试 ==="
echo

# 设置环境变量
export KIRO_MOCK_MODE=true
export AIPROXY_API_KEY=test-key-123
export AIPROXY_HOST=0.0.0.0
export AIPROXY_PORT=3000

# 创建测试用的 main.go
cat > /workspace/go-aiproxy/cmd/test-server/main.go << 'EOF'
package main

import (
    "fmt"
    "log"
    "os"
    "github.com/gin-gonic/gin"
    "github.com/aiproxy/go-aiproxy/pkg/models"
)

func main() {
    gin.SetMode(gin.ReleaseMode)
    
    fmt.Println("=== Kiro Provider 模拟测试服务器 ===")
    fmt.Println()
    fmt.Println("配置:")
    fmt.Println("- 服务地址: 0.0.0.0:3000")
    fmt.Println("- API Key: test-key-123")
    fmt.Println("- 模式: 模拟模式 (KIRO_MOCK_MODE=true)")
    fmt.Println()
    fmt.Println("支持的模型:")
    fmt.Println("- claude-3-opus-20240229")
    fmt.Println("- claude-3-sonnet-20240229")
    fmt.Println("- claude-3-haiku-20240307")
    fmt.Println()
    
    router := gin.New()
    router.Use(gin.Recovery())
    
    // 健康检查
    router.GET("/health", func(c *gin.Context) {
        c.JSON(200, gin.H{
            "status": "healthy",
            "provider": "kiro-mock",
            "mode": "simulation",
        })
    })
    
    // 模型列表
    router.GET("/v1/models", func(c *gin.Context) {
        provider := c.GetHeader("X-Model-Provider")
        if provider != "kiro-api" {
            provider = "kiro-api"
        }
        
        c.JSON(200, gin.H{
            "object": "list",
            "data": []gin.H{
                {"id": "claude-3-opus-20240229", "object": "model", "owned_by": "kiro-mock"},
                {"id": "claude-3-sonnet-20240229", "object": "model", "owned_by": "kiro-mock"},
                {"id": "claude-3-haiku-20240307", "object": "model", "owned_by": "kiro-mock"},
            },
        })
    })
    
    // 聊天完成
    router.POST("/v1/chat/completions", func(c *gin.Context) {
        var req models.OpenAIRequest
        if err := c.ShouldBindJSON(&req); err != nil {
            c.JSON(400, gin.H{"error": err.Error()})
            return
        }
        
        // 简单的模拟响应
        content := "这是 Kiro 模拟响应。"
        if len(req.Messages) > 0 {
            lastMsg := req.Messages[len(req.Messages)-1]
            if msg, ok := lastMsg.Content.(string); ok {
                content = fmt.Sprintf("您说: '%s'。我是 Kiro 模拟 Claude 助手。", msg)
            }
        }
        
        if req.Stream {
            // 流式响应
            c.Header("Content-Type", "text/event-stream")
            c.Header("Cache-Control", "no-cache")
            c.Header("Connection", "keep-alive")
            
            words := []string{"这", "是", "Kiro", "流", "式", "响", "应", "。"}
            for _, word := range words {
                c.SSEvent("", gin.H{
                    "choices": []gin.H{
                        {
                            "delta": gin.H{"content": word},
                            "index": 0,
                        },
                    },
                })
                c.Writer.Flush()
            }
            c.SSEvent("", "[DONE]")
            c.Writer.Flush()
        } else {
            // 非流式响应
            c.JSON(200, gin.H{
                "id": "chatcmpl-123",
                "object": "chat.completion",
                "model": req.Model,
                "choices": []gin.H{
                    {
                        "message": gin.H{
                            "role": "assistant",
                            "content": content,
                        },
                        "finish_reason": "stop",
                        "index": 0,
                    },
                },
                "usage": gin.H{
                    "prompt_tokens": 10,
                    "completion_tokens": 20,
                    "total_tokens": 30,
                },
            })
        }
    })
    
    fmt.Println("服务器启动中...")
    if err := router.Run(":3000"); err != nil {
        log.Fatal("服务器启动失败:", err)
    }
}
EOF

# 编译并运行
echo "编译测试服务器..."
cd /workspace/go-aiproxy
go build -o test-server cmd/test-server/main.go

if [ $? -eq 0 ]; then
    echo "启动服务器..."
    ./test-server
else
    echo "编译失败！"
    exit 1
fi