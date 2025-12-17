import React from 'react';
import { cn } from '@/lib/utils';

interface LayoutProps {
  children: React.ReactNode;
}

export const Layout: React.FC<LayoutProps> = ({ children }) => {
  return (
    <div className="min-h-screen w-full bg-background text-foreground font-mono relative overflow-hidden selection:bg-primary selection:text-primary-foreground">
      {/* Scanline effect */}
      <div className="scanline" />
      
      {/* Vignette */}
      <div className="pointer-events-none fixed inset-0 z-40 bg-[radial-gradient(circle_at_center,transparent_50%,rgba(0,0,0,0.4)_100%)]" />
      
      {/* Top Bar */}
      <header className="fixed top-0 left-0 right-0 h-12 border-b border-border bg-background/80 backdrop-blur-sm z-30 flex items-center px-4 justify-between">
        <div className="flex items-center gap-2">
          <div className="w-3 h-3 bg-primary animate-pulse" />
          <span className="font-display font-bold text-lg tracking-wider">AGENT_UI_SYSTEM</span>
        </div>
        <div className="flex items-center gap-4 text-xs text-muted-foreground">
          <div className="flex items-center gap-1">
            <span className="w-2 h-2 rounded-full bg-green-500" />
            <span>ONLINE</span>
          </div>
          <span>v1.0.0</span>
        </div>
      </header>

      {/* Main Content */}
      <main className="pt-16 pb-8 px-4 container min-h-screen flex flex-col relative z-10">
        {children}
      </main>
      
      {/* Corner Decorations */}
      <div className="fixed top-12 left-0 w-4 h-4 border-l border-t border-primary/50 z-20" />
      <div className="fixed top-12 right-0 w-4 h-4 border-r border-t border-primary/50 z-20" />
      <div className="fixed bottom-0 left-0 w-4 h-4 border-l border-b border-primary/50 z-20" />
      <div className="fixed bottom-0 right-0 w-4 h-4 border-r border-b border-primary/50 z-20" />
    </div>
  );
};
