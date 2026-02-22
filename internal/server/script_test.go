package server

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-go-golems/plz-confirm/internal/store"
	"github.com/go-go-golems/plz-confirm/proto/generated/go/plz_confirm/v1"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/known/structpb"
)

const scriptWizard = `
module.exports = {
  describe: function () {
    return { name: "deploy-wizard", version: "1.0.0", apiVersion: "v1", capabilities: ["submit"] };
  },
  init: function () {
    return { step: "confirm" };
  },
  view: function (state) {
    if (state.step === "confirm") {
      return {
        widgetType: "confirm",
        stepId: "confirm",
        input: {
          title: "Ship to production?",
          message: "This affects live traffic",
          approveText: "Ship",
          rejectText: "Review"
        }
      };
    }
    return {
      widgetType: "select",
      stepId: "pick-env",
      input: {
        title: "Pick env",
        options: ["staging", "prod"],
        multi: false,
        searchable: false
      }
    };
  },
  update: function (state, event) {
    if (state.step === "confirm") {
      if (event.type === "submit" && event.data && event.data.approved === true) {
        return { done: true, result: { approved: true, env: "prod" } };
      }
      state.step = "pick";
      return state;
    }

    if (event.type === "submit") {
      return {
        done: true,
        result: {
          approved: false,
          env: event.data ? event.data.selectedSingle : "staging"
        }
      };
    }

    return state;
  }
};
`

func TestScriptRequestLifecycle(t *testing.T) {
	t.Parallel()

	s := New(store.New())
	h := s.Handler()

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
	if created.Status != v1.RequestStatus_pending {
		t.Fatalf("expected pending request, got %v", created.Status)
	}
	if created.GetScriptView() == nil {
		t.Fatalf("expected script_view on create")
	}
	if got := created.GetScriptView().GetWidgetType(); got != "confirm" {
		t.Fatalf("unexpected initial widget type: %q", got)
	}

	// First event: reject confirm, move to select screen.
	firstEvent := &v1.ScriptEvent{
		Type: "submit",
		Data: mustStruct(t, map[string]any{"approved": false}),
	}
	updated := postScriptEvent(t, h, created.Id, firstEvent)
	if updated.Status != v1.RequestStatus_pending {
		t.Fatalf("expected pending after first event, got %v", updated.Status)
	}
	if got := updated.GetScriptView().GetWidgetType(); got != "select" {
		t.Fatalf("expected select view after first event, got %q", got)
	}
	pending := getRequest(t, h, created.Id)
	if pending.Status != v1.RequestStatus_pending {
		t.Fatalf("expected pending status on GET after patch, got %v", pending.Status)
	}
	if got := pending.GetScriptView().GetWidgetType(); got != "select" {
		t.Fatalf("expected select view on GET after patch, got %q", got)
	}
	if step := pending.GetScriptState().AsMap()["step"]; step != "pick" {
		t.Fatalf("unexpected persisted script state step after patch: %v", step)
	}

	// Second event: pick environment, complete flow.
	secondEvent := &v1.ScriptEvent{
		Type: "submit",
		Data: mustStruct(t, map[string]any{"selectedSingle": "staging"}),
	}
	completed := postScriptEvent(t, h, created.Id, secondEvent)
	if completed.Status != v1.RequestStatus_completed {
		t.Fatalf("expected completed status, got %v", completed.Status)
	}
	if completed.GetScriptOutput() == nil || completed.GetScriptOutput().GetResult() == nil {
		t.Fatalf("expected script output result on completion")
	}
	if env := completed.GetScriptOutput().GetResult().AsMap()["env"]; env != "staging" {
		t.Fatalf("unexpected script result env: %v", env)
	}
}

func TestScriptCreateRequiresDescribe(t *testing.T) {
	t.Parallel()

	s := New(store.New())
	h := s.Handler()

	badScript := `module.exports = { init: function(){ return { step: "x" }; }, view: function(){ return { widgetType: "confirm", input: { title: "x" } }; }, update: function(s){ return s; } };`
	createReq := &v1.UIRequest{
		Type:      v1.WidgetType_script,
		SessionId: "global",
		Input: &v1.UIRequest_ScriptInput{
			ScriptInput: &v1.ScriptInput{Title: "Bad", Script: badScript},
		},
	}

	body, err := protojson.Marshal(createReq)
	if err != nil {
		t.Fatalf("marshal create req: %v", err)
	}
	req := httptest.NewRequest(http.MethodPost, "/api/requests", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for missing describe, got %d body=%s", rr.Code, rr.Body.String())
	}
}

func TestScriptCreateTimeoutMapsTo504(t *testing.T) {
	t.Parallel()

	s := New(store.New())
	h := s.Handler()

	slowScript := `
module.exports = {
  describe: function() { return { name: "slow", version: "1.0.0" }; },
  init: function() { while (true) {} },
  view: function() { return { widgetType: "confirm", input: { title: "x" } }; },
  update: function(s) { return s; }
};
`
	createReq := &v1.UIRequest{
		Type:      v1.WidgetType_script,
		SessionId: "global",
		Input: &v1.UIRequest_ScriptInput{
			ScriptInput: &v1.ScriptInput{Title: "Slow", Script: slowScript, TimeoutMs: toPtr(int64(25))},
		},
	}

	body, err := protojson.Marshal(createReq)
	if err != nil {
		t.Fatalf("marshal create req: %v", err)
	}
	req := httptest.NewRequest(http.MethodPost, "/api/requests", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusGatewayTimeout {
		t.Fatalf("expected 504 timeout mapping, got %d body=%s", rr.Code, rr.Body.String())
	}
}

func TestScriptUpdateRuntimeFaultMapsTo422(t *testing.T) {
	t.Parallel()

	s := New(store.New())
	h := s.Handler()

	badUpdateScript := `
module.exports = {
  describe: function () { return { name: "faulty", version: "1.0.0" }; },
  init: function () { return { step: "confirm" }; },
  view: function () { return { widgetType: "confirm", input: { title: "x" } }; },
  update: function () { throw new Error("boom"); }
};
`
	createReq := &v1.UIRequest{
		Type:      v1.WidgetType_script,
		SessionId: "global",
		Input: &v1.UIRequest_ScriptInput{
			ScriptInput: &v1.ScriptInput{Title: "Faulty", Script: badUpdateScript},
		},
	}
	created := postUIRequest(t, h, "/api/requests", createReq)

	ev := &v1.ScriptEvent{
		Type: "submit",
		Data: mustStruct(t, map[string]any{"approved": true}),
	}
	body, err := protojson.Marshal(ev)
	if err != nil {
		t.Fatalf("marshal ScriptEvent: %v", err)
	}
	req := httptest.NewRequest(http.MethodPost, "/api/requests/"+created.Id+"/event", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusUnprocessableEntity {
		t.Fatalf("expected 422 runtime mapping, got %d body=%s", rr.Code, rr.Body.String())
	}
}

func postUIRequest(t *testing.T, h http.Handler, path string, reqProto *v1.UIRequest) *v1.UIRequest {
	t.Helper()

	body, err := protojson.Marshal(reqProto)
	if err != nil {
		t.Fatalf("marshal UIRequest: %v", err)
	}
	req := httptest.NewRequest(http.MethodPost, path, bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code < 200 || rr.Code >= 300 {
		t.Fatalf("request failed status=%d body=%s", rr.Code, rr.Body.String())
	}
	out := &v1.UIRequest{}
	if err := protojson.Unmarshal(rr.Body.Bytes(), out); err != nil {
		t.Fatalf("unmarshal UIRequest response: %v body=%s", err, rr.Body.String())
	}
	return out
}

func postScriptEvent(t *testing.T, h http.Handler, id string, ev *v1.ScriptEvent) *v1.UIRequest {
	t.Helper()

	body, err := protojson.Marshal(ev)
	if err != nil {
		t.Fatalf("marshal ScriptEvent: %v", err)
	}
	req := httptest.NewRequest(http.MethodPost, "/api/requests/"+id+"/event", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code < 200 || rr.Code >= 300 {
		t.Fatalf("event request failed status=%d body=%s", rr.Code, rr.Body.String())
	}
	out := &v1.UIRequest{}
	if err := protojson.Unmarshal(rr.Body.Bytes(), out); err != nil {
		t.Fatalf("unmarshal event response: %v body=%s", err, rr.Body.String())
	}
	return out
}

func getRequest(t *testing.T, h http.Handler, id string) *v1.UIRequest {
	t.Helper()

	req := httptest.NewRequest(http.MethodGet, "/api/requests/"+id, nil)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code < 200 || rr.Code >= 300 {
		t.Fatalf("get request failed status=%d body=%s", rr.Code, rr.Body.String())
	}

	out := &v1.UIRequest{}
	if err := protojson.Unmarshal(rr.Body.Bytes(), out); err != nil {
		t.Fatalf("unmarshal get response: %v body=%s", err, rr.Body.String())
	}
	return out
}

func mustStruct(t *testing.T, m map[string]any) *structpb.Struct {
	t.Helper()
	st, err := structpb.NewStruct(m)
	if err != nil {
		t.Fatalf("structpb.NewStruct: %v", err)
	}
	return st
}

func toPtr[T any](v T) *T {
	return &v
}
