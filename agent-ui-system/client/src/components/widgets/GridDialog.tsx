import React from "react";
import { Loader2 } from "lucide-react";

import {
  GridCell,
  GridInput,
  GridSelection,
} from "@/proto/generated/plz_confirm/v1/widgets";
import { cn } from "@/lib/utils";

interface Props {
  requestId: string;
  input: GridInput;
  onSubmit: (output: GridSelection) => Promise<void>;
  loading?: boolean;
}

const styleClassMap: Record<string, string> = {
  empty: "border-border text-muted-foreground hover:border-primary/40 hover:bg-primary/5",
  filled: "border-primary/40 text-primary bg-primary/10 hover:bg-primary/15",
  highlighted:
    "border-yellow-400/70 text-yellow-100 bg-yellow-500/20 hover:bg-yellow-500/25",
  disabled: "border-border/40 text-muted-foreground/40 bg-muted/10 cursor-not-allowed",
};

const cellSizeClass = (size?: string) => {
  switch (String(size || "").toLowerCase()) {
    case "small":
      return "h-14 text-base";
    case "large":
      return "h-24 text-2xl";
    case "medium":
    default:
      return "h-20 text-xl";
  }
};

const normalizeGridDimension = (value: unknown): number => {
  const n = Number(value);
  if (!Number.isFinite(n) || n < 1) return 0;
  return Math.floor(n);
};

const normalizeGridCells = (cells: GridCell[] | undefined, totalCells: number): GridCell[] => {
  const source = Array.isArray(cells) ? cells : [];
  const output: GridCell[] = [];
  for (let i = 0; i < totalCells; i += 1) {
    output.push(source[i] ?? { value: "" });
  }
  return output;
};

export const GridDialog: React.FC<Props> = ({ input, onSubmit, loading }) => {
  const [submittingCell, setSubmittingCell] = React.useState<number | null>(null);

  const rows = normalizeGridDimension(input.rows);
  const cols = normalizeGridDimension(input.cols);
  const totalCells = rows * cols;
  const cells = normalizeGridCells(input.cells, totalCells);

  const submitCell = async (cellIndex: number) => {
    if (rows < 1 || cols < 1) return;
    setSubmittingCell(cellIndex);
    try {
      await onSubmit({
        row: Math.floor(cellIndex / cols),
        col: cellIndex % cols,
        cellIndex,
      });
    } finally {
      setSubmittingCell(null);
    }
  };

  if (rows < 1 || cols < 1) {
    return (
      <div className="bg-background p-6 md:p-8 min-h-[320px] flex flex-col">
        <h2 className="text-2xl font-display font-bold tracking-tight text-primary uppercase">
          {input.title || "GRID"}
        </h2>
        <div className="mt-4 rounded border border-destructive/50 bg-destructive/10 p-3 text-sm font-mono text-destructive">
          INVALID_GRID_DIMENSIONS
        </div>
      </div>
    );
  }

  return (
    <div className="bg-background p-6 md:p-8 min-h-[360px] flex flex-col relative">
      <div className="space-y-4 mb-6">
        <h2 className="text-2xl font-display font-bold tracking-tight text-primary uppercase">
          {input.title}
        </h2>
        <div className="h-px w-full bg-border" />
      </div>

      <div className="overflow-auto rounded border border-border bg-black/20 p-3">
        <div
          className="grid gap-2"
          style={{ gridTemplateColumns: `repeat(${cols}, minmax(0, 1fr))` }}
        >
          {cells.map((cell, index) => {
            const styleKey = String(cell.style || "empty").toLowerCase();
            const isCellDisabled =
              Boolean(cell.disabled) ||
              styleKey === "disabled" ||
              Boolean(loading) ||
              submittingCell !== null;
            return (
              <button
                key={index}
                type="button"
                title={cell.label || undefined}
                disabled={isCellDisabled}
                onClick={() => submitCell(index)}
                className={cn(
                  "relative rounded border font-bold font-mono transition-colors",
                  "focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary/50",
                  cellSizeClass(input.cellSize),
                  styleClassMap[styleKey] || styleClassMap.empty
                )}
                style={cell.color ? { color: cell.color } : undefined}
              >
                {submittingCell === index ? (
                  <Loader2 className="mx-auto h-5 w-5 animate-spin" />
                ) : (
                  cell.value || <span className="opacity-30">·</span>
                )}
              </button>
            );
          })}
        </div>
      </div>
    </div>
  );
};
