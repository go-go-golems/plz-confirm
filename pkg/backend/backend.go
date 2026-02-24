package backend

import (
	"context"
	"net/http"
	"strings"

	internalserver "github.com/go-go-golems/plz-confirm/internal/server"
	"github.com/go-go-golems/plz-confirm/internal/store"
)

// Server wraps the plz-confirm backend with a public embeddable API.
type Server struct {
	server *internalserver.Server
}

type ListenOptions struct {
	Addr string
}

func NewServer() *Server {
	return &Server{server: internalserver.New(store.New())}
}

func (s *Server) Handler() http.Handler {
	return s.server.Handler()
}

func (s *Server) ListenAndServe(ctx context.Context, opts ListenOptions) error {
	return s.server.ListenAndServe(ctx, internalserver.Options{
		Addr: opts.Addr,
	})
}

// Mount registers the backend handler onto mux at the provided prefix.
// Example: prefix "/confirm" mounts under /confirm/api/* and /confirm/ws.
func (s *Server) Mount(mux *http.ServeMux, prefix string) {
	Mount(mux, prefix, s.Handler())
}

func Mount(mux *http.ServeMux, prefix string, handler http.Handler) {
	if mux == nil {
		panic("backend: nil mux")
	}
	if handler == nil {
		panic("backend: nil handler")
	}

	normalized := normalizePrefix(prefix)
	if normalized == "" {
		mux.Handle("/", handler)
		return
	}

	mux.Handle(normalized+"/", http.StripPrefix(normalized, handler))
	mux.HandleFunc(normalized, func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, normalized+"/", http.StatusTemporaryRedirect)
	})
}

func normalizePrefix(prefix string) string {
	trimmed := strings.TrimSpace(prefix)
	if trimmed == "" || trimmed == "/" {
		return ""
	}
	return "/" + strings.Trim(trimmed, "/")
}
