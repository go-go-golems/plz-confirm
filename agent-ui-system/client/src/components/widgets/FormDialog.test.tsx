import React from "react";
import { describe, expect, it, vi } from "vitest";
import { renderToStaticMarkup } from "react-dom/server";

import { FormDialog } from "@/components/widgets/FormDialog";

describe("FormDialog defaults", () => {
  it("prefills form fields from input.defaults", () => {
    const onSubmit = vi.fn().mockResolvedValue(undefined);
    const html = renderToStaticMarkup(
      React.createElement(FormDialog, {
        requestId: "req-form-1",
        onSubmit,
        input: {
          title: "Profile",
          schema: {
            properties: { name: { type: "string" } },
            required: ["name"],
          },
          defaults: { name: "alice" },
        } as any,
      })
    );

    expect(html).toContain('value="alice"');
  });
});
