import { RequestStatus, UIRequest, WidgetType } from "@/proto/generated/plz_confirm/v1/request";

function coerceEnumValue<E extends Record<string, any>>(
  e: E,
  v: unknown,
  fallback: number,
): number {
  if (typeof v === "number" && Number.isFinite(v)) return v;
  if (typeof v === "string" && (e as any)[v] !== undefined) return (e as any)[v] as number;
  return fallback;
}

export function normalizeUIRequest(req: any): UIRequest {
  return {
    ...(req as UIRequest),
    type: coerceEnumValue(WidgetType as any, req?.type, WidgetType.widget_type_unspecified),
    status: coerceEnumValue(
      RequestStatus as any,
      req?.status,
      RequestStatus.request_status_unspecified,
    ),
  };
}


