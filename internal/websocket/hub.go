package websocket

import (
	"sync"
)

type Client struct {
	ReportID uint
	UserID   string
	Send     chan []byte
}

type Hub struct {
	clients map[uint][]*Client
	mu      sync.Mutex
}

func NewHub() *Hub {
	return &Hub{
		clients: make(map[uint][]*Client),
	}
}

func (h *Hub) Register(reportID uint, client *Client) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.clients[reportID] = append(h.clients[reportID], client)
}

func (h *Hub) Unregister(reportID uint, client *Client) {
	h.mu.Lock()
	defer h.mu.Unlock()
	clients := h.clients[reportID]
	for i, c := range clients {
		if c == client {
			h.clients[reportID] = append(clients[:i], clients[i+1:]...)
			break
		}
	}
}

func (h *Hub) Broadcast(reportID uint, message []byte) {
	h.mu.Lock()
	defer h.mu.Unlock()
	for _, c := range h.clients[reportID] {
		select {
		case c.Send <- message:
		default:
		}
	}
}
