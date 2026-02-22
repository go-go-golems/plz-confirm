Below is one concrete way to think about “JS sandbox + React-like UI DSL” inside **plz-confirm**, plus a bunch of **complex, multi-step** UI examples, and then a **Goja-based implementation map**.

I’m going to assume:

* The *agent/caller* sends a “script request” to plz-confirm (new widget type).
* The script runs in a **Goja** VM on the server side (Go), **not** in the browser.
* The script **builds a UI tree** (pure JSON-like data) using a React-ish DSL (`ui.h(...)` / `ui.*(...)` helpers).
* The browser renders that UI tree using your existing React frontend, and posts events back (“submit step”, “click secondary action”) until the flow completes.

Goja note: Goja is ES5.1-focused with “most of ES6” in progress, and (per maintainer discussions) does **not** plan to support generators/async-await in the core engine. So the flow API should be ES5-friendly and event/state driven, not `async function*`-driven. ([GitHub][1])

---

## Proposed “script widget” contract

### 1) What the agent sends (new widget type)

A new request type like `WidgetType.script` (or `WidgetType.flow`) with input:

```proto
message ScriptInput {
  string title = 1;
  string script = 2; // JS source
  google.protobuf.Struct props = 3; // agent-provided context
  repeated FileMount mounts = 4;    // optional allowed file roots/attachments
  optional ScriptLimits limits = 5;
}
```

And output:

```proto
message ScriptOutput {
  google.protobuf.Struct result = 1;
  repeated ScriptLog logs = 2; // console.log capture (optional)
}
```

### 2) What the script exports

ES5-compatible module interface:

```js
// wizard.js
module.exports = {
  init: function (ctx) { return { step: "start", answers: {} }; },

  view: function (state, ctx) {
    // return a UI tree (JSON-ish)
  },

  update: function (state, event, ctx) {
    // return next state
    // or: return { done: true, result: {...} }
  }
};
```

### 3) The runtime “ctx” you inject

```js
ctx = {
  props: {...},           // agent-provided
  ui: { ... },            // UI DSL builders (pure JS helpers)
  fs: { readText, glob, stat, ... },  // capability-limited, sandboxed
  console: { log, warn, error },      // captured
  now: () => "...",       // deterministic or server time
};
```

### 4) Events from the browser

```js
event = {
  type: "submit" | "action",
  stepId: "reviewPlan",
  data: {...},      // widget output payload
  actionId: "...",  // for secondary buttons
};
```

The server runs:

1. `init(ctx)` → `state0`
2. `view(state0, ctx)` → `ui0` (send to browser)
3. Browser submits `event1`
4. `update(state0, event1, ctx)` → `state1` or `{done,result}`
5. If not done: `view(state1, ctx)` → `ui1` … repeat

---

## UI DSL shape (React-like, but serializable)

Two styles you can support:

### A) “Hyperscript” (`h(type, props, children)`)

No JSX required:

```js
var h = ctx.ui.h;

return h("Wizard", { title: "Deploy" }, [
  h("Step", { id: "env", title: "Choose env" }, [
    h("Select", { id: "envSelect", options: ["dev","staging","prod"] })
  ])
]);
```

### B) “Builder helpers” (less React-ish, more ergonomic)

```js
return ctx.ui.wizard({
  title: "Deploy",
  step: "env",
  steps: [
    ctx.ui.step("env", { title: "Choose env" }, [
      ctx.ui.select({ id: "env", options: ["dev","staging","prod"] })
    ])
  ]
});
```

Either way: the output is plain JSON the frontend can render.

---

# A bunch of complex, multi-step UI examples

Each example below is “what an agent would send” as a script (plus some props). I’ll keep the DSL consistent:

* `init/view/update`
* `ui.*` is your DSL
* `fs.*` is sandboxed

I’ll also show the *final* result shape (what the CLI gets back).

---

## Example 1: “Review and approve a deployment plan” (diff + gating)

**Use case:** Agent generated a plan file; user must review diff and pick an environment; prod requires typed acknowledgement.

```js
module.exports = {
  init: function (ctx) {
    var planText = ctx.fs.readText(ctx.props.planPath);
    var plan = JSON.parse(planText);
    return { step: "chooseEnv", plan: plan, answers: {} };
  },

  view: function (s, ctx) {
    var ui = ctx.ui;

    if (s.step === "chooseEnv") {
      return ui.page({
        title: "Deploy Plan",
        body: [
          ui.markdown("Pick the target environment."),
          ui.select({ id: "env", options: ["dev", "staging", "prod"], searchable: false })
        ],
        primary: ui.submit("Next")
      });
    }

    if (s.step === "reviewDiff") {
      return ui.page({
        title: "Review Changes",
        body: [
          ui.callout("info", "This is the exact diff that will be applied."),
          ui.diff({
            leftLabel: "Current",
            rightLabel: "Proposed",
            leftText: JSON.stringify(s.plan.current, null, 2),
            rightText: JSON.stringify(s.plan.proposed, null, 2),
            language: "json"
          }),
          ui.confirm({
            id: "approve",
            message: "Approve this plan?",
            approveText: "Approve",
            rejectText: "Cancel"
          })
        ],
        primary: ui.submit("Continue")
      });
    }

    // prod gate
    if (s.step === "prodAck") {
      return ui.page({
        title: "Production Safety Gate",
        body: [
          ui.callout("warning", "Production deploy requires explicit acknowledgement."),
          ui.form({
            id: "ack",
            schema: {
              type: "object",
              properties: {
                phrase: { type: "string", title: "Type DEPLOY to continue" },
                ticket: { type: "string", title: "Change Ticket (optional)" }
              },
              required: ["phrase"]
            }
          })
        ],
        primary: ui.submit("Finalize")
      });
    }

    return ui.page({ title: "Unknown state", body: [ui.markdown("Bad step: " + s.step)] });
  },

  update: function (s, event, ctx) {
    if (s.step === "chooseEnv" && event.type === "submit") {
      s.answers.env = event.data.env;
      s.step = "reviewDiff";
      return s;
    }

    if (s.step === "reviewDiff" && event.type === "submit") {
      if (!event.data.approve.approved) {
        return { done: true, result: { approved: false, reason: "user_cancelled" } };
      }
      if (s.answers.env === "prod") {
        s.step = "prodAck";
        return s;
      }
      return { done: true, result: { approved: true, env: s.answers.env, ticket: null } };
    }

    if (s.step === "prodAck" && event.type === "submit") {
      var phrase = (event.data.ack.phrase || "").trim();
      if (phrase !== "DEPLOY") {
        // stay on same step with error
        s.answers.error = "Phrase mismatch";
        return s;
      }
      return {
        done: true,
        result: { approved: true, env: "prod", ticket: event.data.ack.ticket || null }
      };
    }

    return s;
  }
};
```

**Final output:**

```json
{ "approved": true, "env": "prod", "ticket": "CHG-1234" }
```

---

## Example 2: “Select a subset of files from a repo, then confirm a patch set”

**Use case:** Agent wants the user to choose which files to touch.

```js
module.exports = {
  init: function (ctx) {
    var files = ctx.fs.glob(ctx.props.repoRoot, "**/*.{go,ts,tsx,md}");
    return { step: "pickFiles", files: files, picked: [] };
  },

  view: function (s, ctx) {
    var ui = ctx.ui;

    if (s.step === "pickFiles") {
      return ui.page({
        title: "Choose Files",
        body: [
          ui.markdown("Select the files the agent is allowed to modify."),
          ui.table({
            id: "files",
            columns: ["path", "size"],
            rows: s.files.map(function (f) { return { path: f.path, size: f.size }; }),
            multiSelect: true,
            searchable: true
          })
        ],
        primary: ui.submit("Next")
      });
    }

    if (s.step === "confirm") {
      return ui.page({
        title: "Confirm Scope",
        body: [
          ui.callout("info", "Agent will only modify the selected files."),
          ui.code({ language: "text", value: s.picked.join("\n") }),
          ui.confirm({ id: "ok", message: "Confirm this scope?" })
        ],
        primary: ui.submit("Finish")
      });
    }

    return ui.page({ title: "Bad step", body: [ui.markdown(s.step)] });
  },

  update: function (s, event, ctx) {
    if (s.step === "pickFiles" && event.type === "submit") {
      s.picked = (event.data.files.selected || []).map(function (row) { return row.path; });
      s.step = "confirm";
      return s;
    }
    if (s.step === "confirm" && event.type === "submit") {
      if (!event.data.ok.approved) return { done: true, result: { cancelled: true } };
      return { done: true, result: { allowedFiles: s.picked } };
    }
    return s;
  }
};
```

**Final output:**

```json
{ "allowedFiles": ["internal/server/server.go", "README.md"] }
```

---

## Example 3: “CSV import mapping wizard” (preview → mapping → validation → approve)

**Use case:** User needs to map CSV columns to required fields.

```js
module.exports = {
  init: function (ctx) {
    // Either a path from props or a prior upload step in a larger flow
    var csv = ctx.fs.readText(ctx.props.csvPath);
    var lines = csv.split(/\r?\n/).filter(Boolean);
    var header = lines[0].split(",");
    var preview = lines.slice(1, 6).map(function (l) { return l.split(","); });
    return { step: "map", header: header, preview: preview };
  },

  view: function (s, ctx) {
    var ui = ctx.ui;

    if (s.step === "map") {
      var required = ["email", "first_name", "last_name"];
      var schema = {
        type: "object",
        properties: {
          email: { type: "string", enum: s.header, title: "Email column" },
          first_name: { type: "string", enum: s.header, title: "First name column" },
          last_name: { type: "string", enum: s.header, title: "Last name column" }
        },
        required: required
      };

      return ui.page({
        title: "Map CSV Columns",
        body: [
          ui.markdown("Preview (first 5 rows):"),
          ui.table({
            id: "preview",
            columns: s.header,
            rows: s.preview.map(function (r) {
              var obj = {};
              for (var i=0;i<s.header.length;i++) obj[s.header[i]] = r[i] || "";
              return obj;
            }),
            multiSelect: false,
            searchable: false
          }),
          ui.form({ id: "mapping", schema: schema })
        ],
        primary: ui.submit("Validate")
      });
    }

    if (s.step === "validate") {
      return ui.page({
        title: "Validation",
        body: [
          ui.callout("success", "Mapping looks valid."),
          ui.confirm({ id: "approve", message: "Proceed with this import mapping?" })
        ],
        primary: ui.submit("Finish")
      });
    }

    return ui.page({ title: "Bad step", body: [ui.markdown(s.step)] });
  },

  update: function (s, event, ctx) {
    if (s.step === "map" && event.type === "submit") {
      s.mapping = event.data.mapping;
      // basic validation: ensure distinct columns
      var vals = [s.mapping.email, s.mapping.first_name, s.mapping.last_name];
      var seen = {};
      for (var i=0;i<vals.length;i++) {
        if (seen[vals[i]]) {
          s.error = "Two fields map to the same column: " + vals[i];
          return s; // frontend could display s.error if you include it in view()
        }
        seen[vals[i]] = true;
      }
      s.step = "validate";
      return s;
    }

    if (s.step === "validate" && event.type === "submit") {
      if (!event.data.approve.approved) return { done: true, result: { cancelled: true } };
      return { done: true, result: { mapping: s.mapping } };
    }

    return s;
  }
};
```

**Final output:**

```json
{ "mapping": { "email": "EmailAddress", "first_name": "FirstName", "last_name": "LastName" } }
```

---

## Example 4: “Command plan review UI” (edit/disable/reorder commands)

**Use case:** Agent plans to run multiple shell commands; user reviews, disables some, edits others.

```js
module.exports = {
  init: function (ctx) {
    return { step: "review", commands: ctx.props.commands };
  },

  view: function (s, ctx) {
    var ui = ctx.ui;
    return ui.page({
      title: "Review Command Plan",
      body: [
        ui.markdown("Edit or disable commands before the agent runs them."),
        ui.commandListEditor({
          id: "cmds",
          items: s.commands
          // items: [{ id, label, command, enabled, cwd, env }]
        })
      ],
      primary: ui.submit("Approve")
    });
  },

  update: function (s, event, ctx) {
    if (event.type === "submit") {
      var edited = event.data.cmds.items;
      // enforce: cannot enable destructive commands unless acknowledged
      return { done: true, result: { approvedCommands: edited } };
    }
    return s;
  }
};
```

**Final output:**

```json
{ "approvedCommands": [{ "id":"1","command":"make test","enabled":true }, ...] }
```

---

## Example 5: “Log triage” (cluster errors → pick categories → add notes)

**Use case:** Script reads a log file, groups similar errors, user tags priority/owner.

```js
module.exports = {
  init: function (ctx) {
    var text = ctx.fs.readText(ctx.props.logPath);
    var lines = text.split(/\r?\n/);
    // toy grouping
    var groups = {};
    for (var i=0;i<lines.length;i++) {
      var l = lines[i];
      if (!l) continue;
      var key = l.replace(/\d+/g, "#"); // collapse numbers
      groups[key] = (groups[key] || 0) + 1;
    }
    var rows = Object.keys(groups).map(function (k) { return { signature: k, count: groups[k] }; });
    rows.sort(function(a,b){ return b.count-a.count; });
    return { step: "pick", rows: rows.slice(0, 50) };
  },

  view: function (s, ctx) {
    var ui = ctx.ui;

    if (s.step === "pick") {
      return ui.page({
        title: "Log Triage",
        body: [
          ui.markdown("Select the error signatures to escalate."),
          ui.table({
            id: "errors",
            columns: ["count", "signature"],
            rows: s.rows,
            multiSelect: true,
            searchable: true
          })
        ],
        primary: ui.submit("Next")
      });
    }

    if (s.step === "tag") {
      return ui.page({
        title: "Add Triage Tags",
        body: [
          ui.form({
            id: "meta",
            schema: {
              type: "object",
              properties: {
                priority: { type: "string", enum: ["p0","p1","p2","p3"], title: "Priority" },
                owner: { type: "string", title: "Owner/Team" },
                note: { type: "string", title: "Notes" }
              },
              required: ["priority", "owner"]
            }
          })
        ],
        primary: ui.submit("Finish")
      });
    }

    return ui.page({ title: "Bad step", body: [ui.markdown(s.step)] });
  },

  update: function (s, event, ctx) {
    if (s.step === "pick" && event.type === "submit") {
      s.selected = event.data.errors.selected || [];
      s.step = "tag";
      return s;
    }
    if (s.step === "tag" && event.type === "submit") {
      return {
        done: true,
        result: {
          selectedSignatures: s.selected.map(function (r) { return r.signature; }),
          meta: event.data.meta
        }
      };
    }
    return s;
  }
};
```

---

## Example 6: “Image labeling with per-image multi-tags” (iterative step loop)

**Use case:** User tags each image; script advances index until all labeled.

```js
module.exports = {
  init: function (ctx) {
    return { idx: 0, labels: {}, images: ctx.props.images, step: "labelOne" };
  },

  view: function (s, ctx) {
    var ui = ctx.ui;
    var img = s.images[s.idx];

    return ui.page({
      title: "Label Images (" + (s.idx + 1) + "/" + s.images.length + ")",
      body: [
        ui.image({ src: img.src, alt: img.alt || "", caption: img.caption || "" }),
        ui.select({
          id: "tags",
          options: ctx.props.tagOptions,
          multi: true,
          searchable: true
        })
      ],
      primary: ui.submit(s.idx === s.images.length - 1 ? "Finish" : "Next"),
      secondary: [ui.action("skip", "Skip")]
    });
  },

  update: function (s, event, ctx) {
    if (event.type === "action" && event.actionId === "skip") {
      s.idx++;
      if (s.idx >= s.images.length) return { done: true, result: { labels: s.labels } };
      return s;
    }

    if (event.type === "submit") {
      var selected = event.data.tags.selected_multi
        ? event.data.tags.selected_multi.values
        : [event.data.tags.selected_single];

      s.labels[s.images[s.idx].id] = selected;
      s.idx++;
      if (s.idx >= s.images.length) return { done: true, result: { labels: s.labels } };
      return s;
    }

    return s;
  }
};
```

---

## Example 7: “Policy exception request” (collect → summarize → approve)

**Use case:** Make the user fill a justification, then show a generated summary they approve.

```js
module.exports = {
  init: function () { return { step: "collect" }; },

  view: function (s, ctx) {
    var ui = ctx.ui;

    if (s.step === "collect") {
      return ui.page({
        title: "Request Policy Exception",
        body: [
          ui.form({
            id: "req",
            schema: {
              type: "object",
              properties: {
                policy: { type: "string", title: "Which policy?" },
                reason: { type: "string", title: "Justification" },
                durationDays: { type: "number", title: "Duration (days)", minimum: 1, maximum: 90 },
                approver: { type: "string", title: "Approver" }
              },
              required: ["policy", "reason", "durationDays", "approver"]
            }
          })
        ],
        primary: ui.submit("Next")
      });
    }

    if (s.step === "confirm") {
      return ui.page({
        title: "Confirm Summary",
        body: [
          ui.markdown("**Policy:** " + s.req.policy),
          ui.markdown("**Duration:** " + s.req.durationDays + " days"),
          ui.markdown("**Approver:** " + s.req.approver),
          ui.markdown("**Reason:**\n\n" + s.req.reason),
          ui.confirm({ id: "ok", message: "Submit this exception request?" })
        ],
        primary: ui.submit("Submit")
      });
    }

    return ui.page({ title: "Bad step", body: [ui.markdown(s.step)] });
  },

  update: function (s, event) {
    if (s.step === "collect" && event.type === "submit") {
      s.req = event.data.req;
      s.step = "confirm";
      return s;
    }
    if (s.step === "confirm" && event.type === "submit") {
      if (!event.data.ok.approved) return { done: true, result: { cancelled: true } };
      return { done: true, result: { request: s.req } };
    }
    return s;
  }
};
```

---

## Example 8: “Interactive regex transform preview” (edit regex → preview → approve)

**Use case:** User tunes a regex; UI shows before/after preview on sample lines.

```js
module.exports = {
  init: function (ctx) {
    var sample = ctx.fs.readText(ctx.props.samplePath).split(/\r?\n/).slice(0, 20);
    return { step: "edit", sample: sample, regex: ctx.props.defaultRegex, repl: ctx.props.defaultReplacement };
  },

  view: function (s, ctx) {
    var ui = ctx.ui;

    if (s.step === "edit") {
      // preview computed in frontend or server; here: server-side toy preview
      var re = new RegExp(s.regex, "g");
      var out = s.sample.map(function (l) { return l.replace(re, s.repl); });

      return ui.page({
        title: "Transform Preview",
        body: [
          ui.form({
            id: "cfg",
            schema: {
              type: "object",
              properties: {
                regex: { type: "string", title: "Regex" },
                replacement: { type: "string", title: "Replacement" }
              },
              required: ["regex", "replacement"]
            },
            data: { regex: s.regex, replacement: s.repl }
          }),
          ui.diff({
            leftLabel: "Before",
            rightLabel: "After",
            leftText: s.sample.join("\n"),
            rightText: out.join("\n"),
            language: "text"
          }),
          ui.confirm({ id: "ok", message: "Use this transformation?" })
        ],
        primary: ui.submit("Finish")
      });
    }

    return ui.page({ title: "Bad step", body: [ui.markdown(s.step)] });
  },

  update: function (s, event) {
    if (event.type === "submit") {
      s.regex = event.data.cfg.regex;
      s.repl = event.data.cfg.replacement;
      if (!event.data.ok.approved) return s; // user can tweak and resubmit
      return { done: true, result: { regex: s.regex, replacement: s.repl } };
    }
    return s;
  }
};
```

---

## Example 9: “Merge conflict resolver” (choose strategy per file)

**Use case:** Agent detected conflicts. User chooses “ours/theirs/manual” per file.

```js
module.exports = {
  init: function (ctx) {
    return { step: "pick", conflicts: ctx.props.conflicts };
  },

  view: function (s, ctx) {
    var ui = ctx.ui;
    return ui.page({
      title: "Resolve Conflicts",
      body: [
        ui.markdown("Choose a resolution strategy per file."),
        ui.table({
          id: "choices",
          columns: ["file", "status"],
          rows: s.conflicts.map(function (c) { return { file: c.path, status: "conflict" }; }),
          multiSelect: true
        }),
        ui.form({
          id: "strategy",
          schema: {
            type: "object",
            properties: {
              defaultStrategy: { type: "string", enum: ["ours","theirs","manual"], title: "Default strategy" }
            },
            required: ["defaultStrategy"]
          }
        }),
        ui.confirm({ id: "ok", message: "Apply this strategy?" })
      ],
      primary: ui.submit("Finish")
    });
  },

  update: function (s, event) {
    if (event.type === "submit" && event.data.ok.approved) {
      var selected = event.data.choices.selected || [];
      return {
        done: true,
        result: {
          files: selected.map(function (r) { return r.file; }),
          defaultStrategy: event.data.strategy.defaultStrategy
        }
      };
    }
    return s;
  }
};
```

---

## Example 10: “Release notes approval” (edit markdown + confirm)

**Use case:** Agent wrote release notes; user edits and approves.

```js
module.exports = {
  init: function (ctx) {
    var md = ctx.fs.readText(ctx.props.releaseNotesPath);
    return { md: md };
  },

  view: function (s, ctx) {
    var ui = ctx.ui;
    return ui.page({
      title: "Approve Release Notes",
      body: [
        ui.markdown("Edit below, then approve."),
        ui.markdownEditor({ id: "notes", value: s.md }),
        ui.confirm({ id: "ok", message: "Approve these release notes?" })
      ],
      primary: ui.submit("Finish")
    });
  },

  update: function (s, event) {
    if (event.type === "submit") {
      var md = event.data.notes.value;
      if (!event.data.ok.approved) return s;
      return { done: true, result: { releaseNotes: md } };
    }
    return s;
  }
};
```

---

# Mapping out how to implement this with Goja

## 0) Constraints you need to design around

* **Goja runtime is not goroutine-safe**: one runtime instance must be used by a single goroutine at a time. ([GitHub][1])
* Goja is ES5.1-first; “most ES6” exists but is not “modern JS in full.” ([GitHub][1])
* The maintainer has explicitly said they don’t plan to support **generators / async-await** in core (so avoid designs that require them). ([GitHub][2])

So: implement a **pure state machine** (`init/view/update`) instead of generators.

---

## 1) Add a new widget type in protobuf

### A) Extend enums and oneofs

In `proto/plz_confirm/v1/request.proto`:

* Add `script = 7;` (or `flow = 7;`) to `WidgetType`.

In `proto/plz_confirm/v1/widgets.proto`:

* Add `ScriptInput`, `ScriptOutput`.

In `UIRequest`:

* Add `ScriptInput script_input = ...` in the `input` oneof.
* Add `ScriptOutput script_output = ...` in the `output` oneof.

### B) Decide where “current rendered UI” lives

You have two workable patterns:

**Pattern 1 (recommended):** Store *current view* in the request so all clients see it.

* Add `google.protobuf.Struct script_view = N;`
* Add `google.protobuf.Struct script_state = N;` (serialized state machine state)

**Pattern 2:** Keep state server-side only and add a `/render` endpoint returning the current view.

* Less syncing, but simpler proto.

For multi-client parity and debugging, Pattern 1 is nicer.

---

## 2) Server: add a script engine package

Create something like:

```
internal/scriptengine/
  engine.go
  sandbox_fs.go
  ui_prelude.js
```

### A) Compile + execute flow steps

Pseudo-API:

```go
type Engine struct {
  Limits Limits
  Files  *SandboxFS
}

func (e *Engine) InitAndView(script string, props map[string]any) (state any, view any, logs []Log, err error)
func (e *Engine) UpdateAndView(script string, props map[string]any, state any, event any) (newState any, view any, done *DoneResult, logs []Log, err error)
```

Implementation approach:

1. `r := goja.New()`
2. Inject globals: `module`, `exports`, `console`, `fs`, `ui`, `__props`
3. Run a **prelude JS** that defines `ui.*` helper builders (pure JS returning objects)
4. Run the user script
5. Grab `module.exports.init/view/update` as callables
6. Call them, export results back to Go with `Export()`

### B) Time limits (prevent runaway scripts)

The common Goja pattern is:

* Run evaluation in a goroutine.
* If deadline hits, call `runtime.Interrupt()` from another goroutine.

This is exactly what other projects do (k6, etc.). ([GitHub][3])

You’ll also want to:

* Set a max wall time per `init/view/update` (e.g. 50–200ms).
* Cap script size and output size.
* Cap file read bytes.

(Goja can be interrupted, but hard “memory sandboxing” is not something Goja provides natively; if you truly need hard memory isolation, run scripts in a separate process.)

---

## 3) File access: capability-based and traversal-resistant

You said “reading files and stuff.” That’s the sharp edge.

### A) Don’t expose `os.Open` style access directly

Instead expose a *capability object* representing one or more allowed roots:

* mount roots from the request: `[{name:"repo", path:"/home/me/repo", mode:"ro"}]`
* or from server config: `AllowedRoots = [...]`

### B) Use Go 1.24 `os.Root` for safe filesystem scoping

Go 1.24 introduced `os.Root` / `os.OpenRoot` specifically to prevent path traversal issues when you want to restrict access to a directory subtree. ([Go][4])

So implement:

* `root, _ := os.OpenRoot("/allowed/root")`
* `root.Open("relative/path")` (no `../` escapes)
* enforce per-call byte limits

Expose to JS:

```js
ctx.fs.readText("repo:internal/server/server.go")  // mount name + relpath
ctx.fs.glob("repo:**/*.go")
```

No absolute paths in user scripts; no `../../`.

---

## 4) Request lifecycle changes (multi-step within one request)

### A) New endpoint for intermediate events

Add:

* `POST /api/requests/{id}/event`

Body:

```json
{ "type": "submit", "stepId": "chooseEnv", "data": {...}, "actionId": null }
```

Server flow:

1. Load request
2. Load script, props, current `script_state`
3. Call `update`
4. If `done`: set request output and mark completed (existing store.Complete)
5. Else: set updated `script_state` and `script_view` in request (still pending)
6. Return updated request or just `{view, state}`

### B) Broadcast updates to all connected clients (optional but nice)

Add a websocket event type like:

* `request_updated` with new request payload

So if the user has plz-confirm UI open in two tabs, they stay in sync.

---

## 5) Frontend: add a ScriptDialog / FlowDialog + a generic renderer

### A) Extend WidgetRenderer switch

Add:

* `case WidgetType.script: return <ScriptDialog ... />`

### B) ScriptDialog responsibilities

* Render `request.scriptView` using a `renderNode(node)` function.
* When user submits:

  * Post `/api/requests/{id}/event`
  * Update local UI with the server response (or wait for `request_updated`)

### C) Generic renderer: map DSL nodes → React components

A pragmatic mapping:

* `page`, `markdown`, `callout`, `tabs`, `diff`, `code`
* interactive nodes reuse your existing widgets internally:

  * `confirm`, `select`, `form`, `table`, `image`, `upload`

So you’re not re-inventing input validation—your current widget components can be embedded inside the script-driven page.

---

## 6) CLI: add `plz-confirm script` command

Similar to current commands:

* flags:

  * `--title`
  * `--script @wizard.js` (or `--script -` stdin)
  * `--props @props.json` (optional)
  * `--mount name:path:ro` repeatable (optional)
  * normal `--timeout`, `--wait-timeout`, `--session-id`, `--base-url`

CLI behavior:

* Create request → wait until status completed → print `script_output.result` in yaml/json/table.

---

## 7) Sandboxing posture (practical defaults)

If you treat “agent scripts” as potentially untrusted, make the default capability set extremely small:

* ✅ deterministic helpers: `ui`, `console`
* ✅ read-only file mounts (explicit)
* ❌ network access (don’t provide `fetch`)
* ❌ process/env access (`os.Environ`, etc.)
* ❌ arbitrary `require()` from disk

Then add server config to selectively enable more.

---

## 8) Testing strategy

* Unit test `init/view/update` with golden JSON views.
* Fuzz test event payloads → ensure no panics, bounded time.
* Security tests:

  * attempts to escape mounts (`../`)
  * huge file reads
  * infinite loops (ensure interrupt works)
* Frontend snapshot tests for node renderer.

---

# Summary of the “best” architecture choice

If you want **multi-step** *inside a single request*, and you want JS to compute steps dynamically, the cleanest design with Goja is:

* **State machine script** (`init/view/update`)
* **Server-run Goja** per event
* **UI tree is pure JSON**
* **Frontend is a generic renderer**
* **File access is capability-based using `os.Root`** for safe scoping ([Go][4])
* **Execution timeouts via `Runtime.Interrupt()`** pattern ([GitHub][3])

If you tell me which direction you prefer:

1. “script runs once and returns a static wizard spec” (simpler, fewer endpoints), **or**
2. “script is a real state machine that runs on every step” (most powerful)

…I can tighten the proto layout + event schema so it fits your repo’s existing patterns cleanly.

[1]: https://github.com/dop251/goja "GitHub - dop251/goja: ECMAScript/JavaScript engine in pure Go"
[2]: https://github.com/dop251/goja/issues/167?utm_source=chatgpt.com "roadmap and some coordination questions · Issue #167"
[3]: https://github.com/dop251/goja/issues/120?utm_source=chatgpt.com "Some way to reset an interrupted VM/runtime · Issue #120"
[4]: https://go.dev/blog/osroot?utm_source=chatgpt.com "Traversal-resistant file APIs"

