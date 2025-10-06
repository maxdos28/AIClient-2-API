package websocket

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"

	"github.com/aiproxy/go-aiproxy/internal/convert"
	"github.com/aiproxy/go-aiproxy/internal/providers"
	"github.com/aiproxy/go-aiproxy/pkg/models"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

// WebSocketHandler handles WebSocket connections
type WebSocketHandler struct {
	hub        *Hub
	upgrader   websocket.Upgrader
	providers  map[string]providers.Provider
	converter  convert.Converter
	authKey    string
}

// NewWebSocketHandler creates a new WebSocket handler
func NewWebSocketHandler(hub *Hub, providers map[string]providers.Provider, authKey string) *WebSocketHandler {
	return &WebSocketHandler{
		hub: hub,
		upgrader: websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool {
				// Allow connections from any origin
				// In production, implement proper origin checking
				return true
			},
			ReadBufferSize:  1024,
			WriteBufferSize: 1024,
		},
		providers: providers,
		converter: convert.NewConverter(),
		authKey:   authKey,
	}
}

// HandleWebSocket handles WebSocket upgrade and connection
func (h *WebSocketHandler) HandleWebSocket(c *gin.Context) {
	// Check authentication
	token := c.Query("token")
	if token == "" {
		token = c.GetHeader("Sec-WebSocket-Protocol")
	}
	
	if token != h.authKey {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid authentication token"})
		return
	}

	// Upgrade HTTP connection to WebSocket
	conn, err := h.upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.Printf("Failed to upgrade connection: %v", err)
		return
	}

	// Create new client
	client := &Client{
		ID:   uuid.New().String(),
		hub:  h.hub,
		conn: conn,
		send: make(chan []byte, 256),
	}

	// Register client
	client.hub.register <- client

	// Start client goroutines
	go client.writePump()
	go h.readPump(client)
}

// readPump pumps messages from the WebSocket connection to the hub
func (h *WebSocketHandler) readPump(client *Client) {
	defer func() {
		client.hub.unregister <- client
		client.conn.Close()
	}()

	client.conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	client.conn.SetPongHandler(func(string) error {
		client.conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})

	for {
		_, message, err := client.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("WebSocket error: %v", err)
			}
			break
		}

		// Parse message
		var msg Message
		if err := json.Unmarshal(message, &msg); err != nil {
			h.sendError(client, "Invalid message format", err)
			continue
		}

		// Handle message based on type
		switch msg.Type {
		case MessageTypeRequest:
			h.handleRequest(client, &msg)
		case MessageTypeHeartbeat:
			// Echo heartbeat back
			client.SendMessage(&Message{
				Type:      MessageTypeHeartbeat,
				ID:        msg.ID,
				Timestamp: time.Now().Unix(),
			})
		case MessageTypeAuthenticate:
			// Handle re-authentication if needed
			if auth, ok := msg.Metadata["token"].(string); ok && auth == h.authKey {
				client.SendMessage(&Message{
					Type:      MessageTypeAuthenticated,
					ID:        msg.ID,
					Timestamp: time.Now().Unix(),
				})
			} else {
				h.sendError(client, "Authentication failed", nil)
			}
		default:
			h.sendError(client, fmt.Sprintf("Unknown message type: %s", msg.Type), nil)
		}
	}
}

// writePump pumps messages from the hub to the WebSocket connection
func (c *Client) writePump() {
	ticker := time.NewTicker(54 * time.Second)
	defer func() {
		ticker.Stop()
		c.conn.Close()
	}()

	for {
		select {
		case message, ok := <-c.send:
			c.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if !ok {
				// The hub closed the channel
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			w, err := c.conn.NextWriter(websocket.TextMessage)
			if err != nil {
				return
			}
			w.Write(message)

			// Add queued messages to the current WebSocket frame
			n := len(c.send)
			for i := 0; i < n; i++ {
				w.Write([]byte{'\n'})
				w.Write(<-c.send)
			}

			if err := w.Close(); err != nil {
				return
			}

		case <-ticker.C:
			c.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

// handleRequest processes incoming AI requests
func (h *WebSocketHandler) handleRequest(client *Client, msg *Message) {
	// Extract provider and model
	provider := msg.Provider
	if provider == "" {
		provider = client.provider
	}
	if provider == "" {
		// Use first available provider
		for name := range h.providers {
			provider = name
			break
		}
	}

	// Get provider instance
	providerInstance, ok := h.providers[provider]
	if !ok {
		h.sendError(client, fmt.Sprintf("Provider %s not found", provider), nil)
		return
	}

	// Update client's provider and model
	client.provider = provider
	if msg.Model != "" {
		client.model = msg.Model
	}

	// Determine if this is a streaming request
	isStream := false
	if metadata, ok := msg.Metadata["stream"].(bool); ok {
		isStream = metadata
	}

	// Get protocol prefixes
	fromProtocol := models.ProtocolOpenAI // Assume OpenAI format by default
	toProtocol := providerInstance.GetProtocolPrefix()

	// Convert request if needed
	request := msg.Request
	if fromProtocol != toProtocol {
		var err error
		request, err = h.converter.ConvertRequest(msg.Request, fromProtocol, toProtocol)
		if err != nil {
			h.sendError(client, "Failed to convert request", err)
			return
		}
	}

	// Handle streaming vs non-streaming
	if isStream {
		h.handleStreamingRequest(client, msg, providerInstance, request, fromProtocol, toProtocol)
	} else {
		h.handleNonStreamingRequest(client, msg, providerInstance, request, fromProtocol, toProtocol)
	}
}

// handleNonStreamingRequest processes non-streaming requests
func (h *WebSocketHandler) handleNonStreamingRequest(client *Client, msg *Message, provider providers.Provider, request interface{}, fromProtocol, toProtocol models.ProtocolPrefix) {
	ctx := context.Background()
	
	// Make request
	response, err := provider.GenerateContent(ctx, client.model, request)
	if err != nil {
		h.sendError(client, "Provider request failed", err)
		return
	}

	// Convert response if needed
	if fromProtocol != toProtocol {
		response, err = h.converter.ConvertResponse(response, toProtocol, fromProtocol, client.model)
		if err != nil {
			h.sendError(client, "Failed to convert response", err)
			return
		}
	}

	// Send response
	responseMsg := &Message{
		Type:      MessageTypeResponse,
		ID:        msg.ID,
		ClientID:  client.ID,
		Provider:  client.provider,
		Model:     client.model,
		Response:  response,
		Timestamp: time.Now().Unix(),
	}

	client.SendMessage(responseMsg)
}

// handleStreamingRequest processes streaming requests
func (h *WebSocketHandler) handleStreamingRequest(client *Client, msg *Message, provider providers.Provider, request interface{}, fromProtocol, toProtocol models.ProtocolPrefix) {
	ctx := context.Background()
	
	// Get stream
	stream, err := provider.GenerateContentStream(ctx, client.model, request)
	if err != nil {
		h.sendError(client, "Failed to start stream", err)
		return
	}
	defer stream.Close()

	// Read and forward stream chunks
	go func() {
		buffer := make([]byte, 4096)
		for {
			n, err := stream.Read(buffer)
			if err != nil {
				if err != io.EOF {
					h.sendError(client, "Stream read error", err)
				}
				// Send stream end message
				endMsg := &Message{
					Type:      MessageTypeStreamEnd,
					ID:        msg.ID,
					ClientID:  client.ID,
					Provider:  client.provider,
					Model:     client.model,
					Timestamp: time.Now().Unix(),
				}
				client.SendMessage(endMsg)
				break
			}

			if n > 0 {
				chunk := string(buffer[:n])
				
				// Convert chunk if needed
				if fromProtocol != toProtocol {
					convertedChunk, err := h.converter.ConvertStreamChunk(chunk, toProtocol, fromProtocol, client.model)
					if err == nil && convertedChunk != nil {
						chunk = fmt.Sprintf("%v", convertedChunk)
					}
				}

				// Send stream chunk
				streamMsg := &Message{
					Type:      MessageTypeStream,
					ID:        msg.ID,
					ClientID:  client.ID,
					Provider:  client.provider,
					Model:     client.model,
					Response:  chunk,
					Timestamp: time.Now().Unix(),
				}
				client.SendMessage(streamMsg)
			}
		}
	}()
}

// sendError sends an error message to the client
func (h *WebSocketHandler) sendError(client *Client, message string, err error) {
	errMsg := message
	if err != nil {
		errMsg = fmt.Sprintf("%s: %v", message, err)
	}

	errorMessage := &Message{
		Type:      MessageTypeError,
		ID:        uuid.New().String(),
		ClientID:  client.ID,
		Error:     errMsg,
		Timestamp: time.Now().Unix(),
	}

	client.SendMessage(errorMessage)
}