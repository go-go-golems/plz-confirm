module.exports = {
  describe: function () { return { name: "pc02-rating-defaults", version: "1.0.0" }; },
  init: function () { return { step: "rate" }; },
  view: function () {
    return {
      widgetType: "rating",
      stepId: "rate",
      title: "Rating Defaults",
      input: {
        title: "Default starts at 7",
        style: "slider",
        scale: 10,
        defaultValue: 3,
        defaults: { value: 7 },
        labels: { low: "low", high: "high" }
      }
    };
  },
  update: function (state, event) {
    return { done: true, result: { value: event.data ? event.data.value : null } };
  }
};
