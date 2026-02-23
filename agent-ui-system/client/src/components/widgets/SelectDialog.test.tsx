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
});
