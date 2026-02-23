module.exports = {
  describe: function () {
    return { name: "pc02-toast-visibility-isolated", version: "1.0.0" };
  },
  init: function () {
    return { step: "intro", seq: 1 };
  },
  view: function (state) {
    if (state.step === "intro") {
      return {
        widgetType: "confirm",
        stepId: "intro",
        title: "Toast Visibility Isolated",
        toast: {
          message: "TOAST_CREATE_VISIBLE_CHECK",
          style: "warning",
          durationMs: 6000,
        },
        input: {
          title: "Did the create-time toast appear?",
          approveText: "Yes",
          rejectText: "No",
        },
      };
    }

    return {
      widgetType: "select",
      stepId: "loop",
      title: "Toast Visibility Isolated",
      toast: {
        message: "TOAST_TRANSITION_VISIBLE_CHECK_" + state.seq,
        style: "success",
        durationMs: 6000,
      },
      input: {
        title: "Transition toast should appear now. Choose next action:",
        options: ["repeat-toast", "finish-test"],
        multi: false,
        searchable: false,
      },
    };
  },
  update: function (state, event) {
    if (state.step === "intro") {
      state.step = "loop";
      state.sawCreateToast = !!(event.data && event.data.approved);
      return state;
    }

    var choice = event && event.data ? event.data.selectedSingle : "";
    if (choice === "repeat-toast") {
      state.seq = (state.seq || 1) + 1;
      return state;
    }

    return {
      done: true,
      result: {
        sawCreateToast: !!state.sawCreateToast,
        loopCount: state.seq || 1,
        finalChoice: choice || "finish-test",
      },
    };
  },
};
