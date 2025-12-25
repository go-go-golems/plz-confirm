import React from 'react';
import { useSelector, useDispatch } from 'react-redux';
import { RootState, completeRequest } from '@/store/store';
import { submitResponse } from '@/services/websocket';
import { ConfirmDialog } from './widgets/ConfirmDialog';
import { SelectDialog } from './widgets/SelectDialog';
import { TableDialog } from './widgets/TableDialog';
import { FormDialog } from './widgets/FormDialog';
import { UploadDialog } from './widgets/UploadDialog';
import { ImageDialog } from './widgets/ImageDialog';
import { Loader2 } from 'lucide-react';

export const WidgetRenderer: React.FC = () => {
  const { active, loading } = useSelector((state: RootState) => state.request);
  const dispatch = useDispatch();

  if (!active) {
    return (
      <div className="flex flex-col items-center justify-center h-[60vh] text-muted-foreground animate-in fade-in duration-700">
        <div className="w-24 h-24 mb-6 opacity-20 relative">
           <div className="absolute inset-0 border-2 border-primary animate-[spin_10s_linear_infinite]" />
           <div className="absolute inset-4 border border-primary/50 animate-[spin_5s_linear_infinite_reverse]" />
        </div>
        <h2 className="text-xl font-display tracking-widest mb-2">SYSTEM_IDLE</h2>
        <p className="text-sm font-mono opacity-60">WAITING_FOR_INCOMING_TRANSMISSION...</p>
      </div>
    );
  }

  const handleSubmit = async (output: any) => {
    try {
      await submitResponse(active.id, output);
      dispatch(completeRequest({ id: active.id, output }));
    } catch (error) {
      console.error("Failed to submit response", error);
      // Ideally show error toast here
    }
  };

  const commonProps = {
    requestId: active.id,
    input: active.input,
    onSubmit: handleSubmit,
    loading: loading
  };

  const renderWidget = () => {
    switch (active.type) {
      case 'confirm':
        return <ConfirmDialog {...commonProps} input={active.input} />;
      case 'select':
        return <SelectDialog {...commonProps} input={active.input} />;
      case 'table':
        return <TableDialog {...commonProps} input={active.input} />;
      case 'form':
        return <FormDialog {...commonProps} input={active.input} />;
      case 'upload':
        return <UploadDialog {...commonProps} input={active.input} />;
      case 'image':
        return <ImageDialog {...commonProps} input={active.input} />;
      default:
        return (
          <div className="p-8 border border-destructive/50 bg-destructive/10 text-destructive">
            ERROR: UNKNOWN_WIDGET_TYPE [{active.type}]
          </div>
        );
    }
  };

  return (
    <div className="w-full max-w-3xl mx-auto animate-in slide-in-from-bottom-4 duration-500">
      <div className="mb-2 flex justify-between items-end text-xs text-muted-foreground font-mono uppercase">
        <span>REQ_ID: {active.id.substring(0, 8)}</span>
        <span>TYPE: {active.type.toUpperCase()}</span>
      </div>
      
      <div className="cyber-card p-1">
        {renderWidget()}
      </div>
      
      <div className="mt-2 flex justify-between text-[10px] text-muted-foreground/50 font-mono">
        <span>SECURE_CONNECTION</span>
        <span>ENCRYPTED_AES_256</span>
      </div>
    </div>
  );
};
