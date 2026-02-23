package scriptengine

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

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

func TestInitAndViewSupportsGridWidget(t *testing.T) {
	t.Parallel()

	e := New()
	out, err := e.InitAndView(context.Background(), &v1.ScriptInput{
		Script: `
module.exports = {
  describe: function () { return { name: "grid-flow", version: "1.0.0" }; },
  init: function () { return { step: "board" }; },
  view: function () {
    return {
      widgetType: "grid",
      input: {
        title: "Play",
        rows: 2,
        cols: 2,
        cells: [{value:""}, {value:""}, {value:""}, {value:""}]
      }
    };
  },
  update: function (state, event) { return { done: true, result: event.data || {} }; }
};
`,
	})
	if err != nil {
		t.Fatalf("InitAndView returned error: %v", err)
	}
	if got := out.View["widgetType"]; got != "grid" {
		t.Fatalf("unexpected widgetType: %v", got)
	}
}

func TestSeededRandomContextIsDeterministic(t *testing.T) {
	t.Parallel()

	script := `
module.exports = {
  describe: function () { return { name: "seeded", version: "1.0.0" }; },
  init: function (ctx) {
    return {
      seed: ctx.seed,
      r1: ctx.random(),
      r2: ctx.randomInt(1, 10)
    };
  },
  view: function (state) {
    return { widgetType: "confirm", input: { title: "seed" } };
  },
  update: function (state, event, ctx) {
    return { done: true, result: { seed: ctx.seed, r1: ctx.random(), r2: ctx.randomInt(1, 10) } };
  }
};
`

	props, err := structpb.NewStruct(map[string]any{contextSeedPropKey: float64(12345)})
	if err != nil {
		t.Fatalf("NewStruct: %v", err)
	}

	e := New()
	first, err := e.InitAndView(context.Background(), &v1.ScriptInput{
		Script: script,
		Props:  props,
	})
	if err != nil {
		t.Fatalf("InitAndView first failed: %v", err)
	}
	second, err := e.InitAndView(context.Background(), &v1.ScriptInput{
		Script: script,
		Props:  props,
	})
	if err != nil {
		t.Fatalf("InitAndView second failed: %v", err)
	}

	if first.State["r1"] != second.State["r1"] || first.State["r2"] != second.State["r2"] {
		t.Fatalf("expected deterministic random values, got first=%v second=%v", first.State, second.State)
	}
}

func TestBranchHelperSupportsRoutesAndPredicates(t *testing.T) {
	t.Parallel()

	script := `
module.exports = {
  describe: function () { return { name: "branch-helper", version: "1.0.0" }; },
  init: function () { return { step: "confirm" }; },
  view: function (state) {
    if (state.step === "details") {
      return { widgetType: "select", input: { title: "Details", options: ["a"] } };
    }
    if (state.step === "reason") {
      return { widgetType: "form", input: { title: "Reason", schema: { properties: {} } } };
    }
    if (state.step === "positive") {
      return { widgetType: "confirm", input: { title: "Positive" } };
    }
    return { widgetType: "confirm", input: { title: "Confirm" } };
  },
  update: function (state, event, ctx) {
    if (state.step === "confirm") {
      return ctx.branch(state, event, { approved: "details", rejected: "reason", default: "reason" });
    }
    return ctx.branch(state, event, {
      rules: [
        { when: function(ev) { return ev && ev.data && ev.data.score >= 4; }, step: "positive" }
      ],
      default: "reason"
    });
  }
};
`

	e := New()

	out1, err := e.UpdateAndView(
		context.Background(),
		&v1.ScriptInput{Script: script},
		map[string]any{"step": "confirm"},
		map[string]any{"type": "submit", "data": map[string]any{"approved": true}},
	)
	if err != nil {
		t.Fatalf("UpdateAndView route-table failed: %v", err)
	}
	if got := out1.State["step"]; got != "details" {
		t.Fatalf("expected step=details from route table, got %v", got)
	}
	if got := out1.View["widgetType"]; got != "select" {
		t.Fatalf("expected select view after route table branch, got %v", got)
	}

	out2, err := e.UpdateAndView(
		context.Background(),
		&v1.ScriptInput{Script: script},
		map[string]any{"step": "details"},
		map[string]any{"type": "submit", "data": map[string]any{"score": float64(5)}},
	)
	if err != nil {
		t.Fatalf("UpdateAndView predicate rules failed: %v", err)
	}
	if got := out2.State["step"]; got != "positive" {
		t.Fatalf("expected step=positive from predicate rule, got %v", got)
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
	if !errors.Is(err, ErrScriptTimeout) {
		t.Fatalf("expected ErrScriptTimeout, got: %v", err)
	}
}

func TestContextCancel(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	e := New()
	_, err := e.InitAndView(ctx, &v1.ScriptInput{
		Script: `
module.exports = {
  describe: function() { return { name: "cancel", version: "1" }; },
  init: function() { while (true) {} },
  view: function(s) { return { widgetType: "confirm", input: { title: "x" } }; },
  update: function(s, e) { return s; }
};
`,
		TimeoutMs: toPtr(int64(1000)),
	})
	if err == nil {
		t.Fatalf("expected cancellation error")
	}
	if !errors.Is(err, ErrScriptCancelled) {
		t.Fatalf("expected ErrScriptCancelled, got: %v", err)
	}
}

func TestSandboxAllowsRequireAndConsole(t *testing.T) {
	t.Parallel()

	e := New()
	out, err := e.InitAndView(context.Background(), &v1.ScriptInput{
		Script: `
module.exports = {
  describe: function() {
    return { name: "sandbox", version: "1.0.0" };
  },
  init: function(ctx) {
    return {
      hasRequire: typeof require === "function",
      hasConsole: typeof console === "object",
      noProcess: typeof process === "undefined"
    };
  },
  view: function(state) {
    return { widgetType: "confirm", input: { title: "sandbox" } };
  },
  update: function(state, event) {
    return { done: true, result: state };
  }
};
`,
		TimeoutMs: toPtr(int64(100)),
	})
	if err != nil {
		t.Fatalf("InitAndView returned error: %v", err)
	}

	if out.State["hasRequire"] != true {
		t.Fatalf("expected require to be available, got: %v", out.State["hasRequire"])
	}
	if out.State["hasConsole"] != true {
		t.Fatalf("expected console to be available, got: %v", out.State["hasConsole"])
	}
	if out.State["noProcess"] != true {
		t.Fatalf("expected process to be unavailable, got: %v", out.State["noProcess"])
	}
}

func TestConsoleLogsCapturedFromScriptRun(t *testing.T) {
	t.Parallel()

	e := New()
	out, err := e.InitAndView(context.Background(), &v1.ScriptInput{
		Script: `
module.exports = {
  describe: function() {
    console.log("describe-start", 1);
    return { name: "logs", version: "1.0.0" };
  },
  init: function() {
    console.warn("init-warn");
    return { step: "confirm" };
  },
  view: function() {
    console.error("view-error");
    return { widgetType: "confirm", input: { title: "x" } };
  },
  update: function(state, event) {
    return { done: true, result: { ok: true } };
  }
};
`,
	})
	if err != nil {
		t.Fatalf("InitAndView returned error: %v", err)
	}
	if len(out.Logs) < 3 {
		t.Fatalf("expected at least 3 logs, got %d (%v)", len(out.Logs), out.Logs)
	}
	if !strings.Contains(strings.Join(out.Logs, "\n"), "describe-start") {
		t.Fatalf("expected describe-start log, got %v", out.Logs)
	}
	if !strings.Contains(strings.Join(out.Logs, "\n"), "init-warn") {
		t.Fatalf("expected init-warn log, got %v", out.Logs)
	}
	if !strings.Contains(strings.Join(out.Logs, "\n"), "view-error") {
		t.Fatalf("expected view-error log, got %v", out.Logs)
	}
}

func TestConsoleLogsAreTruncatedWhenOverLimit(t *testing.T) {
	t.Parallel()

	e := New()
	out, err := e.InitAndView(context.Background(), &v1.ScriptInput{
		Script: `
module.exports = {
  describe: function() { return { name: "truncate", version: "1.0.0" }; },
  init: function() {
    for (var i = 0; i < 500; i++) {
      console.log("line-" + i);
    }
    return { step: "confirm" };
  },
  view: function() { return { widgetType: "confirm", input: { title: "x" } }; },
  update: function(state, event) { return { done: true, result: { ok: true } }; }
};
`,
	})
	if err != nil {
		t.Fatalf("InitAndView returned error: %v", err)
	}
	if len(out.Logs) == 0 {
		t.Fatalf("expected logs to be captured")
	}
	if !strings.Contains(out.Logs[len(out.Logs)-1], scriptLogTruncatedLine) {
		t.Fatalf("expected truncation sentinel at end, got %q", out.Logs[len(out.Logs)-1])
	}
}

func TestCancelPathReturnsQuickly(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(context.Background())
	e := New()

	done := make(chan error, 1)
	go func() {
		_, err := e.InitAndView(ctx, &v1.ScriptInput{
			Script: `
module.exports = {
  describe: function() { return { name: "cancel-quick", version: "1" }; },
  init: function() { while (true) {} },
  view: function(s) { return { widgetType: "confirm", input: { title: "x" } }; },
  update: function(s, e) { return s; }
};
`,
			TimeoutMs: toPtr(int64(1000)),
		})
		done <- err
	}()

	time.Sleep(10 * time.Millisecond)
	cancel()

	select {
	case err := <-done:
		if err == nil {
			t.Fatalf("expected cancellation error")
		}
	case <-time.After(500 * time.Millisecond):
		t.Fatalf("expected cancellation path to return quickly")
	}
}

func toPtr[T any](v T) *T {
	return &v
}
