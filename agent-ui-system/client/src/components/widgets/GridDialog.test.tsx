import React from "react";
import { describe, expect, it, vi } from "vitest";
import { renderToStaticMarkup } from "react-dom/server";

import { GridDialog } from "@/components/widgets/GridDialog";

describe("GridDialog", () => {
  it("renders title and expected number of grid buttons", () => {
    const onSubmit = vi.fn().mockResolvedValue(undefined);
    const html = renderToStaticMarkup(
      React.createElement(GridDialog, {
        requestId: "req-grid-1",
        onSubmit,
        input: {
          title: "Board",
          rows: 2,
          cols: 3,
          cells: Array.from({ length: 6 }).map(() => ({ value: "" })),
          cellSize: "small",
        },
      })
    );

    expect(html).toContain("Board");
    expect((html.match(/<button/g) || []).length).toBe(6);
  });
});
