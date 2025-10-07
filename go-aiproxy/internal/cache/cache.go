package cache

import (
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/patrickmn/go-cache"
)

// CacheManager manages request/response caching
type CacheManager struct {
	cache       *cache.Cache
	mu          sync.RWMutex
	stats       *CacheStats
	enabled     bool
	maxSize     int64
	currentSize int64
}

// CacheStats tracks cache performance
type CacheStats struct {
	Hits       int64
	Misses     int64
	Evictions  int64
	TotalBytes int64
}

// CacheEntry represents a cached item
type CacheEntry struct {
	Key        string
	Value      interface{}
	Size       int64
	CreatedAt  time.Time
	AccessedAt time.Time
	HitCount   int64
}

// NewCacheManager creates a new cache manager
func NewCacheManager(defaultExpiration, cleanupInterval time.Duration, maxSizeMB int64) *CacheManager {
	return &CacheManager{
		cache:   cache.New(defaultExpiration, cleanupInterval),
		stats:   &CacheStats{},
		enabled: true,
		maxSize: maxSizeMB * 1024 * 1024, // Convert MB to bytes
	}
}

// GenerateCacheKey creates a unique cache key from request data
func (cm *CacheManager) GenerateCacheKey(provider, model string, request interface{}) (string, error) {
	// Serialize request to JSON for consistent hashing
	data, err := json.Marshal(map[string]interface{}{
		"provider": provider,
		"model":    model,
		"request":  request,
	})
	if err != nil {
		return "", fmt.Errorf("failed to marshal request for cache key: %w", err)
	}

	// Generate MD5 hash
	hash := md5.Sum(data)
	return hex.EncodeToString(hash[:]), nil
}

// Get retrieves an item from cache
func (cm *CacheManager) Get(key string) (interface{}, bool) {
	if !cm.enabled {
		return nil, false
	}

	cm.mu.RLock()
	defer cm.mu.RUnlock()

	if item, found := cm.cache.Get(key); found {
		cm.stats.Hits++
		
		// Update access time if it's a CacheEntry
		if entry, ok := item.(*CacheEntry); ok {
			entry.AccessedAt = time.Now()
			entry.HitCount++
			return entry.Value, true
		}
		
		return item, true
	}

	cm.stats.Misses++
	return nil, false
}

// Set stores an item in cache with size tracking
func (cm *CacheManager) Set(key string, value interface{}, duration time.Duration) error {
	if !cm.enabled {
		return nil
	}

	// Estimate size of the cached item
	size := cm.estimateSize(value)

	cm.mu.Lock()
	defer cm.mu.Unlock()

	// Check if we need to evict items to make space
	if cm.currentSize+size > cm.maxSize {
		cm.evictLRU(size)
	}

	entry := &CacheEntry{
		Key:        key,
		Value:      value,
		Size:       size,
		CreatedAt:  time.Now(),
		AccessedAt: time.Now(),
		HitCount:   0,
	}

	cm.cache.Set(key, entry, duration)
	cm.currentSize += size
	cm.stats.TotalBytes = cm.currentSize

	return nil
}

// Delete removes an item from cache
func (cm *CacheManager) Delete(key string) {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	if item, found := cm.cache.Get(key); found {
		if entry, ok := item.(*CacheEntry); ok {
			cm.currentSize -= entry.Size
			cm.stats.TotalBytes = cm.currentSize
		}
		cm.cache.Delete(key)
	}
}

// Clear removes all items from cache
func (cm *CacheManager) Clear() {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	cm.cache.Flush()
	cm.currentSize = 0
	cm.stats.TotalBytes = 0
}

// GetStats returns cache statistics
func (cm *CacheManager) GetStats() CacheStats {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	return *cm.stats
}

// SetEnabled enables or disables caching
func (cm *CacheManager) SetEnabled(enabled bool) {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	cm.enabled = enabled
}

// IsEnabled returns whether caching is enabled
func (cm *CacheManager) IsEnabled() bool {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	return cm.enabled
}

// estimateSize estimates the size of a value in bytes
func (cm *CacheManager) estimateSize(value interface{}) int64 {
	// Simple estimation using JSON serialization
	data, err := json.Marshal(value)
	if err != nil {
		return 1024 // Default 1KB if we can't estimate
	}
	return int64(len(data))
}

// evictLRU evicts least recently used items to make space
func (cm *CacheManager) evictLRU(neededSize int64) {
	items := cm.cache.Items()
	
	// Convert to slice for sorting
	type cacheItem struct {
		key   string
		entry *CacheEntry
	}
	
	var sortedItems []cacheItem
	for k, v := range items {
		if entry, ok := v.Object.(*CacheEntry); ok {
			sortedItems = append(sortedItems, cacheItem{key: k, entry: entry})
		}
	}
	
	// Sort by access time (oldest first)
	for i := 0; i < len(sortedItems)-1; i++ {
		for j := i + 1; j < len(sortedItems); j++ {
			if sortedItems[i].entry.AccessedAt.After(sortedItems[j].entry.AccessedAt) {
				sortedItems[i], sortedItems[j] = sortedItems[j], sortedItems[i]
			}
		}
	}
	
	// Evict items until we have enough space
	freedSpace := int64(0)
	for _, item := range sortedItems {
		if freedSpace >= neededSize {
			break
		}
		
		cm.cache.Delete(item.key)
		cm.currentSize -= item.entry.Size
		freedSpace += item.entry.Size
		cm.stats.Evictions++
	}
}

// CacheMiddleware provides HTTP middleware for caching
type CacheMiddleware struct {
	manager *CacheManager
}

// NewCacheMiddleware creates a new cache middleware
func NewCacheMiddleware(manager *CacheManager) *CacheMiddleware {
	return &CacheMiddleware{
		manager: manager,
	}
}

// ShouldCache determines if a request should be cached
func (m *CacheMiddleware) ShouldCache(method, path string, isStream bool) bool {
	// Only cache GET requests and non-streaming POST completions
	if method == "GET" {
		return true
	}
	
	if method == "POST" && !isStream {
		// Cache completion requests but not streaming ones
		return strings.Contains(path, "/completions") || 
		       strings.Contains(path, "/generateContent")
	}
	
	return false
}

// CacheDuration returns the cache duration for different request types
func (m *CacheMiddleware) CacheDuration(path string) time.Duration {
	// Model listings can be cached longer
	if strings.Contains(path, "/models") {
		return 1 * time.Hour
	}
	
	// Completion requests cached for shorter duration
	return 5 * time.Minute
}