package tests

import (
	"testing"

	"github.com/aiproxy/go-aiproxy/internal/convert"
	"github.com/aiproxy/go-aiproxy/pkg/models"
)

func TestOpenAIToClaudeConversion(t *testing.T) {
	converter := convert.NewConverter()

	// Test OpenAI request
	openaiReq := &models.OpenAIRequest{
		Model: "gpt-3.5-turbo",
		Messages: []models.OpenAIMessage{
			{
				Role:    "system",
				Content: "You are a helpful assistant.",
			},
			{
				Role:    "user",
				Content: "Hello, how are you?",
			},
		},
		MaxTokens:   100,
		Temperature: 0.7,
	}

	// Convert to Claude
	result, err := converter.ConvertRequest(openaiReq, models.ProtocolOpenAI, models.ProtocolClaude)
	if err != nil {
		t.Fatalf("Conversion failed: %v", err)
	}

	claudeReq, ok := result.(*models.ClaudeRequest)
	if !ok {
		t.Fatalf("Expected ClaudeRequest, got %T", result)
	}

	// Verify conversion
	if claudeReq.System != "You are a helpful assistant." {
		t.Errorf("System message not converted correctly: %s", claudeReq.System)
	}

	if len(claudeReq.Messages) != 1 {
		t.Errorf("Expected 1 message, got %d", len(claudeReq.Messages))
	}

	if claudeReq.Messages[0].Role != "user" {
		t.Errorf("Expected user role, got %s", claudeReq.Messages[0].Role)
	}

	if claudeReq.MaxTokens != 100 {
		t.Errorf("Expected MaxTokens 100, got %d", claudeReq.MaxTokens)
	}
}

func TestClaudeToGeminiConversion(t *testing.T) {
	converter := convert.NewConverter()

	// Test Claude request
	claudeReq := &models.ClaudeRequest{
		Model:  "claude-3-opus-20240229",
		System: "You are a helpful assistant.",
		Messages: []models.ClaudeMessage{
			{
				Role: "user",
				Content: []models.ClaudeContent{
					{
						Type: "text",
						Text: "What is 2+2?",
					},
				},
			},
		},
		MaxTokens: 100,
	}

	// Convert to Gemini
	result, err := converter.ConvertRequest(claudeReq, models.ProtocolClaude, models.ProtocolGemini)
	if err != nil {
		t.Fatalf("Conversion failed: %v", err)
	}

	geminiReq, ok := result.(*models.GeminiRequest)
	if !ok {
		t.Fatalf("Expected GeminiRequest, got %T", result)
	}

	// Verify conversion
	if geminiReq.SystemInstruction == nil {
		t.Error("System instruction not converted")
	}

	if len(geminiReq.Contents) != 1 {
		t.Errorf("Expected 1 content, got %d", len(geminiReq.Contents))
	}

	if geminiReq.GenerationConfig.MaxOutputTokens != 100 {
		t.Errorf("Expected MaxOutputTokens 100, got %d", geminiReq.GenerationConfig.MaxOutputTokens)
	}
}

func TestStreamChunkConversion(t *testing.T) {
	converter := convert.NewConverter()

	// Test stream chunk
	chunk := "Hello, world!"

	result, err := converter.ConvertStreamChunk(chunk, models.ProtocolGemini, models.ProtocolOpenAI, "test-model")
	if err != nil {
		t.Fatalf("Stream chunk conversion failed: %v", err)
	}

	streamChunk, ok := result.(*models.StreamChunk)
	if !ok {
		t.Fatalf("Expected StreamChunk, got %T", result)
	}

	// Verify chunk
	if streamChunk.Model != "test-model" {
		t.Errorf("Expected model test-model, got %s", streamChunk.Model)
	}

	if len(streamChunk.Choices) != 1 {
		t.Errorf("Expected 1 choice, got %d", len(streamChunk.Choices))
	}

	content, ok := streamChunk.Choices[0].Delta.Content.(string)
	if !ok || content != chunk {
		t.Errorf("Expected content %s, got %v", chunk, streamChunk.Choices[0].Delta.Content)
	}
}

func BenchmarkOpenAIToClaudeConversion(b *testing.B) {
	converter := convert.NewConverter()

	openaiReq := &models.OpenAIRequest{
		Model: "gpt-3.5-turbo",
		Messages: []models.OpenAIMessage{
			{Role: "system", Content: "You are a helpful assistant."},
			{Role: "user", Content: "Hello!"},
			{Role: "assistant", Content: "Hi there! How can I help you today?"},
			{Role: "user", Content: "What's the weather like?"},
		},
		MaxTokens: 100,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := converter.ConvertRequest(openaiReq, models.ProtocolOpenAI, models.ProtocolClaude)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func TestMultimodalContentConversion(t *testing.T) {
	converter := convert.NewConverter()

	// Test OpenAI multimodal request
	openaiReq := &models.OpenAIRequest{
		Model: "gpt-4-vision-preview",
		Messages: []models.OpenAIMessage{
			{
				Role: "user",
				Content: []models.ContentPart{
					{
						Type: "text",
						Text: "What's in this image?",
					},
					{
						Type: "image_url",
						ImageURL: &models.ImageURL{
							URL: "data:image/jpeg;base64,/9j/4AAQ...",
						},
					},
				},
			},
		},
	}

	// Convert to Claude
	result, err := converter.ConvertRequest(openaiReq, models.ProtocolOpenAI, models.ProtocolClaude)
	if err != nil {
		t.Fatalf("Multimodal conversion failed: %v", err)
	}

	claudeReq, ok := result.(*models.ClaudeRequest)
	if !ok {
		t.Fatalf("Expected ClaudeRequest, got %T", result)
	}

	// Verify multimodal content
	if len(claudeReq.Messages) != 1 {
		t.Errorf("Expected 1 message, got %d", len(claudeReq.Messages))
	}

	if len(claudeReq.Messages[0].Content) != 2 {
		t.Errorf("Expected 2 content parts, got %d", len(claudeReq.Messages[0].Content))
	}

	// Check text part
	if claudeReq.Messages[0].Content[0].Type != "text" {
		t.Errorf("Expected text type, got %s", claudeReq.Messages[0].Content[0].Type)
	}

	// Check image part
	if claudeReq.Messages[0].Content[1].Type != "image" {
		t.Errorf("Expected image type, got %s", claudeReq.Messages[0].Content[1].Type)
	}
}

func TestToolCallConversion(t *testing.T) {
	converter := convert.NewConverter()

	// Test OpenAI request with tools
	openaiReq := &models.OpenAIRequest{
		Model: "gpt-3.5-turbo",
		Messages: []models.OpenAIMessage{
			{
				Role:    "user",
				Content: "What's the weather in Tokyo?",
			},
		},
		Tools: []models.Tool{
			{
				Type: "function",
				Function: models.ToolFunction{
					Name:        "get_weather",
					Description: "Get the current weather",
					Parameters: map[string]interface{}{
						"type": "object",
						"properties": map[string]interface{}{
							"location": map[string]interface{}{
								"type":        "string",
								"description": "The city name",
							},
						},
						"required": []string{"location"},
					},
				},
			},
		},
	}

	// Convert to Claude
	result, err := converter.ConvertRequest(openaiReq, models.ProtocolOpenAI, models.ProtocolClaude)
	if err != nil {
		t.Fatalf("Tool conversion failed: %v", err)
	}

	claudeReq, ok := result.(*models.ClaudeRequest)
	if !ok {
		t.Fatalf("Expected ClaudeRequest, got %T", result)
	}

	// Verify tools
	if len(claudeReq.Tools) != 1 {
		t.Errorf("Expected 1 tool, got %d", len(claudeReq.Tools))
	}

	if claudeReq.Tools[0].Name != "get_weather" {
		t.Errorf("Expected tool name get_weather, got %s", claudeReq.Tools[0].Name)
	}
}
