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
});
