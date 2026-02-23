module.exports = {
  describe: function () { return { name: "pc02-progress-back", version: "1.0.0" }; },
  init: function () { return { step: "confirm", approved: false }; },
  view: function (state) {
    if (state.step === "confirm") {
      return {
        widgetType: "confirm",
        stepId: "confirm",
        title: "Back/Progress Flow",
        progress: { current: 1, total: 3, label: "Step 1 of 3" },
        input: {
          title: "Enable blue-green deploy?",
          message: "Next step lets you pick the target environment.",
          approveText: "Yes",
          rejectText: "No"
        }
      };
    }

    return {
      widgetType: "select",
      stepId: "pick-env",
      title: "Back/Progress Flow",
      progress: { current: 2, total: 3, label: "Step 2 of 3" },
      allowBack: true,
      backLabel: "Back to confirm",
      input: {
        title: "Choose target env",
        options: ["staging", "prod-us", "prod-eu"],
        multi: false,
        searchable: false
      }
    };
  },
  update: function (state, event) {
    if (state.step === "confirm") {
      if (event.type === "submit") {
        state.approved = !!(event.data && event.data.approved);
        state.step = "pick";
      }
      return state;
    }

    if (event.type === "back") {
      state.step = "confirm";
      return state;
    }

    return {
      done: true,
      result: {
        approved: state.approved,
        env: event.data ? event.data.selectedSingle : null
      }
    };
  }
};
