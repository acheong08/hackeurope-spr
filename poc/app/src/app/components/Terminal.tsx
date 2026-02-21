import { useEffect, useRef } from 'react';

interface TerminalProps {
  isDark: boolean;
  logs: string[];
}

export function Terminal({ isDark, logs }: TerminalProps) {
  const scrollRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    if (scrollRef.current) {
      scrollRef.current.scrollTop = scrollRef.current.scrollHeight;
    }
  }, [logs]);

  const getLogColor = (log: string) => {
    if (log.includes('✓') || log.includes('SAFE') || log.includes('Promoted')) {
      return isDark ? '#22c55e' : '#16a34a';
    }
    if (log.includes('⚠') || log.includes('SUSPICIOUS') || log.includes('Flagged')) {
      return '#ef4444';
    }
    if (log.includes('→') || log.includes('Analyzing')) {
      return isDark ? '#60a5fa' : '#3b82f6';
    }
    return isDark ? '#4ade80' : '#22c55e';
  };

  return (
    <div className="h-full flex flex-col">
      <div className="p-4 border-b flex justify-between items-center" style={{ 
        borderColor: isDark ? '#374151' : '#e5e7eb',
        background: isDark ? '#111827' : '#f9fafb'
      }}>
        <h2 className="text-lg" style={{ color: isDark ? '#22c55e' : '#16a34a' }}>
          SPR Analysis Terminal
        </h2>
        <div className="flex gap-2">
          <div className="w-3 h-3 rounded-full bg-red-500" />
          <div className="w-3 h-3 rounded-full bg-yellow-500" />
          <div className="w-3 h-3 rounded-full" style={{ background: isDark ? '#22c55e' : '#16a34a' }} />
        </div>
      </div>
      <div 
        ref={scrollRef}
        className="flex-1 p-4 font-mono text-sm overflow-y-auto"
        style={{ 
          background: isDark ? '#000000' : '#1a1a1a',
          color: isDark ? '#4ade80' : '#22c55e'
        }}
      >
        <div className="space-y-1">
          {logs.map((log, index) => (
            <div 
              key={index} 
              className="whitespace-pre-wrap break-words"
              style={{ color: getLogColor(log) }}
            >
              {log}
            </div>
          ))}
          {logs.length > 0 && (
            <div className="flex items-center gap-1 mt-2">
              <span style={{ color: isDark ? '#22c55e' : '#16a34a' }}>$</span>
              <div className="w-2 h-4 animate-pulse" style={{ background: isDark ? '#22c55e' : '#16a34a' }} />
            </div>
          )}
        </div>
      </div>
    </div>
  );
}