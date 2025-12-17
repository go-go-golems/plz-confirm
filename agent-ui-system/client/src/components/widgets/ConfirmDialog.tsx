import React from 'react';
import { ConfirmInput, ConfirmOutput } from '@/types/schemas';
import { Button } from '@/components/ui/button';
import { Loader2, AlertTriangle, Check, X } from 'lucide-react';

interface Props {
  requestId: string;
  input: ConfirmInput;
  onSubmit: (output: ConfirmOutput) => Promise<void>;
  loading?: boolean;
}

export const ConfirmDialog: React.FC<Props> = ({ input, onSubmit, loading }) => {
  const [submitting, setSubmitting] = React.useState<'approve' | 'reject' | null>(null);

  const handleAction = async (approved: boolean) => {
    setSubmitting(approved ? 'approve' : 'reject');
    await onSubmit({
      approved,
      timestamp: new Date().toISOString()
    });
    setSubmitting(null);
  };

  return (
    <div className="bg-background p-6 md:p-8 min-h-[300px] flex flex-col justify-between relative overflow-hidden">
      {/* Background decoration */}
      <div className="absolute top-0 right-0 p-4 opacity-5">
        <AlertTriangle size={120} />
      </div>

      <div className="space-y-6 relative z-10">
        <div className="space-y-2">
          <h2 className="text-2xl md:text-3xl font-display font-bold tracking-tight text-primary">
            {input.title}
          </h2>
          <div className="h-1 w-12 bg-primary/50" />
        </div>
        
        {input.message && (
          <p className="text-lg text-muted-foreground font-mono leading-relaxed max-w-xl">
            {input.message}
          </p>
        )}
      </div>

      <div className="flex flex-col sm:flex-row gap-4 mt-8 pt-8 border-t border-border/50">
        <Button 
          variant="outline" 
          className="cyber-button flex-1 h-14 text-lg border-destructive/50 text-destructive hover:bg-destructive/10 hover:text-destructive"
          onClick={() => handleAction(false)}
          disabled={loading || submitting !== null}
        >
          {submitting === 'reject' ? (
            <Loader2 className="mr-2 h-5 w-5 animate-spin" />
          ) : (
            <X className="mr-2 h-5 w-5" />
          )}
          {input.rejectText || 'REJECT'}
        </Button>
        
        <Button 
          className="cyber-button flex-1 h-14 text-lg bg-primary text-primary-foreground hover:bg-primary/90"
          onClick={() => handleAction(true)}
          disabled={loading || submitting !== null}
        >
          {submitting === 'approve' ? (
            <Loader2 className="mr-2 h-5 w-5 animate-spin" />
          ) : (
            <Check className="mr-2 h-5 w-5" />
          )}
          {input.approveText || 'APPROVE'}
        </Button>
      </div>
    </div>
  );
};
