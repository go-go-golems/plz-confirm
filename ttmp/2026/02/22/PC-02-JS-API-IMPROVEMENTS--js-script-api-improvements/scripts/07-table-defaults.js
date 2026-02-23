module.exports = {
  describe: function () { return { name: "pc02-table-defaults", version: "1.0.0" }; },
  init: function () { return { step: "table" }; },
  view: function () {
    return {
      widgetType: "table",
      stepId: "table",
      title: "Table Defaults",
      input: {
        title: "Pick nodes",
        columns: ["id", "name", "status", "region"],
        multiSelect: true,
        searchable: true,
        data: [
          { id: "srv-1", name: "alpha", status: "ready", region: "us-east" },
          { id: "srv-2", name: "beta", status: "ready", region: "us-west" },
          { id: "srv-3", name: "gamma", status: "drain", region: "eu-west" }
        ],
        defaults: {
          selectedMulti: { values: ["srv-1", "srv-3"] }
        }
      }
    };
  },
  update: function (state, event) {
    return { done: true, result: { selected: event.data ? event.data.selectedMulti : null } };
  }
};
