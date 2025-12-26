import React from 'react';
import { Button } from '@/components/ui/button';
import { Collapsible, CollapsibleContent, CollapsibleTrigger } from '@/components/ui/collapsible';
import { Textarea } from '@/components/ui/textarea';
import { cn } from '@/lib/utils';
import { ChevronDown, MessageSquarePlus } from 'lucide-react';

export function normalizeOptionalComment(raw: string): string | undefined {
  const trimmed = raw.trim();
  return trimmed.length > 0 ? trimmed : undefined;
}

interface Props {
  value: string;
  onChange: (next: string) => void;
  disabled?: boolean;
  placeholder?: string;
  className?: string;
}

export const OptionalComment: React.FC<Props> = ({ value, onChange, disabled, placeholder, className }) => {
  const [open, setOpen] = React.useState(false);

  return (
    <Collapsible open={open} onOpenChange={setOpen} className={cn("space-y-2", className)}>
      <CollapsibleTrigger asChild>
        <Button
          type="button"
          variant="ghost"
          size="sm"
          disabled={disabled}
          className="w-full justify-between font-mono text-xs uppercase text-muted-foreground hover:text-primary"
        >
          <span className="flex items-center gap-2">
            <MessageSquarePlus className="h-4 w-4" />
            Add comment (optional)
          </span>
          <ChevronDown className={cn("h-4 w-4 transition-transform", open && "rotate-180")} />
        </Button>
      </CollapsibleTrigger>

      <CollapsibleContent className="space-y-2">
        <Textarea
          value={value}
          onChange={(e) => onChange(e.target.value)}
          placeholder={placeholder || "Write a short note for the agent..."}
          className="cyber-input font-mono text-sm min-h-[90px]"
          disabled={disabled}
        />
        <div className="text-[10px] font-mono text-muted-foreground/60">
          This will be returned to the CLI as <span className="text-primary">output.comment</span>.
        </div>
      </CollapsibleContent>
    </Collapsible>
  );
};


