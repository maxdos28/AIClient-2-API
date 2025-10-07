use std::collections::HashMap;
use std::sync::Arc;
use std::time::{Duration, Instant};
use tokio::sync::RwLock;

#[derive(Clone)]
pub struct CacheEntry {
    pub data: String,
    pub expires_at: Instant,
}

pub struct Cache {
    store: Arc<RwLock<HashMap<String, CacheEntry>>>,
    default_ttl: Duration,
}

impl Cache {
    pub fn new(default_ttl: Duration) -> Self {
        Self {
            store: Arc::new(RwLock::new(HashMap::new())),
            default_ttl,
        }
    }

    pub async fn get(&self, key: &str) -> Option<String> {
        let store = self.store.read().await;
        if let Some(entry) = store.get(key) {
            if entry.expires_at > Instant::now() {
                return Some(entry.data.clone());
            }
        }
        None
    }

    pub async fn set(&self, key: String, data: String) {
        self.set_with_ttl(key, data, self.default_ttl).await;
    }

    pub async fn set_with_ttl(&self, key: String, data: String, ttl: Duration) {
        let entry = CacheEntry {
            data,
            expires_at: Instant::now() + ttl,
        };
        
        let mut store = self.store.write().await;
        store.insert(key, entry);
    }

    pub async fn delete(&self, key: &str) {
        let mut store = self.store.write().await;
        store.remove(key);
    }

    pub async fn clear(&self) {
        let mut store = self.store.write().await;
        store.clear();
    }

    pub async fn cleanup_expired(&self) {
        let mut store = self.store.write().await;
        let now = Instant::now();
        store.retain(|_, entry| entry.expires_at > now);
    }

    pub fn generate_key(&self, provider: &str, model: &str, request: &str) -> String {
        format!("{:x}", md5::compute(format!("{}:{}:{}", provider, model, request)))
    }
}

impl Default for Cache {
    fn default() -> Self {
        Self::new(Duration::from_secs(300)) // 5 minutes default
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[tokio::test]
    async fn test_cache_set_get() {
        let cache = Cache::new(Duration::from_secs(60));
        
        cache.set("key1".to_string(), "value1".to_string()).await;
        
        let result = cache.get("key1").await;
        assert_eq!(result, Some("value1".to_string()));
    }

    #[tokio::test]
    async fn test_cache_expiration() {
        let cache = Cache::new(Duration::from_millis(100));
        
        cache.set("key1".to_string(), "value1".to_string()).await;
        tokio::time::sleep(Duration::from_millis(150)).await;
        
        let result = cache.get("key1").await;
        assert_eq!(result, None);
    }

    #[tokio::test]
    async fn test_cache_delete() {
        let cache = Cache::new(Duration::from_secs(60));
        
        cache.set("key1".to_string(), "value1".to_string()).await;
        cache.delete("key1").await;
        
        let result = cache.get("key1").await;
        assert_eq!(result, None);
    }

    #[test]
    fn test_generate_key() {
        let cache = Cache::default();
        
        let key1 = cache.generate_key("openai", "gpt-3.5", "hello");
        let key2 = cache.generate_key("openai", "gpt-3.5", "hello");
        let key3 = cache.generate_key("openai", "gpt-4", "hello");
        
        assert_eq!(key1, key2);
        assert_ne!(key1, key3);
    }
}
