import { RotateCcw, Wifi, WifiOff } from "lucide-react";
import { FileUpload } from "./FileUpload";

interface HeaderProps {
  startAnalysis: () => void;
  uploadedFile: File | null;
  onFileUpload: (file: File) => void;
  onFileRemove: () => void;
  isConnected: boolean;
}

function Header({
  startAnalysis,
  uploadedFile,
  onFileUpload,
  onFileRemove,
  isConnected,
}: HeaderProps) {
  return (
    <header
      className="border-b px-4 md:px-6 py-4 flex flex-col md:flex-row justify-between items-start md:items-center gap-3"
      style={{
        borderColor: "#374151",
        background: "#111827",
      }}
    >
      <div className="flex items-center gap-3 flex-wrap">
        <div className="flex items-center gap-2">
          <div
            className="w-8 h-8 rounded flex items-center justify-center"
            style={{ background: "#22c55e" }}
          >
            <span className="text-lg" style={{ color: "#000" }}>
              S
            </span>
          </div>
          <h1 className="text-xl md:text-2xl" style={{ color: "#22c55e" }}>
            HackEurope SPR
          </h1>
        </div>
        <span
          className="text-xs md:text-sm px-2 py-1 rounded"
          style={{
            background: "#374151",
            color: "#9ca3af",
          }}
        >
          Supply Chain Package Registry
        </span>
        <div
          className="flex items-center gap-1 text-xs px-2 py-1 rounded"
          style={{
            background: isConnected ? "#064e3b" : "#7f1d1d",
            color: isConnected ? "#22c55e" : "#ef4444",
          }}
        >
          {isConnected ? <Wifi className="w-3 h-3" /> : <WifiOff className="w-3 h-3" />}
          <span>{isConnected ? "Connected" : "Disconnected"}</span>
        </div>
      </div>

      <div className="flex gap-3 items-center self-end md:self-auto">
        <FileUpload
          uploadedFile={uploadedFile}
          onFileUpload={onFileUpload}
          onFileRemove={onFileRemove}
        />

        <button
          onClick={startAnalysis}
          className="p-2 rounded-lg transition-colors hover:bg-opacity-80 cursor-pointer"
          style={{
            background: "#374151",
            color: "#22c55e",
          }}
          aria-label="Restart Analysis"
        >
          <RotateCcw className="w-5 h-5" />
        </button>
      </div>
    </header>
  );
}

export default Header;
