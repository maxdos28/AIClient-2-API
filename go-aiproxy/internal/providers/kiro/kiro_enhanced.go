package kiro

import (
	"os"

	"github.com/aiproxy/go-aiproxy/internal/providers"
	"github.com/aiproxy/go-aiproxy/pkg/models"
)

// NewClientWithMock creates a Kiro client with mock support
func NewClientWithMock(config *models.ProviderConfig) (*Client, error) {
	// Check if mock mode is enabled
	if os.Getenv("KIRO_MOCK_MODE") == "true" {
		// For mock mode, we still need to return a regular Client but mark it as mock
		client := &Client{
			BaseProvider: providers.BaseProvider{
				Config:   config,
				Protocol: models.ProtocolClaude,
			},
			baseURL:       "mock://kiro",
			isInitialized: true,
		}
		return client, nil
	}

	// Return regular client
	return NewClient(config)
}
