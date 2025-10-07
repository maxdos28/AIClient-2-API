package providers

import (
	"context"
	"io"

	"github.com/aiproxy/go-aiproxy/pkg/models"
)

// Provider defines the interface for all AI service providers
type Provider interface {
	// GenerateContent sends a completion request and returns the response
	GenerateContent(ctx context.Context, model string, request interface{}) (interface{}, error)

	// GenerateContentStream sends a streaming completion request
	GenerateContentStream(ctx context.Context, model string, request interface{}) (io.ReadCloser, error)

	// ListModels returns available models
	ListModels(ctx context.Context) (interface{}, error)

	// RefreshToken refreshes authentication token if needed
	RefreshToken(ctx context.Context) error

	// GetProtocolPrefix returns the protocol prefix for this provider
	GetProtocolPrefix() models.ProtocolPrefix

	// IsHealthy checks if the provider is healthy
	IsHealthy() bool
}

// BaseProvider provides common functionality for all providers
type BaseProvider struct {
	Config   *models.ProviderConfig
	Protocol models.ProtocolPrefix
}

// GetProtocolPrefix returns the protocol prefix
func (p *BaseProvider) GetProtocolPrefix() models.ProtocolPrefix {
	return p.Protocol
}

// IsHealthy returns the health status
func (p *BaseProvider) IsHealthy() bool {
	if p.Config == nil {
		return false
	}
	return p.Config.IsHealthy
}

// RefreshToken default implementation (no-op for providers without token refresh)
func (p *BaseProvider) RefreshToken(ctx context.Context) error {
	return nil
}
