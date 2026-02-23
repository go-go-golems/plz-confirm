module.exports = {
  describe: function () { return { name: "pc02-grid", version: "1.0.0" }; },
  init: function () { return { step: "board" }; },
  view: function () {
    return {
      widgetType: "grid",
      stepId: "board",
      title: "Grid Hotspot Picker",
      description: "Click one active cell to choose deployment zone.",
      input: {
        title: "Choose a hotspot",
        rows: 3,
        cols: 3,
        cellSize: "medium",
        cells: [
          { value: "A1", style: "filled" },
          { value: "A2", style: "highlighted" },
          { value: "A3", style: "filled" },
          { value: "B1", style: "empty" },
          { value: "B2", style: "disabled", disabled: true },
          { value: "B3", style: "empty" },
          { value: "C1", style: "filled" },
          { value: "C2", style: "empty" },
          { value: "C3", style: "highlighted" }
        ]
      }
    };
  },
  update: function (state, event) {
    return { done: true, result: { picked: event.data || null } };
  }
};
