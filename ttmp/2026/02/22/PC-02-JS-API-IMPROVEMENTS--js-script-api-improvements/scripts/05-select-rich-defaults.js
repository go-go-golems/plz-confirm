module.exports = {
  describe: function () { return { name: "pc02-select-rich-defaults", version: "1.0.0" }; },
  init: function () { return { step: "pick" }; },
  view: function () {
    return {
      widgetType: "select",
      stepId: "pick",
      title: "Rich Select + Defaults",
      input: {
        title: "Select deployment lane",
        searchable: true,
        multi: false,
        options: [
          { value: "staging", label: "Staging", description: "Low risk validation lane", badge: "safe", icon: "lab" },
          { value: "prod-us", label: "Production US", description: "Main US traffic", badge: "hot", icon: "us" },
          { value: "prod-eu", label: "Production EU", description: "Main EU traffic", badge: "warm", icon: "eu", disabled: true }
        ],
        defaults: { selectedSingle: "staging" }
      }
    };
  },
  update: function (state, event) {
    return { done: true, result: { selected: event.data ? event.data.selectedSingle : null } };
  }
};
