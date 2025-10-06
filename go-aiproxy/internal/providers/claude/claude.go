package claude

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

	"github.com/aiproxy/go-aiproxy/internal/providers"
	"github.com/aiproxy/go-aiproxy/pkg/models"
)

// Client implements the Claude provider
type Client struct {
	providers.BaseProvider
	httpClient *http.Client
	baseURL    string
	apiKey     string
	version    string
}

// NewClient creates a new Claude client
func NewClient(config *models.ProviderConfig) (*Client, error) {
	baseURL := config.BaseURL
	if baseURL == "" {
		baseURL = "https://api.anthropic.com"
	}

	return &Client{
		BaseProvider: providers.BaseProvider{
			Config:   config,
			Protocol: models.ProtocolClaude,
		},
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		baseURL: strings.TrimSuffix(baseURL, "/"),
		apiKey:  config.APIKey,
		version: "2023-06-01", // Claude API version
	}, nil
}

// GenerateContent implements the Provider interface
func (c *Client) GenerateContent(ctx context.Context, model string, request interface{}) (interface{}, error) {
	claudeReq, ok := request.(*models.ClaudeRequest)
	if !ok {
		return nil, fmt.Errorf("invalid request type for Claude provider")
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

// GenerateContentStream implements streaming for Claude
func (c *Client) GenerateContentStream(ctx context.Context, model string, request interface{}) (io.ReadCloser, error) {
	claudeReq, ok := request.(*models.ClaudeRequest)
	if !ok {
		return nil, fmt.Errorf("invalid request type for Claude provider")
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
	return &claudeStreamReader{
		reader:  bufio.NewReader(resp.Body),
		closer:  resp.Body,
		model:   model,
	}, nil
}

// ListModels implements the Provider interface
func (c *Client) ListModels(ctx context.Context) (interface{}, error) {
	// Claude doesn't have a public models endpoint, so we return a static list
	models := []models.ModelInfo{
		{
			ID:      "claude-3-opus-20240229",
			Object:  "model",
			Created: time.Now().Unix(),
			OwnedBy: "anthropic",
		},
		{
			ID:      "claude-3-sonnet-20240229",
			Object:  "model",
			Created: time.Now().Unix(),
			OwnedBy: "anthropic",
		},
		{
			ID:      "claude-3-haiku-20240307",
			Object:  "model",
			Created: time.Now().Unix(),
			OwnedBy: "anthropic",
		},
		{
			ID:      "claude-3-5-sonnet-20241022",
			Object:  "model",
			Created: time.Now().Unix(),
			OwnedBy: "anthropic",
		},
	}

	return &models.ModelList{
		Object: "list",
		Data:   models,
	}, nil
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
	req.Header.Set("x-api-key", c.apiKey)
	req.Header.Set("anthropic-version", c.version)
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
		return nil, fmt.Errorf("API error %d: %s", resp.StatusCode, string(bodyBytes))
	}

	return resp, nil
}

// claudeStreamReader handles SSE stream parsing for Claude
type claudeStreamReader struct {
	reader  *bufio.Reader
	closer  io.Closer
	model   string
	buffer  []byte
}

func (r *claudeStreamReader) Read(p []byte) (n int, err error) {
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

			// Handle different event types
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

func (r *claudeStreamReader) Close() error {
	return r.closer.Close()
}