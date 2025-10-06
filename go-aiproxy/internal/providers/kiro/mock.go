package kiro

import (
	"context"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/aiproxy/go-aiproxy/internal/providers"
	"github.com/aiproxy/go-aiproxy/pkg/models"
	"github.com/google/uuid"
)

// MockClient implements a mock Kiro provider for testing
type MockClient struct {
	providers.BaseProvider
}

// NewMockClient creates a new mock Kiro client
func NewMockClient(config *models.ProviderConfig) (*MockClient, error) {
	return &MockClient{
		BaseProvider: providers.BaseProvider{
			Config:   config,
			Protocol: models.ProtocolClaude,
		},
	}, nil
}

// GenerateContent implements mock content generation
func (c *MockClient) GenerateContent(ctx context.Context, model string, request interface{}) (interface{}, error) {
	claudeReq, ok := request.(*models.ClaudeRequest)
	if !ok {
		return nil, fmt.Errorf("invalid request type for Kiro mock provider")
	}

	// Simulate API delay
	time.Sleep(100 * time.Millisecond)

	// Generate mock response
	response := &models.ClaudeResponse{
		ID:   fmt.Sprintf("msg_%s", uuid.New().String()),
		Type: "message",
		Role: "assistant",
		Content: []models.ClaudeContent{
			{
				Type: "text",
				Text: c.generateMockResponse(claudeReq),
			},
		},
		Model:      model,
		StopReason: "end_turn",
		Usage: &models.ClaudeUsage{
			InputTokens:  10,
			OutputTokens: 20,
		},
	}

	return response, nil
}

// GenerateContentStream implements mock streaming
func (c *MockClient) GenerateContentStream(ctx context.Context, model string, request interface{}) (io.ReadCloser, error) {
	claudeReq, ok := request.(*models.ClaudeRequest)
	if !ok {
		return nil, fmt.Errorf("invalid request type for Kiro mock provider")
	}

	// Create mock stream
	response := c.generateMockResponse(claudeReq)
	words := strings.Split(response, " ")

	return &mockStreamReader{
		words:    words,
		position: 0,
		delay:    50 * time.Millisecond,
	}, nil
}

// ListModels returns mock models
func (c *MockClient) ListModels(ctx context.Context) (interface{}, error) {
	return &models.ModelList{
		Object: "list",
		Data: []models.ModelInfo{
			{
				ID:      "claude-3-opus-20240229",
				Object:  "model",
				Created: time.Now().Unix(),
				OwnedBy: "kiro-mock",
			},
			{
				ID:      "claude-3-sonnet-20240229",
				Object:  "model",
				Created: time.Now().Unix(),
				OwnedBy: "kiro-mock",
			},
			{
				ID:      "claude-3-haiku-20240307",
				Object:  "model",
				Created: time.Now().Unix(),
				OwnedBy: "kiro-mock",
			},
		},
	}, nil
}

// generateMockResponse generates a mock response based on the request
func (c *MockClient) generateMockResponse(req *models.ClaudeRequest) string {
	if len(req.Messages) == 0 {
		return "没有收到消息。"
	}

	lastMessage := req.Messages[len(req.Messages)-1]
	userContent := ""

	for _, content := range lastMessage.Content {
		if content.Type == "text" {
			userContent += content.Text
		}
	}

	// Generate context-aware responses
	lowerContent := strings.ToLower(userContent)

	switch {
	case strings.Contains(lowerContent, "你好") || strings.Contains(lowerContent, "hello"):
		return "你好！我是通过 Kiro API 提供的 Claude 助手。很高兴为您服务！"
	
	case strings.Contains(lowerContent, "介绍"):
		return "我是 Claude，一个由 Anthropic 开发的 AI 助手。通过 Kiro API，我可以帮助您处理各种任务。"
	
	case strings.Contains(lowerContent, "数到"):
		return "1\n2\n3\n4\n5"
	
	case strings.Contains(lowerContent, "weather"):
		return "I'd be happy to help you check the weather, but I would need to use the get_weather tool to provide accurate information. Based on your request for Tokyo, I can see you're interested in weather information for that location."
	
	default:
		return fmt.Sprintf("我收到了您的消息：'%s'。这是一个模拟响应，用于测试 Kiro API 集成。", userContent)
	}
}

// mockStreamReader implements a mock stream reader
type mockStreamReader struct {
	words    []string
	position int
	delay    time.Duration
	buffer   []byte
}

func (r *mockStreamReader) Read(p []byte) (n int, err error) {
	// Return buffered data first
	if len(r.buffer) > 0 {
		n = copy(p, r.buffer)
		r.buffer = r.buffer[n:]
		return n, nil
	}

	// Check if we've sent all words
	if r.position >= len(r.words) {
		return 0, io.EOF
	}

	// Simulate streaming delay
	time.Sleep(r.delay)

	// Get next word
	word := r.words[r.position]
	if r.position < len(r.words)-1 {
		word += " "
	}
	r.position++

	// Copy to buffer
	r.buffer = []byte(word)
	n = copy(p, r.buffer)
	r.buffer = r.buffer[n:]

	return n, nil
}

func (r *mockStreamReader) Close() error {
	return nil
}