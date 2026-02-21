import { useEffect, useRef, useContext } from "react";
import { SocketContext } from "../providers/SocketProvider";

interface TerminalProps {
  logs: string[];
  addLog: (log: string) => void;
}

export function Terminal({ logs, addLog }: TerminalProps) {
  const scrollRef = useRef<HTMLDivElement>(null);
  const socket = useContext(SocketContext);

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

  useEffect(() => {
    socket?.on("log", addLog);
    return () => {
      socket?.off("log", addLog);
    };
  }, [socket]);

  return (
    <div className="h-full flex flex-col">
      <div
        className="p-4 border-b flex justify-between items-center"
        style={{
          borderColor: "#374151",
          background: "#111827",
        }}
      >
        <h2 className="text-lg" style={{ color: "#22c55e" }}>
          SPR Analysis Terminal
        </h2>
      </div>
      <div
        ref={scrollRef}
        className="flex-1 p-4 font-mono text-sm overflow-y-auto"
        style={{
          background: "#000000",
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
