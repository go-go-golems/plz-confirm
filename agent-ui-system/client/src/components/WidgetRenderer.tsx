import React from "react";
import { useSelector } from "react-redux";
import { RootState } from "@/store/store";
import { submitResponse, submitScriptEvent, touchRequest } from "@/services/websocket";
import { ConfirmDialog } from "./widgets/ConfirmDialog";
import { SelectDialog } from "./widgets/SelectDialog";
import { TableDialog } from "./widgets/TableDialog";
import { FormDialog } from "./widgets/FormDialog";
import { UploadDialog } from "./widgets/UploadDialog";
import { ImageDialog } from "./widgets/ImageDialog";
import { GridDialog } from "./widgets/GridDialog";
import { DisplayWidget } from "./widgets/DisplayWidget";
import { Loader2 } from "lucide-react";
import { WidgetType } from "@/proto/generated/plz_confirm/v1/request";

export const WidgetRenderer: React.FC = () => {
  const { active, loading } = useSelector((state: RootState) => state.request);
  const lastTouchedId = React.useRef<string | null>(null);
  const [nowMs, setNowMs] = React.useState(() => Date.now());

  React.useEffect(() => {
    setNowMs(Date.now());
    if (!active) return;
    if (active.expiryDisabled) return;
    const t = setInterval(() => setNowMs(Date.now()), 1000);
    return () => clearInterval(t);
  }, [active?.id, active?.expiryDisabled]);

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

  const parseRFC3339NanoToMs = (s: string): number | null => {
    if (!s) return null;
    const m = s.match(
      /^(\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2})(?:\.(\d+))?(Z|[+-]\d{2}:\d{2})$/
    );
    if (!m) {
      const t = Date.parse(s);
      return Number.isFinite(t) ? t : null;
    }
    const base = m[1];
    const frac = (m[2] ?? "0").padEnd(3, "0").slice(0, 3);
    const tz = m[3];
    const t = Date.parse(`${base}.${frac}${tz}`);
    return Number.isFinite(t) ? t : null;
  };

  const expiresAtMs = parseRFC3339NanoToMs(active.expiresAt) ?? null;
  const createdAtMs = parseRFC3339NanoToMs(active.createdAt) ?? null;
  const totalMs =
    expiresAtMs != null && createdAtMs != null
      ? Math.max(0, expiresAtMs - createdAtMs)
      : null;
  const remainingMs =
    expiresAtMs == null ? null : Math.max(0, expiresAtMs - nowMs);
  const remainingS =
    remainingMs == null ? null : Math.max(0, Math.ceil(remainingMs / 1000));
  const remainingPct =
    totalMs == null || totalMs === 0 || remainingMs == null
      ? null
      : Math.max(0, Math.min(100, (remainingMs / totalMs) * 100));
  const formatRemaining = (totalS: number) => {
    const s = Math.max(0, totalS);
    const h = Math.floor(s / 3600);
    const m = Math.floor((s % 3600) / 60);
    const ss = s % 60;
    if (h > 0) {
      return `${String(h).padStart(2, "0")}:${String(m).padStart(2, "0")}:${String(
        ss
      ).padStart(2, "0")}`;
    }
    return `${String(m).padStart(2, "0")}:${String(ss).padStart(2, "0")}`;
  };

  const handleFirstInteraction = () => {
    if (!active) return;
    if (active.expiryDisabled) return;
    if (lastTouchedId.current === active.id) return;
    lastTouchedId.current = active.id;
    void touchRequest(active.id);
  };

  const handleSubmit = async (output: any) => {
    try {
      await submitResponse(active.id, active.type, output);
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

  const handleScriptSubmit = async (output: any) => {
    try {
      await submitScriptEvent(active.id, {
        type: "submit",
        stepId: active.scriptView?.stepId,
        data: output,
      });
    } catch (error) {
      console.error("Failed to submit script event", error);
    }
  };

  const renderScriptView = () => {
    if (!active.scriptView) {
      return (
        <div className="p-8 border border-destructive/50 bg-destructive/10 text-destructive">
          ERROR: SCRIPT_VIEW_MISSING
        </div>
      );
    }

    const scriptCommonProps = {
      requestId: active.id,
      onSubmit: handleScriptSubmit,
      loading: loading,
    };

    const renderInteractiveScriptWidget = (widgetType: string, input: any) => {
      switch (widgetType) {
        case "confirm":
          return <ConfirmDialog {...scriptCommonProps} input={input} />;
        case "select":
          return <SelectDialog {...scriptCommonProps} input={input} />;
        case "table":
          return <TableDialog {...scriptCommonProps} input={input} />;
        case "form":
          return <FormDialog {...scriptCommonProps} input={input} />;
        case "upload":
          return <UploadDialog {...scriptCommonProps} input={input} />;
        case "image":
          return <ImageDialog {...scriptCommonProps} input={input} />;
        case "grid":
          return <GridDialog {...scriptCommonProps} input={input} />;
        default:
          return (
            <div className="p-8 border border-destructive/50 bg-destructive/10 text-destructive">
              ERROR: UNSUPPORTED_SCRIPT_WIDGET [{widgetType || "unknown"}]
            </div>
          );
      }
    };

    const sections = Array.isArray(active.scriptView.sections)
      ? active.scriptView.sections.map(section => ({
          widgetType: String(section.widgetType || "")
            .trim()
            .toLowerCase(),
          input: (section.input ?? {}) as any,
        }))
      : [];

    if (sections.length === 0) {
      const widgetType = String(active.scriptView.widgetType || "")
        .trim()
        .toLowerCase();
      const input = (active.scriptView.input ?? {}) as any;
      return renderInteractiveScriptWidget(widgetType, input);
    }

    const interactiveSections = sections.filter(
      section => section.widgetType !== "display"
    );
    if (interactiveSections.length !== 1) {
      return (
        <div className="p-8 border border-destructive/50 bg-destructive/10 text-destructive">
          ERROR: INVALID_SCRIPT_SECTIONS [exactly one interactive section is required]
        </div>
      );
    }

    return (
      <div className="space-y-3">
        {sections.map((section, idx) => {
          if (section.widgetType === "display") {
            return (
              <DisplayWidget
                key={`display-${idx}`}
                input={section.input}
              />
            );
          }
          return (
            <React.Fragment key={`interactive-${idx}`}>
              {renderInteractiveScriptWidget(section.widgetType, section.input)}
            </React.Fragment>
          );
        })}
      </div>
    );
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
      case WidgetType.script:
        return renderScriptView();
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
        <span className="flex flex-col items-end gap-1">
          {active.expiryDisabled ? (
            <span className="text-[10px] text-green-500">TIMEOUT_DISABLED</span>
          ) : remainingS != null ? (
            <>
              <span className="text-[10px] text-yellow-500">
                EXPIRES_IN: {formatRemaining(remainingS)}
              </span>
              {remainingPct != null && (
                <span className="h-1 w-28 rounded bg-muted/40 overflow-hidden">
                  <span
                    className="block h-full bg-yellow-500"
                    style={{ width: `${remainingPct}%` }}
                  />
                </span>
              )}
            </>
          ) : (
            <span />
          )}
        </span>
        <span>
          TYPE:{" "}
          {String(
            (WidgetType as any)[active.type] ?? active.type
          ).toUpperCase()}
        </span>
      </div>

      <div
        className="cyber-card p-1"
        onPointerDownCapture={handleFirstInteraction}
        onKeyDownCapture={handleFirstInteraction}
      >
        {renderWidget()}
      </div>

      <div className="mt-2 flex justify-between text-[10px] text-muted-foreground/50 font-mono">
        <span>SECURE_CONNECTION</span>
        <span>ENCRYPTED_AES_256</span>
      </div>
    </div>
  );
};
