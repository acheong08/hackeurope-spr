import { useEffect, useRef, useContext } from "react";
import { SocketContext } from "../providers/SocketProvider";

interface TerminalProps {
  logs: string[];
  addLog: (log: string) => void;
}

export function Terminal({ logs, addLog }: TerminalProps) {
  const scrollRef = useRef<HTMLDivElement>(null);
  const { lastMessage } = useContext(SocketContext);

  useEffect(() => {
    if (scrollRef.current) {
      scrollRef.current.scrollTop = scrollRef.current.scrollHeight;
    }
  }, [logs]);

  const getLogColor = (log: string) => {
    if (log.includes("✓") || log.includes("SAFE") || log.includes("Promoted")) {
      return "#22c55e";
    }
    if (
      log.includes("⚠") ||
      log.includes("SUSPICIOUS") ||
      log.includes("Flagged")
    ) {
      return "#ef4444";
    }
    if (log.includes("→") || log.includes("Analyzing")) {
      return "#60a5fa";
    }
    return "#4ade80";
  };

  // Listen for log messages from WebSocket
  useEffect(() => {
    if (lastMessage && lastMessage.type === "log") {
      const payload = lastMessage.payload as { message: string; level?: string };
      const prefix = payload.level === "success" ? "✓ " : 
                      payload.level === "warning" ? "⚠ " : 
                      payload.level === "error" ? "✗ " : 
                      payload.level === "info" ? "→ " : "";
      addLog(`${prefix}${payload.message}`);
    }
  }, [lastMessage, addLog]);

  return (
    <div className="h-full flex flex-col">
      <div
        ref={scrollRef}
        className="flex-1 p-4 font-mono text-sm overflow-y-auto"
        style={{
          background: "#0a0a0a",
          color: "#4ade80",
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
              <span style={{ color: "#22c55e" }}>$</span>
              <div
                className="w-2 h-4 animate-pulse"
                style={{ background: "#22c55e" }}
              />
            </div>
          )}
        </div>
      </div>
    </div>
  );
}
