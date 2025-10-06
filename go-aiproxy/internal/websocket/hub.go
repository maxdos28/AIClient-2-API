package websocket

import (
	"context"
	"encoding/json"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

// Hub maintains the set of active clients and broadcasts messages to clients
type Hub struct {
	// Registered clients
	clients map[string]*Client

	// Inbound messages from clients
	broadcast chan *Message

	// Register requests from clients
	register chan *Client

	// Unregister requests from clients
	unregister chan *Client

	// Mutex for thread-safe operations
	mu sync.RWMutex

	// Metrics callback
	onMetricsUpdate func(activeConnections int)
}

// Client represents a WebSocket client connection
type Client struct {
	ID       string
	hub      *Hub
	conn     *websocket.Conn
	send     chan []byte
	provider string
	model    string
}

// Message represents a WebSocket message
type Message struct {
	Type      string                 `json:"type"`
	ID        string                 `json:"id"`
	ClientID  string                 `json:"client_id,omitempty"`
	Provider  string                 `json:"provider,omitempty"`
	Model     string                 `json:"model,omitempty"`
	Request   interface{}            `json:"request,omitempty"`
	Response  interface{}            `json:"response,omitempty"`
	Error     string                 `json:"error,omitempty"`
	Timestamp int64                  `json:"timestamp"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
}

// MessageType constants
const (
	MessageTypeRequest       = "request"
	MessageTypeResponse      = "response"
	MessageTypeStream        = "stream"
	MessageTypeStreamEnd     = "stream_end"
	MessageTypeError         = "error"
	MessageTypeHeartbeat     = "heartbeat"
	MessageTypeAuthenticate  = "authenticate"
	MessageTypeAuthenticated = "authenticated"
)

// NewHub creates a new WebSocket hub
func NewHub() *Hub {
	return &Hub{
		clients:    make(map[string]*Client),
		broadcast:  make(chan *Message, 256),
		register:   make(chan *Client),
		unregister: make(chan *Client),
	}
}

// Run starts the hub's main event loop
func (h *Hub) Run(ctx context.Context) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			// Shutdown all clients
			h.shutdown()
			return

		case client := <-h.register:
			h.mu.Lock()
			h.clients[client.ID] = client
			h.mu.Unlock()
			
			// Send welcome message
			welcome := &Message{
				Type:      MessageTypeAuthenticated,
				ID:        uuid.New().String(),
				ClientID:  client.ID,
				Timestamp: time.Now().Unix(),
				Metadata: map[string]interface{}{
					"version": "1.0",
					"capabilities": []string{"streaming", "multimodal", "tools"},
				},
			}
			client.SendMessage(welcome)
			
			// Update metrics
			if h.onMetricsUpdate != nil {
				h.onMetricsUpdate(len(h.clients))
			}

		case client := <-h.unregister:
			h.mu.Lock()
			if _, ok := h.clients[client.ID]; ok {
				delete(h.clients, client.ID)
				close(client.send)
				h.mu.Unlock()
				
				// Update metrics
				if h.onMetricsUpdate != nil {
					h.onMetricsUpdate(len(h.clients))
				}
			} else {
				h.mu.Unlock()
			}

		case message := <-h.broadcast:
			// Broadcast to specific client or all clients
			if message.ClientID != "" {
				// Send to specific client
				h.mu.RLock()
				if client, ok := h.clients[message.ClientID]; ok {
					h.mu.RUnlock()
					client.SendMessage(message)
				} else {
					h.mu.RUnlock()
				}
			} else {
				// Broadcast to all clients
				h.broadcastToAll(message)
			}

		case <-ticker.C:
			// Send heartbeat to all clients
			heartbeat := &Message{
				Type:      MessageTypeHeartbeat,
				ID:        uuid.New().String(),
				Timestamp: time.Now().Unix(),
			}
			h.broadcastToAll(heartbeat)
		}
	}
}

// broadcastToAll sends a message to all connected clients
func (h *Hub) broadcastToAll(message *Message) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	data, err := json.Marshal(message)
	if err != nil {
		return
	}

	for _, client := range h.clients {
		select {
		case client.send <- data:
		default:
			// Client's send channel is full, close it
			close(client.send)
			delete(h.clients, client.ID)
		}
	}
}

// GetClient returns a client by ID
func (h *Hub) GetClient(clientID string) (*Client, bool) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	
	client, ok := h.clients[clientID]
	return client, ok
}

// GetClients returns all connected clients
func (h *Hub) GetClients() map[string]*Client {
	h.mu.RLock()
	defer h.mu.RUnlock()
	
	// Return a copy to avoid race conditions
	clients := make(map[string]*Client)
	for k, v := range h.clients {
		clients[k] = v
	}
	return clients
}

// SendToClient sends a message to a specific client
func (h *Hub) SendToClient(clientID string, message *Message) error {
	message.ClientID = clientID
	
	select {
	case h.broadcast <- message:
		return nil
	default:
		return fmt.Errorf("broadcast channel full")
	}
}

// SetMetricsCallback sets the callback for metrics updates
func (h *Hub) SetMetricsCallback(callback func(int)) {
	h.onMetricsUpdate = callback
}

// shutdown closes all client connections
func (h *Hub) shutdown() {
	h.mu.Lock()
	defer h.mu.Unlock()

	for _, client := range h.clients {
		close(client.send)
		client.conn.Close()
	}
	h.clients = make(map[string]*Client)
}

// SendMessage sends a message to the client
func (c *Client) SendMessage(message *Message) error {
	data, err := json.Marshal(message)
	if err != nil {
		return err
	}

	select {
	case c.send <- data:
		return nil
	default:
		return fmt.Errorf("client send channel full")
	}
}

// SendJSON sends a JSON object to the client
func (c *Client) SendJSON(v interface{}) error {
	data, err := json.Marshal(v)
	if err != nil {
		return err
	}

	select {
	case c.send <- data:
		return nil
	default:
		return fmt.Errorf("client send channel full")
	}
}

// Close closes the client connection
func (c *Client) Close() {
	c.hub.unregister <- c
}