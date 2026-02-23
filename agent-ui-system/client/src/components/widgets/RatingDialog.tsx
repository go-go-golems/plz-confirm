import React from "react";
import { Loader2 } from "lucide-react";

import { Button } from "@/components/ui/button";
import { OptionalComment, normalizeOptionalComment } from "./OptionalComment";
import { RatingInput, RatingOutput } from "@/proto/generated/plz_confirm/v1/widgets";
import { cn } from "@/lib/utils";

interface Props {
  requestId: string;
  input: RatingInput;
  onSubmit: (output: RatingOutput) => Promise<void>;
  loading?: boolean;
}

const normalizeScale = (raw: unknown): number => {
  const n = Number(raw ?? 5);
  if (!Number.isFinite(n)) return 5;
  return Math.max(2, Math.min(10, Math.floor(n)));
};

const normalizeStyle = (raw: unknown): "stars" | "numbers" | "emoji" | "slider" => {
  const style = String(raw || "numbers").toLowerCase();
  switch (style) {
    case "stars":
    case "numbers":
    case "emoji":
    case "slider":
      return style;
    default:
      return "numbers";
  }
};

export const RatingDialog: React.FC<Props> = ({ input, onSubmit, loading }) => {
  const [submitting, setSubmitting] = React.useState(false);
  const [comment, setComment] = React.useState("");
  const scale = normalizeScale(input.scale);
  const style = normalizeStyle(input.style);
  const defaultValue = Math.max(
    1,
    Math.min(scale, Number(input.defaultValue ?? Math.ceil(scale / 2)))
  );
  const [value, setValue] = React.useState(defaultValue);

  const values = React.useMemo(
    () => Array.from({ length: scale }, (_, idx) => idx + 1),
    [scale]
  );

  const symbolForValue = (rating: number) => {
    if (style === "stars") return rating <= value ? "★" : "☆";
    if (style === "emoji") {
      const emojis = ["😡", "🙁", "😐", "🙂", "😄", "🤩", "🚀", "🔥", "✨", "🏆"];
      return emojis[rating - 1] ?? "🙂";
    }
    return String(rating);
  };

  const handleSubmit = async () => {
    setSubmitting(true);
    try {
      const c = normalizeOptionalComment(comment);
      await onSubmit({
        value,
        ...(c ? { comment: c } : {}),
      });
    } finally {
      setSubmitting(false);
    }
  };

  return (
    <div className="bg-background p-6 md:p-8 min-h-[360px] flex flex-col relative">
      <div className="space-y-4 mb-6">
        <h2 className="text-2xl font-display font-bold tracking-tight text-primary uppercase">
          {input.title}
        </h2>
        <div className="h-px w-full bg-border" />
      </div>

      <div className="flex-1 flex flex-col justify-center gap-5">
        {style === "slider" ? (
          <div className="space-y-3">
            <input
              type="range"
              min={1}
              max={scale}
              step={1}
              value={value}
              onChange={event => setValue(Number(event.target.value))}
              className="w-full accent-primary"
              disabled={loading || submitting}
            />
            <div className="text-center text-xl font-mono text-primary">{value}</div>
          </div>
        ) : (
          <div className="grid grid-cols-5 gap-2">
            {values.map(rating => (
              <button
                key={rating}
                type="button"
                onClick={() => setValue(rating)}
                disabled={loading || submitting}
                className={cn(
                  "h-12 rounded border font-mono transition-colors",
                  "focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary/50",
                  value === rating
                    ? "border-primary bg-primary/15 text-primary"
                    : "border-border hover:border-primary/40 hover:bg-primary/5"
                )}
              >
                {symbolForValue(rating)}
              </button>
            ))}
          </div>
        )}

        {(input.labels?.low || input.labels?.high) && (
          <div className="flex items-center justify-between text-xs font-mono text-muted-foreground uppercase">
            <span>{input.labels?.low || "LOW"}</span>
            <span>{input.labels?.high || "HIGH"}</span>
          </div>
        )}
      </div>

      <div className="mt-6 pt-4 border-t border-border space-y-3">
        <OptionalComment
          value={comment}
          onChange={setComment}
          disabled={loading || submitting}
        />

        <div className="flex justify-end">
          <Button
            className="cyber-button min-w-[160px]"
            onClick={handleSubmit}
            disabled={loading || submitting}
          >
            {submitting ? <Loader2 className="mr-2 h-4 w-4 animate-spin" /> : null}
            SUBMIT_RATING
          </Button>
        </div>
      </div>
    </div>
  );
};
