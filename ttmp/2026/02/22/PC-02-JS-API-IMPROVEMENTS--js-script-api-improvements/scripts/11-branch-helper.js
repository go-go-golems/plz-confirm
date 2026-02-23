module.exports = {
  describe: function () { return { name: "pc02-branch-helper", version: "1.0.0" }; },
  init: function () { return { step: "confirm" }; },
  view: function (state) {
    if (state.step === "details") {
      return {
        widgetType: "select",
        stepId: "details",
        title: "Branch Helper",
        input: {
          title: "Approved path: pick release tier",
          options: ["tier-1", "tier-2", "tier-3"],
          multi: false,
          searchable: false
        }
      };
    }
    if (state.step === "reason") {
      return {
        widgetType: "form",
        stepId: "reason",
        title: "Branch Helper",
        input: {
          title: "Rejected path: explain why",
          schema: {
            type: "object",
            properties: {
              reason: { type: "string", minLength: 3 }
            },
            required: ["reason"]
          }
        }
      };
    }
    return {
      widgetType: "confirm",
      stepId: "confirm",
      title: "Branch Helper",
      input: {
        title: "Approve deployment request?",
        approveText: "Approve",
        rejectText: "Reject"
      }
    };
  },
  update: function (state, event, ctx) {
    if (state.step === "confirm") {
      return ctx.branch(state, event, {
        approved: "details",
        rejected: "reason",
        default: "reason"
      });
    }
    return { done: true, result: { finalStep: state.step, data: event.data || null } };
  }
};
