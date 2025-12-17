import express from "express";
import { createServer } from "http";
import { WebSocketServer, WebSocket } from "ws";
import cors from "cors";
import bodyParser from "body-parser";
import path from "path";
import { fileURLToPath } from "url";
import { nanoid } from "nanoid";

const __filename = fileURLToPath(import.meta.url);
const __dirname = path.dirname(__filename);

// Types
interface UIRequest {
  id: string;
  type: string;
  sessionId: string;
  input: any;
  output?: any;
  status: 'pending' | 'completed' | 'timeout' | 'error';
  createdAt: string;
  completedAt?: string;
  expiresAt: string;
}

// In-memory storage
const requests = new Map<string, UIRequest>();
const sessions = new Map<string, WebSocket[]>();

async function startServer() {
  const app = express();
  const server = createServer(app);
  const wss = new WebSocketServer({ server, path: "/ws" });

  app.use(cors());
  app.use(bodyParser.json());

  // Serve static files
  const staticPath = process.env.NODE_ENV === "production"
      ? path.resolve(__dirname, "public")
      : path.resolve(__dirname, "..", "dist", "public");
  
  app.use(express.static(staticPath));

  // --- WebSocket Handling ---
  wss.on("connection", (ws, req) => {
    const url = new URL(req.url || "", `http://${req.headers.host}`);
    const sessionId = url.searchParams.get("sessionId");

    if (!sessionId) {
      ws.close(1008, "Session ID required");
      return;
    }

    console.log(`[WS] Client connected: ${sessionId}`);

    if (!sessions.has(sessionId)) {
      sessions.set(sessionId, []);
    }
    sessions.get(sessionId)?.push(ws);

    // Send pending requests for this session
    const pendingRequests = Array.from(requests.values())
      .filter(r => r.sessionId === sessionId && r.status === 'pending');
    
    pendingRequests.forEach(req => {
      ws.send(JSON.stringify({ type: "new_request", request: req }));
    });

    ws.on("close", () => {
      console.log(`[WS] Client disconnected: ${sessionId}`);
      const clients = sessions.get(sessionId) || [];
      const index = clients.indexOf(ws);
      if (index > -1) {
        clients.splice(index, 1);
      }
      if (clients.length === 0) {
        sessions.delete(sessionId);
      }
    });
  });

  // --- REST API ---

  // Create a new request (called by CLI/Agent)
  app.post("/api/requests", (req, res) => {
    const { type, sessionId, input, timeout = 300 } = req.body;

    if (!type || !sessionId || !input) {
      res.status(400).json({ error: "Missing required fields" });
      return;
    }

    const id = nanoid();
    const newRequest: UIRequest = {
      id,
      type,
      sessionId,
      input,
      status: 'pending',
      createdAt: new Date().toISOString(),
      expiresAt: new Date(Date.now() + timeout * 1000).toISOString()
    };

    requests.set(id, newRequest);

    // Notify connected clients via WebSocket
    const clients = sessions.get(sessionId);
    if (clients) {
      clients.forEach(client => {
        if (client.readyState === WebSocket.OPEN) {
          client.send(JSON.stringify({ type: "new_request", request: newRequest }));
        }
      });
    }

    console.log(`[API] Created request ${id} (${type}) for session ${sessionId}`);
    res.status(201).json(newRequest);
  });

  // Get request status (polling fallback)
  app.get("/api/requests/:id", (req, res) => {
    const request = requests.get(req.params.id);
    if (!request) {
      res.status(404).json({ error: "Request not found" });
      return;
    }
    res.json(request);
  });

  // Submit response (called by Frontend)
  app.post("/api/requests/:id/response", (req, res) => {
    const { output } = req.body;
    const id = req.params.id;
    const request = requests.get(id);

    if (!request) {
      res.status(404).json({ error: "Request not found" });
      return;
    }

    if (request.status !== 'pending') {
      res.status(409).json({ error: "Request already completed" });
      return;
    }

    // Update request
    request.output = output;
    request.status = 'completed';
    request.completedAt = new Date().toISOString();
    requests.set(id, request);

    console.log(`[API] Request ${id} completed`);

    // Notify frontend clients (to update UI history)
    const clients = sessions.get(request.sessionId);
    if (clients) {
      clients.forEach(client => {
        if (client.readyState === WebSocket.OPEN) {
          client.send(JSON.stringify({ type: "request_completed", request }));
        }
      });
    }

    res.json(request);
  });

  // Long-poll endpoint for CLI to wait for completion
  app.get("/api/requests/:id/wait", async (req, res) => {
    const id = req.params.id;
    const timeout = parseInt(req.query.timeout as string) || 60;
    const startTime = Date.now();

    const checkStatus = () => {
      const request = requests.get(id);
      if (!request) {
        res.status(404).json({ error: "Request not found" });
        return;
      }

      if (request.status === 'completed') {
        res.json(request);
        return;
      }

      if (Date.now() - startTime > timeout * 1000) {
        res.status(408).json({ error: "Timeout waiting for response" });
        return;
      }

      setTimeout(checkStatus, 500); // Poll every 500ms
    };

    checkStatus();
  });

  // Handle client-side routing
  app.get("*", (_req, res) => {
    res.sendFile(path.join(staticPath, "index.html"));
  });

  const port = 3001;
  server.listen(port, () => {
    console.log(`Server running on http://localhost:${port}/`);
  });
}

startServer().catch(console.error);
