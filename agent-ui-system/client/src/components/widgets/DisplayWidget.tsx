import React from "react";

import { DisplayInput } from "@/proto/generated/plz_confirm/v1/widgets";

interface Props {
  input: DisplayInput;
}

export const DisplayWidget: React.FC<Props> = ({ input }) => {
  const content = String(input.content || "");
  const format = String(input.format || "markdown").toLowerCase();

  if (!content.trim()) {
    return null;
  }

  if (format === "html") {
    return (
      <div
        className="rounded border border-border/60 bg-muted/20 p-4 text-sm leading-relaxed font-mono"
        dangerouslySetInnerHTML={{ __html: content }}
      />
    );
  }

  return (
    <div className="rounded border border-border/60 bg-muted/20 p-4 text-sm leading-relaxed font-mono whitespace-pre-wrap">
      {content}
    </div>
  );
};
