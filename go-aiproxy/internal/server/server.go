package server

import (
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/aiproxy/go-aiproxy/internal/config"
	"github.com/aiproxy/go-aiproxy/internal/convert"
	"github.com/aiproxy/go-aiproxy/internal/middleware"
	"github.com/aiproxy/go-aiproxy/internal/pool"
	"github.com/aiproxy/go-aiproxy/internal/providers"
	"github.com/aiproxy/go-aiproxy/internal/providers/claude"
	"github.com/aiproxy/go-aiproxy/internal/providers/gemini"
	"github.com/aiproxy/go-aiproxy/internal/providers/openai"
	"github.com/aiproxy/go-aiproxy/pkg/models"
	"github.com/gin-gonic/gin"
)

// Server represents the HTTP server
type Server struct {
	config      *config.Config
	router      *gin.Engine
	providers   map[string]providers.Provider
	poolManager *pool.Manager
	converter   convert.Converter
}

// New creates a new server instance
func New(cfg *config.Config) (*Server, error) {
	// Validate configuration
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	// Create server instance
	s := &Server{
		config:    cfg,
		providers: make(map[string]providers.Provider),
		converter: convert.NewConverter(),
	}

	// Initialize providers
	if err := s.initializeProviders(); err != nil {
		return nil, fmt.Errorf("failed to initialize providers: %w", err)
	}

	// Initialize pool manager if configured
	if cfg.ProviderPoolsFile != "" {
		// Pool manager would be initialized here
		// s.poolManager = pool.NewManager(cfg.ProviderPoolsFile)
	}

	// Setup router
	s.setupRouter()

	return s, nil
}

// initializeProviders creates provider instances based on configuration
func (s *Server) initializeProviders() error {
	for name, cfg := range s.config.ProviderConfigs {
		var provider providers.Provider
		var err error

		switch cfg.Provider {
		case models.ProviderOpenAI:
			provider, err = openai.NewClient(cfg)
		case models.ProviderClaude:
			provider, err = claude.NewClient(cfg)
		case models.ProviderGemini:
			provider, err = gemini.NewClient(cfg)
		default:
			return fmt.Errorf("unsupported provider: %s", cfg.Provider)
		}

		if err != nil {
			return fmt.Errorf("failed to create provider %s: %w", name, err)
		}

		s.providers[name] = provider
	}

	return nil
}

// setupRouter configures the HTTP routes
func (s *Server) setupRouter() {
	// Set Gin to release mode for production
	gin.SetMode(gin.ReleaseMode)

	s.router = gin.New()
	
	// Add middleware
	s.router.Use(gin.Recovery())
	s.router.Use(middleware.Logger())
	s.router.Use(middleware.CORS())

	// Health check endpoint
	s.router.GET("/health", s.handleHealth)

	// API routes with authentication
	api := s.router.Group("/")
	api.Use(middleware.APIKeyAuth(s.config.APIKey))

	// OpenAI compatible endpoints
	api.POST("/v1/chat/completions", s.handleChatCompletions)
	api.GET("/v1/models", s.handleListModels)

	// Gemini native endpoints
	api.POST("/v1beta/models/:model:generateContent", s.handleGeminiGenerate)
	api.POST("/v1beta/models/:model:streamGenerateContent", s.handleGeminiStream)
	api.GET("/v1beta/models", s.handleGeminiListModels)

	// Claude native endpoints (if needed)
	api.POST("/v1/messages", s.handleClaudeMessages)
}

// Start starts the HTTP server
func (s *Server) Start(addr string) error {
	return s.router.Run(addr)
}

// handleHealth handles health check requests
func (s *Server) handleHealth(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status": "healthy",
		"time":   time.Now().Unix(),
	})
}

// handleChatCompletions handles OpenAI-style chat completion requests
func (s *Server) handleChatCompletions(c *gin.Context) {
	// Parse request
	var req models.OpenAIRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Determine provider from header or use default
	providerName := c.GetHeader("X-Model-Provider")
	if providerName == "" {
		providerName = s.config.ModelProviders[0]
	}

	provider, ok := s.providers[providerName]
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid provider"})
		return
	}

	// Get protocol prefixes
	fromProtocol := models.ProtocolOpenAI
	toProtocol := provider.GetProtocolPrefix()

	// Convert request if needed
	var convertedReq interface{} = &req
	if fromProtocol != toProtocol {
		var err error
		convertedReq, err = s.converter.ConvertRequest(&req, fromProtocol, toProtocol)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("conversion error: %v", err)})
			return
		}
	}

	// Handle streaming
	if req.Stream {
		s.handleStreamingResponse(c, provider, req.Model, convertedReq, fromProtocol, toProtocol)
		return
	}

	// Make non-streaming request
	ctx := c.Request.Context()
	resp, err := provider.GenerateContent(ctx, req.Model, convertedReq)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Convert response if needed
	if fromProtocol != toProtocol {
		resp, err = s.converter.ConvertResponse(resp, toProtocol, fromProtocol, req.Model)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("response conversion error: %v", err)})
			return
		}
	}

	c.JSON(http.StatusOK, resp)
}

// handleStreamingResponse handles streaming responses
func (s *Server) handleStreamingResponse(c *gin.Context, provider providers.Provider, model string, request interface{}, fromProtocol, toProtocol models.ProtocolPrefix) {
	ctx := c.Request.Context()
	
	// Get stream from provider
	stream, err := provider.GenerateContentStream(ctx, model, request)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer stream.Close()

	// Set headers for SSE
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("Transfer-Encoding", "chunked")

	// Create a channel for sending data
	dataChan := make(chan string)
	doneChan := make(chan bool)

	// Start goroutine to read from stream
	go func() {
		defer close(dataChan)
		defer close(doneChan)

		buffer := make([]byte, 4096)
		for {
			n, err := stream.Read(buffer)
			if err != nil {
				if err != io.EOF {
					log.Printf("Stream read error: %v", err)
				}
				break
			}

			if n > 0 {
				chunk := string(buffer[:n])
				
				// Convert chunk if needed
				if fromProtocol != toProtocol {
					convertedChunk, err := s.converter.ConvertStreamChunk(chunk, toProtocol, fromProtocol, model)
					if err == nil && convertedChunk != nil {
						if chunkData, ok := convertedChunk.(*models.StreamChunk); ok {
							// Format as SSE
							jsonData, _ := json.Marshal(chunkData)
							dataChan <- fmt.Sprintf("data: %s\n\n", string(jsonData))
						}
					}
				} else {
					// Send raw chunk
					dataChan <- fmt.Sprintf("data: %s\n\n", chunk)
				}
			}
		}
		
		// Send done signal
		dataChan <- "data: [DONE]\n\n"
	}()

	// Stream to client
	c.Stream(func(w io.Writer) bool {
		select {
		case data, ok := <-dataChan:
			if !ok {
				return false
			}
			w.Write([]byte(data))
			return true
		case <-ctx.Done():
			return false
		}
	})
}

// handleListModels handles model listing requests
func (s *Server) handleListModels(c *gin.Context) {
	// Determine provider
	providerName := c.GetHeader("X-Model-Provider")
	if providerName == "" {
		providerName = s.config.ModelProviders[0]
	}

	provider, ok := s.providers[providerName]
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid provider"})
		return
	}

	ctx := c.Request.Context()
	models, err := provider.ListModels(ctx)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Convert to OpenAI format if needed
	fromProtocol := provider.GetProtocolPrefix()
	if fromProtocol != models.ProtocolOpenAI {
		models, err = s.converter.ConvertModelList(models, fromProtocol, models.ProtocolOpenAI)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("model list conversion error: %v", err)})
			return
		}
	}

	c.JSON(http.StatusOK, models)
}

// handleGeminiGenerate handles Gemini-style generation requests
func (s *Server) handleGeminiGenerate(c *gin.Context) {
	modelName := c.Param("model")
	
	// Parse request
	var req models.GeminiRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Find Gemini provider
	var provider providers.Provider
	for name, p := range s.providers {
		if p.GetProtocolPrefix() == models.ProtocolGemini {
			provider = p
			break
		}
	}

	if provider == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Gemini provider not configured"})
		return
	}

	// Make request
	ctx := c.Request.Context()
	resp, err := provider.GenerateContent(ctx, modelName, &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, resp)
}

// handleGeminiStream handles Gemini-style streaming requests
func (s *Server) handleGeminiStream(c *gin.Context) {
	modelName := c.Param("model")
	
	// Parse request
	var req models.GeminiRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Find Gemini provider
	var provider providers.Provider
	for name, p := range s.providers {
		if p.GetProtocolPrefix() == models.ProtocolGemini {
			provider = p
			break
		}
	}

	if provider == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Gemini provider not configured"})
		return
	}

	// Handle streaming
	s.handleStreamingResponse(c, provider, modelName, &req, models.ProtocolGemini, models.ProtocolGemini)
}

// handleGeminiListModels handles Gemini model listing
func (s *Server) handleGeminiListModels(c *gin.Context) {
	// Find Gemini provider
	var provider providers.Provider
	for name, p := range s.providers {
		if p.GetProtocolPrefix() == models.ProtocolGemini {
			provider = p
			break
		}
	}

	if provider == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Gemini provider not configured"})
		return
	}

	ctx := c.Request.Context()
	models, err := provider.ListModels(ctx)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, models)
}

// handleClaudeMessages handles Claude-style message requests
func (s *Server) handleClaudeMessages(c *gin.Context) {
	// Parse request
	var req models.ClaudeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Find Claude provider
	var provider providers.Provider
	for name, p := range s.providers {
		if p.GetProtocolPrefix() == models.ProtocolClaude {
			provider = p
			break
		}
	}

	if provider == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Claude provider not configured"})
		return
	}

	// Handle streaming
	if req.Stream {
		s.handleStreamingResponse(c, provider, req.Model, &req, models.ProtocolClaude, models.ProtocolClaude)
		return
	}

	// Make request
	ctx := c.Request.Context()
	resp, err := provider.GenerateContent(ctx, req.Model, &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, resp)
}