module.exports = {
  describe: function () { return { name: "pc02-form-defaults", version: "1.0.0" }; },
  init: function () { return { step: "form" }; },
  view: function () {
    return {
      widgetType: "form",
      stepId: "form",
      title: "Form Defaults",
      input: {
        title: "Connection settings",
        schema: {
          type: "object",
          properties: {
            host: { type: "string", title: "Host" },
            port: { type: "number", title: "Port", minimum: 1, maximum: 65535 },
            useTLS: { type: "boolean", title: "Use TLS" }
          },
          required: ["host", "port"]
        },
        defaults: {
          host: "api.internal.local",
          port: 8443,
          useTLS: true
        }
      }
    };
  },
  update: function (state, event) {
    return { done: true, result: { data: event.data ? event.data.data : null } };
  }
};
