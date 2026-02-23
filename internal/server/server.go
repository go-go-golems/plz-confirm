package server

import (
	"bytes"
	"context"
	"encoding/json"
	stderrors "errors"
	"io"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-go-golems/plz-confirm/internal/scriptengine"
	"github.com/go-go-golems/plz-confirm/internal/store"
	"github.com/go-go-golems/plz-confirm/proto/generated/go/plz_confirm/v1"
	"golang.org/x/sync/errgroup"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

type Server struct {
	store            *store.Store
	ws               *wsBroadcaster
	images           *ImageStore
	scripts          *scriptengine.Engine
	scriptEventLocks *keyedLock
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
		store:            s,
		ws:               newWSBroadcaster(),
		images:           imgStore,
		scripts:          scriptengine.New(),
		scriptEventLocks: newKeyedLock(),
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
	// This server can also serve everything (API, WS, and static files) on :3000 by default.
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
			defer func() {
				_ = indexFile.Close()
			}()

			// Read and serve index.html
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			_, _ = io.Copy(w, indexFile)
			return
		}
		_ = file.Close()

		// File exists - serve it normally
		fileServer.ServeHTTP(w, r)
	}))
}

func (s *Server) ListenAndServe(ctx context.Context, opts Options) error {
	addr := opts.Addr
	if addr == "" {
		addr = ":3000"
	}

	srv := &http.Server{
		Addr:              addr,
		Handler:           s.Handler(),
		ReadHeaderTimeout: 10 * time.Second,
	}

	g, gctx := errgroup.WithContext(ctx)

	g.Go(func() error {
		t := time.NewTicker(1 * time.Second)
		defer t.Stop()
		for {
			select {
			case <-gctx.Done():
				return nil
			case <-t.C:
				expired := s.store.Expire(time.Now().UTC())
				for _, req := range expired {
					if msg, err := marshalWSEvent("request_completed", req); err == nil {
						s.ws.BroadcastRawJSON(req.SessionId, msg)
					} else {
						log.Printf("[WS] marshal request_completed (timeout) failed: %v", err)
					}
				}
			}
		}
	})

	if s.images != nil {
		g.Go(func() error {
			t := time.NewTicker(30 * time.Second)
			defer t.Stop()
			for {
				select {
				case <-gctx.Done():
					return nil
				case <-t.C:
					deleted := s.images.Cleanup(context.Background(), time.Now().UTC())
					if deleted > 0 {
						log.Printf("[IMG] cleaned up %d expired images", deleted)
					}
				}
			}
		})
	}

	g.Go(func() error {
		log.Printf("plz-confirm server listening on http://localhost%s", addr)
		err := srv.ListenAndServe()
		if stderrors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return err
	})

	g.Go(func() error {
		<-gctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = srv.Shutdown(shutdownCtx)
		return nil
	})

	if err := g.Wait(); err != nil {
		return err
	}
	// Ctrl+C / signal cancellation should be a clean shutdown (exit 0).
	if stderrors.Is(ctx.Err(), context.Canceled) {
		return nil
	}
	return ctx.Err()
}

// --- REST handlers ---

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
	bodyBytes, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
	if err != nil {
		http.Error(w, "failed to read request body", http.StatusBadRequest)
		return
	}

	reqProto := &v1.UIRequest{}
	if err := protojson.Unmarshal(bodyBytes, reqProto); err != nil {
		http.Error(w, "invalid protojson UIRequest: "+err.Error(), http.StatusBadRequest)
		return
	}
	if reqProto.Type == v1.WidgetType_widget_type_unspecified || reqProto.Input == nil {
		http.Error(w, "missing required fields (type + widget input oneof)", http.StatusBadRequest)
		return
	}
	inputType, ok := widgetTypeFromInputOneof(reqProto)
	if !ok {
		http.Error(w, "invalid input oneof for UIRequest", http.StatusBadRequest)
		return
	}
	if inputType != reqProto.Type {
		http.Error(w, "input widget type does not match request type", http.StatusBadRequest)
		return
	}

	if reqProto.Type == v1.WidgetType_script {
		seed, err := newScriptSeed()
		if err != nil {
			http.Error(w, "failed to allocate script seed", http.StatusInternalServerError)
			return
		}
		seededInput, err := scriptInputWithSeed(reqProto.GetScriptInput(), seed)
		if err != nil {
			http.Error(w, "invalid script input: "+err.Error(), http.StatusBadRequest)
			return
		}
		initResult, err := s.scripts.InitAndView(r.Context(), seededInput)
		if err != nil {
			http.Error(w, "script init failed: "+err.Error(), statusForScriptError(err))
			return
		}
		reqProto.Input = &v1.UIRequest_ScriptInput{ScriptInput: seededInput}
		initResult.State = ensureSeedInState(initResult.State, seed)

		scriptState, scriptView, scriptDescribe, err := scriptInitResultToProto(initResult)
		if err != nil {
			http.Error(w, "script init result invalid: "+err.Error(), http.StatusBadRequest)
			return
		}
		reqProto.ScriptState = scriptState
		reqProto.ScriptView = scriptView
		reqProto.ScriptDescribe = scriptDescribe
		reqProto.ScriptLogs = append([]string(nil), initResult.Logs...)
	}
	if reqProto.Metadata != nil || r.RemoteAddr != "" || r.UserAgent() != "" {
		if reqProto.Metadata == nil {
			reqProto.Metadata = &v1.RequestMetadata{}
		}
		if reqProto.Metadata.RemoteAddr == nil && r.RemoteAddr != "" {
			ra := r.RemoteAddr
			reqProto.Metadata.RemoteAddr = &ra
		}
		if reqProto.Metadata.UserAgent == nil && r.UserAgent() != "" {
			ua := r.UserAgent()
			reqProto.Metadata.UserAgent = &ua
		}
	}

	// Create in store
	req, err := s.store.Create(r.Context(), reqProto)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Broadcast new_request to WS clients in this session.
	if msg, err := marshalWSEvent("new_request", req); err == nil {
		s.ws.BroadcastRawJSON(req.SessionId, msg)
	} else {
		log.Printf("[WS] marshal new_request failed: %v", err)
	}

	// #nosec G706 -- req.Id is server-generated and quoted for log safety.
	log.Printf("[API] Created request %q (%s)", req.Id, req.Type.String())
	writeProtoJSON(w, http.StatusCreated, req)
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
		writeProtoJSON(w, http.StatusOK, req)
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
	case "event":
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		s.handleScriptEvent(w, r, id)
		return
	case "touch":
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		s.handleTouch(w, r, id)
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

func (s *Server) handleTouch(w http.ResponseWriter, r *http.Request, id string) {
	req, err := s.store.Touch(r.Context(), id, time.Now().UTC())
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

	writeProtoJSON(w, http.StatusOK, req)
}

func (s *Server) handleSubmitResponse(w http.ResponseWriter, r *http.Request, id string) {
	// Get the request to determine widget type
	existingReq, err := s.store.Get(r.Context(), id)
	if err != nil {
		if stderrors.Is(err, store.ErrNotFound) {
			http.Error(w, "request not found", http.StatusNotFound)
			return
		}
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	bodyBytes, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
	if err != nil {
		http.Error(w, "failed to read request body", http.StatusBadRequest)
		return
	}
	incoming := &v1.UIRequest{}
	if err := protojson.Unmarshal(bodyBytes, incoming); err != nil {
		http.Error(w, "invalid protojson UIRequest: "+err.Error(), http.StatusBadRequest)
		return
	}
	if incoming.Output == nil {
		http.Error(w, "missing required fields (widget output oneof)", http.StatusBadRequest)
		return
	}

	outputType, ok := widgetTypeFromOutputOneof(incoming)
	if !ok {
		http.Error(w, "invalid output oneof for UIRequest", http.StatusBadRequest)
		return
	}
	if outputType != existingReq.Type {
		http.Error(w, "output widget type does not match request type", http.StatusBadRequest)
		return
	}

	outputReq := &v1.UIRequest{
		Type:   existingReq.Type,
		Output: incoming.Output,
	}
	ensureOutputTimestamps(outputReq, time.Now().UTC())

	req, err := s.store.Complete(r.Context(), id, outputReq)
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

	// Broadcast completion to WS clients in this session.
	if msg, err := marshalWSEvent("request_completed", req); err == nil {
		s.ws.BroadcastRawJSON(req.SessionId, msg)
	} else {
		log.Printf("[WS] marshal request_completed failed: %v", err)
	}

	// #nosec G706 -- req.Id is server-generated and quoted for log safety.
	log.Printf("[API] Request %q completed", req.Id)
	writeProtoJSON(w, http.StatusOK, req)
}

func ensureOutputTimestamps(req *v1.UIRequest, now time.Time) {
	if req == nil || req.Output == nil {
		return
	}

	switch output := req.Output.(type) {
	case *v1.UIRequest_ConfirmOutput:
		if output.ConfirmOutput == nil {
			return
		}
		if strings.TrimSpace(output.ConfirmOutput.Timestamp) == "" {
			output.ConfirmOutput.Timestamp = now.Format(time.RFC3339Nano)
		}
	case *v1.UIRequest_ImageOutput:
		if output.ImageOutput == nil {
			return
		}
		if strings.TrimSpace(output.ImageOutput.Timestamp) == "" {
			output.ImageOutput.Timestamp = now.Format(time.RFC3339Nano)
		}
	}
}

func widgetTypeFromOutputOneof(req *v1.UIRequest) (v1.WidgetType, bool) {
	switch req.Output.(type) {
	case *v1.UIRequest_ConfirmOutput:
		return v1.WidgetType_confirm, true
	case *v1.UIRequest_SelectOutput:
		return v1.WidgetType_select, true
	case *v1.UIRequest_FormOutput:
		return v1.WidgetType_form, true
	case *v1.UIRequest_UploadOutput:
		return v1.WidgetType_upload, true
	case *v1.UIRequest_TableOutput:
		return v1.WidgetType_table, true
	case *v1.UIRequest_ImageOutput:
		return v1.WidgetType_image, true
	case *v1.UIRequest_ScriptOutput:
		return v1.WidgetType_script, true
	default:
		return v1.WidgetType_widget_type_unspecified, false
	}
}

func widgetTypeFromInputOneof(req *v1.UIRequest) (v1.WidgetType, bool) {
	switch req.Input.(type) {
	case *v1.UIRequest_ConfirmInput:
		return v1.WidgetType_confirm, true
	case *v1.UIRequest_SelectInput:
		return v1.WidgetType_select, true
	case *v1.UIRequest_FormInput:
		return v1.WidgetType_form, true
	case *v1.UIRequest_UploadInput:
		return v1.WidgetType_upload, true
	case *v1.UIRequest_TableInput:
		return v1.WidgetType_table, true
	case *v1.UIRequest_ImageInput:
		return v1.WidgetType_image, true
	case *v1.UIRequest_ScriptInput:
		return v1.WidgetType_script, true
	default:
		return v1.WidgetType_widget_type_unspecified, false
	}
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

	writeProtoJSON(w, http.StatusOK, req)
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
	// IMPORTANT: ParseMultipartForm may spill large parts to disk; ensure we clean up temp files.
	if r.MultipartForm != nil {
		defer func() {
			_ = r.MultipartForm.RemoveAll()
		}()
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
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
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

	f, err := s.images.Open(r.Context(), id)
	if err != nil {
		s.images.Delete(context.Background(), id)
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	defer func() {
		_ = f.Close()
	}()

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
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

// writeProtoJSON writes a protobuf message as JSON using protojson
func writeProtoJSON(w http.ResponseWriter, status int, msg proto.Message) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.WriteHeader(status)
	b, err := protojson.MarshalOptions{
		EmitUnpopulated: true,
		UseProtoNames:   false, // Use json_name tags (camelCase)
	}.Marshal(msg)
	if err != nil {
		// Best-effort: keep response shape JSON, but signal failure.
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	// #nosec G705 -- payload is protojson with application/json content-type.
	_, _ = w.Write(b)
}
