module.exports = {
  describe: function () { return { name: "pc02-toast-create", version: "1.0.0" }; },
  init: function () { return { step: "x" }; },
  view: function () {
    return {
      widgetType: "confirm",
      stepId: "x",
      title: "Toast On Create",
      toast: { message: "Fresh request loaded", style: "info", durationMs: 2000 },
      input: {
        title: "Did you see the toast?",
        approveText: "Yes",
        rejectText: "No"
      }
    };
  },
  update: function (state, event) {
    return { done: true, result: { sawToast: !!(event.data && event.data.approved) } };
  }
};
