import React from "react";
import { describe, expect, it, vi } from "vitest";
import { renderToStaticMarkup } from "react-dom/server";

import { TableDialog } from "@/components/widgets/TableDialog";

describe("TableDialog defaults", () => {
  it("initializes selected rows from defaults.selectedSingle", () => {
    const onSubmit = vi.fn().mockResolvedValue(undefined);
    const html = renderToStaticMarkup(
      React.createElement(TableDialog, {
        requestId: "req-table-1",
        onSubmit,
        input: {
          title: "Pick row",
          data: [
            { id: 1, name: "one" },
            { id: 2, name: "two" },
          ],
          columns: ["name"],
          multiSelect: false,
          defaults: { selectedSingle: 2 },
        } as any,
      })
    );

    expect(html).toContain("1 SELECTED / 2 TOTAL");
  });
});
