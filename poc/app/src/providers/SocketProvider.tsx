import { createContext, useState, useEffect, useCallback, useRef } from "react";

export interface WebSocketMessage {
  type: string;
  payload: any;
  _id?: number; // Internal ID for deduplication (incrementing counter)
}

interface SocketContextType {
  socket: WebSocket | null;
  isConnected: boolean;
  send: (message: any) => void;
  subscribe: (callback: (message: WebSocketMessage) => void) => () => void;
}

export const SocketContext = createContext<SocketContextType>({
  socket: null,
  isConnected: false,
  send: () => {},
  subscribe: () => () => {},
});

function SocketProvider({ children }: { children?: React.ReactNode }) {
  const [socket, setSocket] = useState<WebSocket | null>(null);
  const [isConnected, setIsConnected] = useState(false);
  const messageIdRef = useRef(0);
  const subscribersRef = useRef<Set<(message: WebSocketMessage) => void>>(new Set());

  useEffect(() => {
    // Connect to local server
    const ws = new WebSocket("ws://localhost:8080/ws");
    
    ws.onopen = () => {
      console.log("WebSocket connected");
      setIsConnected(true);
    };
    
    ws.onmessage = (event) => {
      try {
        const data = JSON.parse(event.data) as WebSocketMessage;
        // Add unique incrementing ID to each message for deduplication
        data._id = ++messageIdRef.current;
        
        // Notify all subscribers immediately
        subscribersRef.current.forEach((callback) => {
          try {
            callback(data);
          } catch (err) {
            console.error("Subscriber error:", err);
          }
        });
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

  const subscribe = useCallback((callback: (message: WebSocketMessage) => void) => {
    subscribersRef.current.add(callback);
    // Return unsubscribe function
    return () => {
      subscribersRef.current.delete(callback);
    };
  }, []);

  return (
    <SocketContext.Provider value={{ socket, isConnected, send, subscribe }}>
      {children}
    </SocketContext.Provider>
  );
}

export default SocketProvider;
