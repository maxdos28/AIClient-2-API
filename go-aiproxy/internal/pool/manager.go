package pool

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"sync"
	"time"

	"github.com/aiproxy/go-aiproxy/pkg/models"
)

// Manager manages pools of providers
type Manager struct {
	mu                  sync.RWMutex
	pools               map[string][]*models.ProviderConfig
	roundRobinIndex     map[string]int
	maxErrorCount       int
	healthCheckInterval time.Duration
}

// NewManager creates a new pool manager
func NewManager(configFile string, options ...Option) (*Manager, error) {
	m := &Manager{
		pools:               make(map[string][]*models.ProviderConfig),
		roundRobinIndex:     make(map[string]int),
		maxErrorCount:       3,
		healthCheckInterval: 30 * time.Minute,
	}

	// Apply options
	for _, opt := range options {
		opt(m)
	}

	// Load configuration
	if err := m.LoadConfig(configFile); err != nil {
		return nil, err
	}

	// Start health check routine
	go m.healthCheckLoop()

	return m, nil
}

// Option is a configuration option for the manager
type Option func(*Manager)

// WithMaxErrorCount sets the maximum error count before marking unhealthy
func WithMaxErrorCount(count int) Option {
	return func(m *Manager) {
		m.maxErrorCount = count
	}
}

// WithHealthCheckInterval sets the health check interval
func WithHealthCheckInterval(interval time.Duration) Option {
	return func(m *Manager) {
		m.healthCheckInterval = interval
	}
}

// LoadConfig loads provider pools from a JSON file
func (m *Manager) LoadConfig(filename string) error {
	data, err := ioutil.ReadFile(filename)
	if err != nil {
		return fmt.Errorf("failed to read config file: %w", err)
	}

	var config map[string][]*models.ProviderConfig
	if err := json.Unmarshal(data, &config); err != nil {
		return fmt.Errorf("failed to parse config: %w", err)
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	m.pools = config

	// Initialize round-robin indices
	for providerType := range config {
		m.roundRobinIndex[providerType] = 0
	}

	return nil
}

// SelectProvider selects a healthy provider using round-robin
func (m *Manager) SelectProvider(providerType string) (*models.ProviderConfig, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	providers, ok := m.pools[providerType]
	if !ok || len(providers) == 0 {
		return nil, fmt.Errorf("no providers available for type: %s", providerType)
	}

	// Find healthy providers
	var healthyProviders []*models.ProviderConfig
	for _, p := range providers {
		if p.IsHealthy {
			healthyProviders = append(healthyProviders, p)
		}
	}

	if len(healthyProviders) == 0 {
		return nil, fmt.Errorf("no healthy providers available for type: %s", providerType)
	}

	// Round-robin selection
	index := m.roundRobinIndex[providerType] % len(healthyProviders)
	selected := healthyProviders[index]

	// Update round-robin index
	m.roundRobinIndex[providerType] = (index + 1) % len(healthyProviders)

	// Update usage statistics
	selected.LastUsed = timePtr(time.Now())
	selected.UsageCount++

	return selected, nil
}

// MarkProviderUnhealthy marks a provider as unhealthy
func (m *Manager) MarkProviderUnhealthy(providerType string, providerUUID string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	providers, ok := m.pools[providerType]
	if !ok {
		return
	}

	for _, p := range providers {
		if p.UUID == providerUUID {
			p.ErrorCount++
			p.LastErrorTime = timePtr(time.Now())

			if p.ErrorCount >= m.maxErrorCount {
				p.IsHealthy = false
			}
			break
		}
	}
}

// MarkProviderHealthy marks a provider as healthy
func (m *Manager) MarkProviderHealthy(providerType string, providerUUID string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	providers, ok := m.pools[providerType]
	if !ok {
		return
	}

	for _, p := range providers {
		if p.UUID == providerUUID {
			p.IsHealthy = true
			p.ErrorCount = 0
			p.LastErrorTime = nil
			break
		}
	}
}

// healthCheckLoop performs periodic health checks
func (m *Manager) healthCheckLoop() {
	ticker := time.NewTicker(m.healthCheckInterval)
	defer ticker.Stop()

	for range ticker.C {
		m.performHealthChecks()
	}
}

// performHealthChecks checks the health of all providers
func (m *Manager) performHealthChecks() {
	m.mu.RLock()
	allProviders := make(map[string][]*models.ProviderConfig)
	for k, v := range m.pools {
		allProviders[k] = v
	}
	m.mu.RUnlock()

	// Perform health checks (implementation would depend on specific provider types)
	// For now, this is a placeholder
	for providerType, providers := range allProviders {
		for _, provider := range providers {
			if !provider.IsHealthy && provider.LastErrorTime != nil {
				// Check if enough time has passed since last error
				if time.Since(*provider.LastErrorTime) > m.healthCheckInterval {
					// In a real implementation, we would test the provider here
					// For now, we'll just mark it as healthy
					m.MarkProviderHealthy(providerType, provider.UUID)
				}
			}
		}
	}
}

// timePtr is a helper function to get a pointer to a time
func timePtr(t time.Time) *time.Time {
	return &t
}
