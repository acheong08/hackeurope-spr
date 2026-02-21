import { createRoot } from "react-dom/client";
import App from "./App.tsx";
import SocketProvider from "./providers/SocketProvider";
import "./styles/index.css";

createRoot(document.getElementById("root")!).render(
  <SocketProvider>
    <App />
  </SocketProvider>
);
