import React from "react";
import { useSelector, useDispatch } from "react-redux";
import { RootState, completeRequest } from "@/store/store";
import { submitResponse } from "@/services/websocket";
import { ConfirmDialog } from "./widgets/ConfirmDialog";
import { SelectDialog } from "./widgets/SelectDialog";
import { TableDialog } from "./widgets/TableDialog";
import { FormDialog } from "./widgets/FormDialog";
import { UploadDialog } from "./widgets/UploadDialog";
import { ImageDialog } from "./widgets/ImageDialog";
import { Loader2 } from "lucide-react";
import { WidgetType } from "@/proto/generated/plz_confirm/v1/request";

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
        <h2 className="text-xl font-display tracking-widest mb-2">
          SYSTEM_IDLE
        </h2>
        <p className="text-sm font-mono opacity-60">
          WAITING_FOR_INCOMING_TRANSMISSION...
        </p>
      </div>
    );
  }

  const handleSubmit = async (output: any) => {
    try {
      const completedReq = await submitResponse(active.id, active.type, output);
      dispatch(completeRequest(completedReq));
    } catch (error) {
      console.error("Failed to submit response", error);
      // Ideally show error toast here
    }
  };

  const commonProps = {
    requestId: active.id,
    onSubmit: handleSubmit,
    loading: loading,
  };

  const renderWidget = () => {
    const typeLabel = (WidgetType as any)[active.type] ?? "unknown";
    switch (active.type) {
      case WidgetType.confirm:
        return active.confirmInput ? (
          <ConfirmDialog {...commonProps} input={active.confirmInput} />
        ) : null;
      case WidgetType.select:
        return active.selectInput ? (
          <SelectDialog {...commonProps} input={active.selectInput} />
        ) : null;
      case WidgetType.table:
        return active.tableInput ? (
          <TableDialog {...commonProps} input={active.tableInput} />
        ) : null;
      case WidgetType.form:
        return active.formInput ? (
          <FormDialog {...commonProps} input={active.formInput} />
        ) : null;
      case WidgetType.upload:
        return active.uploadInput ? (
          <UploadDialog {...commonProps} input={active.uploadInput} />
        ) : null;
      case WidgetType.image:
        return active.imageInput ? (
          <ImageDialog {...commonProps} input={active.imageInput} />
        ) : null;
      default:
        return (
          <div className="p-8 border border-destructive/50 bg-destructive/10 text-destructive">
            ERROR: UNKNOWN_WIDGET_TYPE [{String(typeLabel)}]
          </div>
        );
    }
  };

  return (
    <div className="w-full max-w-3xl mx-auto animate-in slide-in-from-bottom-4 duration-500">
      <div className="mb-2 flex justify-between items-end text-xs text-muted-foreground font-mono uppercase">
        <span>REQ_ID: {active.id.substring(0, 8)}</span>
        <span>
          TYPE:{" "}
          {String(
            (WidgetType as any)[active.type] ?? active.type
          ).toUpperCase()}
        </span>
      </div>

      <div className="cyber-card p-1">{renderWidget()}</div>

      <div className="mt-2 flex justify-between text-[10px] text-muted-foreground/50 font-mono">
        <span>SECURE_CONNECTION</span>
        <span>ENCRYPTED_AES_256</span>
      </div>
    </div>
  );
};
