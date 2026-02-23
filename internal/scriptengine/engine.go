package scriptengine

import (
	"context"
	"errors"
	"fmt"
	"math"
	"math/rand"
	"strconv"
	"strings"
	"time"

	"github.com/dop251/goja"
	ggjengine "github.com/go-go-golems/go-go-goja/engine"
	"github.com/go-go-golems/plz-confirm/proto/generated/go/plz_confirm/v1"
)

const defaultTimeout = 2 * time.Second
const contextSeedPropKey = "__pc_seed"
const maxScriptLogLines = 200
const maxScriptLogBytes = 64 * 1024
const scriptLogTruncatedLine = "[system] log output truncated"

var (
	ErrScriptSetup      = errors.New("script setup failed")
	ErrScriptValidation = errors.New("script validation failed")
	ErrScriptRuntime    = errors.New("script runtime failed")
	ErrScriptTimeout    = errors.New("script execution timeout")
	ErrScriptCancelled  = errors.New("script execution cancelled")
)

type Engine struct {
	runtimeFactory *ggjengine.Factory
	factoryErr     error
}

func New() *Engine {
	factory, err := ggjengine.NewBuilder().Build()
	return &Engine{
		runtimeFactory: factory,
		factoryErr:     err,
	}
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

type runLogCollector struct {
	lines     []string
	bytes     int
	truncated bool
}

func newRunLogCollector() *runLogCollector {
	return &runLogCollector{
		lines: make([]string, 0, 8),
	}
}

func (c *runLogCollector) Add(level string, args ...goja.Value) {
	if c == nil {
		return
	}
	if c.truncated {
		return
	}
	line := "[" + level + "] " + formatConsoleArgs(args)
	if len(c.lines) >= maxScriptLogLines || c.bytes+len(line) > maxScriptLogBytes {
		c.lines = append(c.lines, scriptLogTruncatedLine)
		c.bytes += len(scriptLogTruncatedLine)
		c.truncated = true
		return
	}
	c.lines = append(c.lines, line)
	c.bytes += len(line)
}

func (c *runLogCollector) Snapshot() []string {
	if c == nil || len(c.lines) == 0 {
		return []string{}
	}
	out := make([]string, len(c.lines))
	copy(out, c.lines)
	return out
}

func formatConsoleArgs(args []goja.Value) string {
	if len(args) == 0 {
		return ""
	}
	parts := make([]string, 0, len(args))
	for _, arg := range args {
		parts = append(parts, formatConsoleValue(arg))
	}
	return strings.Join(parts, " ")
}

func formatConsoleValue(v goja.Value) string {
	if v == nil {
		return "undefined"
	}
	if goja.IsUndefined(v) {
		return "undefined"
	}
	if goja.IsNull(v) {
		return "null"
	}
	switch value := v.Export().(type) {
	case string:
		return value
	case bool:
		return strconv.FormatBool(value)
	case int:
		return strconv.Itoa(value)
	case int64:
		return strconv.FormatInt(value, 10)
	case int32:
		return strconv.FormatInt(int64(value), 10)
	case float64:
		return strconv.FormatFloat(value, 'f', -1, 64)
	case float32:
		return strconv.FormatFloat(float64(value), 'f', -1, 32)
	default:
		return fmt.Sprint(value)
	}
}

func installConsoleCapture(vm *goja.Runtime, collector *runLogCollector) error {
	consoleObj := vm.NewObject()
	for _, level := range []string{"log", "info", "warn", "error"} {
		logLevel := level
		if err := consoleObj.Set(logLevel, func(call goja.FunctionCall) goja.Value {
			collector.Add(logLevel, call.Arguments...)
			return goja.Undefined()
		}); err != nil {
			return fmt.Errorf("%w: set console.%s: %v", ErrScriptSetup, logLevel, err)
		}
	}
	if err := vm.Set("console", consoleObj); err != nil {
		return fmt.Errorf("%w: set console object: %v", ErrScriptSetup, err)
	}
	return nil
}

func (e *Engine) newRuntime(ctx context.Context, collector *runLogCollector) (*ggjengine.Runtime, error) {
	if e == nil {
		return nil, fmt.Errorf("%w: engine is nil", ErrScriptSetup)
	}
	if e.factoryErr != nil {
		return nil, fmt.Errorf("%w: runtime factory build: %v", ErrScriptSetup, e.factoryErr)
	}
	if e.runtimeFactory == nil {
		return nil, fmt.Errorf("%w: runtime factory is nil", ErrScriptSetup)
	}
	rt, err := e.runtimeFactory.NewRuntime(ctx)
	if err != nil {
		return nil, fmt.Errorf("%w: new runtime: %v", ErrScriptSetup, err)
	}
	if err := installConsoleCapture(rt.VM, collector); err != nil {
		_ = rt.Close(ctx)
		return nil, err
	}
	return rt, nil
}

func (e *Engine) InitAndView(ctx context.Context, in *v1.ScriptInput) (*InitAndViewResult, error) {
	if in == nil {
		return nil, fmt.Errorf("%w: script input is required", ErrScriptValidation)
	}
	if strings.TrimSpace(in.GetScript()) == "" {
		return nil, fmt.Errorf("%w: script source is required", ErrScriptValidation)
	}

	collector := newRunLogCollector()
	rt, err := e.newRuntime(ctx, collector)
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = rt.Close(ctx)
	}()

	var out InitAndViewResult
	run := func() error {
		if _, err := rt.VM.RunString(buildExportsProgram(in.GetScript())); err != nil {
			return fmt.Errorf("%w: script load failed: %v", ErrScriptValidation, err)
		}

		hasDescribe, err := evalBool(rt.VM.RunString(`typeof __pc_exports.describe === "function"`))
		if err != nil {
			return err
		}
		hasInit, err := evalBool(rt.VM.RunString(`typeof __pc_exports.init === "function"`))
		if err != nil {
			return err
		}
		hasView, err := evalBool(rt.VM.RunString(`typeof __pc_exports.view === "function"`))
		if err != nil {
			return err
		}
		hasUpdate, err := evalBool(rt.VM.RunString(`typeof __pc_exports.update === "function"`))
		if err != nil {
			return err
		}
		if !hasDescribe || !hasInit || !hasView || !hasUpdate {
			return fmt.Errorf("%w: script must export describe/init/view/update functions", ErrScriptValidation)
		}

		scriptCtx := defaultScriptContext(in.GetProps())
		if err := rt.VM.Set("__pc_ctx", scriptCtx); err != nil {
			return fmt.Errorf("%w: set ctx failed: %v", ErrScriptSetup, err)
		}
		if err := attachContextHelpers(rt.VM); err != nil {
			return err
		}

		describeVal, err := rt.VM.RunString(`__pc_exports.describe(__pc_ctx)`)
		if err != nil {
			return fmt.Errorf("%w: describe() failed: %v", ErrScriptRuntime, err)
		}
		describeMap, err := expectMap(describeVal.Export(), "describe result")
		if err != nil {
			return err
		}
		out.Describe = describeMap

		stateVal, err := rt.VM.RunString(`__pc_exports.init(__pc_ctx)`)
		if err != nil {
			return fmt.Errorf("%w: init() failed: %v", ErrScriptRuntime, err)
		}
		stateMap, err := expectMap(stateVal.Export(), "init result")
		if err != nil {
			return err
		}
		out.State = stateMap

		if err := rt.VM.Set("__pc_state", stateMap); err != nil {
			return fmt.Errorf("%w: set state failed: %v", ErrScriptSetup, err)
		}

		viewVal, err := rt.VM.RunString(`__pc_exports.view(__pc_state, __pc_ctx)`)
		if err != nil {
			return fmt.Errorf("%w: view() failed: %v", ErrScriptRuntime, err)
		}
		viewMap, err := expectMap(viewVal.Export(), "view result")
		if err != nil {
			return err
		}
		out.View = viewMap

		return nil
	}

	if err := runWithTimeout(ctx, rt.VM, timeoutFromInput(in), run); err != nil {
		return nil, err
	}
	out.Logs = collector.Snapshot()

	return &out, nil
}

func (e *Engine) UpdateAndView(
	ctx context.Context,
	in *v1.ScriptInput,
	state map[string]any,
	event map[string]any,
) (*UpdateAndViewResult, error) {
	if in == nil {
		return nil, fmt.Errorf("%w: script input is required", ErrScriptValidation)
	}
	if strings.TrimSpace(in.GetScript()) == "" {
		return nil, fmt.Errorf("%w: script source is required", ErrScriptValidation)
	}
	if state == nil {
		state = map[string]any{}
	}
	if event == nil {
		event = map[string]any{}
	}

	collector := newRunLogCollector()
	rt, err := e.newRuntime(ctx, collector)
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = rt.Close(ctx)
	}()

	var out UpdateAndViewResult
	run := func() error {
		if _, err := rt.VM.RunString(buildExportsProgram(in.GetScript())); err != nil {
			return fmt.Errorf("%w: script load failed: %v", ErrScriptValidation, err)
		}

		hasUpdate, err := evalBool(rt.VM.RunString(`typeof __pc_exports.update === "function"`))
		if err != nil {
			return err
		}
		hasView, err := evalBool(rt.VM.RunString(`typeof __pc_exports.view === "function"`))
		if err != nil {
			return err
		}
		if !hasUpdate || !hasView {
			return fmt.Errorf("%w: script must export update/view functions", ErrScriptValidation)
		}

		scriptCtx := defaultScriptContext(in.GetProps())
		if err := rt.VM.Set("__pc_ctx", scriptCtx); err != nil {
			return fmt.Errorf("%w: set ctx failed: %v", ErrScriptSetup, err)
		}
		if err := attachContextHelpers(rt.VM); err != nil {
			return err
		}
		if err := rt.VM.Set("__pc_state", state); err != nil {
			return fmt.Errorf("%w: set state failed: %v", ErrScriptSetup, err)
		}
		if err := rt.VM.Set("__pc_event", event); err != nil {
			return fmt.Errorf("%w: set event failed: %v", ErrScriptSetup, err)
		}

		updateVal, err := rt.VM.RunString(`__pc_exports.update(__pc_state, __pc_event, __pc_ctx)`)
		if err != nil {
			return fmt.Errorf("%w: update() failed: %v", ErrScriptRuntime, err)
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
		if err := rt.VM.Set("__pc_state", out.State); err != nil {
			return fmt.Errorf("%w: set next state failed: %v", ErrScriptSetup, err)
		}

		viewVal, err := rt.VM.RunString(`__pc_exports.view(__pc_state, __pc_ctx)`)
		if err != nil {
			return fmt.Errorf("%w: view() failed: %v", ErrScriptRuntime, err)
		}
		viewMap, err := expectMap(viewVal.Export(), "view result")
		if err != nil {
			return err
		}
		out.View = viewMap
		return nil
	}

	if err := runWithTimeout(ctx, rt.VM, timeoutFromInput(in), run); err != nil {
		return nil, err
	}
	out.Logs = collector.Snapshot()

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
		return fmt.Errorf("%w: attach ctx helpers failed: %v", ErrScriptSetup, err)
	}
	return nil
}

func evalBool(v goja.Value, err error) (bool, error) {
	if err != nil {
		return false, fmt.Errorf("%w: bool evaluation failed: %v", ErrScriptValidation, err)
	}
	if v == nil {
		return false, fmt.Errorf("%w: expected bool result, got nil", ErrScriptValidation)
	}
	b, ok := v.Export().(bool)
	if !ok {
		return false, fmt.Errorf("%w: expected bool result", ErrScriptValidation)
	}
	return b, nil
}

func expectMap(v any, name string) (map[string]any, error) {
	m, ok := v.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("%w: %s must be an object", ErrScriptValidation, name)
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
				return errors.Join(ErrScriptTimeout, runErr, err)
			}
			return errors.Join(ErrScriptTimeout, runErr)
		}
		if err != nil {
			return errors.Join(ErrScriptCancelled, runErr, err)
		}
		return errors.Join(ErrScriptCancelled, runErr)
	}
	return err
}
