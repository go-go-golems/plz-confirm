package client

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/go-go-golems/plz-confirm/proto/generated/go/plz_confirm/v1"
	"google.golang.org/protobuf/encoding/protojson"
)

func TestWaitRequest_RetriesOn408(t *testing.T) {
	t.Parallel()

	var calls int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasSuffix(r.URL.Path, "/api/requests/req-1/wait") {
			http.NotFound(w, r)
			return
		}
		n := atomic.AddInt32(&calls, 1)
		if n <= 2 {
			http.Error(w, "timeout waiting for response", http.StatusRequestTimeout)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		resp := &v1.UIRequest{
			Id:        "req-1",
			Type:      v1.WidgetType_confirm,
			SessionId: "global",
			Input: &v1.UIRequest_ConfirmInput{
				ConfirmInput: &v1.ConfirmInput{Title: "t"},
			},
			Status:    v1.RequestStatus_completed,
			CreatedAt: time.Now().UTC().Format(time.RFC3339Nano),
			ExpiresAt: time.Now().UTC().Add(10 * time.Second).Format(time.RFC3339Nano),
		}
		b, _ := protojson.MarshalOptions{EmitUnpopulated: true}.Marshal(resp)
		_, _ = w.Write(b)
	}))
	defer srv.Close()

	c := New(srv.URL)
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	got, err := c.WaitRequest(ctx, "req-1", 1)
	if err != nil {
		t.Fatalf("WaitRequest returned error: %v", err)
	}
	if got.Id != "req-1" {
		t.Fatalf("unexpected request id: %q", got.Id)
	}
	if atomic.LoadInt32(&calls) < 3 {
		t.Fatalf("expected at least 3 calls, got %d", atomic.LoadInt32(&calls))
	}
}

func TestWaitRequest_WaitForeverHonorsContextCancel(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Always 408 quickly (simulating "not completed yet").
		http.Error(w, "timeout waiting for response", http.StatusRequestTimeout)
	}))
	defer srv.Close()

	c := New(srv.URL)
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	_, err := c.WaitRequest(ctx, "req-1", 0) // wait forever, but ctx will cancel
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
}
