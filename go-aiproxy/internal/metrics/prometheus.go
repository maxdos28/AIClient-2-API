package metrics

import (
	"time"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// Metrics holds all Prometheus metrics
type Metrics struct {
	// HTTP metrics
	RequestsTotal   *prometheus.CounterVec
	RequestDuration *prometheus.HistogramVec
	ResponseSize    *prometheus.HistogramVec
	ActiveRequests  prometheus.Gauge

	// Provider metrics
	ProviderRequestsTotal   *prometheus.CounterVec
	ProviderRequestDuration *prometheus.HistogramVec
	ProviderErrors          *prometheus.CounterVec
	ProviderTokensUsed      *prometheus.CounterVec

	// Cache metrics
	CacheHits       *prometheus.CounterVec
	CacheMisses     *prometheus.CounterVec
	CacheEvictions  prometheus.Counter
	CacheSizeBytes  prometheus.Gauge

	// System metrics
	GoRoutines      prometheus.Gauge
	MemoryUsageBytes prometheus.Gauge
	CPUUsagePercent  prometheus.Gauge

	// WebSocket metrics
	WSConnections    prometheus.Gauge
	WSMessagesTotal  *prometheus.CounterVec
	WSBytesTotal     *prometheus.CounterVec

	// Pool metrics
	PoolProviders      *prometheus.GaugeVec
	PoolHealthyProviders *prometheus.GaugeVec
	PoolFailovers       *prometheus.CounterVec
}

// NewMetrics creates and registers all Prometheus metrics
func NewMetrics() *Metrics {
	return &Metrics{
		// HTTP metrics
		RequestsTotal: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "aiproxy_http_requests_total",
				Help: "Total number of HTTP requests",
			},
			[]string{"method", "endpoint", "status"},
		),
		RequestDuration: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "aiproxy_http_request_duration_seconds",
				Help:    "HTTP request duration in seconds",
				Buckets: prometheus.DefBuckets,
			},
			[]string{"method", "endpoint"},
		),
		ResponseSize: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "aiproxy_http_response_size_bytes",
				Help:    "HTTP response size in bytes",
				Buckets: prometheus.ExponentialBuckets(100, 10, 7), // 100B to 100MB
			},
			[]string{"method", "endpoint"},
		),
		ActiveRequests: promauto.NewGauge(
			prometheus.GaugeOpts{
				Name: "aiproxy_http_active_requests",
				Help: "Number of active HTTP requests",
			},
		),

		// Provider metrics
		ProviderRequestsTotal: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "aiproxy_provider_requests_total",
				Help: "Total number of requests to providers",
			},
			[]string{"provider", "model", "status"},
		),
		ProviderRequestDuration: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "aiproxy_provider_request_duration_seconds",
				Help:    "Provider request duration in seconds",
				Buckets: prometheus.DefBuckets,
			},
			[]string{"provider", "model"},
		),
		ProviderErrors: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "aiproxy_provider_errors_total",
				Help: "Total number of provider errors",
			},
			[]string{"provider", "error_type"},
		),
		ProviderTokensUsed: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "aiproxy_provider_tokens_used_total",
				Help: "Total number of tokens used by provider",
			},
			[]string{"provider", "model", "token_type"},
		),

		// Cache metrics
		CacheHits: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "aiproxy_cache_hits_total",
				Help: "Total number of cache hits",
			},
			[]string{"cache_type"},
		),
		CacheMisses: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "aiproxy_cache_misses_total",
				Help: "Total number of cache misses",
			},
			[]string{"cache_type"},
		),
		CacheEvictions: promauto.NewCounter(
			prometheus.CounterOpts{
				Name: "aiproxy_cache_evictions_total",
				Help: "Total number of cache evictions",
			},
		),
		CacheSizeBytes: promauto.NewGauge(
			prometheus.GaugeOpts{
				Name: "aiproxy_cache_size_bytes",
				Help: "Current cache size in bytes",
			},
		),

		// System metrics
		GoRoutines: promauto.NewGauge(
			prometheus.GaugeOpts{
				Name: "aiproxy_goroutines",
				Help: "Number of active goroutines",
			},
		),
		MemoryUsageBytes: promauto.NewGauge(
			prometheus.GaugeOpts{
				Name: "aiproxy_memory_usage_bytes",
				Help: "Current memory usage in bytes",
			},
		),
		CPUUsagePercent: promauto.NewGauge(
			prometheus.GaugeOpts{
				Name: "aiproxy_cpu_usage_percent",
				Help: "Current CPU usage percentage",
			},
		),

		// WebSocket metrics
		WSConnections: promauto.NewGauge(
			prometheus.GaugeOpts{
				Name: "aiproxy_websocket_connections",
				Help: "Number of active WebSocket connections",
			},
		),
		WSMessagesTotal: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "aiproxy_websocket_messages_total",
				Help: "Total number of WebSocket messages",
			},
			[]string{"direction"}, // "sent" or "received"
		),
		WSBytesTotal: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "aiproxy_websocket_bytes_total",
				Help: "Total bytes transferred via WebSocket",
			},
			[]string{"direction"},
		),

		// Pool metrics
		PoolProviders: promauto.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "aiproxy_pool_providers_total",
				Help: "Total number of providers in pool",
			},
			[]string{"provider_type"},
		),
		PoolHealthyProviders: promauto.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "aiproxy_pool_healthy_providers",
				Help: "Number of healthy providers in pool",
			},
			[]string{"provider_type"},
		),
		PoolFailovers: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "aiproxy_pool_failovers_total",
				Help: "Total number of provider failovers",
			},
			[]string{"from_provider", "to_provider"},
		),
	}
}

// PrometheusMiddleware creates a Gin middleware for Prometheus metrics
func PrometheusMiddleware(m *Metrics) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		method := c.Request.Method

		// Increment active requests
		m.ActiveRequests.Inc()
		defer m.ActiveRequests.Dec()

		// Process request
		c.Next()

		// Record metrics
		duration := time.Since(start).Seconds()
		status := fmt.Sprintf("%d", c.Writer.Status())
		
		m.RequestsTotal.WithLabelValues(method, path, status).Inc()
		m.RequestDuration.WithLabelValues(method, path).Observe(duration)
		m.ResponseSize.WithLabelValues(method, path).Observe(float64(c.Writer.Size()))
	}
}

// RecordProviderMetrics records metrics for provider requests
func (m *Metrics) RecordProviderMetrics(provider, model string, duration time.Duration, err error, tokens map[string]int) {
	status := "success"
	if err != nil {
		status = "error"
		errorType := "unknown"
		if strings.Contains(err.Error(), "timeout") {
			errorType = "timeout"
		} else if strings.Contains(err.Error(), "401") || strings.Contains(err.Error(), "403") {
			errorType = "auth"
		} else if strings.Contains(err.Error(), "429") {
			errorType = "rate_limit"
		}
		m.ProviderErrors.WithLabelValues(provider, errorType).Inc()
	}

	m.ProviderRequestsTotal.WithLabelValues(provider, model, status).Inc()
	m.ProviderRequestDuration.WithLabelValues(provider, model).Observe(duration.Seconds())

	// Record token usage
	for tokenType, count := range tokens {
		m.ProviderTokensUsed.WithLabelValues(provider, model, tokenType).Add(float64(count))
	}
}

// RecordCacheMetrics records cache hit/miss metrics
func (m *Metrics) RecordCacheMetrics(cacheType string, hit bool) {
	if hit {
		m.CacheHits.WithLabelValues(cacheType).Inc()
	} else {
		m.CacheMisses.WithLabelValues(cacheType).Inc()
	}
}

// UpdateSystemMetrics updates system resource metrics
func (m *Metrics) UpdateSystemMetrics(goroutines int, memoryMB float64, cpuPercent float64) {
	m.GoRoutines.Set(float64(goroutines))
	m.MemoryUsageBytes.Set(memoryMB * 1024 * 1024)
	m.CPUUsagePercent.Set(cpuPercent)
}

// Handler returns the Prometheus metrics handler
func Handler() gin.HandlerFunc {
	return gin.WrapH(promhttp.Handler())
}

// CollectSystemMetrics starts a goroutine to collect system metrics
func (m *Metrics) CollectSystemMetrics(interval time.Duration) {
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		for range ticker.C {
			var memStats runtime.MemStats
			runtime.ReadMemStats(&memStats)

			goroutines := runtime.NumGoroutine()
			memoryMB := float64(memStats.Alloc) / 1024 / 1024
			
			// CPU usage would require platform-specific implementation
			// For now, we'll use a placeholder
			cpuPercent := 0.0

			m.UpdateSystemMetrics(goroutines, memoryMB, cpuPercent)
		}
	}()
}