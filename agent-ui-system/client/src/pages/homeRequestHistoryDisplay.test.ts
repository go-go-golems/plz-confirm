import { describe, expect, it } from "vitest";

import {
  RequestStatus,
  UIRequest,
  WidgetType,
} from "@/proto/generated/plz_confirm/v1/request";
import { getRequestHistoryDisplay } from "@/pages/homeRequestHistoryDisplay";

const buildRequest = (overrides: Partial<UIRequest>): UIRequest => ({
  id: "req-history-1",
  type: WidgetType.confirm,
  sessionId: "global",
  status: RequestStatus.pending,
  createdAt: "2026-02-23T00:00:00Z",
  expiresAt: "2026-02-23T00:05:00Z",
  ...overrides,
});

describe("getRequestHistoryDisplay", () => {
  it("uses script title and widget badge for script requests", () => {
    const display = getRequestHistoryDisplay(
      buildRequest({
        type: WidgetType.script,
        scriptInput: { title: "Ship release", script: "module.exports = {}" },
        scriptView: { widgetType: "confirm", input: {}, stepId: "confirm-1" },
      })
    );

    expect(display.typeLabel).toBe("SCRIPT");
    expect(display.title).toBe("Ship release");
    expect(display.scriptWidgetBadge).toBe("confirm");
    expect(display.scriptCompletedMeta).toBeUndefined();
    expect(display.isScript).toBe(true);
  });

  it("uses script describe metadata for completed script requests", () => {
    const display = getRequestHistoryDisplay(
      buildRequest({
        type: WidgetType.script,
        status: RequestStatus.completed,
        scriptInput: { title: "Rate this flow", script: "module.exports = {}" },
        scriptDescribe: { name: "survey", version: "1.2.3", capabilities: [] },
      })
    );

    expect(display.scriptCompletedMeta).toBe("survey v1.2.3");
  });

  it("falls back to standard widget titles for non-script requests", () => {
    const display = getRequestHistoryDisplay(
      buildRequest({
        type: WidgetType.select,
        selectInput: {
          title: "Pick one",
          options: ["a", "b"],
        },
      })
    );

    expect(display.title).toBe("Pick one");
    expect(display.scriptWidgetBadge).toBeUndefined();
    expect(display.isScript).toBe(false);
  });
});
