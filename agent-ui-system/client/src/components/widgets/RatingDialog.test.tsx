import React from "react";
import { describe, expect, it, vi } from "vitest";
import { renderToStaticMarkup } from "react-dom/server";

import { RatingDialog } from "@/components/widgets/RatingDialog";

describe("RatingDialog", () => {
  it("renders title and rating controls", () => {
    const onSubmit = vi.fn().mockResolvedValue(undefined);
    const html = renderToStaticMarkup(
      React.createElement(RatingDialog, {
        requestId: "req-rating-1",
        onSubmit,
        input: {
          title: "Rate docs",
          scale: 5,
          style: "numbers",
        },
      })
    );
    expect(html).toContain("Rate docs");
    expect((html.match(/<button/g) || []).length).toBeGreaterThanOrEqual(5);
  });

  it("supports defaults.value initialization", () => {
    const onSubmit = vi.fn().mockResolvedValue(undefined);
    const html = renderToStaticMarkup(
      React.createElement(RatingDialog, {
        requestId: "req-rating-2",
        onSubmit,
        input: {
          title: "Rate defaults",
          scale: 5,
          style: "slider",
          defaults: { value: 4 },
        } as any,
      })
    );
    expect(html).toContain(">4</div>");
  });
});
