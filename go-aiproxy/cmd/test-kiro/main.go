package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
)

// Kiro 测试程序
func main() {
	fmt.Println("=== Kiro Provider 测试程序 ===")
	fmt.Println()

	// 检查服务器是否运行
	serverURL := "http://localhost:3000"
	apiKey := "test-key-123"

	// 1. 健康检查
	fmt.Println("1. 健康检查...")
	resp, err := http.Get(serverURL + "/health")
	if err != nil {
		fmt.Printf("错误: 服务器未运行。请先启动服务器:\n")
		fmt.Printf("./aiproxy --model-provider kiro-api --api-key %s\n", apiKey)
		os.Exit(1)
	}
	resp.Body.Close()
	fmt.Println("✓ 服务器正在运行")
	fmt.Println()

	// 2. 列出模型
	fmt.Println("2. 列出 Kiro 支持的模型...")
	models := listModels(serverURL, apiKey)
	fmt.Printf("✓ 可用模型: %v\n", models)
	fmt.Println()

	// 3. 非流式聊天
	fmt.Println("3. 测试非流式聊天...")
	chatResponse := testChat(serverURL, apiKey, false)
	fmt.Printf("✓ AI 回复: %s\n", chatResponse)
	fmt.Println()

	// 4. 流式聊天
	fmt.Println("4. 测试流式聊天...")
	fmt.Print("✓ AI 流式回复: ")
	testStreamChat(serverURL, apiKey)
	fmt.Println("\n")

	// 5. 测试工具调用
	fmt.Println("5. 测试工具调用...")
	toolResponse := testToolCall(serverURL, apiKey)
	fmt.Printf("✓ 工具调用结果: %s\n", toolResponse)
	fmt.Println()

	fmt.Println("所有测试完成！")
}

// 列出模型
func listModels(serverURL, apiKey string) []string {
	req, _ := http.NewRequest("GET", serverURL+"/v1/models", nil)
	req.Header.Set("Authorization", "Bearer "+apiKey)
	req.Header.Set("X-Model-Provider", "kiro-api")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		fmt.Printf("列出模型失败: %v\n", err)
		return nil
	}
	defer resp.Body.Close()

	var result struct {
		Data []struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	json.NewDecoder(resp.Body).Decode(&result)

	models := []string{}
	for _, m := range result.Data {
		models = append(models, m.ID)
	}
	return models
}

// 测试聊天
func testChat(serverURL, apiKey string, stream bool) string {
	payload := map[string]interface{}{
		"model": "claude-3-sonnet-20240229",
		"messages": []map[string]string{
			{"role": "user", "content": "你好！请用一句话介绍你自己。"},
		},
		"max_tokens": 100,
		"stream":     stream,
	}

	jsonData, _ := json.Marshal(payload)
	req, _ := http.NewRequest("POST", serverURL+"/v1/chat/completions", bytes.NewReader(jsonData))
	req.Header.Set("Authorization", "Bearer "+apiKey)
	req.Header.Set("X-Model-Provider", "kiro-api")
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Sprintf("请求失败: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Sprintf("API 错误 (%d): %s", resp.StatusCode, string(body))
	}

	var result struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}
	json.NewDecoder(resp.Body).Decode(&result)

	if len(result.Choices) > 0 {
		return result.Choices[0].Message.Content
	}
	return "无响应"
}

// 测试流式聊天
func testStreamChat(serverURL, apiKey string) {
	payload := map[string]interface{}{
		"model": "claude-3-sonnet-20240229",
		"messages": []map[string]string{
			{"role": "user", "content": "请从1数到5，每个数字单独一行。"},
		},
		"max_tokens": 100,
		"stream":     true,
	}

	jsonData, _ := json.Marshal(payload)
	req, _ := http.NewRequest("POST", serverURL+"/v1/chat/completions", bytes.NewReader(jsonData))
	req.Header.Set("Authorization", "Bearer "+apiKey)
	req.Header.Set("X-Model-Provider", "kiro-api")
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		fmt.Printf("流式请求失败: %v", err)
		return
	}
	defer resp.Body.Close()

	reader := bufio.NewReader(resp.Body)
	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			break
		}

		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "data: ") {
			data := strings.TrimPrefix(line, "data: ")
			if data == "[DONE]" {
				break
			}

			var chunk struct {
				Choices []struct {
					Delta struct {
						Content string `json:"content"`
					} `json:"delta"`
				} `json:"choices"`
			}
			if err := json.Unmarshal([]byte(data), &chunk); err == nil {
				if len(chunk.Choices) > 0 && chunk.Choices[0].Delta.Content != "" {
					fmt.Print(chunk.Choices[0].Delta.Content)
				}
			}
		}
	}
}

// 测试工具调用
func testToolCall(serverURL, apiKey string) string {
	payload := map[string]interface{}{
		"model": "claude-3-sonnet-20240229",
		"messages": []map[string]string{
			{"role": "user", "content": "What's the weather in Tokyo?"},
		},
		"tools": []map[string]interface{}{
			{
				"type": "function",
				"function": map[string]interface{}{
					"name":        "get_weather",
					"description": "Get the current weather in a location",
					"parameters": map[string]interface{}{
						"type": "object",
						"properties": map[string]interface{}{
							"location": map[string]interface{}{
								"type":        "string",
								"description": "The city and state",
							},
						},
						"required": []string{"location"},
					},
				},
			},
		},
		"tool_choice": "auto",
		"max_tokens":  100,
	}

	jsonData, _ := json.Marshal(payload)
	req, _ := http.NewRequest("POST", serverURL+"/v1/chat/completions", bytes.NewReader(jsonData))
	req.Header.Set("Authorization", "Bearer "+apiKey)
	req.Header.Set("X-Model-Provider", "kiro-api")
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Sprintf("工具调用请求失败: %v", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	return string(body)
}