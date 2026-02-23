package scriptengine

import (
	"context"
	"errors"
	"fmt"
	"math"
	"math/rand"
	"strings"
	"time"

	"github.com/dop251/goja"
	"github.com/go-go-golems/plz-confirm/proto/generated/go/plz_confirm/v1"
)

const defaultTimeout = 2 * time.Second
const contextSeedPropKey = "__pc_seed"

type Engine struct{}

func New() *Engine {
	return &Engine{}
}

type InitAndViewResult struct {
	Describe map[string]any
	State    map[string]any
	View     map[string]any
	Logs     []string
}

type UpdateAndViewResult struct {
	Done   bool
	Result map[string]any
	State  map[string]any
	View   map[string]any
	Logs   []string
}

func (e *Engine) InitAndView(ctx context.Context, in *v1.ScriptInput) (*InitAndViewResult, error) {
	if in == nil {
		return nil, errors.New("script input is required")
	}
	if strings.TrimSpace(in.GetScript()) == "" {
		return nil, errors.New("script source is required")
	}

	vm := goja.New()

	var out InitAndViewResult
	run := func() error {
		if _, err := vm.RunString(buildExportsProgram(in.GetScript())); err != nil {
			return fmt.Errorf("script load failed: %w", err)
		}

		hasDescribe, err := evalBool(vm.RunString(`typeof __pc_exports.describe === "function"`))
		if err != nil {
			return err
		}
		hasInit, err := evalBool(vm.RunString(`typeof __pc_exports.init === "function"`))
		if err != nil {
			return err
		}
		hasView, err := evalBool(vm.RunString(`typeof __pc_exports.view === "function"`))
		if err != nil {
			return err
		}
		hasUpdate, err := evalBool(vm.RunString(`typeof __pc_exports.update === "function"`))
		if err != nil {
			return err
		}
		if !hasDescribe || !hasInit || !hasView || !hasUpdate {
			return errors.New("script must export describe/init/view/update functions")
		}

		scriptCtx := defaultScriptContext(in.GetProps())
		if err := vm.Set("__pc_ctx", scriptCtx); err != nil {
			return fmt.Errorf("set ctx failed: %w", err)
		}
		if err := attachContextHelpers(vm); err != nil {
			return err
		}

		describeVal, err := vm.RunString(`__pc_exports.describe(__pc_ctx)`)
		if err != nil {
			return fmt.Errorf("describe() failed: %w", err)
		}
		describeMap, err := expectMap(describeVal.Export(), "describe result")
		if err != nil {
			return err
		}
		out.Describe = describeMap

		stateVal, err := vm.RunString(`__pc_exports.init(__pc_ctx)`)
		if err != nil {
			return fmt.Errorf("init() failed: %w", err)
		}
		stateMap, err := expectMap(stateVal.Export(), "init result")
		if err != nil {
			return err
		}
		out.State = stateMap

		if err := vm.Set("__pc_state", stateMap); err != nil {
			return fmt.Errorf("set state failed: %w", err)
		}

		viewVal, err := vm.RunString(`__pc_exports.view(__pc_state, __pc_ctx)`)
		if err != nil {
			return fmt.Errorf("view() failed: %w", err)
		}
		viewMap, err := expectMap(viewVal.Export(), "view result")
		if err != nil {
			return err
		}
		out.View = viewMap

		return nil
	}

	if err := runWithTimeout(ctx, vm, timeoutFromInput(in), run); err != nil {
		return nil, err
	}

	return &out, nil
}

func (e *Engine) UpdateAndView(
	ctx context.Context,
	in *v1.ScriptInput,
	state map[string]any,
	event map[string]any,
) (*UpdateAndViewResult, error) {
	if in == nil {
		return nil, errors.New("script input is required")
	}
	if strings.TrimSpace(in.GetScript()) == "" {
		return nil, errors.New("script source is required")
	}
	if state == nil {
		state = map[string]any{}
	}
	if event == nil {
		event = map[string]any{}
	}

	vm := goja.New()

	var out UpdateAndViewResult
	run := func() error {
		if _, err := vm.RunString(buildExportsProgram(in.GetScript())); err != nil {
			return fmt.Errorf("script load failed: %w", err)
		}

		hasUpdate, err := evalBool(vm.RunString(`typeof __pc_exports.update === "function"`))
		if err != nil {
			return err
		}
		hasView, err := evalBool(vm.RunString(`typeof __pc_exports.view === "function"`))
		if err != nil {
			return err
		}
		if !hasUpdate || !hasView {
			return errors.New("script must export update/view functions")
		}

		scriptCtx := defaultScriptContext(in.GetProps())
		if err := vm.Set("__pc_ctx", scriptCtx); err != nil {
			return fmt.Errorf("set ctx failed: %w", err)
		}
		if err := attachContextHelpers(vm); err != nil {
			return err
		}
		if err := vm.Set("__pc_state", state); err != nil {
			return fmt.Errorf("set state failed: %w", err)
		}
		if err := vm.Set("__pc_event", event); err != nil {
			return fmt.Errorf("set event failed: %w", err)
		}

		updateVal, err := vm.RunString(`__pc_exports.update(__pc_state, __pc_event, __pc_ctx)`)
		if err != nil {
			return fmt.Errorf("update() failed: %w", err)
		}

		updateMap, err := expectMap(updateVal.Export(), "update result")
		if err != nil {
			return err
		}

		if done, _ := updateMap["done"].(bool); done {
			out.Done = true
			resultMap, err := expectMap(updateMap["result"], "update.result")
			if err != nil {
				return err
			}
			out.Result = resultMap
			return nil
		}

		out.State = updateMap
		if err := vm.Set("__pc_state", out.State); err != nil {
			return fmt.Errorf("set next state failed: %w", err)
		}

		viewVal, err := vm.RunString(`__pc_exports.view(__pc_state, __pc_ctx)`)
		if err != nil {
			return fmt.Errorf("view() failed: %w", err)
		}
		viewMap, err := expectMap(viewVal.Export(), "view result")
		if err != nil {
			return err
		}
		out.View = viewMap
		return nil
	}

	if err := runWithTimeout(ctx, vm, timeoutFromInput(in), run); err != nil {
		return nil, err
	}

	return &out, nil
}

func buildExportsProgram(script string) string {
	return `
var __pc_module = { exports: {} };
var module = __pc_module;
var exports = __pc_module.exports;
function __pc_getPath(obj, path) {
  if (!obj || typeof path !== "string" || path.length === 0) return undefined;
  var parts = path.split(".");
  var cur = obj;
  for (var i = 0; i < parts.length; i++) {
    if (cur == null || typeof cur !== "object") return undefined;
    cur = cur[parts[i]];
  }
  return cur;
}
function __pc_routeKey(event) {
  if (!event || typeof event !== "object") return "";
  if (typeof event.actionId === "string" && event.actionId.length > 0) return event.actionId;
  var data = event.data;
  if (data && typeof data === "object" && typeof data.approved === "boolean") {
    return data.approved ? "approved" : "rejected";
  }
  if (typeof event.type === "string" && event.type.length > 0) return event.type;
  return "";
}
function __pc_applyStep(state, target) {
  if (!state || typeof state !== "object") return state;
  if (typeof target !== "string" || target.length === 0) return state;
  state.step = target;
  return state;
}
function __pc_branch(state, event, spec) {
  if (!state || typeof state !== "object") return state;
  if (!spec || typeof spec !== "object") return state;

  var rules = spec.rules;
  if (Array.isArray(rules)) {
    for (var i = 0; i < rules.length; i++) {
      var rule = rules[i];
      if (!rule || typeof rule !== "object") continue;
      var matched = false;
      if (typeof rule.when === "function") {
        try { matched = !!rule.when(event, state); } catch (e) { matched = false; }
      } else if (typeof rule.when === "string") {
        matched = !!__pc_getPath({ event: event, state: state }, rule.when);
      } else if (typeof rule.when === "boolean") {
        matched = rule.when;
      }
      if (matched) return __pc_applyStep(state, rule.step || rule.to);
    }
  }

  var routes = spec.routes;
  if (!routes || typeof routes !== "object" || Array.isArray(routes)) routes = spec;
  var key = __pc_routeKey(event);
  if (key && Object.prototype.hasOwnProperty.call(routes, key)) {
    return __pc_applyStep(state, routes[key]);
  }
  if (Object.prototype.hasOwnProperty.call(routes, "default")) {
    return __pc_applyStep(state, routes["default"]);
  }
  return state;
}
` + script + `
var __pc_exports = __pc_module.exports;
`
}

func attachContextHelpers(vm *goja.Runtime) error {
	if _, err := vm.RunString(`
if (__pc_ctx && typeof __pc_ctx === "object") {
  __pc_ctx.branch = function(state, event, spec) {
    return __pc_branch(state, event, spec);
  };
}
`); err != nil {
		return fmt.Errorf("attach ctx helpers failed: %w", err)
	}
	return nil
}

func evalBool(v goja.Value, err error) (bool, error) {
	if err != nil {
		return false, err
	}
	if v == nil {
		return false, errors.New("expected bool result, got nil")
	}
	b, ok := v.Export().(bool)
	if !ok {
		return false, errors.New("expected bool result")
	}
	return b, nil
}

func expectMap(v any, name string) (map[string]any, error) {
	m, ok := v.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("%s must be an object", name)
	}
	return m, nil
}

func defaultScriptContext(propsStruct interface{ AsMap() map[string]any }) map[string]any {
	props := map[string]any{}
	if propsStruct != nil {
		props = propsStruct.AsMap()
	}
	seed := contextSeed(props)
	// ctx.random/ctx.randomInt are deterministic scripting helpers, not
	// cryptographic primitives for secrets or auth flows.
	// #nosec G404 -- intentional non-crypto PRNG for reproducible script behavior.
	rng := rand.New(rand.NewSource(seed))
	return map[string]any{
		"props": props,
		"now":   time.Now().UTC().Format(time.RFC3339Nano),
		"seed":  float64(seed),
		"random": func() float64 {
			return rng.Float64()
		},
		"randomInt": func(low, high float64) int64 {
			lo := int64(math.Floor(low))
			hi := int64(math.Floor(high))
			if hi < lo {
				lo, hi = hi, lo
			}
			if lo == hi {
				return lo
			}
			return lo + rng.Int63n(hi-lo+1)
		},
	}
}

func contextSeed(props map[string]any) int64 {
	if props == nil {
		return time.Now().UnixNano()
	}
	raw, ok := props[contextSeedPropKey]
	if !ok {
		return time.Now().UnixNano()
	}
	switch v := raw.(type) {
	case int64:
		return v
	case int:
		return int64(v)
	case int32:
		return int64(v)
	case float64:
		return int64(v)
	case float32:
		return int64(v)
	default:
		return time.Now().UnixNano()
	}
}

func timeoutFromInput(in *v1.ScriptInput) time.Duration {
	if in != nil {
		if ms := in.GetTimeoutMs(); ms > 0 {
			return time.Duration(ms) * time.Millisecond
		}
	}
	return defaultTimeout
}

func runWithTimeout(
	ctx context.Context,
	vm interface{ Interrupt(any) },
	timeout time.Duration,
	fn func() error,
) error {
	runCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	stop := make(chan struct{})
	defer close(stop)

	go func() {
		select {
		case <-runCtx.Done():
			vm.Interrupt(runCtx.Err())
		case <-stop:
		}
	}()

	err := fn()
	if runErr := runCtx.Err(); runErr != nil {
		if errors.Is(runErr, context.DeadlineExceeded) {
			if err != nil {
				return fmt.Errorf("script execution timeout: %w", err)
			}
			return errors.New("script execution timeout")
		}
		if err != nil {
			return fmt.Errorf("script execution cancelled: %w", err)
		}
		return runErr
	}
	return err
}
