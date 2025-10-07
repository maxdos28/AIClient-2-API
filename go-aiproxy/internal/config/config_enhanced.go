package config

import (
	"os"

	"github.com/aiproxy/go-aiproxy/pkg/models"
)

// Enhanced configuration fields
type EnhancedConfig struct {
	*Config

	// Cache configuration
	CacheEnabled bool
	CacheMaxSize int64
	CacheTTL     int

	// Redis configuration
	RedisAddr     string
	RedisPassword string
	RedisDB       int

	// Metrics configuration
	MetricsEnabled bool
	MetricsPort    int

	// WebSocket configuration
	WebSocketEnabled        bool
	WebSocketMaxConnections int

	// Load balancer configuration
	LoadBalancerEnabled   bool
	LoadBalancerAlgorithm string

	// Cluster configuration
	ClusterEnabled bool
	NodeID         string
	NodeAddress    string
	SeedNodes      []string

	// Kiro configuration
	KiroOAuthCredsBase64 string
	KiroOAuthCredsFile   string

	// Qwen configuration
	QwenOAuthCredsBase64 string
	QwenOAuthCredsFile   string
}

// AddKiroConfig adds Kiro provider configuration
func (c *Config) AddKiroConfig() {
	if c.ProviderConfigs == nil {
		c.ProviderConfigs = make(map[string]*models.ProviderConfig)
	}

	// Check if we have Kiro credentials
	kiroCredsFile := getEnvOrDefault("KIRO_OAUTH_CREDS_FILE", "")
	kiroCredsBase64 := getEnvOrDefault("KIRO_OAUTH_CREDS_BASE64", "")

	if kiroCredsFile != "" || kiroCredsBase64 != "" {
		c.ProviderConfigs["kiro-api"] = &models.ProviderConfig{
			Provider:         models.ProviderKiro,
			OAuthCredsFile:   kiroCredsFile,
			OAuthCredsBase64: kiroCredsBase64,
			BaseURL:          getEnvOrDefault("KIRO_BASE_URL", "https://api.kiro.com"),
		}

		// Add to model providers if not already present
		hasKiro := false
		for _, p := range c.ModelProviders {
			if p == "kiro-api" {
				hasKiro = true
				break
			}
		}
		if !hasKiro {
			c.ModelProviders = append(c.ModelProviders, "kiro-api")
		}
	}
}

// getEnvOrDefault gets environment variable or returns default
func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
