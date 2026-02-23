import {
  RequestStatus,
  UIRequest,
  WidgetType,
} from "@/proto/generated/plz_confirm/v1/request";

export interface RequestHistoryDisplay {
  typeLabel: string;
  title: string;
  scriptWidgetBadge?: string;
  scriptCompletedMeta?: string;
  isScript: boolean;
}

const UNKNOWN_REQUEST = "UNKNOWN_REQUEST";

const resolveRequestTypeLabel = (req: UIRequest): string =>
  String(
    (WidgetType as unknown as Record<number, string>)[req.type as unknown as number] ??
      req.type
  ).toUpperCase();

const resolveScriptTitle = (req: UIRequest): string =>
  req.scriptInput?.title?.trim() ||
  req.scriptDescribe?.name?.trim() ||
  UNKNOWN_REQUEST;

const resolveStandardTitle = (req: UIRequest): string =>
  req.confirmInput?.title ||
  req.selectInput?.title ||
  req.formInput?.title ||
  req.uploadInput?.title ||
  req.tableInput?.title ||
  req.imageInput?.title ||
  UNKNOWN_REQUEST;

const resolveScriptCompletedMeta = (req: UIRequest): string | undefined => {
  if (req.status !== RequestStatus.completed) return undefined;

  const name = req.scriptDescribe?.name?.trim();
  const version = req.scriptDescribe?.version?.trim();
  if (!name && !version) return undefined;
  if (name && version) return `${name} v${version}`;
  return name || version;
};

export const getRequestHistoryDisplay = (req: UIRequest): RequestHistoryDisplay => {
  const isScript = req.type === WidgetType.script;
  return {
    typeLabel: resolveRequestTypeLabel(req),
    title: isScript ? resolveScriptTitle(req) : resolveStandardTitle(req),
    scriptWidgetBadge: isScript
      ? String(req.scriptView?.widgetType || "unknown").toLowerCase()
      : undefined,
    scriptCompletedMeta: isScript ? resolveScriptCompletedMeta(req) : undefined,
    isScript,
  };
};
