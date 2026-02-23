import React from "react";
import { describe, expect, it, vi } from "vitest";
import { Provider } from "react-redux";
import { renderToStaticMarkup } from "react-dom/server";

import { WidgetRenderer } from "@/components/WidgetRenderer";
import { createAppStore, enqueueRequest } from "@/store/store";
import {
  RequestStatus,
  UIRequest,
  WidgetType,
} from "@/proto/generated/plz_confirm/v1/request";

vi.mock("@/components/widgets/ConfirmDialog", () => ({
  ConfirmDialog: ({ input }: any) => `MOCK_CONFIRM:${input?.title ?? ""}`,
}));
vi.mock("@/components/widgets/SelectDialog", () => ({
  SelectDialog: ({ input }: any) => `MOCK_SELECT:${input?.title ?? ""}`,
}));
vi.mock("@/components/widgets/TableDialog", () => ({
  TableDialog: ({ input }: any) => `MOCK_TABLE:${input?.title ?? ""}`,
}));
vi.mock("@/components/widgets/FormDialog", () => ({
  FormDialog: ({ input }: any) => `MOCK_FORM:${input?.title ?? ""}`,
}));
vi.mock("@/components/widgets/UploadDialog", () => ({
  UploadDialog: ({ input }: any) => `MOCK_UPLOAD:${input?.title ?? ""}`,
}));
vi.mock("@/components/widgets/ImageDialog", () => ({
  ImageDialog: ({ input }: any) => `MOCK_IMAGE:${input?.title ?? ""}`,
}));
vi.mock("@/components/widgets/GridDialog", () => ({
  GridDialog: ({ input }: any) => `MOCK_GRID:${input?.title ?? ""}`,
}));
vi.mock("@/components/widgets/RatingDialog", () => ({
  RatingDialog: ({ input }: any) => `MOCK_RATING:${input?.title ?? ""}`,
}));
vi.mock("@/components/widgets/DisplayWidget", () => ({
  DisplayWidget: ({ input }: any) =>
    `MOCK_DISPLAY:${input?.content ?? ""}:${input?.format ?? ""}`,
}));

const buildScriptRequest = (
  overrides: Partial<UIRequest> = {}
): UIRequest => ({
  id: "req-render-1",
  type: WidgetType.script,
  sessionId: "global",
  status: RequestStatus.pending,
  createdAt: "2026-02-22T00:00:00Z",
  expiresAt: "2026-02-22T00:05:00Z",
  scriptInput: {
    title: "Script Render",
    script: "module.exports = {}",
  },
  scriptView: {
    widgetType: "confirm",
    input: { title: "Confirm step" },
    stepId: "step-confirm",
  },
  scriptDescribe: {
    name: "demo",
    version: "1.0.0",
    capabilities: ["submit"],
  },
  ...overrides,
});

const renderWithStore = (request: UIRequest) => {
  const store = createAppStore();
  store.dispatch(enqueueRequest(request));
  return renderToStaticMarkup(
    React.createElement(
      Provider,
      { store },
      React.createElement(WidgetRenderer)
    )
  );
};

describe("WidgetRenderer script branch", () => {
  it("renders confirm script views via ConfirmDialog mapping", () => {
    const html = renderWithStore(
      buildScriptRequest({
        id: "req-render-confirm",
        scriptView: {
          widgetType: "confirm",
          input: { title: "Ship to prod?" },
          stepId: "confirm",
        },
      })
    );
    expect(html).toContain("MOCK_CONFIRM:Ship to prod?");
  });

  it("renders explicit error for unsupported script widget types", () => {
    const html = renderWithStore(
      buildScriptRequest({
        id: "req-render-unsupported",
        scriptView: {
          widgetType: "custom-widget",
          input: { title: "Custom step" },
          stepId: "custom",
        },
      })
    );
    expect(html).toContain("ERROR: UNSUPPORTED_SCRIPT_WIDGET [custom-widget]");
  });

  it("renders grid script views via GridDialog mapping", () => {
    const html = renderWithStore(
      buildScriptRequest({
        id: "req-render-grid",
        scriptView: {
          widgetType: "grid",
          input: {
            title: "Your move",
            rows: 3,
            cols: 3,
            cells: Array.from({ length: 9 }).map(() => ({ value: "" })),
          },
          stepId: "grid-step",
        },
      })
    );
    expect(html).toContain("MOCK_GRID:Your move");
  });

  it("renders composite sections with display and one interactive widget", () => {
    const html = renderWithStore(
      buildScriptRequest({
        id: "req-render-sections",
        scriptView: {
          widgetType: "confirm",
          input: { title: "Fallback interactive" },
          stepId: "with-sections",
          sections: [
            {
              widgetType: "display",
              input: { content: "## Context", format: "markdown" },
            },
            {
              widgetType: "confirm",
              input: { title: "Approve composite?" },
            },
          ],
        },
      })
    );
    expect(html).toContain("MOCK_DISPLAY:## Context:markdown");
    expect(html).toContain("MOCK_CONFIRM:Approve composite?");
  });

  it("renders explicit error when composite sections are invalid", () => {
    const html = renderWithStore(
      buildScriptRequest({
        id: "req-render-invalid-sections",
        scriptView: {
          widgetType: "confirm",
          input: { title: "Fallback" },
          stepId: "bad-sections",
          sections: [
            {
              widgetType: "display",
              input: { content: "Only display", format: "text" },
            },
          ],
        },
      })
    );
    expect(html).toContain(
      "ERROR: INVALID_SCRIPT_SECTIONS [exactly one interactive section is required]"
    );
  });

  it("renders script progress indicators when progress is provided", () => {
    const html = renderWithStore(
      buildScriptRequest({
        id: "req-render-progress",
        scriptView: {
          widgetType: "confirm",
          input: { title: "Rate docs" },
          stepId: "q3",
          progress: {
            current: 3,
            total: 8,
            label: "QUESTION 3 OF 8",
          },
        },
      })
    );
    expect(html).toContain("QUESTION 3 OF 8");
    expect(html).toContain("3/8");
  });

  it("renders back button when script view enables back navigation", () => {
    const html = renderWithStore(
      buildScriptRequest({
        id: "req-render-back",
        scriptView: {
          widgetType: "confirm",
          input: { title: "Step 2" },
          stepId: "step-2",
          allowBack: true,
          backLabel: "GO_BACK",
        },
      })
    );
    expect(html).toContain("GO_BACK");
  });

  it("renders rating script views via RatingDialog mapping", () => {
    const html = renderWithStore(
      buildScriptRequest({
        id: "req-render-rating",
        scriptView: {
          widgetType: "rating",
          input: {
            title: "How was this flow?",
            scale: 5,
            style: "stars",
          },
          stepId: "rating-step",
        },
      })
    );
    expect(html).toContain("MOCK_RATING:How was this flow?");
  });
});
