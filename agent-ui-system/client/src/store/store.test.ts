import { describe, expect, it } from "vitest";

import {
  completeRequest,
  createAppStore,
  enqueueRequest,
  patchRequest,
} from "@/store/store";
import {
  RequestStatus,
  UIRequest,
  WidgetType,
} from "@/proto/generated/plz_confirm/v1/request";

const buildScriptRequest = (
  overrides: Partial<UIRequest> = {}
): UIRequest => ({
  id: "req-script-1",
  type: WidgetType.script,
  sessionId: "global",
  status: RequestStatus.pending,
  createdAt: "2026-02-22T00:00:00Z",
  expiresAt: "2026-02-22T00:05:00Z",
  scriptInput: {
    title: "Script Flow",
    script: "module.exports = {}",
  },
  scriptView: {
    widgetType: "confirm",
    input: { title: "Confirm" },
    stepId: "step-confirm",
  },
  scriptDescribe: {
    name: "demo",
    version: "1.0.0",
    capabilities: ["submit"],
  },
  ...overrides,
});

describe("request reducer script flow", () => {
  it("patches script view and then completes request into history", () => {
    const store = createAppStore();
    const base = buildScriptRequest();

    store.dispatch(enqueueRequest(base));
    expect(store.getState().request.active?.id).toBe(base.id);
    expect(store.getState().request.active?.scriptView?.widgetType).toBe(
      "confirm"
    );

    store.dispatch(
      patchRequest({
        id: base.id,
        scriptView: {
          widgetType: "select",
          input: { title: "Select env", options: ["staging", "prod"] },
          stepId: "step-select",
        },
      })
    );
    expect(store.getState().request.active?.scriptView?.widgetType).toBe(
      "select"
    );

    store.dispatch(
      completeRequest(
        buildScriptRequest({
          id: base.id,
          status: RequestStatus.completed,
          scriptOutput: {
            result: { env: "staging" },
            logs: [],
          },
          scriptView: {
            widgetType: "select",
            input: { title: "Select env" },
            stepId: "step-select",
          },
        })
      )
    );

    const requestState = store.getState().request;
    expect(requestState.active).toBeNull();
    expect(requestState.pending).toHaveLength(0);
    expect(requestState.history).toHaveLength(1);
    expect(requestState.history[0]?.id).toBe(base.id);
    expect(requestState.history[0]?.status).toBe(RequestStatus.completed);
  });
});
