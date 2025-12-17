import React, { useState } from 'react';
import { SelectInput, SelectOutput } from '@/types/schemas';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { ScrollArea } from '@/components/ui/scroll-area';
import { Loader2, Search, CheckSquare, Square, ChevronRight } from 'lucide-react';
import { cn } from '@/lib/utils';

interface Props {
  requestId: string;
  input: SelectInput;
  onSubmit: (output: SelectOutput) => Promise<void>;
  loading?: boolean;
}

export const SelectDialog: React.FC<Props> = ({ input, onSubmit, loading }) => {
  const [selected, setSelected] = useState<string[]>([]);
  const [search, setSearch] = useState('');
  const [submitting, setSubmitting] = useState(false);

  const filteredOptions = input.options.filter(opt => 
    opt.toLowerCase().includes(search.toLowerCase())
  );

  const toggleSelection = (option: string) => {
    if (input.multi) {
      setSelected(prev => 
        prev.includes(option) 
          ? prev.filter(i => i !== option)
          : [...prev, option]
      );
    } else {
      setSelected([option]);
    }
  };

  const handleSubmit = async () => {
    if (selected.length === 0) return;
    
    setSubmitting(true);
    await onSubmit({
      selected: input.multi ? selected : selected[0]
    });
    setSubmitting(false);
  };

  return (
    <div className="bg-background p-6 md:p-8 min-h-[400px] flex flex-col relative">
      <div className="space-y-4 mb-6">
        <h2 className="text-2xl font-display font-bold tracking-tight text-primary uppercase">
          {input.title}
        </h2>
        <div className="h-px w-full bg-border" />
      </div>

      {input.searchable && (
        <div className="relative mb-4 group">
          <Search className="absolute left-3 top-1/2 -translate-y-1/2 h-4 w-4 text-muted-foreground group-focus-within:text-primary transition-colors" />
          <Input 
            placeholder="FILTER_OPTIONS..." 
            value={search}
            onChange={(e) => setSearch(e.target.value)}
            className="pl-9 cyber-input h-12 font-mono text-sm"
          />
        </div>
      )}

      <ScrollArea className="flex-1 border border-border bg-black/20 -mx-2 px-2 mb-6 h-[300px]">
        <div className="space-y-1 py-2">
          {filteredOptions.length === 0 ? (
            <div className="text-center py-8 text-muted-foreground font-mono text-sm">
              NO_MATCHES_FOUND
            </div>
          ) : (
            filteredOptions.map((option, idx) => {
              const isSelected = selected.includes(option);
              return (
                <div
                  key={idx}
                  onClick={() => toggleSelection(option)}
                  className={cn(
                    "flex items-center p-3 cursor-pointer transition-all duration-200 border border-transparent hover:border-primary/30 hover:bg-primary/5 group",
                    isSelected && "bg-primary/10 border-primary/50"
                  )}
                >
                  <div className={cn(
                    "mr-3 text-muted-foreground transition-colors",
                    isSelected && "text-primary"
                  )}>
                    {input.multi ? (
                      isSelected ? <CheckSquare className="h-5 w-5" /> : <Square className="h-5 w-5" />
                    ) : (
                      <div className={cn(
                        "w-4 h-4 border border-current flex items-center justify-center",
                        isSelected ? "bg-primary text-black" : "bg-transparent"
                      )}>
                        {isSelected && <div className="w-2 h-2 bg-black" />}
                      </div>
                    )}
                  </div>
                  <span className={cn(
                    "font-mono text-sm flex-1",
                    isSelected && "text-primary font-bold"
                  )}>
                    {option}
                  </span>
                  {isSelected && <ChevronRight className="h-4 w-4 text-primary animate-pulse" />}
                </div>
              );
            })
          )}
        </div>
      </ScrollArea>

      <div className="flex justify-end pt-4 border-t border-border">
        <div className="flex items-center gap-4 w-full">
          <div className="text-xs font-mono text-muted-foreground flex-1">
            {selected.length} SELECTED
          </div>
          <Button 
            className="cyber-button min-w-[140px]"
            onClick={handleSubmit}
            disabled={loading || submitting || selected.length === 0}
          >
            {submitting ? <Loader2 className="mr-2 h-4 w-4 animate-spin" /> : null}
            CONFIRM_SELECTION
          </Button>
        </div>
      </div>
    </div>
  );
};
