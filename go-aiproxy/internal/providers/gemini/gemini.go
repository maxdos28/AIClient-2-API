package gemini

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
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

// Client implements the Gemini provider
type Client struct {
	providers.BaseProvider
	httpClient    *http.Client
	baseURL       string
	projectID     string
	apiKey        string
	authClient    *http.Client
	tokenSource   oauth2.TokenSource
	isInitialized bool
}

// NewClient creates a new Gemini client
func NewClient(config *models.ProviderConfig) (*Client, error) {
	baseURL := config.BaseURL
	if baseURL == "" {
		baseURL = "https://generativelanguage.googleapis.com"
	}

	client := &Client{
		BaseProvider: providers.BaseProvider{
			Config:   config,
			Protocol: models.ProtocolGemini,
		},
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		baseURL:   strings.TrimSuffix(baseURL, "/"),
		projectID: config.ProjectID,
		apiKey:    config.APIKey,
	}

	// Initialize OAuth if credentials are provided
	if config.OAuthCredsBase64 != "" || config.OAuthCredsFile != "" {
		if err := client.initializeAuth(context.Background()); err != nil {
			return nil, err
		}
	}

	return client, nil
}

// initializeAuth sets up OAuth authentication
func (c *Client) initializeAuth(ctx context.Context) error {
	var creds []byte
	var err error

	if c.Config.OAuthCredsBase64 != "" {
		// Decode base64 credentials
		creds = []byte(c.Config.OAuthCredsBase64) // In real implementation, decode from base64
	} else if c.Config.OAuthCredsFile != "" {
		// Read from file - implementation would read the actual file
		return fmt.Errorf("file-based OAuth not implemented yet")
	}

	// Create OAuth2 config
	config, err := google.JWTConfigFromJSON(creds, "https://www.googleapis.com/auth/cloud-platform")
	if err != nil {
		return fmt.Errorf("failed to create JWT config: %w", err)
	}

	// Create token source and HTTP client
	c.tokenSource = config.TokenSource(ctx)
	c.authClient = oauth2.NewClient(ctx, c.tokenSource)
	c.isInitialized = true

	return nil
}

// GenerateContent implements the Provider interface
func (c *Client) GenerateContent(ctx context.Context, model string, request interface{}) (interface{}, error) {
	geminiReq, ok := request.(*models.GeminiRequest)
	if !ok {
		return nil, fmt.Errorf("invalid request type for Gemini provider")
	}

	// Build URL
	url := c.buildURL(model, "generateContent", false)

	// Make API request
	resp, err := c.makeRequest(ctx, "POST", url, geminiReq)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	// Parse response
	var geminiResp models.GeminiResponse
	if err := json.NewDecoder(resp.Body).Decode(&geminiResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &geminiResp, nil
}

// GenerateContentStream implements streaming for Gemini
func (c *Client) GenerateContentStream(ctx context.Context, model string, request interface{}) (io.ReadCloser, error) {
	geminiReq, ok := request.(*models.GeminiRequest)
	if !ok {
		return nil, fmt.Errorf("invalid request type for Gemini provider")
	}

	// Build URL for streaming
	url := c.buildURL(model, "streamGenerateContent", true)

	// Make streaming request
	resp, err := c.makeRequest(ctx, "POST", url, geminiReq)
	if err != nil {
		return nil, err
	}

	// Return a custom reader that handles streaming response
	return &geminiStreamReader{
		reader: bufio.NewReader(resp.Body),
		closer: resp.Body,
		model:  model,
	}, nil
}

// ListModels implements the Provider interface
func (c *Client) ListModels(ctx context.Context) (interface{}, error) {
	url := fmt.Sprintf("%s/v1beta/models", c.baseURL)
	
	resp, err := c.makeRequest(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result struct {
		Models []struct {
			Name string `json:"name"`
		} `json:"models"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode model list: %w", err)
	}

	// Convert to standard format
	var modelList []models.ModelInfo
	for _, m := range result.Models {
		modelID := m.Name
		if strings.HasPrefix(modelID, "models/") {
			modelID = strings.TrimPrefix(modelID, "models/")
		}
		modelList = append(modelList, models.ModelInfo{
			ID:      modelID,
			Object:  "model",
			Created: time.Now().Unix(),
			OwnedBy: "google",
		})
	}

	return &models.ModelList{
		Object: "list",
		Data:   modelList,
	}, nil
}

// RefreshToken refreshes the OAuth token if needed
func (c *Client) RefreshToken(ctx context.Context) error {
	if c.tokenSource == nil {
		return nil // No OAuth configured
	}

	// Force token refresh
	_, err := c.tokenSource.Token()
	return err
}

// buildURL constructs the API URL
func (c *Client) buildURL(model string, action string, isStream bool) string {
	if c.projectID != "" && c.isInitialized {
		// Use vertex AI endpoint for OAuth
		return fmt.Sprintf("https://%s-aiplatform.googleapis.com/v1/projects/%s/locations/%s/publishers/google/models/%s:%s",
			"us-central1", c.projectID, "us-central1", model, action)
	}

	// Use public API with API key
	return fmt.Sprintf("%s/v1beta/models/%s:%s", c.baseURL, model, action)
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

	// Use appropriate client and auth
	client := c.httpClient
	if c.authClient != nil {
		client = c.authClient
	} else if c.apiKey != "" {
		// Add API key to URL
		q := req.URL.Query()
		q.Add("key", c.apiKey)
		req.URL.RawQuery = q.Encode()
	}

	// Make request
	resp, err := client.Do(req)
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

// geminiStreamReader handles streaming response parsing
type geminiStreamReader struct {
	reader *bufio.Reader
	closer io.Closer
	model  string
	buffer []byte
}

func (r *geminiStreamReader) Read(p []byte) (n int, err error) {
	// If we have buffered data, return it first
	if len(r.buffer) > 0 {
		n = copy(p, r.buffer)
		r.buffer = r.buffer[n:]
		return n, nil
	}

	// Read next line from response
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

		// Parse JSON response
		var response models.GeminiResponse
		if err := json.Unmarshal([]byte(line), &response); err != nil {
			continue // Skip malformed lines
		}

		// Extract text content
		for _, candidate := range response.Candidates {
			for _, part := range candidate.Content.Parts {
				if part.Text != "" {
					r.buffer = []byte(part.Text)
					n = copy(p, r.buffer)
					r.buffer = r.buffer[n:]
					return n, nil
				}
			}
		}
	}
}

func (r *geminiStreamReader) Close() error {
	return r.closer.Close()
}