import { configureStore, createSlice, PayloadAction } from '@reduxjs/toolkit';
import { UIRequest, Notification } from '@/types/schemas';
import { MOCK_REQUESTS } from '@/services/mockData';

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
  history: UIRequest[];
  loading: boolean;
}

const initialRequestState: RequestState = {
  active: null,
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
    completeRequest: (state: RequestState, action: PayloadAction<{ id: string, output: any }>) => {
      if (state.active && state.active.id === action.payload.id) {
        const completedReq: UIRequest = {
          ...state.active,
          status: 'completed',
          output: action.payload.output,
          completedAt: new Date().toISOString()
        };
        state.history.unshift(completedReq);
        state.active = null;
      }
    },
    addToHistory: (state: RequestState, action: PayloadAction<UIRequest>) => {
      state.history.unshift(action.payload);
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
export const { setActiveRequest, completeRequest, addToHistory } = requestSlice.actions;
export const { addNotification, removeNotification } = notificationSlice.actions;
