package openai

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

// Client implements the OpenAI provider
type Client struct {
	providers.BaseProvider
	httpClient *http.Client
	baseURL    string
	apiKey     string
}

// NewClient creates a new OpenAI client
func NewClient(config *models.ProviderConfig) (*Client, error) {
	baseURL := config.BaseURL
	if baseURL == "" {
		baseURL = "https://api.openai.com/v1"
	}

	return &Client{
		BaseProvider: providers.BaseProvider{
			Config:   config,
			Protocol: models.ProtocolOpenAI,
		},
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		baseURL: strings.TrimSuffix(baseURL, "/"),
		apiKey:  config.APIKey,
	}, nil
}

// GenerateContent implements the Provider interface
func (c *Client) GenerateContent(ctx context.Context, model string, request interface{}) (interface{}, error) {
	// Convert request to OpenAI format if needed
	openaiReq, ok := request.(*models.OpenAIRequest)
	if !ok {
		return nil, fmt.Errorf("invalid request type for OpenAI provider")
	}

	// Override model if specified
	if model != "" {
		openaiReq.Model = model
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

// GenerateContentStream implements streaming for OpenAI
func (c *Client) GenerateContentStream(ctx context.Context, model string, request interface{}) (io.ReadCloser, error) {
	openaiReq, ok := request.(*models.OpenAIRequest)
	if !ok {
		return nil, fmt.Errorf("invalid request type for OpenAI provider")
	}

	// Override model and enable streaming
	if model != "" {
		openaiReq.Model = model
	}
	openaiReq.Stream = true

	// Make streaming request
	url := fmt.Sprintf("%s/chat/completions", c.baseURL)
	resp, err := c.makeRequest(ctx, "POST", url, openaiReq)
	if err != nil {
		return nil, err
	}

	// Return a custom reader that handles SSE parsing
	return &openAIStreamReader{
		reader: bufio.NewReader(resp.Body),
		closer: resp.Body,
		model:  model,
	}, nil
}

// ListModels implements the Provider interface
func (c *Client) ListModels(ctx context.Context) (interface{}, error) {
	url := fmt.Sprintf("%s/models", c.baseURL)
	resp, err := c.makeRequest(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var modelList models.ModelList
	if err := json.NewDecoder(resp.Body).Decode(&modelList); err != nil {
		return nil, fmt.Errorf("failed to decode model list: %w", err)
	}

	return &modelList, nil
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
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.apiKey))
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

// openAIStreamReader handles SSE stream parsing
type openAIStreamReader struct {
	reader *bufio.Reader
	closer io.Closer
	model  string
	buffer []byte
}

func (r *openAIStreamReader) Read(p []byte) (n int, err error) {
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

func (r *openAIStreamReader) Close() error {
	return r.closer.Close()
}
