package server

import (
	"fmt"
	"io/fs"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/pisush/sketch-talk/assets"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

// Server is the HTTP + WebSocket display server.
type Server struct {
	hub        *Hub
	latestPNG  []byte
	latestMu   sync.RWMutex
}

// NewServer creates a Server with the given hub.
func NewServer(hub *Hub) *Server {
	return &Server{hub: hub}
}

// UpdateSnapshot stores the latest PNG for new connections.
func (s *Server) UpdateSnapshot(png []byte) {
	s.latestMu.Lock()
	s.latestPNG = png
	s.latestMu.Unlock()
}

// ListenAndServe starts the HTTP server on addr.
func (s *Server) ListenAndServe(addr string) error {
	webRoot, err := fs.Sub(assets.WebFS, "web")
	if err != nil {
		return fmt.Errorf("embed sub: %w", err)
	}
	mux := http.NewServeMux()
	mux.Handle("/", http.FileServer(http.FS(webRoot)))
	mux.HandleFunc("/ws", s.handleWS)
	mux.HandleFunc("/sketchnote.png", s.handleSnapshot)

	log.Printf("Display server: http://localhost%s", addr)
	return http.ListenAndServe(addr, mux)
}

func (s *Server) handleWS(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("WS upgrade: %v", err)
		return
	}
	c := &client{conn: conn, send: make(chan []byte, 16)}
	s.hub.register <- c

	// Send current snapshot immediately on connect.
	s.latestMu.RLock()
	if len(s.latestPNG) > 0 {
		c.send <- s.latestPNG
	}
	s.latestMu.RUnlock()

	go c.writePump(s.hub)
}

func (s *Server) handleSnapshot(w http.ResponseWriter, r *http.Request) {
	s.latestMu.RLock()
	data := s.latestPNG
	s.latestMu.RUnlock()
	if len(data) == 0 {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Content-Type", "image/png")
	_, _ = w.Write(data)
}

func (c *client) writePump(hub *Hub) {
	defer func() {
		hub.unregister <- c
		c.conn.Close()
	}()
	c.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
	for frame := range c.send {
		c.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
		if err := c.conn.WriteMessage(websocket.BinaryMessage, frame); err != nil {
			return
		}
	}
}
