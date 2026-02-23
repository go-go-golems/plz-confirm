import React from "react";
import { describe, expect, it } from "vitest";
import { renderToStaticMarkup } from "react-dom/server";

import { DisplayWidget } from "@/components/widgets/DisplayWidget";

describe("DisplayWidget", () => {
  it("renders markdown/text content", () => {
    const html = renderToStaticMarkup(
      React.createElement(DisplayWidget, {
        input: { content: "## Heading", format: "markdown" },
      })
    );
    expect(html).toContain("## Heading");
  });

  it("renders html content", () => {
    const html = renderToStaticMarkup(
      React.createElement(DisplayWidget, {
        input: { content: "<strong>Hello</strong>", format: "html" },
      })
    );
    expect(html).toContain("<strong>Hello</strong>");
  });
});
