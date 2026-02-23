module.exports = {
  describe: function () { return { name: "pc02-rating", version: "1.0.0" }; },
  init: function () { return { step: "rate" }; },
  view: function () {
    return {
      widgetType: "rating",
      stepId: "rate",
      title: "Rating Widget",
      progress: { current: 2, total: 5, label: "Step 2 of 5" },
      input: {
        title: "How smooth was this wizard?",
        style: "emoji",
        scale: 5,
        labels: { low: "rough", high: "excellent" }
      }
    };
  },
  update: function (state, event) {
    return { done: true, result: { rating: event.data ? event.data.value : null } };
  }
};
