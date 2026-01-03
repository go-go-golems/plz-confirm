import { configureStore, createSlice, PayloadAction } from '@reduxjs/toolkit';
import { UIRequest } from '@/proto/generated/plz_confirm/v1/request';
import { Notification } from '@/types/notifications';

// Session Slice
interface SessionState {
  id: string | null;
  connected: boolean;
  reconnecting: boolean;
  error: string | null;
}

// Use a fixed session ID for the demo to match the CLI script
const DEMO_SESSION_ID = "550e8400-e29b-41d4-a716-446655440000";

const initialSessionState: SessionState = {
  id: DEMO_SESSION_ID,
  connected: false,
  reconnecting: false,
  error: null
};

const sessionSlice = createSlice({
  name: 'session',
  initialState: initialSessionState,
  reducers: {
    setConnected: (state, action: PayloadAction<boolean>) => {
      state.connected = action.payload;
    },
    setError: (state, action: PayloadAction<string | null>) => {
      state.error = action.payload;
    }
  }
});

// Request Slice
interface RequestState {
  active: UIRequest | null;
  pending: UIRequest[];
  history: UIRequest[];
  loading: boolean;
}

const initialRequestState: RequestState = {
  active: null,
  pending: [],
  history: [],
  loading: false
};

const requestSlice = createSlice({
  name: 'request',
  initialState: initialRequestState,
  reducers: {
    setActiveRequest: (state: RequestState, action: PayloadAction<UIRequest | null>) => {
      state.active = action.payload;
    },
    enqueueRequest: (state: RequestState, action: PayloadAction<UIRequest>) => {
      const incoming = action.payload;

      if (state.active?.id === incoming.id) return;
      if (state.pending.some((r) => r.id === incoming.id)) return;
      if (state.history.some((r) => r.id === incoming.id)) return;

      if (!state.active) {
        state.active = incoming;
        return;
      }
      state.pending.push(incoming);
    },
    completeRequest: (state: RequestState, action: PayloadAction<UIRequest>) => {
      const completedReq = action.payload;
      if (state.active?.id === completedReq.id) {
        state.active = null;
      }

      state.pending = state.pending.filter((r) => r.id !== completedReq.id);

      state.history = state.history.filter((r) => r.id !== completedReq.id);
      state.history.unshift(completedReq);

      if (!state.active && state.pending.length > 0) {
        state.active = state.pending.shift() ?? null;
      }
    },
    addToHistory: (state: RequestState, action: PayloadAction<UIRequest>) => {
      const req = action.payload;
      state.history = state.history.filter((r) => r.id !== req.id);
      state.history.unshift(req);
    }
  }
});

// Notification Slice
interface NotificationState {
  items: Notification[];
}

const initialNotificationState: NotificationState = {
  items: []
};

const notificationSlice = createSlice({
  name: 'notifications',
  initialState: initialNotificationState,
  reducers: {
    addNotification: (state: NotificationState, action: PayloadAction<Notification>) => {
      state.items.unshift(action.payload);
    },
    removeNotification: (state: NotificationState, action: PayloadAction<string>) => {
      state.items = state.items.filter((item: Notification) => item.id !== action.payload);
    }
  }
});

export const store = configureStore({
  reducer: {
    session: sessionSlice.reducer,
    request: requestSlice.reducer,
    notifications: notificationSlice.reducer
  }
});

export type RootState = ReturnType<typeof store.getState>;
export type AppDispatch = typeof store.dispatch;

export const { setConnected, setError } = sessionSlice.actions;
export const { setActiveRequest, enqueueRequest, completeRequest, addToHistory } =
  requestSlice.actions;
export const { addNotification, removeNotification } = notificationSlice.actions;
