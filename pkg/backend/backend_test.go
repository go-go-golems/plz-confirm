package backend

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-go-golems/plz-confirm/proto/generated/go/plz_confirm/v1"
	"google.golang.org/protobuf/encoding/protojson"
)

func TestServerHandler_CreatesConfirmRequest(t *testing.T) {
	srv := NewServer()
	handler := srv.Handler()

	created := createConfirmRequest(t, handler, "/api/requests")
	if created.GetId() == "" {
		t.Fatalf("expected created request id")
	}
	if created.GetType() != v1.WidgetType_confirm {
		t.Fatalf("unexpected type: got=%v want=%v", created.GetType(), v1.WidgetType_confirm)
	}
}

func TestMount_WithPrefix_CreatesConfirmRequest(t *testing.T) {
	srv := NewServer()
	mux := http.NewServeMux()
	srv.Mount(mux, "/confirm")

	created := createConfirmRequest(t, mux, "/confirm/api/requests")
	if created.GetId() == "" {
		t.Fatalf("expected created request id")
	}
	if created.GetSessionId() != "embed-session" {
		t.Fatalf("unexpected session id: got=%q", created.GetSessionId())
	}
}

func createConfirmRequest(t *testing.T, handler http.Handler, path string) *v1.UIRequest {
	t.Helper()

	payload := &v1.UIRequest{
		Type:      v1.WidgetType_confirm,
		SessionId: "embed-session",
		Input: &v1.UIRequest_ConfirmInput{
			ConfirmInput: &v1.ConfirmInput{
				Title:   "Deploy now?",
				Message: stringPtr("Ship build 42"),
			},
		},
	}

	body, err := protojson.Marshal(payload)
	if err != nil {
		t.Fatalf("marshal request: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, path, bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusCreated {
		t.Fatalf("POST %s status=%d body=%s", path, rr.Code, rr.Body.String())
	}

	out := &v1.UIRequest{}
	if err := protojson.Unmarshal(rr.Body.Bytes(), out); err != nil {
		t.Fatalf("unmarshal response: %v body=%s", err, rr.Body.String())
	}
	return out
}

func stringPtr(v string) *string {
	return &v
}
