package qwen

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/aiproxy/go-aiproxy/internal/auth"
	"github.com/aiproxy/go-aiproxy/internal/providers"
	"github.com/aiproxy/go-aiproxy/pkg/models"
	"golang.org/x/oauth2"
)

// Client implements the Qwen provider
type Client struct {
	providers.BaseProvider
	httpClient    *http.Client
	baseURL       string
	tokenManager  *auth.TokenManager
	isInitialized bool
}

// NewClient creates a new Qwen client
func NewClient(config *models.ProviderConfig) (*Client, error) {
	baseURL := config.BaseURL
	if baseURL == "" {
		baseURL = "https://api.qwen.com" // Qwen API endpoint
	}

	client := &Client{
		BaseProvider: providers.BaseProvider{
			Config:   config,
			Protocol: models.ProtocolOpenAI, // Qwen uses OpenAI protocol
		},
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		baseURL: strings.TrimSuffix(baseURL, "/"),
	}

	// Initialize OAuth if credentials are provided
	if config.OAuthCredsFile != "" {
		tokenManager, err := auth.NewTokenManager(config)
		if err != nil {
			return nil, fmt.Errorf("failed to create token manager: %w", err)
		}
		client.tokenManager = tokenManager
		
		// Initialize OAuth client
		if err := client.initialize(context.Background()); err != nil {
			return nil, fmt.Errorf("failed to initialize Qwen OAuth: %w", err)
		}
	}

	return client, nil
}

// initialize sets up OAuth authentication
func (c *Client) initialize(ctx context.Context) error {
	token, err := c.tokenManager.GetToken(ctx)
	if err != nil {
		return fmt.Errorf("failed to get token: %w", err)
	}

	// Create OAuth2 client
	c.httpClient = oauth2.NewClient(ctx, oauth2.StaticTokenSource(token))
	c.isInitialized = true

	return nil
}

// GenerateContent implements the Provider interface
func (c *Client) GenerateContent(ctx context.Context, model string, request interface{}) (interface{}, error) {
	// Ensure initialized
	if !c.isInitialized && c.tokenManager != nil {
		if err := c.initialize(ctx); err != nil {
			return nil, err
		}
	}

	openaiReq, ok := request.(*models.OpenAIRequest)
	if !ok {
		return nil, fmt.Errorf("invalid request type for Qwen provider")
	}

	// Override model if specified
	if model != "" {
		openaiReq.Model = model
	}

	// Qwen specific: Add built-in tools if configured
	if c.Config.Provider == models.ProviderQwen {
		openaiReq = c.enhanceWithBuiltinTools(openaiReq)
	}

	// Make API request
	url := fmt.Sprintf("%s/chat/completions", c.baseURL)
	resp, err := c.makeRequest(ctx, "POST", url, openaiReq)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	// Parse response
	var openaiResp models.OpenAIResponse
	if err := json.NewDecoder(resp.Body).Decode(&openaiResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &openaiResp, nil
}

// GenerateContentStream implements streaming for Qwen
func (c *Client) GenerateContentStream(ctx context.Context, model string, request interface{}) (io.ReadCloser, error) {
	// Ensure initialized
	if !c.isInitialized && c.tokenManager != nil {
		if err := c.initialize(ctx); err != nil {
			return nil, err
		}
	}

	openaiReq, ok := request.(*models.OpenAIRequest)
	if !ok {
		return nil, fmt.Errorf("invalid request type for Qwen provider")
	}

	// Override model and enable streaming
	if model != "" {
		openaiReq.Model = model
	}
	openaiReq.Stream = true

	// Qwen specific: Add built-in tools if configured
	if c.Config.Provider == models.ProviderQwen {
		openaiReq = c.enhanceWithBuiltinTools(openaiReq)
	}

	// Make streaming request
	url := fmt.Sprintf("%s/chat/completions", c.baseURL)
	resp, err := c.makeRequest(ctx, "POST", url, openaiReq)
	if err != nil {
		return nil, err
	}

	// Return a custom reader that handles SSE parsing
	return &qwenStreamReader{
		reader:  bufio.NewReader(resp.Body),
		closer:  resp.Body,
		model:   model,
	}, nil
}

// ListModels implements the Provider interface
func (c *Client) ListModels(ctx context.Context) (interface{}, error) {
	// Qwen supported models
	models := []models.ModelInfo{
		{
			ID:      "qwen-max",
			Object:  "model",
			Created: time.Now().Unix(),
			OwnedBy: "qwen",
		},
		{
			ID:      "qwen-plus",
			Object:  "model",
			Created: time.Now().Unix(),
			OwnedBy: "qwen",
		},
		{
			ID:      "qwen-turbo",
			Object:  "model",
			Created: time.Now().Unix(),
			OwnedBy: "qwen",
		},
		{
			ID:      "qwen-coder-turbo",
			Object:  "model",
			Created: time.Now().Unix(),
			OwnedBy: "qwen",
		},
		{
			ID:      "qwen3-coder-plus",
			Object:  "model",
			Created: time.Now().Unix(),
			OwnedBy: "qwen",
		},
	}

	return &models.ModelList{
		Object: "list",
		Data:   models,
	}, nil
}

// RefreshToken refreshes the OAuth token if needed
func (c *Client) RefreshToken(ctx context.Context) error {
	if c.tokenManager == nil {
		return nil // No OAuth configured
	}

	// Force token refresh
	token, err := c.tokenManager.RefreshToken(ctx)
	if err != nil {
		return fmt.Errorf("failed to refresh token: %w", err)
	}

	// Update HTTP client with new token
	c.httpClient = oauth2.NewClient(ctx, oauth2.StaticTokenSource(token))
	
	return nil
}

// IsHealthy checks if the provider is healthy including token validity
func (c *Client) IsHealthy() bool {
	if !c.BaseProvider.IsHealthy() {
		return false
	}

	if c.tokenManager != nil {
		// Check if token is about to expire
		return c.tokenManager.IsTokenValid()
	}

	return true
}

// enhanceWithBuiltinTools adds Qwen's built-in tools to the request
func (c *Client) enhanceWithBuiltinTools(req *models.OpenAIRequest) *models.OpenAIRequest {
	// Qwen built-in tools
	builtinTools := []models.Tool{
		{
			Type: "function",
			Function: models.ToolFunction{
				Name:        "code_interpreter",
				Description: "Execute Python code and return results",
				Parameters: map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"code": map[string]interface{}{
							"type":        "string",
							"description": "Python code to execute",
						},
					},
					"required": []string{"code"},
				},
			},
		},
		{
			Type: "function",
			Function: models.ToolFunction{
				Name:        "web_search",
				Description: "Search the web for information",
				Parameters: map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"query": map[string]interface{}{
							"type":        "string",
							"description": "Search query",
						},
					},
					"required": []string{"query"},
				},
			},
		},
	}

	// Merge with existing tools
	if req.Tools == nil {
		req.Tools = builtinTools
	} else {
		// Check if built-in tools already exist
		existingNames := make(map[string]bool)
		for _, tool := range req.Tools {
			existingNames[tool.Function.Name] = true
		}
		
		// Add only non-existing built-in tools
		for _, tool := range builtinTools {
			if !existingNames[tool.Function.Name] {
				req.Tools = append(req.Tools, tool)
			}
		}
	}

	return req
}

// makeRequest is a helper method to make HTTP requests
func (c *Client) makeRequest(ctx context.Context, method, url string, body interface{}) (*http.Response, error) {
	var reqBody io.Reader
	if body != nil {
		jsonData, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal request: %w", err)
		}
		reqBody = bytes.NewReader(jsonData)
	}

	req, err := http.NewRequestWithContext(ctx, method, url, reqBody)
	if err != nil {
		return nil, err
	}

	// Set headers
	req.Header.Set("Content-Type", "application/json")

	// Make request
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}

	// Check status code
	if resp.StatusCode >= 400 {
		bodyBytes, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		
		// Handle token expiration
		if resp.StatusCode == 401 && c.tokenManager != nil {
			// Try to refresh token
			if err := c.RefreshToken(ctx); err == nil {
				// Retry request with new token
				return c.makeRequest(ctx, method, url, body)
			}
		}
		
		return nil, fmt.Errorf("API error %d: %s", resp.StatusCode, string(bodyBytes))
	}

	return resp, nil
}

// qwenStreamReader handles SSE stream parsing for Qwen
type qwenStreamReader struct {
	reader  *bufio.Reader
	closer  io.Closer
	model   string
	buffer  []byte
}

func (r *qwenStreamReader) Read(p []byte) (n int, err error) {
	// If we have buffered data, return it first
	if len(r.buffer) > 0 {
		n = copy(p, r.buffer)
		r.buffer = r.buffer[n:]
		return n, nil
	}

	// Read next SSE event
	for {
		line, err := r.reader.ReadString('\n')
		if err != nil {
			if err == io.EOF {
				return 0, io.EOF
			}
			return 0, err
		}

		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		if strings.HasPrefix(line, "data: ") {
			data := strings.TrimPrefix(line, "data: ")
			if data == "[DONE]" {
				return 0, io.EOF
			}

			// Parse the chunk
			var chunk models.StreamChunk
			if err := json.Unmarshal([]byte(data), &chunk); err != nil {
				continue // Skip malformed chunks
			}

			// Extract text content
			if len(chunk.Choices) > 0 && chunk.Choices[0].Delta != nil {
				if content, ok := chunk.Choices[0].Delta.Content.(string); ok && content != "" {
					r.buffer = []byte(content)
					n = copy(p, r.buffer)
					r.buffer = r.buffer[n:]
					return n, nil
				}
			}
		}
	}
}

func (r *qwenStreamReader) Close() error {
	return r.closer.Close()
}