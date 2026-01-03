package server

import (
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-go-golems/plz-confirm/internal/store"
)

func TestStaticServing_Index(t *testing.T) {
	srv := New(store.New())
	h := srv.Handler()

	req := httptest.NewRequest("GET", "http://example.com/", nil)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	if rr.Code != 200 {
		t.Fatalf("GET / status=%d body=%q", rr.Code, rr.Body.String())
	}

	ct := rr.Header().Get("Content-Type")
	if !strings.Contains(ct, "text/html") {
		t.Fatalf("GET / Content-Type=%q, want contains text/html", ct)
	}

	if !strings.Contains(rr.Body.String(), `<div id="root"></div>`) {
		t.Fatalf("GET / body does not look like SPA index.html")
	}
}
