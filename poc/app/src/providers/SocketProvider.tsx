import { createContext, useState, useEffect } from 'react';
import io, { Socket } from "socket.io-client";

export const SocketContext = createContext<Socket | undefined>(undefined);

function SocketProvider({ children }: { children?: React.ReactNode }) {
  const [socket, setSocket] = useState<Socket>();
    
  useEffect(() => {
    if (socket) {
      return;
    }

    const ws = io(`http://localhost:8000`, { withCredentials: true });
    ws.on("connect", () => {
      setSocket(ws);
    });

    return () => {
      ws?.disconnect();
    }
  }, [socket]);

  return (
    <SocketContext.Provider value={socket}>
      {children}
    </SocketContext.Provider>
  );
}
export default SocketProvider;