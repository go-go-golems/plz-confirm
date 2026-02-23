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
` + script + `
var __pc_exports = __pc_module.exports;
`
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
