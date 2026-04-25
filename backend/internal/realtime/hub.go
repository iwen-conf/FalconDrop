package realtime

import (
	"encoding/json"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

type Event struct {
	EventID   string    `json:"eventId,omitempty"`
	Type      string    `json:"type"`
	Payload   any       `json:"payload"`
	CreatedAt time.Time `json:"createdAt"`
}

type Hub struct {
	mu      sync.RWMutex
	clients map[*websocket.Conn]struct{}
}

func NewHub() *Hub {
	return &Hub{
		clients: make(map[*websocket.Conn]struct{}),
	}
}

func (h *Hub) Add(conn *websocket.Conn) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.clients[conn] = struct{}{}
}

func (h *Hub) Remove(conn *websocket.Conn) {
	h.mu.Lock()
	defer h.mu.Unlock()
	delete(h.clients, conn)
	_ = conn.Close()
}

func (h *Hub) Broadcast(eventType string, payload any) {
	event := Event{
		EventID:   uuid.NewString(),
		Type:      eventType,
		Payload:   payload,
		CreatedAt: time.Now().UTC(),
	}
	b, err := json.Marshal(event)
	if err != nil {
		return
	}

	h.mu.RLock()
	clients := make([]*websocket.Conn, 0, len(h.clients))
	for c := range h.clients {
		clients = append(clients, c)
	}
	h.mu.RUnlock()

	for _, c := range clients {
		_ = c.SetWriteDeadline(time.Now().Add(5 * time.Second))
		if err := c.WriteMessage(websocket.TextMessage, b); err != nil {
			h.Remove(c)
		}
	}
}

func (h *Hub) Count() int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.clients)
}
