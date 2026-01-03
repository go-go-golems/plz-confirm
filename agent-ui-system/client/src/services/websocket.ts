import {
  store,
  setConnected,
  setError,
  enqueueRequest,
  completeRequest,
} from "@/store/store";
import { browserNotificationService } from "./notifications";
import {
  UIRequest,
  WidgetType,
} from "@/proto/generated/plz_confirm/v1/request";
import { normalizeUIRequest } from "@/proto/normalize";

let ws: WebSocket | null = null;
let reconnectTimeout: NodeJS.Timeout | null = null;

const MAX_KNOWN_COMPLETIONS = 512;
const completedIds = new Set<string>();
const completedOrder: string[] = [];

const markCompleted = (requestId: string) => {
  if (completedIds.has(requestId)) return;
  completedIds.add(requestId);
  completedOrder.push(requestId);
  while (completedOrder.length > MAX_KNOWN_COMPLETIONS) {
    const oldest = completedOrder.shift();
    if (!oldest) continue;
    completedIds.delete(oldest);
  }
};

export const connectWebSocket = () => {
  const state = store.getState();
  const sessionId = state.session.id;

  if (!sessionId) return;

  const protocol = window.location.protocol === "https:" ? "wss:" : "ws:";
  const host = window.location.host; // Includes port if present
  const wsUrl = `${protocol}//${host}/ws?sessionId=${sessionId}`;

  console.log(`Connecting to WebSocket: ${wsUrl}`);

  ws = new WebSocket(wsUrl);

  ws.onopen = () => {
    console.log("WebSocket connected");
    store.dispatch(setConnected(true));
    store.dispatch(setError(null));
    if (reconnectTimeout) {
      clearTimeout(reconnectTimeout);
      reconnectTimeout = null;
    }
  };

  ws.onmessage = event => {
    try {
      const data = JSON.parse(event.data);
      console.log("WS Message:", data);

      if (data.type === "new_request") {
        const request: UIRequest = normalizeUIRequest(data.request);
        store.dispatch(enqueueRequest(request));

        // Show browser notification for new request
        const requestTitle =
          request.confirmInput?.title ||
          request.selectInput?.title ||
          request.formInput?.title ||
          request.uploadInput?.title ||
          request.tableInput?.title ||
          request.imageInput?.title ||
          "New Request";
        const requestTypeLabel = (WidgetType as any)[request.type] ?? "unknown";
        browserNotificationService.showRequestNotification(
          requestTitle,
          String(requestTypeLabel)
        );
      } else if (data.type === "request_completed") {
        const completedReq: UIRequest = normalizeUIRequest(data.request);
        if (completedIds.has(completedReq.id)) return;
        markCompleted(completedReq.id);
        store.dispatch(completeRequest(completedReq));
      }
    } catch (e) {
      console.error("Failed to parse WS message", e);
    }
  };

  ws.onclose = () => {
    console.log("WebSocket disconnected");
    store.dispatch(setConnected(false));
    ws = null;

    // Try to reconnect
    if (!reconnectTimeout) {
      reconnectTimeout = setTimeout(() => {
        console.log("Attempting reconnect...");
        connectWebSocket();
      }, 3000);
    }
  };

  ws.onerror = error => {
    console.error("WebSocket error", error);
    store.dispatch(setError("Connection error"));
  };
};

function buildSubmitResponseBody(requestType: WidgetType, output: any): any {
  const typeLabel =
    (WidgetType as any)[requestType] ?? "widget_type_unspecified";
  const sessionId = store.getState().session.id ?? "global";

  switch (requestType) {
    case WidgetType.confirm:
      return { type: String(typeLabel), sessionId, confirmOutput: output };
    case WidgetType.select:
      return { type: String(typeLabel), sessionId, selectOutput: output };
    case WidgetType.form:
      return { type: String(typeLabel), sessionId, formOutput: output };
    case WidgetType.upload:
      return { type: String(typeLabel), sessionId, uploadOutput: output };
    case WidgetType.table:
      return { type: String(typeLabel), sessionId, tableOutput: output };
    case WidgetType.image:
      return { type: String(typeLabel), sessionId, imageOutput: output };
    default:
      return { type: String(typeLabel), sessionId };
  }
}

export const submitResponse = async (
  requestId: string,
  requestType: WidgetType,
  output: any
) => {
  try {
    const response = await fetch(`/api/requests/${requestId}/response`, {
      method: "POST",
      headers: {
        "Content-Type": "application/json",
      },
      body: JSON.stringify(buildSubmitResponseBody(requestType, output)),
    });

    if (!response.ok) {
      throw new Error("Failed to submit response");
    }

    const json = await response.json();
    const completedReq = normalizeUIRequest(json);
    if (!completedIds.has(requestId)) {
      markCompleted(requestId);
      store.dispatch(completeRequest(completedReq));
    }
    return completedReq;
  } catch (error) {
    console.error("Error submitting response:", error);
    throw error;
  }
};
