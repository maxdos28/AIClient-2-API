package kiro

import (
	"os"

	"github.com/aiproxy/go-aiproxy/pkg/models"
)

// NewClientWithMock creates a Kiro client with mock support
func NewClientWithMock(config *models.ProviderConfig) (*Client, error) {
	// Check if mock mode is enabled
	if os.Getenv("KIRO_MOCK_MODE") == "true" {
		mockClient, err := NewMockClient(config)
		if err != nil {
			return nil, err
		}
		// Return mock client wrapped as regular client
		return &Client{
			BaseProvider: mockClient.BaseProvider,
			httpClient:   mockClient.httpClient,
			baseURL:      "mock://kiro",
			isInitialized: true,
		}, nil
	}

	// Return regular client
	return NewClient(config)
}