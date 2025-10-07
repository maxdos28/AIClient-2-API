package convert

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/aiproxy/go-aiproxy/pkg/models"
	"github.com/google/uuid"
)

// Default constants
const (
	DefaultMaxTokens       = 8192
	DefaultGeminiMaxTokens = 65536
	DefaultTemperature     = 1.0
	DefaultTopP            = 0.9
)

// Converter interface defines the conversion methods
type Converter interface {
	ConvertRequest(data interface{}, fromProvider, toProvider models.ProtocolPrefix) (interface{}, error)
	ConvertResponse(data interface{}, fromProvider, toProvider models.ProtocolPrefix, model string) (interface{}, error)
	ConvertStreamChunk(data interface{}, fromProvider, toProvider models.ProtocolPrefix, model string) (interface{}, error)
	ConvertModelList(data interface{}, fromProvider, toProvider models.ProtocolPrefix) (interface{}, error)
}

// DefaultConverter implements the Converter interface
type DefaultConverter struct{}

// NewConverter creates a new converter instance
func NewConverter() Converter {
	return &DefaultConverter{}
}

// ConvertRequest converts request between different formats
func (c *DefaultConverter) ConvertRequest(data interface{}, fromProvider, toProvider models.ProtocolPrefix) (interface{}, error) {
	// If same protocol, no conversion needed
	if fromProvider == toProvider {
		return data, nil
	}

	switch toProvider {
	case models.ProtocolOpenAI:
		switch fromProvider {
		case models.ProtocolGemini:
			return c.toOpenAIRequestFromGemini(data)
		case models.ProtocolClaude:
			return c.toOpenAIRequestFromClaude(data)
		}
	case models.ProtocolClaude:
		switch fromProvider {
		case models.ProtocolOpenAI:
			return c.toClaudeRequestFromOpenAI(data)
		case models.ProtocolGemini:
			return c.toClaudeRequestFromGemini(data)
		}
	case models.ProtocolGemini:
		switch fromProvider {
		case models.ProtocolOpenAI:
			return c.toGeminiRequestFromOpenAI(data)
		case models.ProtocolClaude:
			return c.toGeminiRequestFromClaude(data)
		}
	}

	return nil, fmt.Errorf("unsupported conversion from %s to %s", fromProvider, toProvider)
}

// ConvertResponse converts response between different formats
func (c *DefaultConverter) ConvertResponse(data interface{}, fromProvider, toProvider models.ProtocolPrefix, model string) (interface{}, error) {
	if fromProvider == toProvider {
		return data, nil
	}

	switch toProvider {
	case models.ProtocolOpenAI:
		switch fromProvider {
		case models.ProtocolGemini:
			return c.toOpenAIChatCompletionFromGemini(data, model)
		case models.ProtocolClaude:
			return c.toOpenAIChatCompletionFromClaude(data, model)
		}
	case models.ProtocolClaude:
		switch fromProvider {
		case models.ProtocolOpenAI:
			return c.toClaudeChatCompletionFromOpenAI(data, model)
		case models.ProtocolGemini:
			return c.toClaudeChatCompletionFromGemini(data, model)
		}
	}

	return nil, fmt.Errorf("unsupported response conversion from %s to %s", fromProvider, toProvider)
}

// ConvertStreamChunk converts stream chunks between different formats
func (c *DefaultConverter) ConvertStreamChunk(data interface{}, fromProvider, toProvider models.ProtocolPrefix, model string) (interface{}, error) {
	if fromProvider == toProvider {
		return data, nil
	}

	switch toProvider {
	case models.ProtocolOpenAI:
		switch fromProvider {
		case models.ProtocolGemini:
			return c.toOpenAIStreamChunkFromGemini(data, model)
		case models.ProtocolClaude:
			return c.toOpenAIStreamChunkFromClaude(data, model)
		}
	case models.ProtocolClaude:
		switch fromProvider {
		case models.ProtocolOpenAI:
			return c.toClaudeStreamChunkFromOpenAI(data, model)
		case models.ProtocolGemini:
			return c.toClaudeStreamChunkFromGemini(data, model)
		}
	}

	return nil, fmt.Errorf("unsupported stream chunk conversion from %s to %s", fromProvider, toProvider)
}

// ConvertModelList converts model lists between different formats
func (c *DefaultConverter) ConvertModelList(data interface{}, fromProvider, toProvider models.ProtocolPrefix) (interface{}, error) {
	if fromProvider == toProvider {
		return data, nil
	}

	switch toProvider {
	case models.ProtocolOpenAI:
		switch fromProvider {
		case models.ProtocolGemini:
			return c.toOpenAIModelListFromGemini(data)
		case models.ProtocolClaude:
			return c.toOpenAIModelListFromClaude(data)
		}
	}

	return nil, fmt.Errorf("unsupported model list conversion from %s to %s", fromProvider, toProvider)
}

// Helper function to check and assign default values
func checkAndAssignOrDefault[T comparable](value T, defaultValue T) T {
	var zero T
	if value != zero {
		return value
	}
	return defaultValue
}

// OpenAI conversion functions
func (c *DefaultConverter) toOpenAIRequestFromGemini(data interface{}) (*models.OpenAIRequest, error) {
	geminiReq, ok := data.(*models.GeminiRequest)
	if !ok {
		// Try to unmarshal from map
		jsonData, _ := json.Marshal(data)
		geminiReq = &models.GeminiRequest{}
		if err := json.Unmarshal(jsonData, geminiReq); err != nil {
			return nil, err
		}
	}

	openaiReq := &models.OpenAIRequest{
		Model:       "gpt-3.5-turbo", // Default model
		Messages:    []models.OpenAIMessage{},
		MaxTokens:   DefaultMaxTokens,
		Temperature: DefaultTemperature,
		TopP:        DefaultTopP,
	}

	// Process system instruction
	if geminiReq.SystemInstruction != nil && len(geminiReq.SystemInstruction.Parts) > 0 {
		systemContent := c.processGeminiPartsToOpenAIContent(geminiReq.SystemInstruction.Parts)
		if systemContent != "" {
			openaiReq.Messages = append(openaiReq.Messages, models.OpenAIMessage{
				Role:    models.RoleSystem,
				Content: systemContent,
			})
		}
	}

	// Process contents
	for _, content := range geminiReq.Contents {
		openaiContent := c.processGeminiPartsToOpenAIContent(content.Parts)
		if openaiContent != nil {
			role := content.Role
			if role == models.RoleModel {
				role = models.RoleAssistant
			}
			openaiReq.Messages = append(openaiReq.Messages, models.OpenAIMessage{
				Role:    role,
				Content: openaiContent,
			})
		}
	}

	// Process generation config
	if geminiReq.GenerationConfig != nil {
		if geminiReq.GenerationConfig.MaxOutputTokens > 0 {
			openaiReq.MaxTokens = geminiReq.GenerationConfig.MaxOutputTokens
		}
		if geminiReq.GenerationConfig.Temperature > 0 {
			openaiReq.Temperature = geminiReq.GenerationConfig.Temperature
		}
		if geminiReq.GenerationConfig.TopP > 0 {
			openaiReq.TopP = geminiReq.GenerationConfig.TopP
		}
	}

	return openaiReq, nil
}

func (c *DefaultConverter) processGeminiPartsToOpenAIContent(parts []models.GeminiPart) interface{} {
	if len(parts) == 0 {
		return ""
	}

	var contentParts []models.ContentPart
	hasMultimodal := false

	for _, part := range parts {
		if part.Text != "" {
			contentParts = append(contentParts, models.ContentPart{
				Type: "text",
				Text: part.Text,
			})
		}

		if part.InlineData != nil {
			hasMultimodal = true
			contentParts = append(contentParts, models.ContentPart{
				Type: "image_url",
				ImageURL: &models.ImageURL{
					URL: fmt.Sprintf("data:%s;base64,%s", part.InlineData.MimeType, part.InlineData.Data),
				},
			})
		}

		if part.FileData != nil {
			hasMultimodal = true
			if strings.HasPrefix(part.FileData.MimeType, "image/") {
				contentParts = append(contentParts, models.ContentPart{
					Type: "image_url",
					ImageURL: &models.ImageURL{
						URL: part.FileData.FileURI,
					},
				})
			} else if strings.HasPrefix(part.FileData.MimeType, "audio/") {
				contentParts = append(contentParts, models.ContentPart{
					Type: "text",
					Text: fmt.Sprintf("[Audio file: %s]", part.FileData.FileURI),
				})
			}
		}
	}

	// Return string for simple text content
	if len(contentParts) == 1 && contentParts[0].Type == "text" && !hasMultimodal {
		return contentParts[0].Text
	}

	return contentParts
}

func (c *DefaultConverter) toOpenAIChatCompletionFromGemini(data interface{}, model string) (*models.OpenAIResponse, error) {
	geminiResp, ok := data.(*models.GeminiResponse)
	if !ok {
		jsonData, _ := json.Marshal(data)
		geminiResp = &models.GeminiResponse{}
		if err := json.Unmarshal(jsonData, geminiResp); err != nil {
			return nil, err
		}
	}

	content := c.processGeminiResponseContent(geminiResp)
	
	response := &models.OpenAIResponse{
		ID:      fmt.Sprintf("chatcmpl-%s", uuid.New().String()),
		Object:  "chat.completion",
		Created: time.Now().Unix(),
		Model:   model,
		Choices: []models.OpenAIChoice{
			{
				Index: 0,
				Message: &models.OpenAIMessage{
					Role:    models.RoleAssistant,
					Content: content,
				},
				FinishReason: "stop",
			},
		},
	}

	if geminiResp.UsageMetadata != nil {
		response.Usage = &models.Usage{
			PromptTokens:     geminiResp.UsageMetadata.PromptTokenCount,
			CompletionTokens: geminiResp.UsageMetadata.CandidatesTokenCount,
			TotalTokens:      geminiResp.UsageMetadata.TotalTokenCount,
		}
	}

	return response, nil
}

func (c *DefaultConverter) processGeminiResponseContent(resp *models.GeminiResponse) string {
	if resp == nil || len(resp.Candidates) == 0 {
		return ""
	}

	var contents []string
	for _, candidate := range resp.Candidates {
		for _, part := range candidate.Content.Parts {
			if part.Text != "" {
				contents = append(contents, part.Text)
			}
		}
	}

	return strings.Join(contents, "\n")
}

func (c *DefaultConverter) toOpenAIStreamChunkFromGemini(data interface{}, model string) (*models.StreamChunk, error) {
	chunkText, ok := data.(string)
	if !ok {
		return nil, fmt.Errorf("expected string chunk for Gemini stream")
	}

	return &models.StreamChunk{
		ID:      fmt.Sprintf("chatcmpl-%s", uuid.New().String()),
		Object:  "chat.completion.chunk",
		Created: time.Now().Unix(),
		Model:   model,
		Choices: []models.StreamChoice{
			{
				Index: 0,
				Delta: &models.OpenAIMessage{
					Content: chunkText,
				},
				FinishReason: "",
			},
		},
	}, nil
}

func (c *DefaultConverter) toOpenAIModelListFromGemini(data interface{}) (*models.ModelList, error) {
	// Implementation for model list conversion
	// This would parse Gemini's model list format and convert to OpenAI format
	return &models.ModelList{
		Object: "list",
		Data:   []models.ModelInfo{},
	}, nil
}

// Claude conversion implementations
func (c *DefaultConverter) toOpenAIRequestFromClaude(data interface{}) (*models.OpenAIRequest, error) {
	claudeReq, ok := data.(*models.ClaudeRequest)
	if !ok {
		jsonData, _ := json.Marshal(data)
		claudeReq = &models.ClaudeRequest{}
		if err := json.Unmarshal(jsonData, claudeReq); err != nil {
			return nil, err
		}
	}

	openaiReq := &models.OpenAIRequest{
		Model:       claudeReq.Model,
		Messages:    []models.OpenAIMessage{},
		MaxTokens:   checkAndAssignOrDefault(claudeReq.MaxTokens, DefaultMaxTokens),
		Temperature: checkAndAssignOrDefault(claudeReq.Temperature, DefaultTemperature),
		TopP:        checkAndAssignOrDefault(claudeReq.TopP, DefaultTopP),
		Stream:      claudeReq.Stream,
	}

	// Add system message if present
	if claudeReq.System != "" {
		openaiReq.Messages = append(openaiReq.Messages, models.OpenAIMessage{
			Role:    models.RoleSystem,
			Content: claudeReq.System,
		})
	}

	// Process messages
	for _, msg := range claudeReq.Messages {
		openaiMsg := models.OpenAIMessage{
			Role: msg.Role,
		}

		// Process content
		content := c.processClaudeContentToOpenAI(msg.Content)
		openaiMsg.Content = content

		// Handle tool results
		if msg.Role == models.RoleUser {
			for _, item := range msg.Content {
				if item.Type == "tool_result" {
					openaiReq.Messages = append(openaiReq.Messages, models.OpenAIMessage{
						Role:       models.RoleTool,
						ToolCallID: item.ToolUseID,
						Content:    fmt.Sprintf("%v", item.Content),
					})
					continue
				}
			}
		}

		// Handle tool calls
		if msg.Role == models.RoleAssistant {
			var toolCalls []models.ToolCall
			for _, item := range msg.Content {
				if item.Type == "tool_use" {
					toolCalls = append(toolCalls, models.ToolCall{
						ID:   item.ID,
						Type: "function",
						Function: models.ToolCallFunction{
							Name:      item.Name,
							Arguments: c.marshalJSON(item.Input),
						},
					})
				}
			}
			if len(toolCalls) > 0 {
				openaiMsg.ToolCalls = toolCalls
				openaiMsg.Content = ""
			}
		}

		if openaiMsg.Content != nil || len(openaiMsg.ToolCalls) > 0 {
			openaiReq.Messages = append(openaiReq.Messages, openaiMsg)
		}
	}

	// Process tools
	if len(claudeReq.Tools) > 0 {
		openaiReq.Tools = make([]models.Tool, len(claudeReq.Tools))
		for i, tool := range claudeReq.Tools {
			openaiReq.Tools[i] = models.Tool{
				Type: "function",
				Function: models.ToolFunction{
					Name:        tool.Name,
					Description: tool.Description,
					Parameters:  tool.InputSchema,
				},
			}
		}
	}

	return openaiReq, nil
}

func (c *DefaultConverter) processClaudeContentToOpenAI(content []models.ClaudeContent) interface{} {
	if len(content) == 0 {
		return ""
	}

	var parts []models.ContentPart
	hasMultimodal := false

	for _, block := range content {
		switch block.Type {
		case "text":
			if block.Text != "" {
				parts = append(parts, models.ContentPart{
					Type: "text",
					Text: block.Text,
				})
			}
		case "image":
			if block.Source != nil && block.Source.Type == "base64" {
				hasMultimodal = true
				parts = append(parts, models.ContentPart{
					Type: "image_url",
					ImageURL: &models.ImageURL{
						URL: fmt.Sprintf("data:%s;base64,%s", block.Source.MediaType, block.Source.Data),
					},
				})
			}
		}
	}

	// Return string for simple text
	if len(parts) == 1 && parts[0].Type == "text" && !hasMultimodal {
		return parts[0].Text
	}

	return parts
}

func (c *DefaultConverter) toOpenAIChatCompletionFromClaude(data interface{}, model string) (*models.OpenAIResponse, error) {
	claudeResp, ok := data.(*models.ClaudeResponse)
	if !ok {
		jsonData, _ := json.Marshal(data)
		claudeResp = &models.ClaudeResponse{}
		if err := json.Unmarshal(jsonData, claudeResp); err != nil {
			return nil, err
		}
	}

	content := c.processClaudeResponseContent(claudeResp.Content)
	finishReason := c.mapClaudeStopReason(claudeResp.StopReason)

	response := &models.OpenAIResponse{
		ID:      fmt.Sprintf("chatcmpl-%s", uuid.New().String()),
		Object:  "chat.completion",
		Created: time.Now().Unix(),
		Model:   model,
		Choices: []models.OpenAIChoice{
			{
				Index: 0,
				Message: &models.OpenAIMessage{
					Role:    models.RoleAssistant,
					Content: content,
				},
				FinishReason: finishReason,
			},
		},
	}

	if claudeResp.Usage != nil {
		response.Usage = &models.Usage{
			PromptTokens:     claudeResp.Usage.InputTokens,
			CompletionTokens: claudeResp.Usage.OutputTokens,
			TotalTokens:      claudeResp.Usage.InputTokens + claudeResp.Usage.OutputTokens,
		}
	}

	return response, nil
}

func (c *DefaultConverter) processClaudeResponseContent(content []models.ClaudeContent) interface{} {
	if len(content) == 0 {
		return ""
	}

	var parts []models.ContentPart
	for _, block := range content {
		switch block.Type {
		case "text":
			parts = append(parts, models.ContentPart{
				Type: "text",
				Text: block.Text,
			})
		case "thinking":
			// Extract thinking content if needed
			parts = append(parts, models.ContentPart{
				Type: "text",
				Text: fmt.Sprintf("<thinking>%s</thinking>", block.Thinking),
			})
		}
	}

	if len(parts) == 1 && parts[0].Type == "text" {
		return parts[0].Text
	}

	return parts
}

func (c *DefaultConverter) mapClaudeStopReason(reason string) string {
	switch reason {
	case "end_turn":
		return "stop"
	case "max_tokens":
		return "length"
	case "stop_sequence":
		return "stop"
	default:
		return "stop"
	}
}

func (c *DefaultConverter) toOpenAIStreamChunkFromClaude(data interface{}, model string) (*models.StreamChunk, error) {
	claudeChunk, ok := data.(string)
	if !ok {
		return nil, fmt.Errorf("expected string chunk for Claude stream")
	}

	return &models.StreamChunk{
		ID:      fmt.Sprintf("chatcmpl-%s", uuid.New().String()),
		Object:  "chat.completion.chunk",
		Created: time.Now().Unix(),
		Model:   model,
		Choices: []models.StreamChoice{
			{
				Index: 0,
				Delta: &models.OpenAIMessage{
					Content: claudeChunk,
				},
				FinishReason: "",
			},
		},
	}, nil
}

func (c *DefaultConverter) toOpenAIModelListFromClaude(data interface{}) (*models.ModelList, error) {
	// Implementation for Claude model list conversion
	return &models.ModelList{
		Object: "list",
		Data:   []models.ModelInfo{},
	}, nil
}

// Reverse conversions (OpenAI to Claude/Gemini)
func (c *DefaultConverter) toClaudeRequestFromOpenAI(data interface{}) (*models.ClaudeRequest, error) {
	openaiReq, ok := data.(*models.OpenAIRequest)
	if !ok {
		jsonData, _ := json.Marshal(data)
		openaiReq = &models.OpenAIRequest{}
		if err := json.Unmarshal(jsonData, openaiReq); err != nil {
			return nil, err
		}
	}

	claudeReq := &models.ClaudeRequest{
		Model:       openaiReq.Model,
		Messages:    []models.ClaudeMessage{},
		MaxTokens:   checkAndAssignOrDefault(openaiReq.MaxTokens, DefaultMaxTokens),
		Temperature: checkAndAssignOrDefault(openaiReq.Temperature, DefaultTemperature),
		TopP:        checkAndAssignOrDefault(openaiReq.TopP, DefaultTopP),
		Stream:      openaiReq.Stream,
	}

	// Process messages
	for _, msg := range openaiReq.Messages {
		if msg.Role == models.RoleSystem {
			claudeReq.System = msg.GetContentAsString()
			continue
		}

		claudeMsg := models.ClaudeMessage{
			Role: msg.Role,
		}

		// Convert content
		contentParts := msg.GetContentAsParts()
		for _, part := range contentParts {
			switch part.Type {
			case "text":
				claudeMsg.Content = append(claudeMsg.Content, models.ClaudeContent{
					Type: "text",
					Text: part.Text,
				})
			case "image_url":
				if part.ImageURL != nil && strings.HasPrefix(part.ImageURL.URL, "data:") {
					// Parse data URL
					parts := strings.SplitN(part.ImageURL.URL, ",", 2)
					if len(parts) == 2 {
						header := parts[0]
						data := parts[1]
						mediaType := strings.TrimPrefix(strings.Split(header, ";")[0], "data:")
						
						claudeMsg.Content = append(claudeMsg.Content, models.ClaudeContent{
							Type: "image",
							Source: &models.ClaudeImageSource{
								Type:      "base64",
								MediaType: mediaType,
								Data:      data,
							},
						})
					}
				}
			}
		}

		// Handle tool messages
		if msg.Role == models.RoleTool {
			claudeMsg.Role = models.RoleUser
			claudeMsg.Content = []models.ClaudeContent{
				{
					Type:      "tool_result",
					ToolUseID: msg.ToolCallID,
					Content:   msg.GetContentAsString(),
				},
			}
		}

		// Handle tool calls
		if len(msg.ToolCalls) > 0 {
			claudeMsg.Content = []models.ClaudeContent{}
			for _, tc := range msg.ToolCalls {
				var input map[string]interface{}
				json.Unmarshal([]byte(tc.Function.Arguments), &input)
				
				claudeMsg.Content = append(claudeMsg.Content, models.ClaudeContent{
					Type:  "tool_use",
					ID:    tc.ID,
					Name:  tc.Function.Name,
					Input: input,
				})
			}
		}

		if len(claudeMsg.Content) > 0 {
			claudeReq.Messages = append(claudeReq.Messages, claudeMsg)
		}
	}

	// Convert tools
	if len(openaiReq.Tools) > 0 {
		claudeReq.Tools = make([]models.ClaudeTool, len(openaiReq.Tools))
		for i, tool := range openaiReq.Tools {
			claudeReq.Tools[i] = models.ClaudeTool{
				Name:        tool.Function.Name,
				Description: tool.Function.Description,
				InputSchema: tool.Function.Parameters,
			}
		}
	}

	return claudeReq, nil
}

func (c *DefaultConverter) toGeminiRequestFromOpenAI(data interface{}) (*models.GeminiRequest, error) {
	openaiReq, ok := data.(*models.OpenAIRequest)
	if !ok {
		jsonData, _ := json.Marshal(data)
		openaiReq = &models.OpenAIRequest{}
		if err := json.Unmarshal(jsonData, openaiReq); err != nil {
			return nil, err
		}
	}

	geminiReq := &models.GeminiRequest{
		Contents: []models.GeminiContent{},
	}

	// Extract system messages
	var systemTexts []string
	var nonSystemMessages []models.OpenAIMessage

	for _, msg := range openaiReq.Messages {
		if msg.Role == models.RoleSystem {
			systemTexts = append(systemTexts, msg.GetContentAsString())
		} else {
			nonSystemMessages = append(nonSystemMessages, msg)
		}
	}

	// Set system instruction
	if len(systemTexts) > 0 {
		geminiReq.SystemInstruction = &models.GeminiSystemInstruction{
			Parts: []models.GeminiPart{
				{Text: strings.Join(systemTexts, "\n")},
			},
		}
	}

	// Process messages
	for _, msg := range nonSystemMessages {
		role := msg.Role
		if role == models.RoleAssistant {
			role = models.RoleModel
		}

		geminiContent := models.GeminiContent{
			Role:  role,
			Parts: []models.GeminiPart{},
		}

		// Convert content
		contentParts := msg.GetContentAsParts()
		for _, part := range contentParts {
			switch part.Type {
			case "text":
				geminiContent.Parts = append(geminiContent.Parts, models.GeminiPart{
					Text: part.Text,
				})
			case "image_url":
				if part.ImageURL != nil {
					if strings.HasPrefix(part.ImageURL.URL, "data:") {
						// Parse data URL
						parts := strings.SplitN(part.ImageURL.URL, ",", 2)
						if len(parts) == 2 {
							header := parts[0]
							data := parts[1]
							mimeType := strings.TrimPrefix(strings.Split(header, ";")[0], "data:")
							
							geminiContent.Parts = append(geminiContent.Parts, models.GeminiPart{
								InlineData: &models.GeminiInlineData{
									MimeType: mimeType,
									Data:     data,
								},
							})
						}
					} else {
						// Regular URL
						geminiContent.Parts = append(geminiContent.Parts, models.GeminiPart{
							FileData: &models.GeminiFileData{
								MimeType: "image/jpeg",
								FileURI:  part.ImageURL.URL,
							},
						})
					}
				}
			}
		}

		if len(geminiContent.Parts) > 0 {
			geminiReq.Contents = append(geminiReq.Contents, geminiContent)
		}
	}

	// Set generation config
	config := &models.GeminiGenerationConfig{
		Temperature:     checkAndAssignOrDefault(openaiReq.Temperature, DefaultTemperature),
		TopP:            checkAndAssignOrDefault(openaiReq.TopP, DefaultTopP),
		MaxOutputTokens: checkAndAssignOrDefault(openaiReq.MaxTokens, DefaultGeminiMaxTokens),
	}
	geminiReq.GenerationConfig = config

	// Convert tools
	if len(openaiReq.Tools) > 0 {
		var funcDecls []models.GeminiFunctionDeclaration
		for _, tool := range openaiReq.Tools {
			funcDecls = append(funcDecls, models.GeminiFunctionDeclaration{
				Name:        tool.Function.Name,
				Description: tool.Function.Description,
				Parameters:  tool.Function.Parameters,
			})
		}
		geminiReq.Tools = []models.GeminiTool{
			{FunctionDeclarations: funcDecls},
		}
	}

	return geminiReq, nil
}

// Additional conversion methods would be implemented similarly...

// Helper methods
func (c *DefaultConverter) marshalJSON(v interface{}) string {
	data, _ := json.Marshal(v)
	return string(data)
}

// Stub implementations for remaining conversions
func (c *DefaultConverter) toClaudeRequestFromGemini(data interface{}) (*models.ClaudeRequest, error) {
	// Implementation would follow similar pattern
	return &models.ClaudeRequest{}, nil
}

func (c *DefaultConverter) toGeminiRequestFromClaude(data interface{}) (*models.GeminiRequest, error) {
	// Implementation would follow similar pattern
	return &models.GeminiRequest{}, nil
}

func (c *DefaultConverter) toClaudeChatCompletionFromOpenAI(data interface{}, model string) (*models.ClaudeResponse, error) {
	// Implementation would follow similar pattern
	return &models.ClaudeResponse{}, nil
}

func (c *DefaultConverter) toClaudeChatCompletionFromGemini(data interface{}, model string) (*models.ClaudeResponse, error) {
	// Implementation would follow similar pattern
	return &models.ClaudeResponse{}, nil
}

func (c *DefaultConverter) toClaudeStreamChunkFromOpenAI(data interface{}, model string) (interface{}, error) {
	// Implementation would follow similar pattern
	return nil, nil
}

func (c *DefaultConverter) toClaudeStreamChunkFromGemini(data interface{}, model string) (interface{}, error) {
	// Implementation would follow similar pattern
	return nil, nil
}