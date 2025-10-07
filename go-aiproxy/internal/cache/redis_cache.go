package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/go-redis/redis/v8"
)

// RedisCache implements distributed caching using Redis
type RedisCache struct {
	client  *redis.Client
	prefix  string
	enabled bool
}

// RedisConfig holds Redis connection configuration
type RedisConfig struct {
	Addr     string
	Password string
	DB       int
	Prefix   string
}

// NewRedisCache creates a new Redis cache instance
func NewRedisCache(config RedisConfig) (*RedisCache, error) {
	client := redis.NewClient(&redis.Options{
		Addr:     config.Addr,
		Password: config.Password,
		DB:       config.DB,
	})

	// Test connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("failed to connect to Redis: %w", err)
	}

	return &RedisCache{
		client:  client,
		prefix:  config.Prefix,
		enabled: true,
	}, nil
}

// Get retrieves a value from Redis
func (rc *RedisCache) Get(ctx context.Context, key string) (interface{}, error) {
	if !rc.enabled {
		return nil, fmt.Errorf("cache disabled")
	}

	fullKey := rc.prefix + key
	val, err := rc.client.Get(ctx, fullKey).Result()
	if err == redis.Nil {
		return nil, nil // Key not found
	} else if err != nil {
		return nil, fmt.Errorf("failed to get from Redis: %w", err)
	}

	// Deserialize the value
	var result interface{}
	if err := json.Unmarshal([]byte(val), &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal cached value: %w", err)
	}

	// Update access count
	rc.client.HIncrBy(ctx, fullKey+":stats", "hits", 1)

	return result, nil
}

// Set stores a value in Redis with expiration
func (rc *RedisCache) Set(ctx context.Context, key string, value interface{}, expiration time.Duration) error {
	if !rc.enabled {
		return nil
	}

	// Serialize the value
	data, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("failed to marshal value: %w", err)
	}

	fullKey := rc.prefix + key
	if err := rc.client.Set(ctx, fullKey, data, expiration).Err(); err != nil {
		return fmt.Errorf("failed to set in Redis: %w", err)
	}

	// Store metadata
	stats := map[string]interface{}{
		"created": time.Now().Unix(),
		"size":    len(data),
		"hits":    0,
		"expiry":  time.Now().Add(expiration).Unix(),
	}

	rc.client.HMSet(ctx, fullKey+":stats", stats)
	rc.client.Expire(ctx, fullKey+":stats", expiration)

	return nil
}

// Delete removes a value from Redis
func (rc *RedisCache) Delete(ctx context.Context, key string) error {
	fullKey := rc.prefix + key

	// Delete both value and stats
	pipe := rc.client.Pipeline()
	pipe.Del(ctx, fullKey)
	pipe.Del(ctx, fullKey+":stats")

	_, err := pipe.Exec(ctx)
	return err
}

// Clear removes all cached items with the prefix
func (rc *RedisCache) Clear(ctx context.Context) error {
	// Use SCAN to find all keys with our prefix
	iter := rc.client.Scan(ctx, 0, rc.prefix+"*", 0).Iterator()

	pipe := rc.client.Pipeline()
	count := 0

	for iter.Next(ctx) {
		pipe.Del(ctx, iter.Val())
		count++

		// Execute in batches
		if count%100 == 0 {
			if _, err := pipe.Exec(ctx); err != nil {
				return fmt.Errorf("failed to clear cache: %w", err)
			}
			pipe = rc.client.Pipeline()
		}
	}

	// Execute remaining deletes
	if count%100 != 0 {
		if _, err := pipe.Exec(ctx); err != nil {
			return fmt.Errorf("failed to clear cache: %w", err)
		}
	}

	return iter.Err()
}

// GetStats retrieves cache statistics from Redis
func (rc *RedisCache) GetStats(ctx context.Context) (map[string]int64, error) {
	stats := make(map[string]int64)

	// Count total keys
	iter := rc.client.Scan(ctx, 0, rc.prefix+"*", 0).Iterator()
	keyCount := int64(0)
	totalSize := int64(0)
	totalHits := int64(0)

	for iter.Next(ctx) {
		key := iter.Val()
		if !strings.HasSuffix(key, ":stats") {
			keyCount++

			// Get stats for this key
			if statsData, err := rc.client.HGetAll(ctx, key+":stats").Result(); err == nil {
				if size, ok := statsData["size"]; ok {
					if s, err := parseInt64(size); err == nil {
						totalSize += s
					}
				}
				if hits, ok := statsData["hits"]; ok {
					if h, err := parseInt64(hits); err == nil {
						totalHits += h
					}
				}
			}
		}
	}

	stats["keys"] = keyCount
	stats["bytes"] = totalSize
	stats["hits"] = totalHits

	return stats, iter.Err()
}

// SetEnabled enables or disables the cache
func (rc *RedisCache) SetEnabled(enabled bool) {
	rc.enabled = enabled
}

// Close closes the Redis connection
func (rc *RedisCache) Close() error {
	return rc.client.Close()
}

// Helper function to parse int64
func parseInt64(s string) (int64, error) {
	var i int64
	_, err := fmt.Sscanf(s, "%d", &i)
	return i, err
}
