package server

import "github.com/gorilla/websocket"

// client represents a single WebSocket connection.
type client struct {
	conn     *websocket.Conn
	send     chan []byte // binary PNG frames
	sendText chan []byte // JSON text events (questions, etc.)
}

// Hub manages all connected WebSocket clients and broadcasts frames.
type Hub struct {
	clients       map[*client]bool
	register      chan *client
	unregister    chan *client
	Broadcast     chan []byte // binary PNG frames
	BroadcastText chan []byte // JSON text events
}

// NewHub creates a new Hub.
func NewHub() *Hub {
	return &Hub{
		clients:       make(map[*client]bool),
		register:      make(chan *client),
		unregister:    make(chan *client),
		Broadcast:     make(chan []byte, 8),
		BroadcastText: make(chan []byte, 32),
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
				close(c.sendText)
			}
		case frame := <-h.Broadcast:
			for c := range h.clients {
				select {
				case c.send <- frame:
				default:
					// Slow client: drop frame.
				}
			}
		case msg := <-h.BroadcastText:
			for c := range h.clients {
				select {
				case c.sendText <- msg:
				default:
				}
			}
		}
	}
}
