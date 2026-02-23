import { RequestStatus, UIRequest, WidgetType } from "@/proto/generated/plz_confirm/v1/request";
import { nanoid } from "nanoid";

const SESSION_ID = "550e8400-e29b-41d4-a716-446655440000";

export const MOCK_REQUESTS: UIRequest[] = [
  {
    id: nanoid(),
    type: WidgetType.confirm,
    sessionId: SESSION_ID,
    confirmInput: {
      title: "Deploy to production?",
      message: "This will deploy v2.3.1 to the production environment. Are you sure?",
      approveText: "Deploy",
      rejectText: "Cancel"
    },
    status: RequestStatus.pending,
    createdAt: new Date().toISOString(),
    expiresAt: new Date(Date.now() + 300000).toISOString(), // 5 mins
    scriptLogs: [],
  },
  {
    id: nanoid(),
    type: WidgetType.select,
    sessionId: SESSION_ID,
    selectInput: {
      title: "Choose environment",
      options: ["development", "staging", "production", "disaster-recovery"],
      multi: false,
      searchable: true
    },
    status: RequestStatus.completed,
    selectOutput: { selectedSingle: "staging" },
    createdAt: new Date(Date.now() - 3600000).toISOString(),
    completedAt: new Date(Date.now() - 3500000).toISOString(),
    expiresAt: new Date(Date.now() - 3300000).toISOString(),
    scriptLogs: [],
  },
  {
    id: nanoid(),
    type: WidgetType.table,
    sessionId: SESSION_ID,
    tableInput: {
      title: "Select users to notify",
      multiSelect: true,
      searchable: true,
      columns: ["name", "email", "role"],
      data: [
        { id: 1, name: "Alice Johnson", email: "alice@example.com", role: "Admin" },
        { id: 2, name: "Bob Smith", email: "bob@example.com", role: "Editor" },
        { id: 3, name: "Carol Williams", email: "carol@example.com", role: "Admin" },
        { id: 4, name: "David Brown", email: "david@example.com", role: "Viewer" },
        { id: 5, name: "Eve Davis", email: "eve@example.com", role: "Editor" },
      ]
    },
    status: RequestStatus.pending,
    createdAt: new Date().toISOString(),
    expiresAt: new Date(Date.now() + 600000).toISOString(),
    scriptLogs: [],
  }
];

export const getMockSession = () => ({
  id: SESSION_ID,
  connected: true,
  reconnecting: false,
  error: null
});
