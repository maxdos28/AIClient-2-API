package convert

import (
	"testing"

	"github.com/aiproxy/go-aiproxy/pkg/models"
)

func TestConverter_ConvertRequest_SameProtocol(t *testing.T) {
	converter := NewConverter()

	req := &models.OpenAIRequest{
		Model: "gpt-3.5-turbo",
		Messages: []models.OpenAIMessage{
			{Role: "user", Content: "Hello"},
		},
	}

	result, err := converter.ConvertRequest(req, models.ProtocolOpenAI, models.ProtocolOpenAI)
	if err != nil {
		t.Fatalf("ConvertRequest failed: %v", err)
	}

	if result != req {
		t.Error("Expected same request when protocols match")
	}
}

func TestConverter_ConvertGeminiToClaude(t *testing.T) {
	converter := NewConverter()

	geminiReq := &models.GeminiRequest{
		Contents: []models.GeminiContent{
			{
				Role: "user",
				Parts: []models.GeminiPart{
					{Text: "Hello, how are you?"},
				},
			},
		},
		SystemInstruction: &models.GeminiSystemInstruction{
			Parts: []models.GeminiPart{
				{Text: "You are a helpful assistant."},
			},
		},
		GenerationConfig: &models.GeminiGenerationConfig{
			Temperature:     0.7,
			TopP:            0.9,
			MaxOutputTokens: 1024,
		},
	}

	result, err := converter.ConvertRequest(geminiReq, models.ProtocolGemini, models.ProtocolClaude)
	if err != nil {
		t.Fatalf("ConvertRequest failed: %v", err)
	}

	claudeReq, ok := result.(*models.ClaudeRequest)
	if !ok {
		t.Fatalf("Expected *models.ClaudeRequest, got %T", result)
	}

	if claudeReq.System != "You are a helpful assistant." {
		t.Errorf("System instruction not converted correctly: %s", claudeReq.System)
	}

	if len(claudeReq.Messages) != 1 {
		t.Errorf("Expected 1 message, got %d", len(claudeReq.Messages))
	}

	if claudeReq.Temperature != 0.7 {
		t.Errorf("Temperature not converted correctly: %f", claudeReq.Temperature)
	}
}

func TestConverter_ConvertClaudeToGemini_WithTools(t *testing.T) {
	converter := NewConverter()

	claudeReq := &models.ClaudeRequest{
		Model:  "claude-3-opus",
		System: "You are a helpful assistant.",
		Messages: []models.ClaudeMessage{
			{
				Role: "user",
				Content: []models.ClaudeContent{
					{Type: "text", Text: "What's the weather?"},
				},
			},
		},
		MaxTokens:   1024,
		Temperature: 0.7,
		Tools: []models.ClaudeTool{
			{
				Name:        "get_weather",
				Description: "Get weather information",
				InputSchema: map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"location": map[string]interface{}{
							"type": "string",
						},
					},
				},
			},
		},
	}

	result, err := converter.ConvertRequest(claudeReq, models.ProtocolClaude, models.ProtocolGemini)
	if err != nil {
		t.Fatalf("ConvertRequest failed: %v", err)
	}

	geminiReq, ok := result.(*models.GeminiRequest)
	if !ok {
		t.Fatalf("Expected *models.GeminiRequest, got %T", result)
	}

	if len(geminiReq.Tools) != 1 {
		t.Errorf("Expected 1 tool, got %d", len(geminiReq.Tools))
	}

	if geminiReq.Tools[0].FunctionDeclarations[0].Name != "get_weather" {
		t.Errorf("Tool name not converted correctly")
	}
}

func TestConverter_ConvertResponse_OpenAIToClaude(t *testing.T) {
	converter := NewConverter()

	openaiResp := &models.OpenAIResponse{
		ID:      "chatcmpl-123",
		Model:   "gpt-3.5-turbo",
		Created: 1234567890,
		Choices: []models.OpenAIChoice{
			{
				Index: 0,
				Message: &models.OpenAIMessage{
					Role:    "assistant",
					Content: "I'm doing well, thank you!",
				},
				FinishReason: "stop",
			},
		},
		Usage: &models.Usage{
			PromptTokens:     10,
			CompletionTokens: 15,
			TotalTokens:      25,
		},
	}

	result, err := converter.ConvertResponse(openaiResp, models.ProtocolOpenAI, models.ProtocolClaude, "claude-3-opus")
	if err != nil {
		t.Fatalf("ConvertResponse failed: %v", err)
	}

	claudeResp, ok := result.(*models.ClaudeResponse)
	if !ok {
		t.Fatalf("Expected *models.ClaudeResponse, got %T", result)
	}

	if len(claudeResp.Content) == 0 {
		t.Error("Expected content in response")
	}

	if claudeResp.Content[0].Text != "I'm doing well, thank you!" {
		t.Errorf("Content not converted correctly: %s", claudeResp.Content[0].Text)
	}

	if claudeResp.Usage.InputTokens != 10 {
		t.Errorf("Input tokens not converted correctly: %d", claudeResp.Usage.InputTokens)
	}
}

func TestConverter_ConvertResponse_GeminiToClaude(t *testing.T) {
	converter := NewConverter()

	geminiResp := &models.GeminiResponse{
		Candidates: []models.GeminiCandidate{
			{
				Content: models.GeminiContent{
					Role: "model",
					Parts: []models.GeminiPart{
						{Text: "Hello! I'm here to help."},
					},
				},
				FinishReason: "STOP",
			},
		},
		UsageMetadata: &models.GeminiUsage{
			PromptTokenCount:     10,
			CandidatesTokenCount: 15,
			TotalTokenCount:      25,
		},
	}

	result, err := converter.ConvertResponse(geminiResp, models.ProtocolGemini, models.ProtocolClaude, "claude-3-opus")
	if err != nil {
		t.Fatalf("ConvertResponse failed: %v", err)
	}

	claudeResp, ok := result.(*models.ClaudeResponse)
	if !ok {
		t.Fatalf("Expected *models.ClaudeResponse, got %T", result)
	}

	if len(claudeResp.Content) == 0 {
		t.Error("Expected content in response")
	}

	if claudeResp.StopReason != "end_turn" {
		t.Errorf("Stop reason not converted correctly: %s", claudeResp.StopReason)
	}
}

func TestConverter_ConvertStreamChunk(t *testing.T) {
	converter := NewConverter()

	// Test with a simple text chunk
	chunk := "Hello, world!"

	result, err := converter.ConvertStreamChunk(chunk, models.ProtocolGemini, models.ProtocolOpenAI, "gpt-3.5-turbo")
	if err != nil {
		t.Fatalf("ConvertStreamChunk failed: %v", err)
	}

	streamChunk, ok := result.(*models.StreamChunk)
	if !ok {
		t.Fatalf("Expected *models.StreamChunk, got %T", result)
	}

	if streamChunk.Model != "gpt-3.5-turbo" {
		t.Errorf("Model not set correctly: %s", streamChunk.Model)
	}

	if len(streamChunk.Choices) == 0 {
		t.Error("Expected choices in stream chunk")
	}

	// Check that content was set
	if streamChunk.Choices[0].Delta == nil {
		t.Error("Expected delta in stream chunk")
	}
}

func TestConverter_ConvertModelList(t *testing.T) {
	converter := NewConverter()

	modelList := &models.ModelList{
		Object: "list",
		Data: []models.ModelInfo{
			{
				ID:      "gpt-3.5-turbo",
				Object:  "model",
				Created: 1234567890,
				OwnedBy: "openai",
			},
		},
	}

	result, err := converter.ConvertModelList(modelList, models.ProtocolOpenAI, models.ProtocolOpenAI)
	if err != nil {
		t.Fatalf("ConvertModelList failed: %v", err)
	}

	resultList, ok := result.(*models.ModelList)
	if !ok {
		t.Fatalf("Expected *models.ModelList, got %T", result)
	}

	if len(resultList.Data) != 1 {
		t.Errorf("Expected 1 model, got %d", len(resultList.Data))
	}
}

func BenchmarkConverter_ConvertRequest_OpenAIToClaude(b *testing.B) {
	converter := NewConverter()

	req := &models.OpenAIRequest{
		Model: "gpt-3.5-turbo",
		Messages: []models.OpenAIMessage{
			{Role: "system", Content: "You are helpful."},
			{Role: "user", Content: "Hello!"},
		},
		MaxTokens:   100,
		Temperature: 0.7,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = converter.ConvertRequest(req, models.ProtocolOpenAI, models.ProtocolClaude)
	}
}

func BenchmarkConverter_ConvertRequest_ClaudeToGemini(b *testing.B) {
	converter := NewConverter()

	req := &models.ClaudeRequest{
		Model:  "claude-3-opus",
		System: "You are helpful.",
		Messages: []models.ClaudeMessage{
			{
				Role: "user",
				Content: []models.ClaudeContent{
					{Type: "text", Text: "Hello!"},
				},
			},
		},
		MaxTokens:   100,
		Temperature: 0.7,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = converter.ConvertRequest(req, models.ProtocolClaude, models.ProtocolGemini)
	}
}

func BenchmarkConverter_ConvertStreamChunk(b *testing.B) {
	converter := NewConverter()
	chunk := "Test chunk"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = converter.ConvertStreamChunk(chunk, models.ProtocolOpenAI, models.ProtocolOpenAI, "gpt-3.5-turbo")
	}
}
