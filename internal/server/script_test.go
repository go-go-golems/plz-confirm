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

func TestScriptLifecycleWithGridWidget(t *testing.T) {
	t.Parallel()

	s := New(store.New())
	h := s.Handler()

	gridScript := `
module.exports = {
  describe: function () { return { name: "grid-demo", version: "1.0.0" }; },
  init: function () { return { step: "grid" }; },
  view: function () {
    return {
      widgetType: "grid",
      stepId: "board",
      input: {
        title: "Pick a cell",
        rows: 2,
        cols: 2,
        cells: [
          { value: "" },
          { value: "X", disabled: true },
          { value: "" },
          { value: "O", disabled: true }
        ],
        cellSize: "small"
      }
    };
  },
  update: function (state, event) {
    return { done: true, result: event.data || {} };
  }
};
`

	createReq := &v1.UIRequest{
		Type:      v1.WidgetType_script,
		SessionId: "global",
		Input: &v1.UIRequest_ScriptInput{
			ScriptInput: &v1.ScriptInput{
				Title:  "Grid demo",
				Script: gridScript,
			},
		},
	}

	created := postUIRequest(t, h, "/api/requests", createReq)
	if created.GetScriptView() == nil {
		t.Fatalf("expected script_view on create")
	}
	if got := created.GetScriptView().GetWidgetType(); got != "grid" {
		t.Fatalf("expected grid view on create, got %q", got)
	}

	event := &v1.ScriptEvent{
		Type:   "submit",
		StepId: toPtr("board"),
		Data:   mustStruct(t, map[string]any{"row": 1, "col": 0, "cellIndex": 2}),
	}
	completed := postScriptEvent(t, h, created.Id, event)
	if completed.GetStatus() != v1.RequestStatus_completed {
		t.Fatalf("expected completed status, got %v", completed.GetStatus())
	}
	result := completed.GetScriptOutput().GetResult().AsMap()
	if result["cellIndex"] != float64(2) {
		t.Fatalf("unexpected grid result payload: %#v", result)
	}
}

func TestScriptCreateRejectsInvalidGridViewShape(t *testing.T) {
	t.Parallel()

	s := New(store.New())
	h := s.Handler()

	invalidGridScript := `
module.exports = {
  describe: function () { return { name: "grid-bad", version: "1.0.0" }; },
  init: function () { return { step: "grid" }; },
  view: function () {
    return {
      widgetType: "grid",
      input: {
        title: "Broken board",
        rows: 2,
        cols: 2,
        cells: [{ value: "" }]
      }
    };
  },
  update: function (state) { return state; }
};
`

	createReq := &v1.UIRequest{
		Type:      v1.WidgetType_script,
		SessionId: "global",
		Input: &v1.UIRequest_ScriptInput{
			ScriptInput: &v1.ScriptInput{
				Title:  "Invalid grid",
				Script: invalidGridScript,
			},
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
		t.Fatalf("expected 400 for invalid grid view, got %d body=%s", rr.Code, rr.Body.String())
	}
	if !bytes.Contains(rr.Body.Bytes(), []byte("rows*cols")) {
		t.Fatalf("expected rows*cols validation message, got body=%s", rr.Body.String())
	}
}

func TestScriptLifecycleWithCompositeSections(t *testing.T) {
	t.Parallel()

	s := New(store.New())
	h := s.Handler()

	sectionsScript := `
module.exports = {
  describe: function () { return { name: "sections-demo", version: "1.0.0" }; },
  init: function () { return { step: "review" }; },
  view: function () {
    return {
      stepId: "review-step",
      sections: [
        {
          widgetType: "display",
          input: { content: "## Review Context", format: "markdown" }
        },
        {
          widgetType: "confirm",
          input: { title: "Approve changes?" }
        }
      ]
    };
  },
  update: function (state, event) {
    return { done: true, result: { approved: !!(event.data && event.data.approved) } };
  }
};
`

	createReq := &v1.UIRequest{
		Type:      v1.WidgetType_script,
		SessionId: "global",
		Input: &v1.UIRequest_ScriptInput{
			ScriptInput: &v1.ScriptInput{
				Title:  "Sections demo",
				Script: sectionsScript,
			},
		},
	}

	created := postUIRequest(t, h, "/api/requests", createReq)
	view := created.GetScriptView()
	if view == nil {
		t.Fatalf("expected script view on create")
	}
	if got := view.GetWidgetType(); got != "confirm" {
		t.Fatalf("expected derived interactive widget type confirm, got %q", got)
	}
	if len(view.GetSections()) != 2 {
		t.Fatalf("expected 2 sections, got %d", len(view.GetSections()))
	}
	if got := view.GetSections()[0].GetWidgetType(); got != "display" {
		t.Fatalf("expected first section to be display, got %q", got)
	}

	event := &v1.ScriptEvent{
		Type:   "submit",
		StepId: toPtr("review-step"),
		Data:   mustStruct(t, map[string]any{"approved": true}),
	}
	completed := postScriptEvent(t, h, created.Id, event)
	if completed.GetStatus() != v1.RequestStatus_completed {
		t.Fatalf("expected completed status, got %v", completed.GetStatus())
	}
}

func TestScriptCreateRejectsInvalidCompositeSections(t *testing.T) {
	t.Parallel()

	s := New(store.New())
	h := s.Handler()

	invalidSectionsScript := `
module.exports = {
  describe: function () { return { name: "sections-bad", version: "1.0.0" }; },
  init: function () { return { step: "bad" }; },
  view: function () {
    return {
      sections: [
        { widgetType: "confirm", input: { title: "A" } },
        { widgetType: "select", input: { title: "B", options: ["x", "y"] } }
      ]
    };
  },
  update: function (state) { return state; }
};
`

	createReq := &v1.UIRequest{
		Type:      v1.WidgetType_script,
		SessionId: "global",
		Input: &v1.UIRequest_ScriptInput{
			ScriptInput: &v1.ScriptInput{
				Title:  "Invalid sections",
				Script: invalidSectionsScript,
			},
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
		t.Fatalf("expected 400 for invalid sections, got %d body=%s", rr.Code, rr.Body.String())
	}
	if !bytes.Contains(rr.Body.Bytes(), []byte("exactly one interactive")) {
		t.Fatalf("expected interactive section validation message, got body=%s", rr.Body.String())
	}
}

func TestScriptCreateMapsProgressFields(t *testing.T) {
	t.Parallel()

	s := New(store.New())
	h := s.Handler()

	progressScript := `
module.exports = {
  describe: function () { return { name: "progress-demo", version: "1.0.0" }; },
  init: function () { return { step: "q3" }; },
  view: function () {
    return {
      widgetType: "confirm",
      input: { title: "Rate docs?" },
      progress: { current: 3, total: 8, label: "Question 3 of 8" }
    };
  },
  update: function (state, event) { return { done: true, result: event.data || {} }; }
};
`

	createReq := &v1.UIRequest{
		Type:      v1.WidgetType_script,
		SessionId: "global",
		Input: &v1.UIRequest_ScriptInput{
			ScriptInput: &v1.ScriptInput{
				Title:  "Progress demo",
				Script: progressScript,
			},
		},
	}

	created := postUIRequest(t, h, "/api/requests", createReq)
	progress := created.GetScriptView().GetProgress()
	if progress == nil {
		t.Fatalf("expected progress in script view")
	}
	if progress.GetCurrent() != 3 || progress.GetTotal() != 8 {
		t.Fatalf("unexpected progress payload: %+v", progress)
	}
}

func TestScriptCreateRejectsInvalidProgressFields(t *testing.T) {
	t.Parallel()

	s := New(store.New())
	h := s.Handler()

	invalidProgressScript := `
module.exports = {
  describe: function () { return { name: "progress-bad", version: "1.0.0" }; },
  init: function () { return { step: "x" }; },
  view: function () {
    return {
      widgetType: "confirm",
      input: { title: "Bad progress" },
      progress: { current: 9, total: 8 }
    };
  },
  update: function (state) { return state; }
};
`

	createReq := &v1.UIRequest{
		Type:      v1.WidgetType_script,
		SessionId: "global",
		Input: &v1.UIRequest_ScriptInput{
			ScriptInput: &v1.ScriptInput{
				Title:  "Invalid progress",
				Script: invalidProgressScript,
			},
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
		t.Fatalf("expected 400 for invalid progress, got %d body=%s", rr.Code, rr.Body.String())
	}
	if !bytes.Contains(rr.Body.Bytes(), []byte("current must be <=")) {
		t.Fatalf("expected progress validation message, got body=%s", rr.Body.String())
	}
}

func TestScriptCreateMapsBackNavigationFields(t *testing.T) {
	t.Parallel()

	s := New(store.New())
	h := s.Handler()

	backScript := `
module.exports = {
  describe: function () { return { name: "back-demo", version: "1.0.0" }; },
  init: function () { return { step: "details" }; },
  view: function () {
    return {
      widgetType: "confirm",
      input: { title: "Details step" },
      showBack: true,
      backLabel: "Go Back"
    };
  },
  update: function (state, event) {
    if (event.type === "back") {
      state.step = "confirm";
      return state;
    }
    return { done: true, result: event.data || {} };
  }
};
`

	createReq := &v1.UIRequest{
		Type:      v1.WidgetType_script,
		SessionId: "global",
		Input: &v1.UIRequest_ScriptInput{
			ScriptInput: &v1.ScriptInput{
				Title:  "Back demo",
				Script: backScript,
			},
		},
	}

	created := postUIRequest(t, h, "/api/requests", createReq)
	view := created.GetScriptView()
	if view == nil {
		t.Fatalf("expected script view on create")
	}
	if !view.GetAllowBack() {
		t.Fatalf("expected allow_back to be true")
	}
	if got := view.GetBackLabel(); got != "Go Back" {
		t.Fatalf("unexpected back label: %q", got)
	}
}

func TestScriptLifecycleWithRatingWidget(t *testing.T) {
	t.Parallel()

	s := New(store.New())
	h := s.Handler()

	ratingScript := `
module.exports = {
  describe: function () { return { name: "rating-demo", version: "1.0.0" }; },
  init: function () { return { step: "rate" }; },
  view: function () {
    return {
      widgetType: "rating",
      stepId: "rate",
      input: {
        title: "Rate this flow",
        scale: 5,
        style: "stars",
        labels: { low: "poor", high: "great" }
      }
    };
  },
  update: function (state, event) {
    return { done: true, result: { value: event.data ? event.data.value : 0 } };
  }
};
`

	createReq := &v1.UIRequest{
		Type:      v1.WidgetType_script,
		SessionId: "global",
		Input: &v1.UIRequest_ScriptInput{
			ScriptInput: &v1.ScriptInput{
				Title:  "Rating demo",
				Script: ratingScript,
			},
		},
	}
	created := postUIRequest(t, h, "/api/requests", createReq)
	if got := created.GetScriptView().GetWidgetType(); got != "rating" {
		t.Fatalf("expected rating widget type, got %q", got)
	}

	event := &v1.ScriptEvent{
		Type:   "submit",
		StepId: toPtr("rate"),
		Data:   mustStruct(t, map[string]any{"value": 4}),
	}
	completed := postScriptEvent(t, h, created.Id, event)
	if completed.GetStatus() != v1.RequestStatus_completed {
		t.Fatalf("expected completed status, got %v", completed.GetStatus())
	}
}

func TestScriptCreateRejectsInvalidRatingStyle(t *testing.T) {
	t.Parallel()

	s := New(store.New())
	h := s.Handler()

	invalidRatingScript := `
module.exports = {
  describe: function () { return { name: "rating-bad", version: "1.0.0" }; },
  init: function () { return { step: "rate" }; },
  view: function () {
    return {
      widgetType: "rating",
      input: {
        title: "Rate this flow",
        style: "bad-style"
      }
    };
  },
  update: function (state) { return state; }
};
`

	createReq := &v1.UIRequest{
		Type:      v1.WidgetType_script,
		SessionId: "global",
		Input: &v1.UIRequest_ScriptInput{
			ScriptInput: &v1.ScriptInput{
				Title:  "Rating bad",
				Script: invalidRatingScript,
			},
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
		t.Fatalf("expected 400 for invalid rating style, got %d body=%s", rr.Code, rr.Body.String())
	}
	if !bytes.Contains(rr.Body.Bytes(), []byte("style must be")) {
		t.Fatalf("expected rating style validation message, got body=%s", rr.Body.String())
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
