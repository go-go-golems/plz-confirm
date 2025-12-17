import { store, setConnected, setError, setActiveRequest, completeRequest, addToHistory } from '@/store/store';
import { UIRequest } from '@/types/schemas';

let ws: WebSocket | null = null;
let reconnectTimeout: NodeJS.Timeout | null = null;

export const connectWebSocket = () => {
  const state = store.getState();
  const sessionId = state.session.id;

  if (!sessionId) return;

  const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
  const host = window.location.host; // Includes port if present
  const wsUrl = `${protocol}//${host}/ws?sessionId=${sessionId}`;

  console.log(`Connecting to WebSocket: ${wsUrl}`);

  ws = new WebSocket(wsUrl);

  ws.onopen = () => {
    console.log('WebSocket connected');
    store.dispatch(setConnected(true));
    store.dispatch(setError(null));
    if (reconnectTimeout) {
      clearTimeout(reconnectTimeout);
      reconnectTimeout = null;
    }
  };

  ws.onmessage = (event) => {
    try {
      const data = JSON.parse(event.data);
      console.log('WS Message:', data);

      if (data.type === 'new_request') {
        store.dispatch(setActiveRequest(data.request));
      } else if (data.type === 'request_completed') {
        // If another client completed it, or just to sync history
        const currentActive = store.getState().request.active;
        if (currentActive && currentActive.id === data.request.id) {
            store.dispatch(completeRequest({ id: data.request.id, output: data.request.output }));
        } else {
            store.dispatch(addToHistory(data.request));
        }
      }
    } catch (e) {
      console.error('Failed to parse WS message', e);
    }
  };

  ws.onclose = () => {
    console.log('WebSocket disconnected');
    store.dispatch(setConnected(false));
    ws = null;
    
    // Try to reconnect
    if (!reconnectTimeout) {
      reconnectTimeout = setTimeout(() => {
        console.log('Attempting reconnect...');
        connectWebSocket();
      }, 3000);
    }
  };

  ws.onerror = (error) => {
    console.error('WebSocket error', error);
    store.dispatch(setError('Connection error'));
  };
};

export const submitResponse = async (requestId: string, output: any) => {
  try {
    const response = await fetch(`/api/requests/${requestId}/response`, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json'
      },
      body: JSON.stringify({ output })
    });

    if (!response.ok) {
      throw new Error('Failed to submit response');
    }

    return await response.json();
  } catch (error) {
    console.error('Error submitting response:', error);
    throw error;
  }
};
