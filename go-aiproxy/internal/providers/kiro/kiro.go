package kiro

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

// Client implements the Kiro provider (Claude via OAuth)
type Client struct {
	providers.BaseProvider
	httpClient    *http.Client
	baseURL       string
	tokenManager  *auth.TokenManager
	isInitialized bool
}

// NewClient creates a new Kiro client
func NewClient(config *models.ProviderConfig) (*Client, error) {
	baseURL := config.BaseURL
	if baseURL == "" {
		baseURL = "https://api.kiro.com" // Kiro API endpoint
	}

	client := &Client{
		BaseProvider: providers.BaseProvider{
			Config:   config,
			Protocol: models.ProtocolClaude, // Kiro uses Claude protocol
		},
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		baseURL: strings.TrimSuffix(baseURL, "/"),
	}

	// Initialize OAuth if credentials are provided
	if config.OAuthCredsBase64 != "" || config.OAuthCredsFile != "" {
		tokenManager, err := auth.NewTokenManager(config)
		if err != nil {
			return nil, fmt.Errorf("failed to create token manager: %w", err)
		}
		client.tokenManager = tokenManager
		
		// Initialize OAuth client
		if err := client.initialize(context.Background()); err != nil {
			return nil, fmt.Errorf("failed to initialize Kiro OAuth: %w", err)
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

	claudeReq, ok := request.(*models.ClaudeRequest)
	if !ok {
		return nil, fmt.Errorf("invalid request type for Kiro provider")
	}

	// Override model if specified
	if model != "" {
		claudeReq.Model = model
	}

	// Make API request
	url := fmt.Sprintf("%s/v1/messages", c.baseURL)
	resp, err := c.makeRequest(ctx, "POST", url, claudeReq)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	// Parse response
	var claudeResp models.ClaudeResponse
	if err := json.NewDecoder(resp.Body).Decode(&claudeResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &claudeResp, nil
}

// GenerateContentStream implements streaming for Kiro
func (c *Client) GenerateContentStream(ctx context.Context, model string, request interface{}) (io.ReadCloser, error) {
	// Ensure initialized
	if !c.isInitialized && c.tokenManager != nil {
		if err := c.initialize(ctx); err != nil {
			return nil, err
		}
	}

	claudeReq, ok := request.(*models.ClaudeRequest)
	if !ok {
		return nil, fmt.Errorf("invalid request type for Kiro provider")
	}

	// Override model and enable streaming
	if model != "" {
		claudeReq.Model = model
	}
	claudeReq.Stream = true

	// Make streaming request
	url := fmt.Sprintf("%s/v1/messages", c.baseURL)
	resp, err := c.makeRequest(ctx, "POST", url, claudeReq)
	if err != nil {
		return nil, err
	}

	// Return a custom reader that handles SSE parsing
	return &kiroStreamReader{
		reader:  bufio.NewReader(resp.Body),
		closer:  resp.Body,
		model:   model,
	}, nil
}

// ListModels implements the Provider interface
func (c *Client) ListModels(ctx context.Context) (interface{}, error) {
	// Kiro supports Claude models
	modelList := []models.ModelInfo{
		{
			ID:      "claude-3-opus-20240229",
			Object:  "model",
			Created: time.Now().Unix(),
			OwnedBy: "kiro",
		},
		{
			ID:      "claude-3-sonnet-20240229",
			Object:  "model",
			Created: time.Now().Unix(),
			OwnedBy: "kiro",
		},
		{
			ID:      "claude-3-haiku-20240307",
			Object:  "model",
			Created: time.Now().Unix(),
			OwnedBy: "kiro",
		},
		{
			ID:      "claude-3-5-sonnet-20241022",
			Object:  "model",
			Created: time.Now().Unix(),
			OwnedBy: "kiro",
		},
	}

	return &models.ModelList{
		Object: "list",
		Data:   modelList,
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
	req.Header.Set("anthropic-version", "2023-06-01")

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

// kiroStreamReader handles SSE stream parsing for Kiro
type kiroStreamReader struct {
	reader  *bufio.Reader
	closer  io.Closer
	model   string
	buffer  []byte
}

func (r *kiroStreamReader) Read(p []byte) (n int, err error) {
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
			
			// Parse the event
			var event map[string]interface{}
			if err := json.Unmarshal([]byte(data), &event); err != nil {
				continue // Skip malformed events
			}

			// Handle different event types (same as Claude)
			eventType, _ := event["type"].(string)
			switch eventType {
			case "content_block_delta":
				if delta, ok := event["delta"].(map[string]interface{}); ok {
					if deltaType, _ := delta["type"].(string); deltaType == "text_delta" {
						if text, ok := delta["text"].(string); ok && text != "" {
							r.buffer = []byte(text)
							n = copy(p, r.buffer)
							r.buffer = r.buffer[n:]
							return n, nil
						}
					}
				}
			case "message_stop":
				return 0, io.EOF
			}
		}
	}
}

func (r *kiroStreamReader) Close() error {
	return r.closer.Close()
}