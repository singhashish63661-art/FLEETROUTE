// Package hub manages WebSocket connections and subscription-based message dispatch.
package hub

import (
	"context"
	"encoding/json"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"go.uber.org/zap"
)

// Client represents a single WebSocket connection with its subscriptions.
type Client struct {
	conn     *websocket.Conn
	tenantID string
	devices  map[string]bool // subscribed device IDs (empty = all devices in tenant)
	send     chan []byte
}

// Hub manages all active WebSocket clients and routes messages to subscribers.
type Hub struct {
	mu      sync.RWMutex
	clients map[*Client]bool
	logger  *zap.Logger

	register   chan *Client
	unregister chan *Client
	broadcast  chan broadcastMsg
}

type broadcastMsg struct {
	tenantID string
	deviceID string
	data     []byte
}

// New creates an empty Hub.
func New(logger *zap.Logger) *Hub {
	return &Hub{
		clients:    make(map[*Client]bool),
		register:   make(chan *Client, 256),
		unregister: make(chan *Client, 256),
		broadcast:  make(chan broadcastMsg, 4096),
		logger:     logger,
	}
}

// Run processes hub events. Must be called in a goroutine.
func (h *Hub) Run(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case client := <-h.register:
			h.mu.Lock()
			h.clients[client] = true
			h.mu.Unlock()
			h.logger.Debug("ws client registered", zap.String("tenant", client.tenantID))

		case client := <-h.unregister:
			h.mu.Lock()
			if _, ok := h.clients[client]; ok {
				delete(h.clients, client)
				close(client.send)
			}
			h.mu.Unlock()

		case msg := <-h.broadcast:
			h.mu.RLock()
			for client := range h.clients {
				if client.tenantID != msg.tenantID {
					continue
				}
				// If client has device subscriptions, filter; otherwise send all
				if len(client.devices) > 0 && !client.devices[msg.deviceID] {
					continue
				}
				select {
				case client.send <- msg.data:
				default:
					// Slow client — drop and unregister
					h.logger.Warn("slow ws client, dropping", zap.String("tenant", client.tenantID))
					go func(c *Client) { h.unregister <- c }(client)
				}
			}
			h.mu.RUnlock()
		}
	}
}

// Register adds a client to the hub.
func (h *Hub) Register(conn *websocket.Conn, tenantID string, deviceFilter []string) *Client {
	devices := make(map[string]bool)
	for _, d := range deviceFilter {
		devices[d] = true
	}
	client := &Client{
		conn:     conn,
		tenantID: tenantID,
		devices:  devices,
		send:     make(chan []byte, 256),
	}
	h.register <- client
	return client
}

// Unregister removes a client.
func (h *Hub) Unregister(client *Client) {
	h.unregister <- client
}

// Broadcast delivers a message to all matching subscribers.
func (h *Hub) Broadcast(tenantID, deviceID string, data []byte) {
	h.broadcast <- broadcastMsg{tenantID: tenantID, deviceID: deviceID, data: data}
}

// WritePump sends messages from the hub to the WebSocket connection.
func (h *Hub) WritePump(client *Client) {
	defer client.conn.Close()
	for data := range client.send {
		client.conn.SetWriteDeadline(time.Now().Add(10 * time.Second)) //nolint:errcheck
		if err := client.conn.WriteMessage(websocket.TextMessage, data); err != nil {
			break
		}
	}
}

// ReadPump reads control messages (subscribe/unsubscribe) from the client.
func (h *Hub) ReadPump(ctx context.Context, client *Client) {
	defer func() {
		h.Unregister(client)
		client.conn.Close()
	}()
	client.conn.SetReadLimit(512)

	for {
		select {
		case <-ctx.Done():
			return
		default:
		}
		_, msg, err := client.conn.ReadMessage()
		if err != nil {
			break
		}
		// Handle subscription updates: {"action":"subscribe","devices":["dev1","dev2"]}
		var cmd struct {
			Action  string   `json:"action"`
			Devices []string `json:"devices"`
		}
		if err := json.Unmarshal(msg, &cmd); err == nil {
			if cmd.Action == "subscribe" {
				h.mu.Lock()
				client.devices = make(map[string]bool)
				for _, d := range cmd.Devices {
					client.devices[d] = true
				}
				h.mu.Unlock()
			}
		}
	}
}


