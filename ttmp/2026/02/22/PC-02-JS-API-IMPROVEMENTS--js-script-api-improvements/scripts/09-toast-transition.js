module.exports = {
  describe: function () { return { name: "pc02-toast-transition", version: "1.0.0" }; },
  init: function () { return { step: "confirm" }; },
  view: function (state) {
    if (state.step === "confirm") {
      return {
        widgetType: "confirm",
        stepId: "confirm",
        title: "Toast On Transition",
        input: {
          title: "Save this checkpoint?",
          approveText: "Save",
          rejectText: "Skip"
        }
      };
    }
    return {
      widgetType: "select",
      stepId: "choose",
      title: "Toast On Transition",
      toast: { message: "Checkpoint saved", style: "success", durationMs: 1800 },
      input: {
        title: "Now choose rollout mode",
        options: ["canary", "blue-green", "rolling"],
        multi: false,
        searchable: false
      }
    };
  },
  update: function (state, event) {
    if (state.step === "confirm") {
      state.step = "choose";
      state.saved = !!(event.data && event.data.approved);
      return state;
    }
    return {
      done: true,
      result: {
        saved: !!state.saved,
        mode: event.data ? event.data.selectedSingle : null
      }
    };
  }
};
