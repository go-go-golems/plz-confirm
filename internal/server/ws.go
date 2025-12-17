package server

import (
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

type wsBroadcaster struct {
	mu      sync.Mutex
	clients map[*websocket.Conn]struct{}
}

func newWSBroadcaster() *wsBroadcaster {
	return &wsBroadcaster{
		clients: make(map[*websocket.Conn]struct{}),
	}
}

func (b *wsBroadcaster) add(conn *websocket.Conn) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.clients[conn] = struct{}{}
}

func (b *wsBroadcaster) remove(conn *websocket.Conn) {
	b.mu.Lock()
	defer b.mu.Unlock()
	delete(b.clients, conn)
}

func (b *wsBroadcaster) snapshot() []*websocket.Conn {
	b.mu.Lock()
	defer b.mu.Unlock()
	out := make([]*websocket.Conn, 0, len(b.clients))
	for c := range b.clients {
		out = append(out, c)
	}
	return out
}

func (b *wsBroadcaster) BroadcastJSON(msg any) {
	conns := b.snapshot()
	for _, c := range conns {
		_ = c.SetWriteDeadline(time.Now().Add(5 * time.Second))
		if err := c.WriteJSON(msg); err != nil {
			log.Printf("[WS] write failed, dropping client: %v", err)
			_ = c.Close()
			b.remove(c)
		}
	}
}

var wsUpgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		// Dev/proxy and local use: be permissive (matches existing Node demo posture).
		return true
	},
}

func (s *Server) handleWS(w http.ResponseWriter, r *http.Request) {
	// Compatibility: accept sessionId but ignore it (G=no-session).
	// Frontend builds `/ws?sessionId=<id>`; we just tolerate it.
	_ = r.URL.Query().Get("sessionId")

	conn, err := wsUpgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("[WS] upgrade error: %v", err)
		return
	}
	s.ws.add(conn)
	log.Printf("[WS] client connected")

	// On connect, send all currently pending requests.
	pending := s.store.Pending(r.Context())
	for _, req := range pending {
		_ = conn.SetWriteDeadline(time.Now().Add(5 * time.Second))
		if err := conn.WriteJSON(map[string]any{"type": "new_request", "request": req}); err != nil {
			log.Printf("[WS] initial send failed: %v", err)
			_ = conn.Close()
			s.ws.remove(conn)
			return
		}
	}

	// We don't expect messages from the client; just read until close.
	for {
		if _, _, err := conn.ReadMessage(); err != nil {
			_ = conn.Close()
			s.ws.remove(conn)
			log.Printf("[WS] client disconnected")
			return
		}
	}
}
