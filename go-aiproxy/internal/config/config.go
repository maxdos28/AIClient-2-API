package config

import (
	"fmt"
	"os"
	"strings"

	"github.com/aiproxy/go-aiproxy/pkg/models"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// Config holds all configuration for the server
type Config struct {
	// Server configuration
	Host   string
	Port   int
	APIKey string

	// Provider configuration
	ModelProviders []string
	ProviderConfigs map[string]*models.ProviderConfig

	// OpenAI configuration
	OpenAIAPIKey  string
	OpenAIBaseURL string

	// Claude configuration
	ClaudeAPIKey  string
	ClaudeBaseURL string

	// Gemini configuration
	GeminiAPIKey         string
	GeminiOAuthCredsBase64 string
	GeminiOAuthCredsFile   string
	ProjectID            string

	// System prompt configuration
	SystemPromptFile string
	SystemPromptMode string

	// Logging configuration
	LogPrompts         string
	PromptLogBaseName  string

	// Pool configuration
	ProviderPoolsFile string
	RequestMaxRetries int
	RequestBaseDelay  int
	
	// Redis configuration
	RedisAddr     string
	RedisPassword string
	RedisDB       int
	
	// Load balancer configuration
	LoadBalancerEnabled   bool
	LoadBalancerAlgorithm string
	
	// Cluster configuration
	ClusterEnabled bool
	NodeID         string
	NodeAddress    string
	SeedNodes      []string
}

// New creates a new configuration instance
func New() *Config {
	return &Config{
		ProviderConfigs: make(map[string]*models.ProviderConfig),
	}
}

// LoadFromFile loads configuration from a file
func (c *Config) LoadFromFile(filename string) error {
	viper.SetConfigFile(filename)
	return viper.ReadInConfig()
}

// LoadFromFlags loads configuration from command line flags
func (c *Config) LoadFromFlags(cmd *cobra.Command) error {
	// Server flags
	c.Host, _ = cmd.Flags().GetString("host")
	c.Port, _ = cmd.Flags().GetInt("port")
	c.APIKey, _ = cmd.Flags().GetString("api-key")

	// Provider flags
	c.ModelProviders, _ = cmd.Flags().GetStringSlice("model-provider")

	// OpenAI flags
	c.OpenAIAPIKey, _ = cmd.Flags().GetString("openai-api-key")
	c.OpenAIBaseURL, _ = cmd.Flags().GetString("openai-base-url")

	// Claude flags
	c.ClaudeAPIKey, _ = cmd.Flags().GetString("claude-api-key")
	c.ClaudeBaseURL, _ = cmd.Flags().GetString("claude-base-url")

	// Gemini flags
	c.GeminiAPIKey, _ = cmd.Flags().GetString("gemini-api-key")
	c.GeminiOAuthCredsBase64, _ = cmd.Flags().GetString("gemini-oauth-creds-base64")
	c.GeminiOAuthCredsFile, _ = cmd.Flags().GetString("gemini-oauth-creds-file")
	c.ProjectID, _ = cmd.Flags().GetString("project-id")

	// System prompt flags
	c.SystemPromptFile, _ = cmd.Flags().GetString("system-prompt-file")
	c.SystemPromptMode, _ = cmd.Flags().GetString("system-prompt-mode")

	// Logging flags
	c.LogPrompts, _ = cmd.Flags().GetString("log-prompts")
	c.PromptLogBaseName, _ = cmd.Flags().GetString("prompt-log-base-name")

	// Pool flags
	c.ProviderPoolsFile, _ = cmd.Flags().GetString("provider-pools-file")
	c.RequestMaxRetries, _ = cmd.Flags().GetInt("request-max-retries")
	c.RequestBaseDelay, _ = cmd.Flags().GetInt("request-base-delay")

	// Build provider configurations
	c.buildProviderConfigs()

	return nil
}

// LoadFromEnv loads configuration from environment variables
func (c *Config) LoadFromEnv() {
	// Server environment variables
	if host := os.Getenv("AIPROXY_HOST"); host != "" {
		c.Host = host
	}
	if port := os.Getenv("AIPROXY_PORT"); port != "" {
		fmt.Sscanf(port, "%d", &c.Port)
	}
	if apiKey := os.Getenv("AIPROXY_API_KEY"); apiKey != "" {
		c.APIKey = apiKey
	}

	// Provider environment variables
	if providers := os.Getenv("AIPROXY_MODEL_PROVIDERS"); providers != "" {
		c.ModelProviders = strings.Split(providers, ",")
	}

	// OpenAI environment variables
	if key := os.Getenv("OPENAI_API_KEY"); key != "" {
		c.OpenAIAPIKey = key
	}
	if url := os.Getenv("OPENAI_BASE_URL"); url != "" {
		c.OpenAIBaseURL = url
	}

	// Claude environment variables
	if key := os.Getenv("CLAUDE_API_KEY"); key != "" {
		c.ClaudeAPIKey = key
	}
	if url := os.Getenv("CLAUDE_BASE_URL"); url != "" {
		c.ClaudeBaseURL = url
	}

	// Gemini environment variables
	if key := os.Getenv("GEMINI_API_KEY"); key != "" {
		c.GeminiAPIKey = key
	}
	if creds := os.Getenv("GEMINI_OAUTH_CREDS_BASE64"); creds != "" {
		c.GeminiOAuthCredsBase64 = creds
	}
	if file := os.Getenv("GEMINI_OAUTH_CREDS_FILE"); file != "" {
		c.GeminiOAuthCredsFile = file
	}
	if projectID := os.Getenv("GOOGLE_CLOUD_PROJECT"); projectID != "" {
		c.ProjectID = projectID
	}

	// Rebuild provider configurations with environment variables
	c.buildProviderConfigs()
}

// buildProviderConfigs creates provider configurations based on settings
func (c *Config) buildProviderConfigs() {
	for _, provider := range c.ModelProviders {
		switch provider {
		case "openai-custom":
			c.ProviderConfigs["openai-custom"] = &models.ProviderConfig{
				Provider: models.ProviderOpenAI,
				APIKey:   c.OpenAIAPIKey,
				BaseURL:  c.OpenAIBaseURL,
			}
		case "claude-custom":
			c.ProviderConfigs["claude-custom"] = &models.ProviderConfig{
				Provider: models.ProviderClaude,
				APIKey:   c.ClaudeAPIKey,
				BaseURL:  c.ClaudeBaseURL,
			}
		case "gemini-cli", "gemini-cli-oauth":
			c.ProviderConfigs[provider] = &models.ProviderConfig{
				Provider:         models.ProviderGemini,
				APIKey:           c.GeminiAPIKey,
				ProjectID:        c.ProjectID,
				OAuthCredsBase64: c.GeminiOAuthCredsBase64,
				OAuthCredsFile:   c.GeminiOAuthCredsFile,
			}
		}
	}
}

// GetProviderConfig returns configuration for a specific provider
func (c *Config) GetProviderConfig(provider string) (*models.ProviderConfig, error) {
	cfg, ok := c.ProviderConfigs[provider]
	if !ok {
		return nil, fmt.Errorf("provider %s not configured", provider)
	}
	return cfg, nil
}

// Validate checks if the configuration is valid
func (c *Config) Validate() error {
	if c.APIKey == "" {
		return fmt.Errorf("API key is required")
	}

	if len(c.ModelProviders) == 0 {
		return fmt.Errorf("at least one model provider must be configured")
	}

	// Validate each provider configuration
	for name, cfg := range c.ProviderConfigs {
		if err := c.validateProviderConfig(name, cfg); err != nil {
			return fmt.Errorf("invalid configuration for provider %s: %w", name, err)
		}
	}

	return nil
}

// validateProviderConfig validates a specific provider configuration
func (c *Config) validateProviderConfig(name string, cfg *models.ProviderConfig) error {
	switch cfg.Provider {
	case models.ProviderOpenAI, models.ProviderClaude:
		if cfg.APIKey == "" {
			return fmt.Errorf("API key is required")
		}
	case models.ProviderGemini:
		if cfg.APIKey == "" && cfg.OAuthCredsBase64 == "" && cfg.OAuthCredsFile == "" {
			return fmt.Errorf("either API key or OAuth credentials are required")
		}
		if (cfg.OAuthCredsBase64 != "" || cfg.OAuthCredsFile != "") && cfg.ProjectID == "" {
			return fmt.Errorf("project ID is required when using OAuth")
		}
	}
	return nil
}