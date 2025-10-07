package cache

import (
	"testing"
	"time"
)

func TestCacheManager_SetAndGet(t *testing.T) {
	cm := NewCacheManager(5*time.Minute, 10*time.Minute, 100)

	// Test basic set and get
	key := "test-key"
	value := "test-value"

	cm.Set(key, value, 1*time.Minute)

	result, found := cm.Get(key)
	if !found {
		t.Error("Expected to find cached value")
	}

	if result != value {
		t.Errorf("Expected value %s, got %s", value, result)
	}
}

func TestCacheManager_Delete(t *testing.T) {
	cm := NewCacheManager(5*time.Minute, 10*time.Minute, 100)

	key := "test-key"
	value := "test-value"

	cm.Set(key, value, 1*time.Minute)
	cm.Delete(key)

	_, found := cm.Get(key)
	if found {
		t.Error("Expected key to be deleted")
	}
}

func TestCacheManager_Expiration(t *testing.T) {
	cm := NewCacheManager(100*time.Millisecond, 50*time.Millisecond, 100)

	key := "test-key"
	value := "test-value"

	cm.Set(key, value, 100*time.Millisecond)

	// Wait for expiration
	time.Sleep(150 * time.Millisecond)

	_, found := cm.Get(key)
	if found {
		t.Error("Expected key to be expired")
	}
}

func TestCacheManager_Clear(t *testing.T) {
	cm := NewCacheManager(5*time.Minute, 10*time.Minute, 100)

	cm.Set("key1", "value1", 1*time.Minute)
	cm.Set("key2", "value2", 1*time.Minute)
	cm.Set("key3", "value3", 1*time.Minute)

	cm.Clear()

	_, found1 := cm.Get("key1")
	_, found2 := cm.Get("key2")
	_, found3 := cm.Get("key3")

	if found1 || found2 || found3 {
		t.Error("Expected all keys to be cleared")
	}
}

func TestCacheManager_Stats(t *testing.T) {
	cm := NewCacheManager(5*time.Minute, 10*time.Minute, 100)

	key := "test-key"
	value := "test-value"

	cm.Set(key, value, 1*time.Minute)

	// Hit
	_, found := cm.Get(key)
	if !found {
		t.Error("Expected cache hit")
	}

	// Miss
	_, found = cm.Get("nonexistent-key")
	if found {
		t.Error("Expected cache miss")
	}

	stats := cm.GetStats()
	if stats.Hits != 1 {
		t.Errorf("Expected 1 hit, got %d", stats.Hits)
	}
	if stats.Misses != 1 {
		t.Errorf("Expected 1 miss, got %d", stats.Misses)
	}
}

func TestCacheManager_GenerateCacheKey(t *testing.T) {
	cm := NewCacheManager(5*time.Minute, 10*time.Minute, 100)

	req1 := map[string]interface{}{"message": "hello"}
	req2 := map[string]interface{}{"message": "hello"}
	req3 := map[string]interface{}{"message": "world"}

	key1, err := cm.GenerateCacheKey("openai", "gpt-3.5-turbo", req1)
	if err != nil {
		t.Fatalf("GenerateCacheKey failed: %v", err)
	}

	key2, err := cm.GenerateCacheKey("openai", "gpt-3.5-turbo", req2)
	if err != nil {
		t.Fatalf("GenerateCacheKey failed: %v", err)
	}

	key3, err := cm.GenerateCacheKey("openai", "gpt-3.5-turbo", req3)
	if err != nil {
		t.Fatalf("GenerateCacheKey failed: %v", err)
	}

	if key1 != key2 {
		t.Error("Same parameters should generate same key")
	}

	if key1 == key3 {
		t.Error("Different requests should generate different keys")
	}
}

func BenchmarkCacheManager_Set(b *testing.B) {
	cm := NewCacheManager(5*time.Minute, 10*time.Minute, 100)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		cm.Set("test-key", "test-value", 1*time.Minute)
	}
}

func BenchmarkCacheManager_Get(b *testing.B) {
	cm := NewCacheManager(5*time.Minute, 10*time.Minute, 100)
	cm.Set("test-key", "test-value", 1*time.Minute)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		cm.Get("test-key")
	}
}

func BenchmarkCacheManager_GenerateCacheKey(b *testing.B) {
	cm := NewCacheManager(5*time.Minute, 10*time.Minute, 100)
	req := map[string]interface{}{"message": "test"}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = cm.GenerateCacheKey("openai", "gpt-3.5-turbo", req)
	}
}
