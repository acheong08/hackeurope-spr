import { Play, Loader2, Wifi, WifiOff } from "lucide-react";
import { FileUpload } from "./FileUpload";

interface HeaderProps {
  startAnalysis: () => void;
  uploadedFile: File | null;
  onFileUpload: (file: File) => void;
  onFileRemove: () => void;
  isConnected: boolean;
  isAnalyzing: boolean;
}

function Header({
  startAnalysis,
  uploadedFile,
  onFileUpload,
  onFileRemove,
  isConnected,
  isAnalyzing,
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
          disabled={isAnalyzing}
        />

        {uploadedFile && (
          <button
            onClick={startAnalysis}
            disabled={isAnalyzing}
            className="flex items-center gap-2 px-4 py-2 rounded-lg transition-colors hover:bg-opacity-80 cursor-pointer font-medium disabled:opacity-50 disabled:cursor-not-allowed"
            style={{
              background: isAnalyzing ? "#374151" : "#22c55e",
              color: isAnalyzing ? "#9ca3af" : "#000",
            }}
          >
            {isAnalyzing ? (
              <>
                <Loader2 className="w-4 h-4 animate-spin" />
                <span>Analyzing...</span>
              </>
            ) : (
              <>
                <Play className="w-4 h-4" />
                <span>Start Analysis</span>
              </>
            )}
          </button>
        )}
      </div>
    </header>
  );
}

export default Header;
