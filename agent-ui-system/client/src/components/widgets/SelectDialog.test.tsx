import React from "react";
import { describe, expect, it, vi } from "vitest";
import { renderToStaticMarkup } from "react-dom/server";

import { SelectDialog } from "@/components/widgets/SelectDialog";

describe("SelectDialog defaults", () => {
  it("initializes selected state from defaults.selectedSingle", () => {
    const onSubmit = vi.fn().mockResolvedValue(undefined);
    const html = renderToStaticMarkup(
      React.createElement(SelectDialog, {
        requestId: "req-select-1",
        onSubmit,
        input: {
          title: "Pick env",
          options: ["staging", "prod"],
          defaults: { selectedSingle: "prod" },
        } as any,
      })
    );

    expect(html).toContain("1 SELECTED");
  });

  it("renders rich object options with descriptions and badges", () => {
    const onSubmit = vi.fn().mockResolvedValue(undefined);
    const html = renderToStaticMarkup(
      React.createElement(SelectDialog, {
        requestId: "req-select-2",
        onSubmit,
        input: {
          title: "Pick server",
          options: [
            {
              value: "prod-us",
              label: "Production US",
              description: "3 instances, healthy",
              badge: "healthy",
            },
            {
              value: "staging-eu",
              label: "Staging EU",
              description: "1 instance, degraded",
              badge: "warning",
              disabled: true,
            },
          ],
          searchable: true,
        } as any,
      })
    );

    expect(html).toContain("Production US");
    expect(html).toContain("3 instances, healthy");
    expect(html).toContain("healthy");
  });
});
