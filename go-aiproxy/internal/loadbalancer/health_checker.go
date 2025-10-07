package loadbalancer

import (
	"context"
	"sync"
	"time"
)

// HealthChecker performs periodic health checks on instances
type HealthChecker struct {
	mu              sync.RWMutex
	instances       map[string]*Instance
	checkInterval   time.Duration
	checkTimeout    time.Duration
	stopChan        chan struct{}
	updateCallbacks []func(instanceID string, healthy bool)
}

// NewHealthChecker creates a new health checker
func NewHealthChecker(checkInterval time.Duration) *HealthChecker {
	return &HealthChecker{
		instances:       make(map[string]*Instance),
		checkInterval:   checkInterval,
		checkTimeout:    5 * time.Second,
		stopChan:        make(chan struct{}),
		updateCallbacks: make([]func(string, bool), 0),
	}
}

// Register registers an instance for health checking
func (hc *HealthChecker) Register(instance *Instance) {
	hc.mu.Lock()
	defer hc.mu.Unlock()
	hc.instances[instance.ID] = instance
}

// Unregister removes an instance from health checking
func (hc *HealthChecker) Unregister(instance *Instance) {
	hc.mu.Lock()
	defer hc.mu.Unlock()
	delete(hc.instances, instance.ID)
}

// AddUpdateCallback adds a callback for health status updates
func (hc *HealthChecker) AddUpdateCallback(callback func(instanceID string, healthy bool)) {
	hc.mu.Lock()
	defer hc.mu.Unlock()
	hc.updateCallbacks = append(hc.updateCallbacks, callback)
}

// Start begins the health checking routine
func (hc *HealthChecker) Start() {
	ticker := time.NewTicker(hc.checkInterval)
	defer ticker.Stop()

	// Initial check
	hc.checkAll()

	for {
		select {
		case <-ticker.C:
			hc.checkAll()
		case <-hc.stopChan:
			return
		}
	}
}

// Stop stops the health checking routine
func (hc *HealthChecker) Stop() {
	close(hc.stopChan)
}

// checkAll performs health checks on all registered instances
func (hc *HealthChecker) checkAll() {
	hc.mu.RLock()
	instances := make([]*Instance, 0, len(hc.instances))
	for _, inst := range hc.instances {
		instances = append(instances, inst)
	}
	hc.mu.RUnlock()

	// Check each instance concurrently
	var wg sync.WaitGroup
	for _, inst := range instances {
		wg.Add(1)
		go func(instance *Instance) {
			defer wg.Done()
			hc.checkInstance(instance)
		}(inst)
	}
	wg.Wait()
}

// checkInstance performs a health check on a single instance
func (hc *HealthChecker) checkInstance(instance *Instance) {
	ctx, cancel := context.WithTimeout(context.Background(), hc.checkTimeout)
	defer cancel()

	// Perform health check
	healthy := instance.Provider.IsHealthy()

	// Additional check: try to list models as a health check
	if healthy {
		_, err := instance.Provider.ListModels(ctx)
		healthy = err == nil
	}

	// Update health status if changed
	if instance.IsHealthy != healthy {
		instance.IsHealthy = healthy

		// Notify callbacks
		hc.mu.RLock()
		callbacks := make([]func(string, bool), len(hc.updateCallbacks))
		copy(callbacks, hc.updateCallbacks)
		hc.mu.RUnlock()

		for _, callback := range callbacks {
			callback(instance.ID, healthy)
		}
	}
}

// ForceCheck forces an immediate health check on a specific instance
func (hc *HealthChecker) ForceCheck(instanceID string) {
	hc.mu.RLock()
	instance, ok := hc.instances[instanceID]
	hc.mu.RUnlock()

	if ok {
		hc.checkInstance(instance)
	}
}
