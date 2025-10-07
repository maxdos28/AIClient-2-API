package main

import (
	"fmt"
	"log"
	"os"

	"github.com/aiproxy/go-aiproxy/internal/config"
	"github.com/aiproxy/go-aiproxy/internal/server"
	"github.com/spf13/cobra"
)

var (
	cfgFile string
	cfg     *config.Config
)

var rootCmd = &cobra.Command{
	Use:   "go-aiproxy",
	Short: "AI Proxy Server - Bridge between different AI APIs",
	Long: `AI Proxy Server provides a unified interface for multiple AI providers.
It converts between OpenAI, Claude, and Gemini API formats seamlessly.`,
	Run: func(cmd *cobra.Command, args []string) {
		runServer()
	},
}

func init() {
	cobra.OnInitialize(initConfig)

	// Server flags
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is ./config.yaml)")
	rootCmd.Flags().String("host", "localhost", "Server listening address")
	rootCmd.Flags().Int("port", 3000, "Server listening port")
	rootCmd.Flags().String("api-key", "123456", "API key for authentication")

	// Provider flags
	rootCmd.Flags().StringSlice("model-provider", []string{"openai-custom"}, "Model providers (comma-separated)")

	// OpenAI provider flags
	rootCmd.Flags().String("openai-api-key", "", "OpenAI API key")
	rootCmd.Flags().String("openai-base-url", "https://api.openai.com/v1", "OpenAI base URL")

	// Claude provider flags
	rootCmd.Flags().String("claude-api-key", "", "Claude API key")
	rootCmd.Flags().String("claude-base-url", "https://api.anthropic.com", "Claude base URL")

	// Gemini provider flags
	rootCmd.Flags().String("gemini-api-key", "", "Gemini API key")
	rootCmd.Flags().String("gemini-oauth-creds-base64", "", "Gemini OAuth credentials (base64)")
	rootCmd.Flags().String("gemini-oauth-creds-file", "", "Gemini OAuth credentials file")
	rootCmd.Flags().String("project-id", "", "Google Cloud project ID")

	// System prompt flags
	rootCmd.Flags().String("system-prompt-file", "", "System prompt file path")
	rootCmd.Flags().String("system-prompt-mode", "overwrite", "System prompt mode (overwrite/append)")

	// Logging flags
	rootCmd.Flags().String("log-prompts", "none", "Prompt logging mode (none/console/file)")
	rootCmd.Flags().String("prompt-log-base-name", "prompt_log", "Base name for prompt log files")

	// Pool management flags
	rootCmd.Flags().String("provider-pools-file", "", "Provider pools configuration file")
	rootCmd.Flags().Int("request-max-retries", 3, "Maximum retries for failed requests")
	rootCmd.Flags().Int("request-base-delay", 1000, "Base delay between retries (ms)")
}

func initConfig() {
	cfg = config.New()

	// Load from config file if specified
	if cfgFile != "" {
		cfg.LoadFromFile(cfgFile)
	}

	// Override with command line flags
	cfg.LoadFromFlags(rootCmd)

	// Load from environment variables
	cfg.LoadFromEnv()
}

func runServer() {
	// Create and start server
	srv, err := server.New(cfg)
	if err != nil {
		log.Fatalf("Failed to create server: %v", err)
	}

	addr := fmt.Sprintf("%s:%d", cfg.Host, cfg.Port)
	log.Printf("Starting AI Proxy Server on %s", addr)
	log.Printf("API Key: %s", cfg.APIKey)
	log.Printf("Providers: %v", cfg.ModelProviders)

	if err := srv.Start(addr); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
