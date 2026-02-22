package main

import (
	"fmt"

	"github.com/dop251/goja"
)

func must(err error) {
	if err != nil {
		panic(err)
	}
}

func main() {
	vm := goja.New()

	moduleObj := vm.NewObject()
	exportsObj := vm.NewObject()
	must(moduleObj.Set("exports", exportsObj))
	must(vm.Set("module", moduleObj))
	must(vm.Set("exports", exportsObj))

	script := `
module.exports = {
  init: function(ctx) {
    return {
      step: "start",
      answers: {},
      propsEcho: ctx.props,
      tags: ["alpha", "beta"]
    };
  },

  view: function(state, ctx) {
    return {
      kind: "page",
      title: "Experiment",
      stateStep: state.step,
      user: ctx.props.user,
      components: [
        { kind: "markdown", text: "hello" },
        { kind: "select", id: "env", options: ["dev", "prod"] }
      ]
    };
  },

  update: function(state, event, ctx) {
    if (event.type === "submit") {
      state.step = "done";
      state.answers = event.data;
      return { done: true, result: { approved: true, answers: state.answers } };
    }
    return state;
  }
};
`

	_, err := vm.RunString(script)
	must(err)

	exports := moduleObj.Get("exports").ToObject(vm)
	initFn, ok := goja.AssertFunction(exports.Get("init"))
	if !ok {
		panic("init is not callable")
	}
	viewFn, ok := goja.AssertFunction(exports.Get("view"))
	if !ok {
		panic("view is not callable")
	}
	updateFn, ok := goja.AssertFunction(exports.Get("update"))
	if !ok {
		panic("update is not callable")
	}

	ctx := map[string]any{
		"props": map[string]any{
			"user": "intern-1",
			"ticket": "PC-01",
		},
	}

	stateVal, err := initFn(goja.Undefined(), vm.ToValue(ctx))
	must(err)
	state := stateVal.Export()
	fmt.Printf("init type=%T\n", state)
	fmt.Printf("init value=%#v\n", state)

	viewVal, err := viewFn(goja.Undefined(), vm.ToValue(state), vm.ToValue(ctx))
	must(err)
	view := viewVal.Export()
	fmt.Printf("view type=%T\n", view)
	fmt.Printf("view value=%#v\n", view)

	event := map[string]any{
		"type": "submit",
		"data": map[string]any{
			"env": "prod",
			"ticket": "CHG-123",
		},
	}
	updateVal, err := updateFn(goja.Undefined(), vm.ToValue(state), vm.ToValue(event), vm.ToValue(ctx))
	must(err)
	update := updateVal.Export()
	fmt.Printf("update type=%T\n", update)
	fmt.Printf("update value=%#v\n", update)
}
