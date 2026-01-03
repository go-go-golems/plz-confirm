package server

import (
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

type wsBroadcaster struct {
	mu               sync.Mutex
	clientsBySession map[string]map[*websocket.Conn]struct{}
	sessionByConn    map[*websocket.Conn]string
}

func newWSBroadcaster() *wsBroadcaster {
	return &wsBroadcaster{
		clientsBySession: make(map[string]map[*websocket.Conn]struct{}),
		sessionByConn:    make(map[*websocket.Conn]string),
	}
}

func (b *wsBroadcaster) add(sessionID string, conn *websocket.Conn) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if sessionID == "" {
		sessionID = "global"
	}
	if _, ok := b.clientsBySession[sessionID]; !ok {
		b.clientsBySession[sessionID] = make(map[*websocket.Conn]struct{})
	}
	b.clientsBySession[sessionID][conn] = struct{}{}
	b.sessionByConn[conn] = sessionID
}

func (b *wsBroadcaster) remove(conn *websocket.Conn) {
	b.mu.Lock()
	defer b.mu.Unlock()
	sessionID, ok := b.sessionByConn[conn]
	if ok {
		if m, ok := b.clientsBySession[sessionID]; ok {
			delete(m, conn)
			if len(m) == 0 {
				delete(b.clientsBySession, sessionID)
			}
		}
		delete(b.sessionByConn, conn)
		return
	}
	for sid, m := range b.clientsBySession {
		delete(m, conn)
		if len(m) == 0 {
			delete(b.clientsBySession, sid)
		}
	}
}

func (b *wsBroadcaster) snapshot(sessionID string) []*websocket.Conn {
	b.mu.Lock()
	defer b.mu.Unlock()
	if sessionID == "" {
		sessionID = "global"
	}
	m := b.clientsBySession[sessionID]
	out := make([]*websocket.Conn, 0, len(m))
	for c := range m {
		out = append(out, c)
	}
	return out
}

func (b *wsBroadcaster) BroadcastJSON(sessionID string, msg any) {
	conns := b.snapshot(sessionID)
	for _, c := range conns {
		_ = c.SetWriteDeadline(time.Now().Add(5 * time.Second))
		if err := c.WriteJSON(msg); err != nil {
			log.Printf("[WS] write failed, dropping client: %v", err)
			_ = c.Close()
			b.remove(c)
		}
	}
}

func (b *wsBroadcaster) BroadcastRawJSON(sessionID string, msg []byte) {
	conns := b.snapshot(sessionID)
	for _, c := range conns {
		_ = c.SetWriteDeadline(time.Now().Add(5 * time.Second))
		if err := c.WriteMessage(websocket.TextMessage, msg); err != nil {
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
	sessionID := r.URL.Query().Get("sessionId")
	if sessionID == "" {
		sessionID = "global"
	}

	conn, err := wsUpgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("[WS] upgrade error: %v", err)
		return
	}
	s.ws.add(sessionID, conn)
	log.Printf("[WS] client connected (sessionId=%s)", sessionID)

	// On connect, send all currently pending requests.
	pending := s.store.PendingForSession(r.Context(), sessionID)
	for _, req := range pending {
		_ = conn.SetWriteDeadline(time.Now().Add(5 * time.Second))
		msg, err := marshalWSEvent("new_request", req)
		if err != nil {
			log.Printf("[WS] initial marshal failed: %v", err)
			_ = conn.Close()
			s.ws.remove(conn)
			return
		}
		if err := conn.WriteMessage(websocket.TextMessage, msg); err != nil {
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
