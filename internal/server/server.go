package server

import (
	"bytes"
	"context"
	"encoding/json"
	stderrors "errors"
	"io"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/go-go-golems/plz-confirm/internal/store"
	"github.com/go-go-golems/plz-confirm/internal/types"
)

type Server struct {
	store  *store.Store
	ws     *wsBroadcaster
	images *ImageStore
}

type Options struct {
	Addr string
}

func New(s *store.Store) *Server {
	imgStore, err := NewImageStore(ImageStoreOptions{})
	if err != nil {
		log.Printf("[IMG] failed to initialize image store, uploads disabled: %v", err)
	}
	return &Server{
		store:  s,
		ws:     newWSBroadcaster(),
		images: imgStore,
	}
}

func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()

	// API and WebSocket routes (must come before static file serving)
	mux.HandleFunc("/ws", s.handleWS)
	mux.HandleFunc("/api/images", s.handleImagesCollection)
	mux.HandleFunc("/api/images/", s.handleImagesItem)
	mux.HandleFunc("/api/requests", s.handleRequestsCollection)
	mux.HandleFunc("/api/requests/", s.handleRequestsItem)

	// Serve embedded static files (production mode)
	// In dev, Vite serves UI on :3000 and proxies /api and /ws to backend (typically :3001).
	// In production, this server serves everything (API, WS, and static files) on :3000 by default.
	s.handleStaticFiles(mux)

	return withCORS(mux)
}

func (s *Server) handleStaticFiles(mux *http.ServeMux) {
	// Check if embedded filesystem has content
	if embeddedPublicFS == nil {
		// No embedded files - skip static serving (dev mode)
		return
	}

	// Check if embed directory exists and has content
	if _, err := embeddedPublicFS.Open("index.html"); err != nil {
		// No index.html - skip static serving (generate not run)
		return
	}

	// Serve static files with SPA fallback
	fileServer := http.FileServer(http.FS(embeddedPublicFS))
	mux.Handle("/", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Never serve static files for API or WebSocket paths
		if strings.HasPrefix(r.URL.Path, "/api") || strings.HasPrefix(r.URL.Path, "/ws") {
			http.NotFound(w, r)
			return
		}

		// Try to serve the requested file
		path := strings.TrimPrefix(r.URL.Path, "/")
		if path == "" {
			path = "index.html"
		}

		file, err := embeddedPublicFS.Open(path)
		if err != nil {
			// File not found - serve index.html for SPA routing
			indexFile, err := embeddedPublicFS.Open("index.html")
			if err != nil {
				http.NotFound(w, r)
				return
			}
			defer indexFile.Close()

			// Read and serve index.html
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			_, _ = io.Copy(w, indexFile)
			return
		}
		file.Close()

		// File exists - serve it normally
		fileServer.ServeHTTP(w, r)
	}))
}

func (s *Server) ListenAndServe(ctx context.Context, opts Options) error {
	addr := opts.Addr
	if addr == "" {
		addr = ":3000"
	}

	if s.images != nil {
		go func() {
			t := time.NewTicker(30 * time.Second)
			defer t.Stop()
			for {
				select {
				case <-ctx.Done():
					return
				case <-t.C:
					deleted := s.images.Cleanup(context.Background(), time.Now().UTC())
					if deleted > 0 {
						log.Printf("[IMG] cleaned up %d expired images", deleted)
					}
				}
			}
		}()
	}

	srv := &http.Server{
		Addr:              addr,
		Handler:           s.Handler(),
		ReadHeaderTimeout: 10 * time.Second,
	}

	errCh := make(chan error, 1)
	go func() {
		log.Printf("plz-confirm server listening on http://localhost%s", addr)
		errCh <- srv.ListenAndServe()
	}()

	select {
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = srv.Shutdown(shutdownCtx)
		// Ctrl+C / signal cancellation should be a clean shutdown (exit 0).
		if stderrors.Is(ctx.Err(), context.Canceled) {
			return nil
		}
		return ctx.Err()
	case err := <-errCh:
		return err
	}
}

// --- REST handlers ---

type createRequestBody struct {
	Type      types.WidgetType `json:"type"`
	SessionID string           `json:"sessionId"`
	Input     any              `json:"input"`
	Timeout   int              `json:"timeout"` // seconds
}

func (s *Server) handleRequestsCollection(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodPost:
		s.handleCreateRequest(w, r)
		return
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
}

func (s *Server) handleCreateRequest(w http.ResponseWriter, r *http.Request) {
	var body createRequestBody
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "invalid json body", http.StatusBadRequest)
		return
	}
	if body.Type == "" || body.Input == nil {
		http.Error(w, "missing required fields", http.StatusBadRequest)
		return
	}

	req, err := s.store.Create(r.Context(), store.CreateParams{
		Type:      body.Type,
		SessionID: body.SessionID,
		Input:     body.Input,
		TimeoutS:  body.Timeout,
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Broadcast new_request to all connected WS clients (G=no-session).
	s.ws.BroadcastJSON(map[string]any{
		"type":    "new_request",
		"request": req,
	})

	log.Printf("[API] Created request %s (%s)", req.ID, req.Type)
	writeJSON(w, http.StatusCreated, req)
}

func (s *Server) handleRequestsItem(w http.ResponseWriter, r *http.Request) {
	// Paths:
	// - /api/requests/{id}
	// - /api/requests/{id}/response
	// - /api/requests/{id}/wait
	path := strings.TrimPrefix(r.URL.Path, "/api/requests/")
	if path == "" {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	parts := strings.Split(path, "/")
	id := parts[0]
	if id == "" {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}

	if len(parts) == 1 {
		// /api/requests/{id}
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		req, err := s.store.Get(r.Context(), id)
		if err != nil {
			if stderrors.Is(err, store.ErrNotFound) {
				http.Error(w, "request not found", http.StatusNotFound)
				return
			}
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}
		writeJSON(w, http.StatusOK, req)
		return
	}

	switch parts[1] {
	case "response":
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		s.handleSubmitResponse(w, r, id)
		return
	case "wait":
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		s.handleWait(w, r, id)
		return
	default:
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
}

type submitResponseBody struct {
	Output any `json:"output"`
}

func (s *Server) handleSubmitResponse(w http.ResponseWriter, r *http.Request, id string) {
	var body submitResponseBody
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "invalid json body", http.StatusBadRequest)
		return
	}

	req, err := s.store.Complete(r.Context(), id, body.Output)
	if err != nil {
		if stderrors.Is(err, store.ErrNotFound) {
			http.Error(w, "request not found", http.StatusNotFound)
			return
		}
		if stderrors.Is(err, store.ErrAlreadyCompleted) {
			http.Error(w, "request already completed", http.StatusConflict)
			return
		}
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	// Broadcast completion to all connected WS clients.
	s.ws.BroadcastJSON(map[string]any{
		"type":    "request_completed",
		"request": req,
	})

	log.Printf("[API] Request %s completed", req.ID)
	writeJSON(w, http.StatusOK, req)
}

func (s *Server) handleWait(w http.ResponseWriter, r *http.Request, id string) {
	timeoutS := 60
	if raw := r.URL.Query().Get("timeout"); raw != "" {
		if t, err := strconv.Atoi(raw); err == nil && t > 0 {
			timeoutS = t
		}
	}

	ctx, cancel := context.WithTimeout(r.Context(), time.Duration(timeoutS)*time.Second)
	defer cancel()

	req, err := s.store.Wait(ctx, id)
	if err != nil {
		if stderrors.Is(err, store.ErrNotFound) {
			http.Error(w, "request not found", http.StatusNotFound)
			return
		}
		if stderrors.Is(err, store.ErrWaitTimeout) {
			http.Error(w, "timeout waiting for response", http.StatusRequestTimeout)
			return
		}
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusOK, req)
}

// --- Images handlers ---

type uploadImageResponse struct {
	ID       string `json:"id"`
	URL      string `json:"url"`
	MimeType string `json:"mimeType"`
	Size     int64  `json:"size"`
}

func (s *Server) handleImagesCollection(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if s.images == nil {
		http.Error(w, "image uploads not available", http.StatusServiceUnavailable)
		return
	}

	// Prevent overly large payloads.
	r.Body = http.MaxBytesReader(w, r.Body, s.images.MaxUploadBytes()+(1<<20))

	ttlSeconds := int64(3600) // default 1h
	// Parse multipart form; maxMemory only affects in-memory buffering.
	if err := r.ParseMultipartForm(32 << 20); err != nil {
		http.Error(w, "invalid multipart form", http.StatusBadRequest)
		return
	}
	if rawTTL := r.FormValue("ttlSeconds"); rawTTL != "" {
		if t, err := strconv.ParseInt(rawTTL, 10, 64); err == nil && t > 0 {
			ttlSeconds = t
		}
	}

	file, _, err := r.FormFile("file")
	if err != nil {
		http.Error(w, "missing file field", http.StatusBadRequest)
		return
	}
	defer file.Close()

	// Sniff first bytes to determine content type and validate it's an image.
	head, err := io.ReadAll(io.LimitReader(file, 512))
	if err != nil {
		http.Error(w, "failed to read upload", http.StatusBadRequest)
		return
	}
	if len(head) == 0 {
		http.Error(w, "empty file", http.StatusBadRequest)
		return
	}
	mimeType := http.DetectContentType(head)
	if !strings.HasPrefix(mimeType, "image/") {
		http.Error(w, "invalid content-type (expected image/*)", http.StatusBadRequest)
		return
	}

	expiresAt := time.Now().UTC().Add(time.Duration(ttlSeconds) * time.Second)

	// Re-attach the bytes we consumed for sniffing.
	img, err := s.images.Put(r.Context(), io.MultiReader(bytes.NewReader(head), file), mimeType, expiresAt)
	if err != nil {
		http.Error(w, "failed to store image", http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusCreated, uploadImageResponse{
		ID:       img.ID,
		URL:      "/api/images/" + img.ID,
		MimeType: img.MimeType,
		Size:     img.Size,
	})
}

func (s *Server) handleImagesItem(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if s.images == nil {
		http.Error(w, "image serving not available", http.StatusServiceUnavailable)
		return
	}

	path := strings.TrimPrefix(r.URL.Path, "/api/images/")
	if path == "" {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	parts := strings.Split(path, "/")
	id := parts[0]
	if id == "" || len(parts) != 1 {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}

	img, ok := s.images.Get(r.Context(), id)
	if !ok {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	if !img.ExpiresAt.IsZero() && time.Now().UTC().After(img.ExpiresAt) {
		s.images.Delete(context.Background(), id)
		http.Error(w, "not found", http.StatusNotFound)
		return
	}

	f, err := os.Open(img.Path)
	if err != nil {
		s.images.Delete(context.Background(), id)
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	defer f.Close()

	st, err := f.Stat()
	if err != nil {
		s.images.Delete(context.Background(), id)
		http.Error(w, "not found", http.StatusNotFound)
		return
	}

	// Keep caching conservative: these are ephemeral and may be deleted soon.
	w.Header().Set("Content-Type", img.MimeType)
	w.Header().Set("Cache-Control", "private, max-age=60")
	http.ServeContent(w, r, id, st.ModTime(), f)
}

// --- helpers ---

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}
