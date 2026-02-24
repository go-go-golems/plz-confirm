package server

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gorilla/websocket"
)

const defaultWSClientQueueSize = 64

var (
	errWSClientClosed    = errors.New("ws client closed")
	errWSClientQueueFull = errors.New("ws client queue full")
)

type wsClient struct {
	conn      *websocket.Conn
	sessionID string
	send      chan []byte
	done      chan struct{}
	closeOnce sync.Once
	closed    atomic.Bool
}

func newWSClient(conn *websocket.Conn, sessionID string, queueSize int) *wsClient {
	if queueSize <= 0 {
		queueSize = defaultWSClientQueueSize
	}
	return &wsClient{
		conn:      conn,
		sessionID: sessionID,
		send:      make(chan []byte, queueSize),
		done:      make(chan struct{}),
	}
}

func (c *wsClient) start(onWriteError func(error)) {
	go c.writePump(onWriteError)
}

func (c *wsClient) writePump(onWriteError func(error)) {
	for {
		select {
		case <-c.done:
			return
		case msg := <-c.send:
			if c.conn == nil {
				continue
			}
			_ = c.conn.SetWriteDeadline(time.Now().Add(5 * time.Second))
			if err := c.conn.WriteMessage(websocket.TextMessage, msg); err != nil {
				onWriteError(err)
				return
			}
		}
	}
}

func (c *wsClient) enqueue(msg []byte) error {
	return c.enqueueWithTimeout(msg, 0)
}

func (c *wsClient) enqueueWithTimeout(msg []byte, timeout time.Duration) error {
	if c.closed.Load() {
		return errWSClientClosed
	}
	if timeout <= 0 {
		select {
		case c.send <- msg:
			return nil
		case <-c.done:
			return errWSClientClosed
		default:
			return errWSClientQueueFull
		}
	}
	timer := time.NewTimer(timeout)
	defer timer.Stop()
	select {
	case c.send <- msg:
		return nil
	case <-c.done:
		return errWSClientClosed
	case <-timer.C:
		return errWSClientQueueFull
	}
}

func (c *wsClient) stop() {
	c.closeOnce.Do(func() {
		c.closed.Store(true)
		close(c.done)
		if c.conn != nil {
			_ = c.conn.Close()
		}
	})
}

type wsBroadcaster struct {
	mu               sync.Mutex
	clientsBySession map[string]map[*wsClient]struct{}
	clientByConn     map[*websocket.Conn]*wsClient
	writeQueueSize   int
}

func newWSBroadcaster() *wsBroadcaster {
	return &wsBroadcaster{
		clientsBySession: make(map[string]map[*wsClient]struct{}),
		clientByConn:     make(map[*websocket.Conn]*wsClient),
		writeQueueSize:   defaultWSClientQueueSize,
	}
}

func (b *wsBroadcaster) add(sessionID string, conn *websocket.Conn) *wsClient {
	if sessionID == "" {
		sessionID = "global"
	}
	client := newWSClient(conn, sessionID, b.writeQueueSize)

	b.mu.Lock()
	if _, ok := b.clientsBySession[sessionID]; !ok {
		b.clientsBySession[sessionID] = make(map[*wsClient]struct{})
	}
	b.clientsBySession[sessionID][client] = struct{}{}
	b.clientByConn[conn] = client
	b.mu.Unlock()

	client.start(func(err error) {
		log.Printf("[WS] write failed, dropping client: %v", err)
		b.remove(conn)
	})
	return client
}

func (b *wsBroadcaster) remove(conn *websocket.Conn) {
	var client *wsClient
	b.mu.Lock()
	if mapped, ok := b.clientByConn[conn]; ok {
		client = mapped
		if m, exists := b.clientsBySession[mapped.sessionID]; exists {
			delete(m, mapped)
			if len(m) == 0 {
				delete(b.clientsBySession, mapped.sessionID)
			}
		}
		delete(b.clientByConn, conn)
	} else {
		for sid, m := range b.clientsBySession {
			for c := range m {
				if c.conn == conn {
					client = c
					delete(m, c)
					if len(m) == 0 {
						delete(b.clientsBySession, sid)
					}
					break
				}
			}
		}
	}
	b.mu.Unlock()

	if client != nil {
		client.stop()
	}
}

func (b *wsBroadcaster) snapshot(sessionID string) []*wsClient {
	b.mu.Lock()
	defer b.mu.Unlock()
	if sessionID == "" {
		sessionID = "global"
	}
	m := b.clientsBySession[sessionID]
	out := make([]*wsClient, 0, len(m))
	for c := range m {
		out = append(out, c)
	}
	return out
}

func (b *wsBroadcaster) BroadcastJSON(sessionID string, msg any) {
	raw, err := json.Marshal(msg)
	if err != nil {
		log.Printf("[WS] marshal failed, skipping broadcast: %v", err)
		return
	}
	b.BroadcastRawJSON(sessionID, raw)
}

func (b *wsBroadcaster) BroadcastRawJSON(sessionID string, msg []byte) {
	clients := b.snapshot(sessionID)
	for _, c := range clients {
		if err := c.enqueue(msg); err != nil {
			log.Printf("[WS] enqueue failed, dropping client: %v", err)
			b.remove(c.conn)
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
	client := s.ws.add(sessionID, conn)
	// #nosec G706 -- sessionId is quoted to neutralize control characters.
	log.Printf("[WS] client connected (sessionId=%q)", sessionID)

	// On connect, send all currently pending requests.
	pending := s.store.PendingForSession(r.Context(), sessionID)
	for _, req := range pending {
		msg, err := marshalWSEvent("new_request", req)
		if err != nil {
			log.Printf("[WS] initial marshal failed: %v", err)
			s.ws.remove(conn)
			return
		}
		if err := client.enqueueWithTimeout(msg, 5*time.Second); err != nil {
			log.Printf("[WS] initial enqueue failed: %v", err)
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
