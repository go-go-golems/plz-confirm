import React, { useEffect } from 'react';
import { Layout } from '@/components/Layout';
import { WidgetRenderer } from '@/components/WidgetRenderer';
import { useDispatch, useSelector } from 'react-redux';
import { RootState, setActiveRequest, addToHistory } from '@/store/store';
import { MOCK_REQUESTS } from '@/services/mockData';
import { Button } from '@/components/ui/button';
import { ScrollArea } from '@/components/ui/scroll-area';
import { Clock, CheckCircle, XCircle, Terminal } from 'lucide-react';
import { nanoid } from 'nanoid';

export default function Home() {
  const dispatch = useDispatch();
  const { active, history } = useSelector((state: RootState) => state.request);

  // Simulate receiving a new request if none is active
  const simulateNewRequest = (type: string) => {
    if (active) return;
    
    const template = MOCK_REQUESTS.find(r => r.type === type);
    if (template) {
      dispatch(setActiveRequest({
        ...template,
        id: nanoid(),
        status: 'pending',
        createdAt: new Date().toISOString()
      }));
    }
  };

  return (
    <Layout>
      <div className="grid grid-cols-1 lg:grid-cols-12 gap-8 h-full">
        {/* Left Column: Active Request */}
        <div className="lg:col-span-8 flex flex-col">
          <div className="flex-1 flex flex-col justify-center">
            <WidgetRenderer />
          </div>
          
          {/* Debug Controls */}
          <div className="mt-8 p-4 border border-border bg-black/20">
            <div className="text-xs font-mono text-muted-foreground mb-4 uppercase tracking-wider">
              DEV_CONTROLS // SIMULATE_INCOMING_REQUESTS
            </div>
            <div className="flex flex-wrap gap-2">
              <Button 
                variant="outline" 
                size="sm" 
                className="cyber-button text-xs"
                onClick={() => simulateNewRequest('confirm')}
                disabled={!!active}
              >
                CONFIRM_REQ
              </Button>
              <Button 
                variant="outline" 
                size="sm" 
                className="cyber-button text-xs"
                onClick={() => simulateNewRequest('select')}
                disabled={!!active}
              >
                SELECT_REQ
              </Button>
              <Button 
                variant="outline" 
                size="sm" 
                className="cyber-button text-xs"
                onClick={() => simulateNewRequest('table')}
                disabled={!!active}
              >
                TABLE_REQ
              </Button>
              <Button 
                variant="outline" 
                size="sm" 
                className="cyber-button text-xs"
                onClick={() => {
                  if (active) return;
                  dispatch(setActiveRequest({
                    id: nanoid(),
                    type: 'form',
                    sessionId: 'mock',
                    status: 'pending',
                    createdAt: new Date().toISOString(),
                    expiresAt: new Date(Date.now() + 300000).toISOString(),
                    input: {
                      title: "CONFIGURE_DATABASE",
                      schema: {
                        properties: {
                          host: { type: "string" },
                          port: { type: "number", minimum: 1, maximum: 65535 },
                          username: { type: "string" },
                          password: { type: "string", format: "password" },
                          ssl: { type: "boolean" }
                        },
                        required: ["host", "port", "username", "password"]
                      }
                    }
                  }));
                }}
                disabled={!!active}
              >
                FORM_REQ
              </Button>
              <Button 
                variant="outline" 
                size="sm" 
                className="cyber-button text-xs"
                onClick={() => {
                  if (active) return;
                  dispatch(setActiveRequest({
                    id: nanoid(),
                    type: 'upload',
                    sessionId: 'mock',
                    status: 'pending',
                    createdAt: new Date().toISOString(),
                    expiresAt: new Date(Date.now() + 300000).toISOString(),
                    input: {
                      title: "UPLOAD_LOGS",
                      accept: [".log", ".txt"],
                      multiple: true,
                      maxSize: 5 * 1024 * 1024
                    }
                  }));
                }}
                disabled={!!active}
              >
                UPLOAD_REQ
              </Button>
            </div>
          </div>
        </div>

        {/* Right Column: History & Logs */}
        <div className="lg:col-span-4 flex flex-col gap-6">
          {/* History Panel */}
          <div className="cyber-card flex-1 flex flex-col min-h-[300px]">
            <div className="p-3 border-b border-border bg-muted/20 flex items-center justify-between">
              <span className="font-display font-bold text-sm tracking-wider">REQUEST_HISTORY</span>
              <Clock className="h-4 w-4 text-muted-foreground" />
            </div>
            <ScrollArea className="flex-1 p-0">
              {history.length === 0 ? (
                <div className="p-8 text-center text-muted-foreground text-xs font-mono">
                  NO_HISTORY_AVAILABLE
                </div>
              ) : (
                <div className="divide-y divide-border/50">
                  {history.map((req) => (
                    <div key={req.id} className="p-4 hover:bg-primary/5 transition-colors group">
                      <div className="flex items-start justify-between mb-1">
                        <span className="font-mono text-xs text-primary font-bold uppercase">
                          {req.type}
                        </span>
                        <span className="font-mono text-[10px] text-muted-foreground">
                          {new Date(req.completedAt || req.createdAt).toLocaleTimeString()}
                        </span>
                      </div>
                      <div className="text-sm font-mono mb-2 line-clamp-1">
                        {req.input.title}
                      </div>
                      <div className="flex items-center gap-2">
                        {req.status === 'completed' ? (
                          <div className="flex items-center text-[10px] text-green-500">
                            <CheckCircle className="h-3 w-3 mr-1" />
                            COMPLETED
                          </div>
                        ) : (
                          <div className="flex items-center text-[10px] text-red-500">
                            <XCircle className="h-3 w-3 mr-1" />
                            FAILED
                          </div>
                        )}
                        <span className="text-[10px] text-muted-foreground/50 font-mono ml-auto">
                          ID: {req.id.substring(0, 6)}
                        </span>
                      </div>
                    </div>
                  ))}
                </div>
              )}
            </ScrollArea>
          </div>

          {/* System Status Panel */}
          <div className="cyber-card h-[200px] flex flex-col">
            <div className="p-3 border-b border-border bg-muted/20 flex items-center justify-between">
              <span className="font-display font-bold text-sm tracking-wider">SYSTEM_LOGS</span>
              <Terminal className="h-4 w-4 text-muted-foreground" />
            </div>
            <div className="flex-1 p-4 font-mono text-[10px] text-muted-foreground overflow-hidden relative">
              <div className="absolute inset-0 p-4 overflow-auto space-y-1">
                <div className="text-primary">system_init: initializing core modules...</div>
                <div>network: websocket connection established</div>
                <div>auth: session verified [id:550e...55]</div>
                <div>renderer: widget_registry loaded (5 widgets)</div>
                <div className="text-primary">system: ready for input</div>
                {active && (
                  <div className="text-yellow-500">
                    incoming: new request [{active.type}] received
                  </div>
                )}
              </div>
            </div>
          </div>
        </div>
      </div>
    </Layout>
  );
}
