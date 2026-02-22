package scriptengine

import (
	"context"
	"strings"
	"testing"

	"github.com/go-go-golems/plz-confirm/proto/generated/go/plz_confirm/v1"
	"google.golang.org/protobuf/types/known/structpb"
)

const happyPathScript = `
module.exports = {
  describe: function (ctx) {
    return { name: "wizard", version: "1.0.0", apiVersion: "v1", capabilities: ["submit"] };
  },
  init: function (ctx) {
    return { step: "confirm", count: 0 };
  },
  view: function (state, ctx) {
    if (state.step === "confirm") {
      return {
        widgetType: "confirm",
        input: {
          title: "Approve deploy?",
          message: "step=" + state.step,
          approveText: "OK",
          rejectText: "NO"
        },
        stepId: "confirm-step"
      };
    }
    return {
      widgetType: "select",
      input: {
        title: "Choose env",
        options: ["dev", "prod"],
        multi: false,
        searchable: false
      },
      stepId: "select-step"
    };
  },
  update: function (state, event, ctx) {
    if (state.step === "confirm") {
      state.count = state.count + 1;
      if (event && event.type === "submit" && event.data && event.data.approved === true) {
        return { done: true, result: { approved: true, count: state.count } };
      }
      state.step = "select";
      return state;
    }

    return { done: true, result: { approved: false, count: state.count } };
  }
};
`

func TestInitAndView(t *testing.T) {
	t.Parallel()

	props, err := structpb.NewStruct(map[string]any{"ticket": "PC-01"})
	if err != nil {
		t.Fatalf("NewStruct: %v", err)
	}

	e := New()
	out, err := e.InitAndView(context.Background(), &v1.ScriptInput{
		Title:  "Script wizard",
		Script: happyPathScript,
		Props:  props,
	})
	if err != nil {
		t.Fatalf("InitAndView returned error: %v", err)
	}

	if got := out.Describe["name"]; got != "wizard" {
		t.Fatalf("unexpected describe.name: %v", got)
	}
	if got := out.State["step"]; got != "confirm" {
		t.Fatalf("unexpected initial step: %v", got)
	}
	if got := out.View["widgetType"]; got != "confirm" {
		t.Fatalf("unexpected widgetType: %v", got)
	}
}

func TestUpdateAndView(t *testing.T) {
	t.Parallel()

	e := New()
	state := map[string]any{"step": "confirm", "count": float64(0)}
	event := map[string]any{
		"type": "submit",
		"data": map[string]any{"approved": false},
	}

	out, err := e.UpdateAndView(context.Background(), &v1.ScriptInput{Script: happyPathScript}, state, event)
	if err != nil {
		t.Fatalf("UpdateAndView returned error: %v", err)
	}
	if out.Done {
		t.Fatalf("expected non-terminal update")
	}
	if got := out.State["step"]; got != "select" {
		t.Fatalf("unexpected step after update: %v", got)
	}
	if got := out.View["widgetType"]; got != "select" {
		t.Fatalf("unexpected view widgetType: %v", got)
	}
}

func TestUpdateDone(t *testing.T) {
	t.Parallel()

	e := New()
	state := map[string]any{"step": "confirm", "count": float64(0)}
	event := map[string]any{
		"type": "submit",
		"data": map[string]any{"approved": true},
	}

	out, err := e.UpdateAndView(context.Background(), &v1.ScriptInput{Script: happyPathScript}, state, event)
	if err != nil {
		t.Fatalf("UpdateAndView returned error: %v", err)
	}
	if !out.Done {
		t.Fatalf("expected terminal update")
	}
	if got := out.Result["approved"]; got != true {
		t.Fatalf("unexpected result.approved: %v", got)
	}
}

func TestTimeout(t *testing.T) {
	t.Parallel()

	e := New()
	_, err := e.InitAndView(context.Background(), &v1.ScriptInput{
		Script: `
module.exports = {
  describe: function() { return { name: "bad", version: "1" }; },
  init: function() { while (true) {} },
  view: function(s) { return { widgetType: "confirm", input: { title: "x" } }; },
  update: function(s, e) { return s; }
};
`,
		TimeoutMs: toPtr(int64(25)),
	})
	if err == nil {
		t.Fatalf("expected timeout error")
	}
	if !strings.Contains(err.Error(), "timeout") {
		t.Fatalf("expected timeout in error, got: %v", err)
	}
}

func toPtr[T any](v T) *T {
	return &v
}
