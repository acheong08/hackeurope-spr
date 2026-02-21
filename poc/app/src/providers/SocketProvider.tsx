import { createContext, useState, useEffect, useCallback } from "react";

export interface WebSocketMessage {
  type: string;
  payload: any;
}

interface SocketContextType {
  socket: WebSocket | null;
  isConnected: boolean;
  send: (message: any) => void;
  lastMessage: WebSocketMessage | null;
}

export const SocketContext = createContext<SocketContextType>({
  socket: null,
  isConnected: false,
  send: () => {},
  lastMessage: null,
});

function SocketProvider({ children }: { children?: React.ReactNode }) {
  const [socket, setSocket] = useState<WebSocket | null>(null);
  const [isConnected, setIsConnected] = useState(false);
  const [lastMessage, setLastMessage] = useState<WebSocketMessage | null>(null);

  useEffect(() => {
    // Connect to local server
    const ws = new WebSocket("ws://localhost:8080/ws");
    
    ws.onopen = () => {
      console.log("WebSocket connected");
      setIsConnected(true);
    };
    
    ws.onmessage = (event) => {
      try {
        const data = JSON.parse(event.data);
        setLastMessage(data);
      } catch (err) {
        console.error("Failed to parse message:", err);
      }
    };
    
    ws.onclose = () => {
      console.log("WebSocket disconnected");
      setIsConnected(false);
    };
    
    ws.onerror = (error) => {
      console.error("WebSocket error:", error);
    };
    
    setSocket(ws);
    
    return () => {
      ws.close();
    };
  }, []);

  const send = useCallback((message: any) => {
    if (socket && socket.readyState === WebSocket.OPEN) {
      socket.send(JSON.stringify(message));
    } else {
      console.warn("WebSocket not connected");
    }
  }, [socket]);

  return (
    <SocketContext.Provider value={{ socket, isConnected, send, lastMessage }}>
      {children}
    </SocketContext.Provider>
  );
}

export default SocketProvider;
