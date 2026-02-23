module.exports = {
  describe: function () { return { name: "pc02-sections", version: "1.0.0" }; },
  init: function () { return { step: "review" }; },
  view: function () {
    return {
      stepId: "review",
      title: "Composite Sections Review",
      sections: [
        {
          widgetType: "display",
          input: {
            format: "markdown",
            content: "## Change Summary\n- 4 services touched\n- 2 migrations pending\n- 0 failed checks"
          }
        },
        {
          widgetType: "confirm",
          input: {
            title: "Proceed with rollout?",
            message: "This confirms you reviewed the display section above.",
            approveText: "Proceed",
            rejectText: "Hold"
          }
        }
      ]
    };
  },
  update: function (state, event) {
    return { done: true, result: { approved: !!(event.data && event.data.approved) } };
  }
};
