import { Toaster } from "@/components/ui/sonner";
import { TooltipProvider } from "@/components/ui/tooltip";
import NotFound from "@/pages/NotFound";
import { Route, Switch } from "wouter";
import ErrorBoundary from "./components/ErrorBoundary";
import { ThemeProvider } from "./contexts/ThemeContext";
import Home from "./pages/Home";
import { Provider } from "react-redux";
import { store } from "./store/store";
import { connectWebSocket } from "./services/websocket";
import { useEffect } from "react";
import { browserNotificationService } from "./services/notifications";

function Router() {
  useEffect(() => {
    // Request notification permission when app loads
    browserNotificationService.requestPermission().then((permission) => {
      if (permission === 'granted') {
        console.log('Browser notification permission granted');
      } else if (permission === 'denied') {
        console.warn('Browser notification permission denied');
      }
    });
    
    // Connect WebSocket
    connectWebSocket();
  }, []);

  return (
    <Switch>
      <Route path={"/"} component={Home} />
      <Route path={"/404"} component={NotFound} />
      {/* Final fallback route */}
      <Route component={NotFound} />
    </Switch>
  );
}

// NOTE: About Theme
// - First choose a default theme according to your design style (dark or light bg), than change color palette in index.css
//   to keep consistent foreground/background color across components
// - If you want to make theme switchable, pass `switchable` ThemeProvider and use `useTheme` hook

function App() {
  return (
    <ErrorBoundary>
      <Provider store={store}>
        <ThemeProvider
          defaultTheme="dark"
          // switchable
        >
          <TooltipProvider>
            <Toaster />
            <Router />
          </TooltipProvider>
        </ThemeProvider>
      </Provider>
    </ErrorBoundary>
  );
}

export default App;
