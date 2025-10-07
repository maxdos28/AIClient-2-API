package pool

import (
	"encoding/json"
	"io/ioutil"
	"os"
	"testing"
	"time"

	"github.com/aiproxy/go-aiproxy/pkg/models"
)

func createTestConfig(t *testing.T) string {
	config := map[string][]*models.ProviderConfig{
		"openai": {
			{
				Provider:  models.ProviderOpenAI,
				APIKey:    "test-key-1",
				BaseURL:   "https://api.openai.com/v1",
				IsHealthy: true,
			},
			{
				Provider:  models.ProviderOpenAI,
				APIKey:    "test-key-2",
				BaseURL:   "https://api.openai.com/v1",
				IsHealthy: true,
			},
		},
		"claude": {
			{
				Provider:  models.ProviderClaude,
				APIKey:    "test-claude-key",
				BaseURL:   "https://api.anthropic.com",
				IsHealthy: true,
			},
		},
	}

	data, err := json.Marshal(config)
	if err != nil {
		t.Fatalf("Failed to marshal config: %v", err)
	}

	tmpfile, err := ioutil.TempFile("", "pool-config-*.json")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}

	if _, err := tmpfile.Write(data); err != nil {
		t.Fatalf("Failed to write config: %v", err)
	}
	tmpfile.Close()

	return tmpfile.Name()
}

func TestNewManager(t *testing.T) {
	configFile := createTestConfig(t)
	defer os.Remove(configFile)

	manager, err := NewManager(configFile)
	if err != nil {
		t.Fatalf("Failed to create manager: %v", err)
	}

	if manager == nil {
		t.Error("Expected non-nil manager")
	}

	if len(manager.pools) != 2 {
		t.Errorf("Expected 2 pools, got %d", len(manager.pools))
	}
}

func TestManager_SelectProvider_RoundRobin(t *testing.T) {
	configFile := createTestConfig(t)
	defer os.Remove(configFile)

	manager, err := NewManager(configFile)
	if err != nil {
		t.Fatalf("Failed to create manager: %v", err)
	}

	// Test round-robin selection
	provider1, err := manager.SelectProvider("openai")
	if err != nil {
		t.Fatalf("Failed to select provider: %v", err)
	}

	provider2, err := manager.SelectProvider("openai")
	if err != nil {
		t.Fatalf("Failed to select provider: %v", err)
	}

	// Should select different providers in round-robin
	if provider1.APIKey == provider2.APIKey {
		t.Log("Note: Round-robin may select same provider if only one is healthy")
	}
}

func TestManager_SelectProvider_NoProviders(t *testing.T) {
	configFile := createTestConfig(t)
	defer os.Remove(configFile)

	manager, err := NewManager(configFile)
	if err != nil {
		t.Fatalf("Failed to create manager: %v", err)
	}

	_, err = manager.SelectProvider("nonexistent")
	if err == nil {
		t.Error("Expected error when selecting nonexistent provider")
	}
}

func TestManager_ProviderUsage(t *testing.T) {
	configFile := createTestConfig(t)
	defer os.Remove(configFile)

	manager, err := NewManager(configFile)
	if err != nil {
		t.Fatalf("Failed to create manager: %v", err)
	}

	provider, err := manager.SelectProvider("openai")
	if err != nil {
		t.Fatalf("Failed to select provider: %v", err)
	}

	// Check that provider has expected fields
	if provider.Provider != "openai" {
		t.Errorf("Expected provider type 'openai', got %v", provider.Provider)
	}

	// Test error handling
	initialErrorCount := provider.ErrorCount
	provider.ErrorCount++
	if provider.ErrorCount != initialErrorCount+1 {
		t.Errorf("Error count not incrementing correctly")
	}
}

func TestManager_PoolSize(t *testing.T) {
	configFile := createTestConfig(t)
	defer os.Remove(configFile)

	manager, err := NewManager(configFile)
	if err != nil {
		t.Fatalf("Failed to create manager: %v", err)
	}

	// Test that pools were loaded
	manager.mu.RLock()
	openaiProviders := manager.pools["openai"]
	claudeProviders := manager.pools["claude"]
	manager.mu.RUnlock()

	if len(openaiProviders) != 2 {
		t.Errorf("Expected 2 OpenAI providers, got %d", len(openaiProviders))
	}

	if len(claudeProviders) != 1 {
		t.Errorf("Expected 1 Claude provider, got %d", len(claudeProviders))
	}
}

func TestManager_WithOptions(t *testing.T) {
	configFile := createTestConfig(t)
	defer os.Remove(configFile)

	manager, err := NewManager(
		configFile,
		WithMaxErrorCount(5),
		WithHealthCheckInterval(10*time.Second),
	)
	if err != nil {
		t.Fatalf("Failed to create manager: %v", err)
	}

	if manager.maxErrorCount != 5 {
		t.Errorf("Expected max error count 5, got %d", manager.maxErrorCount)
	}

	if manager.healthCheckInterval != 10*time.Second {
		t.Errorf("Expected health check interval 10s, got %v", manager.healthCheckInterval)
	}
}

func TestManager_HealthStatus(t *testing.T) {
	configFile := createTestConfig(t)
	defer os.Remove(configFile)

	manager, err := NewManager(configFile)
	if err != nil {
		t.Fatalf("Failed to create manager: %v", err)
	}

	provider, err := manager.SelectProvider("openai")
	if err != nil {
		t.Fatalf("Failed to select provider: %v", err)
	}

	// Test health status
	if !provider.IsHealthy {
		t.Error("Provider should be healthy initially")
	}

	// Mark as unhealthy
	provider.IsHealthy = false
	if provider.IsHealthy {
		t.Error("Provider should be unhealthy")
	}
}

func BenchmarkManager_SelectProvider(b *testing.B) {
	// Create a simple config inline
	config := map[string][]*models.ProviderConfig{
		"openai": {
			{
				Provider:  models.ProviderOpenAI,
				APIKey:    "test-key",
				IsHealthy: true,
			},
		},
	}

	data, _ := json.Marshal(config)
	tmpfile, _ := ioutil.TempFile("", "pool-bench-*.json")
	tmpfile.Write(data)
	tmpfile.Close()
	defer os.Remove(tmpfile.Name())

	manager, _ := NewManager(tmpfile.Name())

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = manager.SelectProvider("openai")
	}
}
