import React, { useMemo, useState } from 'react';
import { ImageInput, ImageOutput } from '@/types/schemas';
import { Button } from '@/components/ui/button';
import { Checkbox } from '@/components/ui/checkbox';
import { ScrollArea } from '@/components/ui/scroll-area';
import { AspectRatio } from '@/components/ui/aspect-ratio';
import { cn } from '@/lib/utils';
import { Loader2, Check, X, ImageOff } from 'lucide-react';
import { OptionalComment, normalizeOptionalComment } from './OptionalComment';

interface Props {
  requestId: string;
  input: ImageInput;
  onSubmit: (output: ImageOutput) => Promise<void>;
  loading?: boolean;
}

type ImgState = {
  loaded: boolean;
  error: boolean;
};

export const ImageDialog: React.FC<Props> = ({ input, onSubmit, loading }) => {
  const [submitting, setSubmitting] = useState<'submit' | 'approve' | 'reject' | null>(null);
  const [selectedIdx, setSelectedIdx] = useState<number[]>([]);
  const [selectedOptions, setSelectedOptions] = useState<string[]>([]);
  const [imgState, setImgState] = useState<Record<number, ImgState>>({});
  const [comment, setComment] = useState('');

  const isConfirm = input.mode === 'confirm';
  const hasOptions = Array.isArray(input.options) && input.options.length > 0;
  const isMulti = Boolean(input.multi);

  const gridClass = useMemo(() => {
    const n = input.images?.length || 0;
    if (n <= 1) return "grid grid-cols-1 gap-4";
    if (n === 2) return "grid grid-cols-1 md:grid-cols-2 gap-4";
    if (n <= 4) return "grid grid-cols-1 sm:grid-cols-2 gap-4";
    return "grid grid-cols-1 sm:grid-cols-2 md:grid-cols-3 gap-4";
  }, [input.images]);

  const toggleIdx = (idx: number) => {
    if (isMulti) {
      setSelectedIdx(prev => prev.includes(idx) ? prev.filter(i => i !== idx) : [...prev, idx]);
      return;
    }
    setSelectedIdx([idx]);
  };

  const toggleOption = (opt: string) => {
    if (isMulti) {
      setSelectedOptions(prev => prev.includes(opt) ? prev.filter(o => o !== opt) : [...prev, opt]);
      return;
    }
    setSelectedOptions([opt]);
  };

  const submitSelect = async () => {
    if (!hasOptions && selectedIdx.length === 0) return;
    if (hasOptions && selectedOptions.length === 0) return;

    setSubmitting('submit');
    const selected = hasOptions
      ? (isMulti ? selectedOptions : selectedOptions[0])
      : (isMulti ? selectedIdx : selectedIdx[0]);

    const c = normalizeOptionalComment(comment);
    await onSubmit({
      selected,
      timestamp: new Date().toISOString(),
      ...(c ? { comment: c } : {})
    });
    setSubmitting(null);
  };

  const submitConfirm = async (approved: boolean) => {
    setSubmitting(approved ? 'approve' : 'reject');
    const c = normalizeOptionalComment(comment);
    await onSubmit({
      selected: approved,
      timestamp: new Date().toISOString(),
      ...(c ? { comment: c } : {})
    });
    setSubmitting(null);
  };

  const selectionCount = hasOptions ? selectedOptions.length : selectedIdx.length;

  return (
    <div className="bg-background p-6 md:p-8 min-h-[420px] flex flex-col relative">
      <div className="space-y-4 mb-6">
        <h2 className="text-2xl font-display font-bold tracking-tight text-primary uppercase">
          {input.title}
        </h2>
        <div className="h-px w-full bg-border" />
        {input.message && (
          <p className="text-sm text-muted-foreground font-mono leading-relaxed">
            {input.message}
          </p>
        )}
      </div>

      <div className={cn(gridClass, "mb-6")}>
        {(input.images || []).map((img, idx) => {
          const isSelected = selectedIdx.includes(idx);
          const st = imgState[idx];
          const failed = st?.error;

          const tile = (
            <div
              className={cn(
                "border border-border bg-black/20 overflow-hidden transition-all",
                !isConfirm && !hasOptions && "cursor-pointer hover:border-primary/40",
                !isConfirm && !hasOptions && isSelected && "border-primary/70 bg-primary/10",
                failed && "border-destructive/50"
              )}
              onClick={() => {
                if (loading || submitting) return;
                // Only allow tile selection in select mode without options (Variant A).
                if (!isConfirm && !hasOptions) toggleIdx(idx);
              }}
              role={!isConfirm && !hasOptions ? "button" : undefined}
              tabIndex={!isConfirm && !hasOptions ? 0 : -1}
            >
              <AspectRatio ratio={16 / 9}>
                {!failed ? (
                  <img
                    src={img.src}
                    alt={img.alt || img.label || `Image ${idx + 1}`}
                    className="w-full h-full object-contain"
                    onLoad={() => setImgState(prev => ({ ...prev, [idx]: { loaded: true, error: false } }))}
                    onError={() => setImgState(prev => ({ ...prev, [idx]: { loaded: false, error: true } }))}
                  />
                ) : (
                  <div className="w-full h-full flex flex-col items-center justify-center text-destructive/80 gap-2">
                    <ImageOff className="h-8 w-8" />
                    <div className="text-xs font-mono">ERROR_LOADING</div>
                  </div>
                )}
              </AspectRatio>

              {(img.label || img.caption) && (
                <div className="p-3 border-t border-border/50 space-y-1">
                  {img.label && (
                    <div className="text-xs font-mono text-primary uppercase">
                      {img.label}
                    </div>
                  )}
                  {img.caption && (
                    <div className="text-xs text-muted-foreground font-mono">
                      {img.caption}
                    </div>
                  )}
                </div>
              )}
            </div>
          );

          return (
            <div key={idx} className="space-y-2">
              {/* If we're in Variant A multi-select, show a small selection indicator */}
              {!isConfirm && !hasOptions && (
                <div className="text-[10px] font-mono text-muted-foreground flex justify-between">
                  <span>IMG_{idx + 1}</span>
                  {isSelected ? <span className="text-primary">SELECTED</span> : <span> </span>}
                </div>
              )}
              {tile}
            </div>
          );
        })}
      </div>

      {/* Variant B: checkbox options below images */}
      {!isConfirm && hasOptions && (
        <div className="flex-1 mb-6">
          <div className="text-xs font-mono text-muted-foreground mb-3 uppercase">
            MULTI_SELECT
          </div>
          <ScrollArea className="border border-border bg-black/20 h-[220px] px-3">
            <div className="py-3 space-y-3">
              {input.options!.map((opt, idx) => {
                const checked = selectedOptions.includes(opt);
                return (
                  <div
                    key={idx}
                    className={cn(
                      "flex items-center gap-3 p-2 border border-transparent hover:border-primary/30 hover:bg-primary/5 cursor-pointer",
                      checked && "bg-primary/10 border-primary/50"
                    )}
                    onClick={() => {
                      if (loading || submitting) return;
                      toggleOption(opt);
                    }}
                  >
                    <Checkbox checked={checked} onCheckedChange={() => {}} />
                    <span className={cn("font-mono text-sm", checked && "text-primary font-bold")}>
                      {opt}
                    </span>
                  </div>
                );
              })}
            </div>
          </ScrollArea>
        </div>
      )}

      <div className="mt-auto pt-4 border-t border-border/50 space-y-3">
        <OptionalComment value={comment} onChange={setComment} disabled={loading || submitting !== null} />

        {/* Footer actions */}
        {isConfirm ? (
          <div className="flex flex-col sm:flex-row gap-4">
            <Button
              variant="outline"
              className="cyber-button flex-1 h-14 text-lg border-destructive/50 text-destructive hover:bg-destructive/10 hover:text-destructive"
              onClick={() => submitConfirm(false)}
              disabled={loading || submitting !== null}
            >
              {submitting === 'reject' ? <Loader2 className="mr-2 h-5 w-5 animate-spin" /> : <X className="mr-2 h-5 w-5" />}
              REJECT
            </Button>
            <Button
              className="cyber-button flex-1 h-14 text-lg bg-primary text-primary-foreground hover:bg-primary/90"
              onClick={() => submitConfirm(true)}
              disabled={loading || submitting !== null}
            >
              {submitting === 'approve' ? <Loader2 className="mr-2 h-5 w-5 animate-spin" /> : <Check className="mr-2 h-5 w-5" />}
              APPROVE
            </Button>
          </div>
        ) : (
          <div className="flex justify-end">
            <div className="flex items-center gap-4 w-full">
              <div className="text-xs font-mono text-muted-foreground flex-1">
                {selectionCount} SELECTED
              </div>
              <Button
                className="cyber-button min-w-[160px]"
                onClick={submitSelect}
                disabled={loading || submitting !== null || selectionCount === 0}
              >
                {submitting === 'submit' ? <Loader2 className="mr-2 h-4 w-4 animate-spin" /> : null}
                SUBMIT_ANSWER
              </Button>
            </div>
          </div>
        )}
      </div>
    </div>
  );
};


