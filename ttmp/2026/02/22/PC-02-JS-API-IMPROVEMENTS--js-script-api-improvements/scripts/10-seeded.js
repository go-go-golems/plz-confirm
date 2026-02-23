module.exports = {
  describe: function () { return { name: "pc02-seeded", version: "1.0.0" }; },
  init: function (ctx) {
    return {
      step: "confirm",
      seed: ctx.seed,
      ticket: ctx.randomInt(1000, 9999),
      lane: ctx.randomInt(1, 4)
    };
  },
  view: function (state) {
    return {
      stepId: "seed-review",
      title: "Seeded Randomness",
      sections: [
        {
          widgetType: "display",
          input: {
            format: "markdown",
            content: "### Deterministic context\nSeed: `" + state.seed + "`\nTicket: `" + state.ticket + "`\nLane: `" + state.lane + "`"
          }
        },
        {
          widgetType: "confirm",
          input: {
            title: "Accept deterministic assignment?",
            approveText: "Accept",
            rejectText: "Recheck"
          }
        }
      ]
    };
  },
  update: function (state, event, ctx) {
    return {
      done: true,
      result: {
        approved: !!(event.data && event.data.approved),
        seedFromState: state.seed,
        seedFromUpdate: ctx.seed,
        nextDeterministicRoll: ctx.randomInt(1, 100)
      }
    };
  }
};
