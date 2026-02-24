package server

import (
	"encoding/json"
	"errors"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/go-go-golems/plz-confirm/internal/store"
	"github.com/go-go-golems/plz-confirm/proto/generated/go/plz_confirm/v1"
	"github.com/gorilla/websocket"
	"google.golang.org/protobuf/encoding/protojson"
)

type wsEventMessage struct {
	Type    string          `json:"type"`
	Request json.RawMessage `json:"request"`
}

func TestWebSocketScriptLifecycleEventsAreOrdered(t *testing.T) {
	s := New(store.New())
	h := s.Handler()

	ts := httptest.NewServer(h)
	defer ts.Close()

	conn := dialWS(t, ts.URL, "global")
	defer func() {
		_ = conn.Close()
	}()

	createReq := &v1.UIRequest{
		Type:      v1.WidgetType_script,
		SessionId: "global",
		Input: &v1.UIRequest_ScriptInput{
			ScriptInput: &v1.ScriptInput{
				Title:  "Deploy wizard",
				Script: scriptWizard,
			},
		},
	}
	created := postUIRequest(t, h, "/api/requests", createReq)

	eventType, eventReq := readWSEvent(t, conn)
	if eventType != "new_request" {
		t.Fatalf("expected first websocket event to be new_request, got %q", eventType)
	}
	if eventReq.Id != created.Id {
		t.Fatalf("unexpected request id in first websocket event: got=%s want=%s", eventReq.Id, created.Id)
	}

	firstEvent := &v1.ScriptEvent{
		Type: "submit",
		Data: mustStruct(t, map[string]any{"approved": false}),
	}
	updated := postScriptEvent(t, h, created.Id, firstEvent)

	eventType, eventReq = readWSEvent(t, conn)
	if eventType != "request_updated" {
		t.Fatalf("expected second websocket event to be request_updated, got %q", eventType)
	}
	if eventReq.Id != updated.Id {
		t.Fatalf("unexpected request id in second websocket event: got=%s want=%s", eventReq.Id, updated.Id)
	}
	if eventReq.Status != v1.RequestStatus_pending {
		t.Fatalf("expected pending status in request_updated event, got %v", eventReq.Status)
	}

	secondEvent := &v1.ScriptEvent{
		Type: "submit",
		Data: mustStruct(t, map[string]any{"selectedSingle": "staging"}),
	}
	completed := postScriptEvent(t, h, created.Id, secondEvent)

	eventType, eventReq = readWSEvent(t, conn)
	if eventType != "request_completed" {
		t.Fatalf("expected third websocket event to be request_completed, got %q", eventType)
	}
	if eventReq.Id != completed.Id {
		t.Fatalf("unexpected request id in third websocket event: got=%s want=%s", eventReq.Id, completed.Id)
	}
	if eventReq.Status != v1.RequestStatus_completed {
		t.Fatalf("expected completed status in request_completed event, got %v", eventReq.Status)
	}
}

func TestWebSocketInitialPendingEventsFollowCreationOrder(t *testing.T) {
	s := New(store.New())
	h := s.Handler()

	first := postUIRequest(t, h, "/api/requests", &v1.UIRequest{
		Type:      v1.WidgetType_confirm,
		SessionId: "global",
		Input: &v1.UIRequest_ConfirmInput{
			ConfirmInput: &v1.ConfirmInput{Title: "first"},
		},
	})
	time.Sleep(2 * time.Millisecond)
	second := postUIRequest(t, h, "/api/requests", &v1.UIRequest{
		Type:      v1.WidgetType_confirm,
		SessionId: "global",
		Input: &v1.UIRequest_ConfirmInput{
			ConfirmInput: &v1.ConfirmInput{Title: "second"},
		},
	})

	ts := httptest.NewServer(h)
	defer ts.Close()

	conn := dialWS(t, ts.URL, "global")
	defer func() {
		_ = conn.Close()
	}()

	eventType1, eventReq1 := readWSEvent(t, conn)
	eventType2, eventReq2 := readWSEvent(t, conn)

	if eventType1 != "new_request" || eventType2 != "new_request" {
		t.Fatalf("expected initial websocket events to be new_request/new_request, got %q/%q", eventType1, eventType2)
	}
	if eventReq1.Id != first.Id {
		t.Fatalf("expected first pending event id=%s, got %s", first.Id, eventReq1.Id)
	}
	if eventReq2.Id != second.Id {
		t.Fatalf("expected second pending event id=%s, got %s", second.Id, eventReq2.Id)
	}
}

func dialWS(t *testing.T, serverURL, sessionID string) *websocket.Conn {
	t.Helper()

	wsURL := "ws" + strings.TrimPrefix(serverURL, "http") + "/ws?sessionId=" + sessionID
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("dial websocket: %v", err)
	}
	return conn
}

func readWSEvent(t *testing.T, conn *websocket.Conn) (string, *v1.UIRequest) {
	t.Helper()

	_ = conn.SetReadDeadline(time.Now().Add(2 * time.Second))
	_, msg, err := conn.ReadMessage()
	if err != nil {
		t.Fatalf("read websocket message: %v", err)
	}

	ev := &wsEventMessage{}
	if err := json.Unmarshal(msg, ev); err != nil {
		t.Fatalf("unmarshal websocket event: %v body=%s", err, string(msg))
	}
	if ev.Type == "" {
		t.Fatalf("websocket event missing type: %s", string(msg))
	}

	req := &v1.UIRequest{}
	if err := protojson.Unmarshal(ev.Request, req); err != nil {
		t.Fatalf("unmarshal websocket request payload: %v payload=%s", err, string(ev.Request))
	}
	return ev.Type, req
}

func TestWSClientEnqueueWithTimeoutReturnsQueueFull(t *testing.T) {
	client := newWSClient(nil, "global", 1)
	t.Cleanup(client.stop)

	if err := client.enqueue([]byte(`{"type":"new_request"}`)); err != nil {
		t.Fatalf("expected first enqueue to succeed, got error: %v", err)
	}
	err := client.enqueueWithTimeout([]byte(`{"type":"new_request"}`), 5*time.Millisecond)
	if !errors.Is(err, errWSClientQueueFull) {
		t.Fatalf("expected queue full error, got: %v", err)
	}
}

func TestWSClientEnqueueAfterStopReturnsClosed(t *testing.T) {
	client := newWSClient(nil, "global", 1)
	client.stop()

	err := client.enqueue([]byte(`{"type":"new_request"}`))
	if !errors.Is(err, errWSClientClosed) {
		t.Fatalf("expected closed error, got: %v", err)
	}
}
