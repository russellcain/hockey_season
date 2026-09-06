package hub

import (
	"sync"

	"github.com/gorilla/websocket"
)

type Client struct {
	DraftID string
	Conn    *websocket.Conn
	Send    chan []byte
}

type Hub struct {
	mu      sync.RWMutex
	clients map[string]map[*Client]struct{}
}

func New() *Hub {
	return &Hub{
		clients: make(map[string]map[*Client]struct{}),
	}
}

func (h *Hub) Register(c *Client) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.clients[c.DraftID] == nil {
		h.clients[c.DraftID] = make(map[*Client]struct{})
	}
	h.clients[c.DraftID][c] = struct{}{}
}

func (h *Hub) Unregister(c *Client) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if set, ok := h.clients[c.DraftID]; ok {
		if _, exists := set[c]; exists {
			delete(set, c)
			close(c.Send)
		}
	}
}

// Broadcast fans out msg to all clients subscribed to draftID.
// Slow clients that can't receive immediately are silently skipped.
func (h *Hub) Broadcast(draftID string, msg []byte) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	for c := range h.clients[draftID] {
		select {
		case c.Send <- msg:
		default:
		}
	}
}
