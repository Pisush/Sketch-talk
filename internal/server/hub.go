package server

import "github.com/gorilla/websocket"

// client represents a single WebSocket connection.
type client struct {
	conn *websocket.Conn
	send chan []byte
}

// Hub manages all connected WebSocket clients and broadcasts frames.
type Hub struct {
	clients    map[*client]bool
	register   chan *client
	unregister chan *client
	Broadcast  chan []byte
}

// NewHub creates a new Hub.
func NewHub() *Hub {
	return &Hub{
		clients:    make(map[*client]bool),
		register:   make(chan *client),
		unregister: make(chan *client),
		Broadcast:  make(chan []byte, 8),
	}
}

// Run is the hub's main event loop. Call in a goroutine.
func (h *Hub) Run() {
	for {
		select {
		case c := <-h.register:
			h.clients[c] = true
		case c := <-h.unregister:
			if _, ok := h.clients[c]; ok {
				delete(h.clients, c)
				close(c.send)
			}
		case frame := <-h.Broadcast:
			for c := range h.clients {
				select {
				case c.send <- frame:
				default:
					// Slow client: drop frame.
				}
			}
		}
	}
}
