package server

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/pisush/sketch-talk/assets"
	"github.com/pisush/sketch-talk/internal/questions"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

// wsEvent is a typed JSON message sent over the WebSocket alongside PNG frames.
type wsEvent struct {
	Type  string `json:"type"`
	Text  string `json:"text,omitempty"`
	Asker string `json:"asker,omitempty"`
	Count int    `json:"count,omitempty"`
}

// Server is the HTTP + WebSocket display server.
type Server struct {
	hub       *Hub
	questions *questions.Store

	latestPNG []byte
	latestMu  sync.RWMutex
}

// NewServer creates a Server with the given hub and question store.
func NewServer(hub *Hub, qs *questions.Store) *Server {
	return &Server{hub: hub, questions: qs}
}

// UpdateSnapshot stores the latest PNG for new connections.
func (s *Server) UpdateSnapshot(png []byte) {
	s.latestMu.Lock()
	s.latestPNG = png
	s.latestMu.Unlock()
}

// BroadcastQuestion sends a question event to all connected WebSocket clients.
func (s *Server) BroadcastQuestion(q questions.Question) {
	ev := wsEvent{
		Type:  "question",
		Text:  q.Text,
		Asker: q.Asker,
		Count: s.questions.Count(),
	}
	data, _ := json.Marshal(ev)
	s.hub.BroadcastText <- data
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
	mux.HandleFunc("/ask", s.handleAsk)
	mux.HandleFunc("/questions/count", s.handleQuestionCount)

	log.Printf("Display:   http://localhost%s", addr)
	log.Printf("Questions: http://localhost%s/ask  (share this link with the audience)", addr)
	return http.ListenAndServe(addr, mux)
}

func (s *Server) handleWS(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("WS upgrade: %v", err)
		return
	}
	c := &client{conn: conn, send: make(chan []byte, 16), sendText: make(chan []byte, 16)}
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

func (s *Server) handleAsk(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.NotFound(w, r)
		return
	}

	var body struct {
		Text  string `json:"text"`
		Asker string `json:"asker"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.Text == "" {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	q := s.questions.Add(body.Text, body.Asker)
	log.Printf("Question from %q: %s", q.Asker, q.Text)
	s.BroadcastQuestion(q)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{"ok": true})
}

func (s *Server) handleQuestionCount(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]int{"count": s.questions.Count()})
}

func (c *client) writePump(hub *Hub) {
	defer func() {
		hub.unregister <- c
		c.conn.Close()
	}()
	for {
		select {
		case frame, ok := <-c.send:
			if !ok {
				return
			}
			c.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := c.conn.WriteMessage(websocket.BinaryMessage, frame); err != nil {
				return
			}
		case msg, ok := <-c.sendText:
			if !ok {
				return
			}
			c.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := c.conn.WriteMessage(websocket.TextMessage, msg); err != nil {
				return
			}
		}
	}
}
