import { createContext, useState, useEffect } from 'react';
import io, { Socket } from "socket.io-client";

export const UserContext = createContext<Socket | undefined>(undefined);

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
    <UserContext.Provider value={socket}>
      {children}
    </UserContext.Provider>
  );
}
export default SocketProvider;