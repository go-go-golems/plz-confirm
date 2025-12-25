package server

import (
	"bytes"
	"encoding/json"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-go-golems/plz-confirm/internal/store"
)

func TestUploadAndGetImage(t *testing.T) {
	t.Parallel()

	imgStore, err := NewImageStore(ImageStoreOptions{
		Dir:            t.TempDir(),
		MaxUploadBytes: 1 << 20,
	})
	if err != nil {
		t.Fatalf("NewImageStore: %v", err)
	}

	s := &Server{
		store:  store.New(),
		ws:     newWSBroadcaster(),
		images: imgStore,
	}

	// Minimal PNG header is enough for http.DetectContentType.
	pngBytes := []byte{0x89, 0x50, 0x4e, 0x47, 0x0d, 0x0a, 0x1a, 0x0a}

	var body bytes.Buffer
	w := multipart.NewWriter(&body)
	_ = w.WriteField("ttlSeconds", "5")
	part, err := w.CreateFormFile("file", "test.png")
	if err != nil {
		t.Fatalf("CreateFormFile: %v", err)
	}
	if _, err := part.Write(pngBytes); err != nil {
		t.Fatalf("write png: %v", err)
	}
	_ = w.Close()

	req := httptest.NewRequest(http.MethodPost, "/api/images", &body)
	req.Header.Set("Content-Type", w.FormDataContentType())
	rr := httptest.NewRecorder()
	s.Handler().ServeHTTP(rr, req)

	if rr.Code != http.StatusCreated {
		t.Fatalf("expected %d, got %d body=%s", http.StatusCreated, rr.Code, rr.Body.String())
	}

	var resp uploadImageResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if resp.ID == "" || resp.URL == "" {
		t.Fatalf("expected id+url, got %#v", resp)
	}

	rr2 := httptest.NewRecorder()
	req2 := httptest.NewRequest(http.MethodGet, resp.URL, nil)
	s.Handler().ServeHTTP(rr2, req2)

	if rr2.Code != http.StatusOK {
		t.Fatalf("expected %d, got %d body=%s", http.StatusOK, rr2.Code, rr2.Body.String())
	}
	if got := rr2.Header().Get("Content-Type"); got == "" || got[:6] != "image/" {
		t.Fatalf("expected image/* content-type, got %q", got)
	}
	if !bytes.Equal(rr2.Body.Bytes(), pngBytes) {
		t.Fatalf("unexpected body bytes: got=%v want=%v", rr2.Body.Bytes(), pngBytes)
	}
}

func TestUploadRejectsNonImage(t *testing.T) {
	t.Parallel()

	imgStore, err := NewImageStore(ImageStoreOptions{
		Dir:            t.TempDir(),
		MaxUploadBytes: 1 << 20,
	})
	if err != nil {
		t.Fatalf("NewImageStore: %v", err)
	}

	s := &Server{
		store:  store.New(),
		ws:     newWSBroadcaster(),
		images: imgStore,
	}

	var body bytes.Buffer
	w := multipart.NewWriter(&body)
	part, err := w.CreateFormFile("file", "test.txt")
	if err != nil {
		t.Fatalf("CreateFormFile: %v", err)
	}
	if _, err := part.Write([]byte("hello")); err != nil {
		t.Fatalf("write: %v", err)
	}
	_ = w.Close()

	req := httptest.NewRequest(http.MethodPost, "/api/images", &body)
	req.Header.Set("Content-Type", w.FormDataContentType())
	rr := httptest.NewRecorder()
	s.Handler().ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected %d, got %d body=%s", http.StatusBadRequest, rr.Code, rr.Body.String())
	}
}
