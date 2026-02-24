package server

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-go-golems/plz-confirm/internal/store"
	"github.com/go-go-golems/plz-confirm/proto/generated/go/plz_confirm/v1"
	"google.golang.org/protobuf/encoding/protojson"
)

func TestSubmitResponse_AssignsConfirmTimestampWhenMissing(t *testing.T) {
	s := New(store.New())
	h := s.Handler()

	created := postUIRequest(t, h, "/api/requests", &v1.UIRequest{
		Type:      v1.WidgetType_confirm,
		SessionId: "global",
		Input: &v1.UIRequest_ConfirmInput{
			ConfirmInput: &v1.ConfirmInput{Title: "Ship now?"},
		},
	})

	completed := postResponse(t, h, created.Id, &v1.UIRequest{
		Output: &v1.UIRequest_ConfirmOutput{
			ConfirmOutput: &v1.ConfirmOutput{
				Approved: true,
			},
		},
	})

	timestamp := completed.GetConfirmOutput().GetTimestamp()
	if timestamp == "" {
		t.Fatalf("expected confirm timestamp to be populated")
	}
	if _, err := time.Parse(time.RFC3339Nano, timestamp); err != nil {
		t.Fatalf("expected RFC3339 timestamp, got %q err=%v", timestamp, err)
	}
}

func TestSubmitResponse_AssignsImageTimestampWhenMissing(t *testing.T) {
	s := New(store.New())
	h := s.Handler()

	created := postUIRequest(t, h, "/api/requests", &v1.UIRequest{
		Type:      v1.WidgetType_image,
		SessionId: "global",
		Input: &v1.UIRequest_ImageInput{
			ImageInput: &v1.ImageInput{
				Title: "Pick one",
				Mode:  "select",
				Images: []*v1.ImageItem{
					{Src: "https://example.com/a.png", Label: toPtr("A")},
				},
			},
		},
	})

	completed := postResponse(t, h, created.Id, &v1.UIRequest{
		Output: &v1.UIRequest_ImageOutput{
			ImageOutput: &v1.ImageOutput{
				Selected: &v1.ImageOutput_SelectedString{
					SelectedString: "img-1",
				},
			},
		},
	})

	timestamp := completed.GetImageOutput().GetTimestamp()
	if timestamp == "" {
		t.Fatalf("expected image timestamp to be populated")
	}
	if _, err := time.Parse(time.RFC3339Nano, timestamp); err != nil {
		t.Fatalf("expected RFC3339 timestamp, got %q err=%v", timestamp, err)
	}
}

func postResponse(t *testing.T, h http.Handler, id string, reqProto *v1.UIRequest) *v1.UIRequest {
	t.Helper()

	body, err := protojson.Marshal(reqProto)
	if err != nil {
		t.Fatalf("marshal UIRequest response payload: %v", err)
	}
	req := httptest.NewRequest(http.MethodPost, "/api/requests/"+id+"/response", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code < 200 || rr.Code >= 300 {
		t.Fatalf("response request failed status=%d body=%s", rr.Code, rr.Body.String())
	}
	out := &v1.UIRequest{}
	if err := protojson.Unmarshal(rr.Body.Bytes(), out); err != nil {
		t.Fatalf("unmarshal response payload: %v body=%s", err, rr.Body.String())
	}
	return out
}
