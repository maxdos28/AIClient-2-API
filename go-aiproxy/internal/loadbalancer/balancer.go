package loadbalancer

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/aiproxy/go-aiproxy/internal/providers"
	"github.com/aiproxy/go-aiproxy/pkg/models"
)

// Algorithm defines load balancing algorithms
type Algorithm string

const (
	AlgorithmRoundRobin    Algorithm = "round_robin"
	AlgorithmLeastRequests Algorithm = "least_requests"
	AlgorithmWeighted      Algorithm = "weighted"
	AlgorithmRandom        Algorithm = "random"
	AlgorithmIPHash        Algorithm = "ip_hash"
)

// LoadBalancer manages multiple provider instances
type LoadBalancer struct {
	mu            sync.RWMutex
	algorithm     Algorithm
	instances     []*Instance
	currentIndex  uint64
	healthChecker *HealthChecker
	metrics       *BalancerMetrics
}

// Instance represents a provider instance
type Instance struct {
	ID             string
	Provider       providers.Provider
	Config         *models.ProviderConfig
	Weight         int
	ActiveRequests int64
	TotalRequests  int64
	FailedRequests int64
	LastUsed       time.Time
	IsHealthy      bool
	HealthCheckURL string
}

// BalancerMetrics tracks load balancer performance
type BalancerMetrics struct {
	TotalRequests    int64
	FailedRequests   int64
	ActiveRequests   int64
	HealthyInstances int64
	TotalInstances   int64
}

// NewLoadBalancer creates a new load balancer
func NewLoadBalancer(algorithm Algorithm) *LoadBalancer {
	lb := &LoadBalancer{
		algorithm:     algorithm,
		instances:     make([]*Instance, 0),
		healthChecker: NewHealthChecker(30 * time.Second),
		metrics:       &BalancerMetrics{},
	}

	// Start health checking
	go lb.healthChecker.Start()

	return lb
}

// AddInstance adds a provider instance to the load balancer
func (lb *LoadBalancer) AddInstance(id string, provider providers.Provider, config *models.ProviderConfig, weight int) error {
	lb.mu.Lock()
	defer lb.mu.Unlock()

	// Check if instance already exists
	for _, inst := range lb.instances {
		if inst.ID == id {
			return fmt.Errorf("instance %s already exists", id)
		}
	}

	instance := &Instance{
		ID:        id,
		Provider:  provider,
		Config:    config,
		Weight:    weight,
		IsHealthy: true,
		LastUsed:  time.Now(),
	}

	lb.instances = append(lb.instances, instance)
	atomic.AddInt64(&lb.metrics.TotalInstances, 1)
	atomic.AddInt64(&lb.metrics.HealthyInstances, 1)

	// Register with health checker
	lb.healthChecker.Register(instance)

	return nil
}

// RemoveInstance removes a provider instance
func (lb *LoadBalancer) RemoveInstance(id string) error {
	lb.mu.Lock()
	defer lb.mu.Unlock()

	for i, inst := range lb.instances {
		if inst.ID == id {
			// Unregister from health checker
			lb.healthChecker.Unregister(inst)

			// Remove from slice
			lb.instances = append(lb.instances[:i], lb.instances[i+1:]...)
			atomic.AddInt64(&lb.metrics.TotalInstances, -1)
			if inst.IsHealthy {
				atomic.AddInt64(&lb.metrics.HealthyInstances, -1)
			}
			return nil
		}
	}

	return fmt.Errorf("instance %s not found", id)
}

// SelectInstance selects a provider instance based on the algorithm
func (lb *LoadBalancer) SelectInstance(ctx context.Context, clientIP string) (*Instance, error) {
	lb.mu.RLock()
	defer lb.mu.RUnlock()

	// Get healthy instances
	healthyInstances := lb.getHealthyInstances()
	if len(healthyInstances) == 0 {
		return nil, fmt.Errorf("no healthy instances available")
	}

	var selected *Instance

	switch lb.algorithm {
	case AlgorithmRoundRobin:
		selected = lb.selectRoundRobin(healthyInstances)
	case AlgorithmLeastRequests:
		selected = lb.selectLeastRequests(healthyInstances)
	case AlgorithmWeighted:
		selected = lb.selectWeighted(healthyInstances)
	case AlgorithmRandom:
		selected = lb.selectRandom(healthyInstances)
	case AlgorithmIPHash:
		selected = lb.selectIPHash(healthyInstances, clientIP)
	default:
		selected = lb.selectRoundRobin(healthyInstances)
	}

	if selected != nil {
		atomic.AddInt64(&selected.ActiveRequests, 1)
		atomic.AddInt64(&selected.TotalRequests, 1)
		atomic.AddInt64(&lb.metrics.TotalRequests, 1)
		atomic.AddInt64(&lb.metrics.ActiveRequests, 1)
		selected.LastUsed = time.Now()
	}

	return selected, nil
}

// ReleaseInstance releases an instance after request completion
func (lb *LoadBalancer) ReleaseInstance(instance *Instance, failed bool) {
	atomic.AddInt64(&instance.ActiveRequests, -1)
	atomic.AddInt64(&lb.metrics.ActiveRequests, -1)

	if failed {
		atomic.AddInt64(&instance.FailedRequests, 1)
		atomic.AddInt64(&lb.metrics.FailedRequests, 1)
	}
}

// getHealthyInstances returns all healthy instances
func (lb *LoadBalancer) getHealthyInstances() []*Instance {
	var healthy []*Instance
	for _, inst := range lb.instances {
		if inst.IsHealthy {
			healthy = append(healthy, inst)
		}
	}
	return healthy
}

// selectRoundRobin selects instance using round-robin
func (lb *LoadBalancer) selectRoundRobin(instances []*Instance) *Instance {
	if len(instances) == 0 {
		return nil
	}

	index := atomic.AddUint64(&lb.currentIndex, 1) % uint64(len(instances))
	return instances[index]
}

// selectLeastRequests selects instance with least active requests
func (lb *LoadBalancer) selectLeastRequests(instances []*Instance) *Instance {
	if len(instances) == 0 {
		return nil
	}

	selected := instances[0]
	minRequests := atomic.LoadInt64(&selected.ActiveRequests)

	for _, inst := range instances[1:] {
		requests := atomic.LoadInt64(&inst.ActiveRequests)
		if requests < minRequests {
			selected = inst
			minRequests = requests
		}
	}

	return selected
}

// selectWeighted selects instance based on weights
func (lb *LoadBalancer) selectWeighted(instances []*Instance) *Instance {
	if len(instances) == 0 {
		return nil
	}

	// Calculate total weight
	totalWeight := 0
	for _, inst := range instances {
		totalWeight += inst.Weight
	}

	if totalWeight == 0 {
		return lb.selectRoundRobin(instances)
	}

	// Select based on weight
	index := int(atomic.AddUint64(&lb.currentIndex, 1)) % totalWeight
	currentWeight := 0

	for _, inst := range instances {
		currentWeight += inst.Weight
		if index < currentWeight {
			return inst
		}
	}

	return instances[len(instances)-1]
}

// selectRandom selects a random instance
func (lb *LoadBalancer) selectRandom(instances []*Instance) *Instance {
	if len(instances) == 0 {
		return nil
	}

	// Use current time as random seed
	index := time.Now().UnixNano() % int64(len(instances))
	return instances[index]
}

// selectIPHash selects instance based on client IP hash
func (lb *LoadBalancer) selectIPHash(instances []*Instance, clientIP string) *Instance {
	if len(instances) == 0 {
		return nil
	}

	// Simple hash function
	hash := uint32(0)
	for _, b := range []byte(clientIP) {
		hash = hash*31 + uint32(b)
	}

	index := hash % uint32(len(instances))
	return instances[index]
}

// GetMetrics returns current metrics
func (lb *LoadBalancer) GetMetrics() BalancerMetrics {
	return BalancerMetrics{
		TotalRequests:    atomic.LoadInt64(&lb.metrics.TotalRequests),
		FailedRequests:   atomic.LoadInt64(&lb.metrics.FailedRequests),
		ActiveRequests:   atomic.LoadInt64(&lb.metrics.ActiveRequests),
		HealthyInstances: atomic.LoadInt64(&lb.metrics.HealthyInstances),
		TotalInstances:   atomic.LoadInt64(&lb.metrics.TotalInstances),
	}
}

// UpdateInstanceHealth updates the health status of an instance
func (lb *LoadBalancer) UpdateInstanceHealth(instanceID string, healthy bool) {
	lb.mu.Lock()
	defer lb.mu.Unlock()

	for _, inst := range lb.instances {
		if inst.ID == instanceID {
			if inst.IsHealthy != healthy {
				inst.IsHealthy = healthy
				if healthy {
					atomic.AddInt64(&lb.metrics.HealthyInstances, 1)
				} else {
					atomic.AddInt64(&lb.metrics.HealthyInstances, -1)
				}
			}
			break
		}
	}
}

// GetInstances returns all instances
func (lb *LoadBalancer) GetInstances() []*Instance {
	lb.mu.RLock()
	defer lb.mu.RUnlock()

	// Return a copy to avoid race conditions
	instances := make([]*Instance, len(lb.instances))
	copy(instances, lb.instances)
	return instances
}

// SetAlgorithm changes the load balancing algorithm
func (lb *LoadBalancer) SetAlgorithm(algorithm Algorithm) {
	lb.mu.Lock()
	defer lb.mu.Unlock()
	lb.algorithm = algorithm
}

// Close stops the load balancer and health checker
func (lb *LoadBalancer) Close() {
	lb.healthChecker.Stop()
}
