import { createContext, useState, useEffect } from "react";
import { WebSocket } from "ws";

export const SocketContext = createContext<WebSocket | undefined>(undefined);

function SocketProvider({ children }: { children?: React.ReactNode }) {
  const [socket, setSocket] = useState<WebSocket>();

  useEffect(() => {
    if (socket) {
      return;
    }

    const ws = new WebSocket("ws://www.host.com/path");
    ws.on("open", () => {
      setSocket(ws);
    });

    return () => {
      ws?.close();
    };
  }, [socket]);

  return (
    <SocketContext.Provider value={socket}>{children}</SocketContext.Provider>
  );
}
export default SocketProvider;
