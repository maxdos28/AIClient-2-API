package server

import (
	"context"
	"fmt"
	"time"

	"github.com/aiproxy/go-aiproxy/internal/cache"
	"github.com/aiproxy/go-aiproxy/internal/loadbalancer"
	"github.com/aiproxy/go-aiproxy/internal/metrics"
	"github.com/aiproxy/go-aiproxy/internal/providers/kiro"
	"github.com/aiproxy/go-aiproxy/internal/providers/qwen"
	"github.com/aiproxy/go-aiproxy/internal/websocket"
	"github.com/aiproxy/go-aiproxy/pkg/models"
	"github.com/gin-gonic/gin"
)

// EnhancedServer extends the base server with advanced features
type EnhancedServer struct {
	*Server
	cacheManager *cache.CacheManager
	redisCache   *cache.RedisCache
	wsHub        *websocket.Hub
	wsHandler    *websocket.WebSocketHandler
	metrics      *metrics.Metrics
	dashboard    *metrics.MetricsDashboard
	loadBalancer *loadbalancer.LoadBalancer
	cluster      *loadbalancer.Cluster
}

// NewEnhancedServer creates a server with all advanced features
func NewEnhancedServer(cfg *config.Config) (*EnhancedServer, error) {
	// Create base server
	base, err := New(cfg)
	if err != nil {
		return nil, err
	}

	s := &EnhancedServer{
		Server: base,
	}

	// Initialize cache
	if err := s.initializeCache(cfg); err != nil {
		return nil, fmt.Errorf("failed to initialize cache: %w", err)
	}

	// Initialize metrics
	s.initializeMetrics()

	// Initialize WebSocket
	s.initializeWebSocket()

	// Initialize load balancer
	if err := s.initializeLoadBalancer(cfg); err != nil {
		return nil, fmt.Errorf("failed to initialize load balancer: %w", err)
	}

	// Add new providers
	if err := s.addEnhancedProviders(); err != nil {
		return nil, fmt.Errorf("failed to add enhanced providers: %w", err)
	}

	// Setup enhanced routes
	s.setupEnhancedRoutes()

	return s, nil
}

// initializeCache sets up caching system
func (s *EnhancedServer) initializeCache(cfg *config.Config) error {
	// Initialize in-memory cache
	s.cacheManager = cache.NewCacheManager(
		5*time.Minute,  // Default expiration
		10*time.Minute, // Cleanup interval
		100,            // Max size in MB
	)

	// Initialize Redis cache if configured
	if cfg.RedisAddr != "" {
		redisCache, err := cache.NewRedisCache(cache.RedisConfig{
			Addr:     cfg.RedisAddr,
			Password: cfg.RedisPassword,
			DB:       cfg.RedisDB,
			Prefix:   "aiproxy:",
		})
		if err != nil {
			return err
		}
		s.redisCache = redisCache
	}

	return nil
}

// initializeMetrics sets up Prometheus metrics
func (s *EnhancedServer) initializeMetrics() {
	s.metrics = metrics.NewMetrics()
	
	// Start system metrics collector
	s.metrics.CollectSystemMetrics(10 * time.Second)

	// Initialize dashboard
	dashboard, _ := metrics.NewDashboard(&metrics.DashboardConfig{
		Title:           "AI Proxy Metrics",
		RefreshInterval: 5,
		PrometheusURL:   "/metrics",
	})
	s.dashboard = dashboard
}

// initializeWebSocket sets up WebSocket support
func (s *EnhancedServer) initializeWebSocket() {
	s.wsHub = websocket.NewHub()
	s.wsHandler = websocket.NewWebSocketHandler(s.wsHub, s.providers, s.config.APIKey)

	// Set metrics callback
	s.wsHub.SetMetricsCallback(func(connections int) {
		s.metrics.WSConnections.Set(float64(connections))
	})

	// Start WebSocket hub
	go s.wsHub.Run(context.Background())
}

// initializeLoadBalancer sets up load balancing
func (s *EnhancedServer) initializeLoadBalancer(cfg *config.Config) error {
	// Create load balancer
	algorithm := loadbalancer.AlgorithmRoundRobin
	if cfg.LoadBalancerAlgorithm != "" {
		algorithm = loadbalancer.Algorithm(cfg.LoadBalancerAlgorithm)
	}
	
	s.loadBalancer = loadbalancer.NewLoadBalancer(algorithm)

	// Add instances from pool configuration
	for providerType, configs := range s.poolManager.pools {
		for i, config := range configs {
			instanceID := fmt.Sprintf("%s-%d", providerType, i)
			provider := s.providers[providerType]
			
			weight := 1
			if config.Weight > 0 {
				weight = config.Weight
			}
			
			s.loadBalancer.AddInstance(instanceID, provider, config, weight)
		}
	}

	// Set up health check callback
	s.loadBalancer.healthChecker.AddUpdateCallback(func(instanceID string, healthy bool) {
		s.loadBalancer.UpdateInstanceHealth(instanceID, healthy)
		
		// Update metrics
		if healthy {
			s.metrics.PoolHealthyProviders.WithLabelValues(instanceID).Inc()
		} else {
			s.metrics.PoolHealthyProviders.WithLabelValues(instanceID).Dec()
		}
	})

	// Initialize cluster if configured
	if cfg.ClusterEnabled {
		s.cluster = loadbalancer.NewCluster(cfg.NodeID, cfg.NodeAddress)
		if err := s.cluster.Join(cfg.SeedNodes); err != nil {
			return fmt.Errorf("failed to join cluster: %w", err)
		}
	}

	return nil
}

// addEnhancedProviders adds Kiro and Qwen providers
func (s *EnhancedServer) addEnhancedProviders() error {
	// Add Kiro provider
	if kiroConfig, ok := s.config.ProviderConfigs["kiro-api"]; ok {
		kiroClient, err := kiro.NewClient(kiroConfig)
		if err != nil {
			return fmt.Errorf("failed to create Kiro client: %w", err)
		}
		s.providers["kiro-api"] = kiroClient
	}

	// Add Qwen provider
	if qwenConfig, ok := s.config.ProviderConfigs["qwen-api"]; ok {
		qwenClient, err := qwen.NewClient(qwenConfig)
		if err != nil {
			return fmt.Errorf("failed to create Qwen client: %w", err)
		}
		s.providers["qwen-api"] = qwenClient
	}

	return nil
}

// setupEnhancedRoutes adds routes for advanced features
func (s *EnhancedServer) setupEnhancedRoutes() {
	// Metrics endpoint
	s.router.GET("/metrics", metrics.Handler())
	
	// Metrics dashboard
	if s.dashboard != nil {
		s.dashboard.RegisterRoutes(s.router.Group("/"))
	}

	// WebSocket endpoint
	s.router.GET("/ws", s.wsHandler.HandleWebSocket)

	// Cache management endpoints
	cache := s.router.Group("/cache")
	cache.Use(middleware.APIKeyAuth(s.config.APIKey))
	{
		cache.GET("/stats", s.handleCacheStats)
		cache.DELETE("/clear", s.handleCacheClear)
		cache.PUT("/enable", s.handleCacheEnable)
		cache.PUT("/disable", s.handleCacheDisable)
	}

	// Load balancer endpoints
	lb := s.router.Group("/loadbalancer")
	lb.Use(middleware.APIKeyAuth(s.config.APIKey))
	{
		lb.GET("/instances", s.handleLBInstances)
		lb.GET("/metrics", s.handleLBMetrics)
		lb.PUT("/algorithm", s.handleLBSetAlgorithm)
	}

	// Cluster endpoints
	if s.cluster != nil {
		s.cluster.RegisterHandlers(s.router.Group("/cluster"))
	}
}

// Enhanced chat completions with caching
func (s *EnhancedServer) handleChatCompletionsWithCache(c *gin.Context) {
	// Check cache first
	if s.cacheManager.IsEnabled() {
		var req models.OpenAIRequest
		if err := c.ShouldBindJSON(&req); err == nil && !req.Stream {
			// Generate cache key
			providerName := c.GetHeader("X-Model-Provider")
			if providerName == "" {
				providerName = s.config.ModelProviders[0]
			}
			
			cacheKey, _ := s.cacheManager.GenerateCacheKey(providerName, req.Model, &req)
			
			// Check cache
			if cached, found := s.cacheManager.Get(cacheKey); found {
				s.metrics.RecordCacheMetrics("memory", true)
				c.JSON(http.StatusOK, cached)
				return
			}
			
			// Cache miss - continue with normal processing
			s.metrics.RecordCacheMetrics("memory", false)
		}
	}

	// Call original handler
	s.handleChatCompletions(c)
}

// Cache management handlers
func (s *EnhancedServer) handleCacheStats(c *gin.Context) {
	stats := s.cacheManager.GetStats()
	
	response := gin.H{
		"enabled":     s.cacheManager.IsEnabled(),
		"hits":        stats.Hits,
		"misses":      stats.Misses,
		"evictions":   stats.Evictions,
		"total_bytes": stats.TotalBytes,
		"hit_rate":    float64(stats.Hits) / float64(stats.Hits+stats.Misses) * 100,
	}

	// Add Redis stats if available
	if s.redisCache != nil {
		redisStats, _ := s.redisCache.GetStats(c.Request.Context())
		response["redis"] = redisStats
	}

	c.JSON(http.StatusOK, response)
}

func (s *EnhancedServer) handleCacheClear(c *gin.Context) {
	s.cacheManager.Clear()
	
	if s.redisCache != nil {
		s.redisCache.Clear(c.Request.Context())
	}

	c.JSON(http.StatusOK, gin.H{"status": "cache cleared"})
}

func (s *EnhancedServer) handleCacheEnable(c *gin.Context) {
	s.cacheManager.SetEnabled(true)
	c.JSON(http.StatusOK, gin.H{"status": "cache enabled"})
}

func (s *EnhancedServer) handleCacheDisable(c *gin.Context) {
	s.cacheManager.SetEnabled(false)
	c.JSON(http.StatusOK, gin.H{"status": "cache disabled"})
}

// Load balancer handlers
func (s *EnhancedServer) handleLBInstances(c *gin.Context) {
	instances := s.loadBalancer.GetInstances()
	
	response := make([]gin.H, len(instances))
	for i, inst := range instances {
		response[i] = gin.H{
			"id":              inst.ID,
			"weight":          inst.Weight,
			"active_requests": inst.ActiveRequests,
			"total_requests":  inst.TotalRequests,
			"failed_requests": inst.FailedRequests,
			"is_healthy":      inst.IsHealthy,
			"last_used":       inst.LastUsed,
		}
	}

	c.JSON(http.StatusOK, response)
}

func (s *EnhancedServer) handleLBMetrics(c *gin.Context) {
	metrics := s.loadBalancer.GetMetrics()
	
	c.JSON(http.StatusOK, gin.H{
		"total_requests":    metrics.TotalRequests,
		"failed_requests":   metrics.FailedRequests,
		"active_requests":   metrics.ActiveRequests,
		"healthy_instances": metrics.HealthyInstances,
		"total_instances":   metrics.TotalInstances,
	})
}

func (s *EnhancedServer) handleLBSetAlgorithm(c *gin.Context) {
	var req struct {
		Algorithm string `json:"algorithm"`
	}
	
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	s.loadBalancer.SetAlgorithm(loadbalancer.Algorithm(req.Algorithm))
	c.JSON(http.StatusOK, gin.H{"status": "algorithm updated"})
}